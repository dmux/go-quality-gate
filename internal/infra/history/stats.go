package history

import (
	"sort"
	"time"
)

// assumedManualFixDuration is how long we assume a developer would have
// spent fixing an issue by hand, used only to produce a rough "time saved"
// estimate — not a measured value.
const assumedManualFixDuration = 5 * time.Minute

// Stats is a set of aggregate metrics derived from the history log.
type Stats struct {
	TotalRuns       int
	TotalFailedRuns int
	SuccessRate     float64 // 0-100

	StreakDays     int
	StreakHookType string
	// StreakBrokenToday is true when the most recent streakHookType run
	// failed and that failure happened today — the reason StreakDays can
	// be 0 even though history exists (as opposed to 0 just because
	// tracking only started today with no failures at all).
	StreakBrokenToday  bool
	TotalAutoFixes     int
	FixesThisMonth     int
	TimeSavedThisMonth time.Duration
}

// Achievement is a single unlockable badge.
type Achievement struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Unlocked    bool   `json:"unlocked"`
	// Progress and Target describe how close an unlocked/locked achievement
	// is (e.g. Progress=15, Target=30 for a 30-day streak badge).
	Progress int `json:"progress"`
	Target   int `json:"target"`
}

// DailySummary aggregates one calendar day's "run" records, used to chart
// quality evolution over time.
type DailySummary struct {
	Date        string  `json:"date"` // YYYY-MM-DD
	Total       int     `json:"total"`
	Failed      int     `json:"failed"`
	SuccessRate float64 `json:"success_rate"` // 0-100
}

// DailySummaries returns one entry per day for the trailing `days` days
// (oldest first, ending at now's calendar day), zero-filled for days with
// no recorded runs so charts get a continuous, evenly-spaced X axis.
func DailySummaries(records []Record, days int, now time.Time) []DailySummary {
	byDay := make(map[string]*DailySummary, days)
	for _, r := range records {
		if r.Type != "run" {
			continue
		}
		key := r.Timestamp.Format("2006-01-02")
		d, ok := byDay[key]
		if !ok {
			d = &DailySummary{Date: key}
			byDay[key] = d
		}
		d.Total++
		if !r.Success {
			d.Failed++
		}
	}

	result := make([]DailySummary, 0, days)
	for i := days - 1; i >= 0; i-- {
		day := now.AddDate(0, 0, -i)
		key := day.Format("2006-01-02")
		if d, ok := byDay[key]; ok {
			if d.Total > 0 {
				d.SuccessRate = 100 * float64(d.Total-d.Failed) / float64(d.Total)
			}
			result = append(result, *d)
		} else {
			result = append(result, DailySummary{Date: key})
		}
	}
	return result
}

// ComputeStats aggregates the raw history log into the metrics the "stats"
// command displays. streakHookType selects which hook type the streak is
// measured against (the spec's example tracks "pre-commit").
func ComputeStats(records []Record, streakHookType string, now time.Time) Stats {
	stats := Stats{StreakHookType: streakHookType}

	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())

	var lastFailure, firstStreakRun time.Time
	for _, r := range records {
		switch r.Type {
		case "run":
			stats.TotalRuns++
			if !r.Success {
				stats.TotalFailedRuns++
			}
			if r.HookType == streakHookType {
				if firstStreakRun.IsZero() || r.Timestamp.Before(firstStreakRun) {
					firstStreakRun = r.Timestamp
				}
				if !r.Success && r.Timestamp.After(lastFailure) {
					lastFailure = r.Timestamp
				}
			}
		case "fix":
			stats.TotalAutoFixes += r.FixedCount
			if !r.Timestamp.Before(monthStart) {
				stats.FixesThisMonth += r.FixedCount
			}
		}
	}

	if stats.TotalRuns > 0 {
		stats.SuccessRate = 100 * float64(stats.TotalRuns-stats.TotalFailedRuns) / float64(stats.TotalRuns)
	}
	stats.TimeSavedThisMonth = time.Duration(stats.FixesThisMonth) * assumedManualFixDuration

	if !firstStreakRun.IsZero() {
		since := firstStreakRun
		if !lastFailure.IsZero() {
			since = lastFailure
			stats.StreakBrokenToday = daysBetween(lastFailure, now) == 0
		}
		stats.StreakDays = daysBetween(since, now)
	}

	return stats
}

// daysBetween returns the number of full calendar days between from and
// to (truncated to local midnight), so a failure "today" yields a streak
// of 0 and a failure "yesterday" yields a streak of 1.
func daysBetween(from, to time.Time) int {
	fromDay := time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, from.Location())
	toDay := time.Date(to.Year(), to.Month(), to.Day(), 0, 0, 0, 0, to.Location())
	days := int(toDay.Sub(fromDay).Hours() / 24)
	if days < 0 {
		return 0
	}
	return days
}

// Achievements evaluates the fixed rule set against the given stats and
// returns every achievement (locked and unlocked), most-relevant first.
func Achievements(stats Stats) []Achievement {
	achievements := []Achievement{
		{
			ID: "first_steps", Name: "First Steps",
			Description: "Ran quality-gate for the first time",
			Progress:    min(stats.TotalRuns, 1), Target: 1,
		},
		{
			ID: "clean_coder", Name: "Clean Coder",
			Description: "Auto-fixed 50 issues with --fix",
			Progress:    min(stats.TotalAutoFixes, 50), Target: 50,
		},
		{
			ID: "week_streak", Name: "Perfectionist",
			Description: "7 days in a row without breaking " + stats.StreakHookType,
			Progress:    min(stats.StreakDays, 7), Target: 7,
		},
		{
			ID: "month_streak", Name: "Marathon Runner",
			Description: "30 days in a row without breaking " + stats.StreakHookType,
			Progress:    min(stats.StreakDays, 30), Target: 30,
		},
		{
			ID: "centurion", Name: "Centurion",
			Description: "100 quality gate runs",
			Progress:    min(stats.TotalRuns, 100), Target: 100,
		},
	}

	for i := range achievements {
		achievements[i].Unlocked = achievements[i].Progress >= achievements[i].Target
	}

	sort.SliceStable(achievements, func(i, j int) bool {
		if achievements[i].Unlocked != achievements[j].Unlocked {
			return achievements[i].Unlocked // unlocked first
		}
		return false
	})

	return achievements
}

// HookStat aggregates one hook's outcomes across every recorded run.
type HookStat struct {
	Name          string  `json:"name"`
	TotalRuns     int     `json:"total_runs"`
	TotalFailures int     `json:"total_failures"`
	FailureRate   float64 `json:"failure_rate"` // 0-100
	AvgDurationMs float64 `json:"avg_duration_ms"`
}

// HookLeaderboard aggregates per-hook stats (run count, failure rate,
// average duration) across every "run" record's per-hook outcomes, keyed
// by hook name. Order is unspecified — callers sort by whichever metric
// they're ranking (duration for "slowest", failure rate for "flakiest").
func HookLeaderboard(records []Record) []HookStat {
	type acc struct {
		runs, failures  int
		totalDurationMs int64
	}
	byName := make(map[string]*acc)

	for _, r := range records {
		if r.Type != "run" {
			continue
		}
		for _, h := range r.Hooks {
			a, ok := byName[h.Name]
			if !ok {
				a = &acc{}
				byName[h.Name] = a
			}
			a.runs++
			if !h.Success {
				a.failures++
			}
			a.totalDurationMs += h.DurationMs
		}
	}

	stats := make([]HookStat, 0, len(byName))
	for name, a := range byName {
		s := HookStat{Name: name, TotalRuns: a.runs, TotalFailures: a.failures}
		if a.runs > 0 {
			s.FailureRate = 100 * float64(a.failures) / float64(a.runs)
			s.AvgDurationMs = float64(a.totalDurationMs) / float64(a.runs)
		}
		stats = append(stats, s)
	}

	// Deterministic base order (map iteration order is random); callers
	// re-sort by their metric of interest, but ties should still be stable.
	sort.Slice(stats, func(i, j int) bool { return stats[i].Name < stats[j].Name })

	return stats
}
