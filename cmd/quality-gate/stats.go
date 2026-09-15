package main

import (
	"fmt"
	"time"

	"github.com/dmux/go-quality-gate/internal/config"
	"github.com/dmux/go-quality-gate/internal/domain"
	"github.com/dmux/go-quality-gate/internal/infra/history"
)

// streakHookType is the hook type the "stats" command's streak counter
// tracks, matching the flow developers care about most day to day.
const streakHookType = "pre-commit"

// openHistoryStore locates the current repository's history log. It
// returns ok=false when no .git directory can be found, which callers
// treat as "no history available yet" rather than a hard error — history
// tracking is a best-effort, local-only feature.
func openHistoryStore() (*history.Store, bool) {
	gitDir, err := history.FindGitDir()
	if err != nil {
		return nil, false
	}
	return history.NewStore(gitDir), true
}

// recordRunHistory appends a best-effort "run" entry to the history log.
// Failures are ignored: history tracking must never affect the exit code
// or block the developer's actual git hook.
func recordRunHistory(hookType string, results []domain.ExecutionResult, success bool, duration time.Duration) {
	store, ok := openHistoryStore()
	if !ok {
		return
	}

	hooks := make([]history.HookOutcome, 0, len(results))
	for _, r := range results {
		hooks = append(hooks, history.HookOutcome{
			Name:       r.Hook.Name,
			Success:    r.Success,
			DurationMs: r.Duration.Milliseconds(),
		})
	}

	_ = store.Append(history.Record{
		Timestamp:  time.Now(),
		Type:       "run",
		HookType:   hookType,
		Success:    success,
		DurationMs: duration.Milliseconds(),
		Hooks:      hooks,
	})
}

// recordFixHistory appends a best-effort "fix" entry counting how many
// hooks configured with a fix_command for hookType were addressed by this
// --fix invocation. Fix() stops at the first failing fix command and
// returns before this is called, so reaching here means every fixable
// hook for hookType succeeded.
func recordFixHistory(cfg *config.Config, hookType string) {
	fixedCount := 0
	for _, group := range cfg.Hooks {
		for _, h := range group[hookType] {
			if h.FixCommand != "" {
				fixedCount++
			}
		}
	}
	if fixedCount == 0 {
		return
	}

	store, ok := openHistoryStore()
	if !ok {
		return
	}
	_ = store.Append(history.Record{
		Timestamp:  time.Now(),
		Type:       "fix",
		HookType:   hookType,
		FixedCount: fixedCount,
	})
}

// runStatsCommand implements `quality-gate stats`: a terminal summary of
// streaks, achievements, and time saved, computed from the local history
// log. It intentionally works without a quality.yml, since it only reads
// past execution history.
func runStatsCommand(logPrintln func(string)) {
	store, ok := openHistoryStore()
	if !ok {
		logPrintln("Not inside a git repository — no stats to show yet.")
		return
	}

	records, err := store.Load()
	if err != nil {
		logPrintln(fmt.Sprintf("Error reading history: %v", err))
		return
	}
	if len(records) == 0 {
		logPrintln("📊 No history yet — run the quality gate a few times to start tracking stats!")
		return
	}

	stats := history.ComputeStats(records, streakHookType, time.Now())
	achievements := history.Achievements(stats)

	logPrintln("📊 Quality Gate Stats")
	logPrintln("")

	switch {
	case stats.StreakBrokenToday:
		logPrintln(fmt.Sprintf("🔥 Streak: 0 days — the latest %s build failed. Let's get it back!", stats.StreakHookType))
	case stats.StreakDays > 0:
		logPrintln(fmt.Sprintf("🔥 Streak: %d days without breaking the %s build!", stats.StreakDays, stats.StreakHookType))
	default:
		logPrintln(fmt.Sprintf("🔥 Streak: starting today — no failures recorded for %s yet!", stats.StreakHookType))
	}
	logPrintln("")

	logPrintln("🏆 Achievements:")
	for _, a := range achievements {
		if a.Unlocked {
			logPrintln(fmt.Sprintf("   ✅ %s — %s", a.Name, a.Description))
		} else {
			logPrintln(fmt.Sprintf("   🔒 %s — %s (%d/%d)", a.Name, a.Description, a.Progress, a.Target))
		}
	}
	logPrintln("")

	logPrintln(fmt.Sprintf(
		"⚡ Time saved with auto-fixes this month: %.1f hours (%d fixes)",
		stats.TimeSavedThisMonth.Hours(), stats.FixesThisMonth,
	))
	logPrintln("")

	logPrintln(fmt.Sprintf("📈 Total runs: %d | Success rate: %.0f%%", stats.TotalRuns, stats.SuccessRate))
}
