// Package webui serves the local quality-gate web dashboard: a Next.js
// static export embedded in the binary, backed by a small JSON API reading
// the same history log the "stats" command uses.
package webui

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"os/exec"
	"runtime"
	"time"

	"github.com/dmux/go-quality-gate/internal/infra/history"
)

// The embed directory is named "static", not "dist" — the repo's root
// .gitignore has a bare "dist/" pattern (for Go build artifacts elsewhere)
// that would silently exclude a "dist" directory here from version control,
// breaking the build for anyone else checking out the repo.
//
//go:embed all:static
var distFS embed.FS

// recentRunsLimit caps how many recent runs the API returns, newest first.
const recentRunsLimit = 50

// dailySummaryWindowDays is how many trailing days of history the daily
// evolution chart covers.
const dailySummaryWindowDays = 30

// streakHookType mirrors the CLI "stats" command's choice of which hook
// type the streak counter tracks.
const streakHookType = "pre-commit"

type apiStats struct {
	TotalRuns                 int     `json:"total_runs"`
	TotalFailedRuns           int     `json:"total_failed_runs"`
	SuccessRate               float64 `json:"success_rate"`
	StreakDays                int     `json:"streak_days"`
	StreakHookType            string  `json:"streak_hook_type"`
	StreakBrokenToday         bool    `json:"streak_broken_today"`
	TotalAutoFixes            int     `json:"total_auto_fixes"`
	FixesThisMonth            int     `json:"fixes_this_month"`
	TimeSavedThisMonthMinutes float64 `json:"time_saved_this_month_minutes"`
}

type apiRunRecord struct {
	Timestamp  time.Time             `json:"timestamp"`
	HookType   string                `json:"hook_type"`
	Success    bool                  `json:"success"`
	DurationMs int64                 `json:"duration_ms"`
	Hooks      []history.HookOutcome `json:"hooks"`
}

type apiResponse struct {
	Stats        apiStats               `json:"stats"`
	Achievements []history.Achievement  `json:"achievements"`
	Daily        []history.DailySummary `json:"daily"`
	RecentRuns   []apiRunRecord         `json:"recent_runs"`
	// HookStats is per-hook aggregate stats (run count, failure rate,
	// average duration), unsorted — the dashboard derives its "slowest" and
	// "flakiest" leaderboards from the same array by sorting client-side.
	HookStats []history.HookStat `json:"hook_stats"`
}

// newMux builds the dashboard's HTTP routes: the JSON API plus the embedded
// static assets. The "static" directory is embedded at compile time (the
// embed directive above fails the build if it's missing or empty), so
// fs.Sub against it cannot fail at runtime.
func newMux(store *history.Store) *http.ServeMux {
	distRoot, _ := fs.Sub(distFS, "static")

	mux := http.NewServeMux()
	mux.HandleFunc("/api/stats", statsHandler(store))
	mux.Handle("/", http.FileServer(http.FS(distRoot)))
	return mux
}

// serve runs the dashboard's HTTP handlers on an already-bound listener
// until it's closed (or a request-handling loop error occurs), separated
// out from Serve so tests can exercise it on a listener they control
// instead of a real, indefinitely-blocking server on a fixed port.
func serve(listener net.Listener, store *history.Store) error {
	return http.Serve(listener, newMux(store))
}

// serveOn announces the dashboard URL, opens the browser, and serves until
// listener is closed. Split out from Serve so tests can drive it with a
// listener they control (net.Listen's success path itself needs no real
// test beyond Serve's own bind-failure case).
func serveOn(listener net.Listener, store *history.Store) error {
	url := "http://" + listener.Addr().String()
	fmt.Printf("📊 Quality Gate dashboard running at %s (Ctrl+C to stop)\n", url)
	go openBrowser(url)
	return serve(listener, store)
}

// Serve starts the local dashboard server on the given port, opens the
// user's default browser to it, and blocks until the server stops (which
// in practice means until the process is killed with Ctrl+C).
func Serve(gitDir string, port int) error {
	store := history.NewStore(gitDir)

	addr := fmt.Sprintf("127.0.0.1:%d", port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to bind %s: %w", addr, err)
	}

	return serveOn(listener, store)
}

func statsHandler(store *history.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		records, err := store.Load()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		now := time.Now()
		stats := history.ComputeStats(records, streakHookType, now)
		achievements := history.Achievements(stats)
		daily := history.DailySummaries(records, dailySummaryWindowDays, now)
		runs := recentRuns(records, recentRunsLimit)
		hookStats := history.HookLeaderboard(records)

		resp := apiResponse{
			Stats: apiStats{
				TotalRuns:                 stats.TotalRuns,
				TotalFailedRuns:           stats.TotalFailedRuns,
				SuccessRate:               stats.SuccessRate,
				StreakDays:                stats.StreakDays,
				StreakHookType:            stats.StreakHookType,
				StreakBrokenToday:         stats.StreakBrokenToday,
				TotalAutoFixes:            stats.TotalAutoFixes,
				FixesThisMonth:            stats.FixesThisMonth,
				TimeSavedThisMonthMinutes: stats.TimeSavedThisMonth.Minutes(),
			},
			Achievements: achievements,
			Daily:        daily,
			RecentRuns:   runs,
			HookStats:    hookStats,
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

// recentRuns returns up to limit "run" records, most recent first.
func recentRuns(records []history.Record, limit int) []apiRunRecord {
	var runs []apiRunRecord
	for i := len(records) - 1; i >= 0 && len(runs) < limit; i-- {
		r := records[i]
		if r.Type != "run" {
			continue
		}
		runs = append(runs, apiRunRecord{
			Timestamp:  r.Timestamp,
			HookType:   r.HookType,
			Success:    r.Success,
			DurationMs: r.DurationMs,
			Hooks:      r.Hooks,
		})
	}
	if runs == nil {
		runs = []apiRunRecord{}
	}
	return runs
}

// browserCommand returns the OS-appropriate command to open url in the
// user's default browser. Split out from openBrowser so the platform
// dispatch logic is testable without actually starting a process (starting
// "open"/"xdg-open" for real in a test would pop open a real browser).
func browserCommand(url, goos string) *exec.Cmd {
	switch goos {
	case "darwin":
		return exec.Command("open", url)
	case "windows":
		return exec.Command("cmd", "/c", "start", url)
	default:
		return exec.Command("xdg-open", url)
	}
}

// startCmd is cmd.Start, swappable in tests so opening a browser doesn't
// actually spawn one.
var startCmd = func(cmd *exec.Cmd) error { return cmd.Start() }

func openBrowser(url string) {
	_ = startCmd(browserCommand(url, runtime.GOOS))
}
