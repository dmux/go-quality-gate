package history

import (
	"testing"
	"time"
)

func TestDailySummaries_ZeroFillsMissingDays(t *testing.T) {
	now := time.Date(2026, 9, 13, 15, 0, 0, 0, time.UTC)
	records := []Record{
		{Type: "run", HookType: "pre-commit", Success: true, Timestamp: now},
		{Type: "run", HookType: "pre-commit", Success: false, Timestamp: now},
		{Type: "fix", HookType: "pre-commit", FixedCount: 1, Timestamp: now}, // ignored
	}

	daily := DailySummaries(records, 3, now)

	if len(daily) != 3 {
		t.Fatalf("expected 3 days, got %d", len(daily))
	}
	// Oldest first: two zero-filled days, then today with 2 runs/1 failure.
	if daily[0].Total != 0 || daily[1].Total != 0 {
		t.Errorf("expected the first 2 days zero-filled, got %+v", daily[:2])
	}
	today := daily[2]
	if today.Total != 2 || today.Failed != 1 {
		t.Fatalf("expected today's total=2 failed=1, got %+v", today)
	}
	if today.SuccessRate != 50 {
		t.Errorf("expected 50%% success rate, got %v", today.SuccessRate)
	}
	if today.Date != "2026-09-13" {
		t.Errorf("expected date 2026-09-13, got %q", today.Date)
	}
}

func TestDailySummaries_Empty(t *testing.T) {
	now := time.Date(2026, 9, 13, 15, 0, 0, 0, time.UTC)
	daily := DailySummaries(nil, 2, now)

	if len(daily) != 2 {
		t.Fatalf("expected 2 zero-filled days, got %d", len(daily))
	}
	for _, d := range daily {
		if d.Total != 0 || d.SuccessRate != 0 {
			t.Errorf("expected a zero-filled day, got %+v", d)
		}
	}
}

func TestDaysBetween_NegativeClampsToZero(t *testing.T) {
	from := time.Date(2026, 9, 13, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 9, 10, 0, 0, 0, 0, time.UTC) // to is before from

	if got := daysBetween(from, to); got != 0 {
		t.Errorf("expected daysBetween to clamp negative results to 0, got %d", got)
	}
}

func TestComputeStats_NeverFailed_SameDay(t *testing.T) {
	now := time.Date(2026, 9, 13, 15, 0, 0, 0, time.UTC)
	records := []Record{
		{Type: "run", HookType: "pre-commit", Success: true, Timestamp: now.Add(-2 * time.Hour)},
		{Type: "run", HookType: "pre-commit", Success: true, Timestamp: now.Add(-1 * time.Hour)},
	}

	stats := ComputeStats(records, "pre-commit", now)

	if stats.StreakDays != 0 {
		t.Errorf("expected StreakDays=0 (same day), got %d", stats.StreakDays)
	}
	if stats.StreakBrokenToday {
		t.Error("expected StreakBrokenToday=false when there was never a failure")
	}
}

func TestComputeStats_FailedToday(t *testing.T) {
	now := time.Date(2026, 9, 13, 15, 0, 0, 0, time.UTC)
	records := []Record{
		{Type: "run", HookType: "pre-commit", Success: true, Timestamp: now.Add(-48 * time.Hour)},
		{Type: "run", HookType: "pre-commit", Success: false, Timestamp: now.Add(-1 * time.Hour)},
	}

	stats := ComputeStats(records, "pre-commit", now)

	if stats.StreakDays != 0 {
		t.Errorf("expected StreakDays=0 right after today's failure, got %d", stats.StreakDays)
	}
	if !stats.StreakBrokenToday {
		t.Error("expected StreakBrokenToday=true when the most recent failure was today")
	}
}

func TestComputeStats_FailedThreeDaysAgo(t *testing.T) {
	now := time.Date(2026, 9, 13, 15, 0, 0, 0, time.UTC)
	records := []Record{
		{Type: "run", HookType: "pre-commit", Success: false, Timestamp: now.AddDate(0, 0, -3)},
		{Type: "run", HookType: "pre-commit", Success: true, Timestamp: now.AddDate(0, 0, -2)},
		{Type: "run", HookType: "pre-commit", Success: true, Timestamp: now},
	}

	stats := ComputeStats(records, "pre-commit", now)

	if stats.StreakDays != 3 {
		t.Errorf("expected StreakDays=3, got %d", stats.StreakDays)
	}
	if stats.StreakBrokenToday {
		t.Error("expected StreakBrokenToday=false when the failure was 3 days ago")
	}
}

func TestComputeStats_IgnoresOtherHookTypesForStreak(t *testing.T) {
	now := time.Date(2026, 9, 13, 15, 0, 0, 0, time.UTC)
	records := []Record{
		{Type: "run", HookType: "pre-push", Success: false, Timestamp: now.Add(-1 * time.Hour)},
		{Type: "run", HookType: "pre-commit", Success: true, Timestamp: now.AddDate(0, 0, -5)},
	}

	stats := ComputeStats(records, "pre-commit", now)

	if stats.StreakBrokenToday {
		t.Error("a pre-push failure must not affect the pre-commit streak")
	}
	if stats.StreakDays != 5 {
		t.Errorf("expected StreakDays=5 (pre-push failures ignored), got %d", stats.StreakDays)
	}
	// But it still counts toward TotalRuns/SuccessRate.
	if stats.TotalRuns != 2 {
		t.Errorf("expected TotalRuns=2, got %d", stats.TotalRuns)
	}
}

func TestComputeStats_FixesOnlyCountedThisMonth(t *testing.T) {
	now := time.Date(2026, 9, 13, 12, 0, 0, 0, time.UTC)
	records := []Record{
		{Type: "fix", HookType: "pre-commit", FixedCount: 10, Timestamp: now.AddDate(0, -1, 0)}, // last month
		{Type: "fix", HookType: "pre-commit", FixedCount: 4, Timestamp: now},                    // this month
	}

	stats := ComputeStats(records, "pre-commit", now)

	if stats.TotalAutoFixes != 14 {
		t.Errorf("expected TotalAutoFixes=14 (all-time), got %d", stats.TotalAutoFixes)
	}
	if stats.FixesThisMonth != 4 {
		t.Errorf("expected FixesThisMonth=4, got %d", stats.FixesThisMonth)
	}
	wantSaved := 4 * assumedManualFixDuration
	if stats.TimeSavedThisMonth != wantSaved {
		t.Errorf("expected TimeSavedThisMonth=%v, got %v", wantSaved, stats.TimeSavedThisMonth)
	}
}

func TestAchievements_UnlockedSortedFirst(t *testing.T) {
	stats := Stats{TotalRuns: 1, TotalAutoFixes: 60, StreakDays: 0, StreakHookType: "pre-commit"}

	achievements := Achievements(stats)

	if !achievements[0].Unlocked {
		t.Fatalf("expected the first achievement to be unlocked, got %+v", achievements[0])
	}

	sawLocked := false
	for _, a := range achievements {
		if !a.Unlocked {
			sawLocked = true
		}
		if a.Unlocked && sawLocked {
			t.Errorf("found an unlocked achievement (%s) after a locked one — unlocked must sort first", a.Name)
		}
	}

	byID := map[string]Achievement{}
	for _, a := range achievements {
		byID[a.ID] = a
	}
	if !byID["first_steps"].Unlocked {
		t.Error("expected first_steps to be unlocked with TotalRuns=1")
	}
	if !byID["clean_coder"].Unlocked {
		t.Error("expected clean_coder to be unlocked with TotalAutoFixes=60")
	}
	if byID["centurion"].Unlocked {
		t.Error("did not expect centurion to be unlocked with TotalRuns=1")
	}
}

func TestStore_AppendAndLoad(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)

	if records, err := store.Load(); err != nil || records != nil {
		t.Fatalf("expected no error and nil records for a missing log, got %v, %v", records, err)
	}

	rec1 := Record{Timestamp: time.Now(), Type: "run", HookType: "pre-commit", Success: true}
	rec2 := Record{Timestamp: time.Now(), Type: "fix", HookType: "pre-commit", FixedCount: 2}

	if err := store.Append(rec1); err != nil {
		t.Fatalf("Append failed: %v", err)
	}
	if err := store.Append(rec2); err != nil {
		t.Fatalf("Append failed: %v", err)
	}

	records, err := store.Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("expected 2 records, got %d", len(records))
	}
	if records[0].Type != "run" || records[1].Type != "fix" {
		t.Errorf("expected records in append order, got %+v", records)
	}
}

func TestHookLeaderboard(t *testing.T) {
	records := []Record{
		{
			Type: "run", HookType: "pre-commit",
			Hooks: []HookOutcome{
				{Name: "Gitleaks", Success: true, DurationMs: 100},
				{Name: "Pytest", Success: true, DurationMs: 4000},
			},
		},
		{
			Type: "run", HookType: "pre-commit",
			Hooks: []HookOutcome{
				{Name: "Gitleaks", Success: true, DurationMs: 200},
				{Name: "Pytest", Success: false, DurationMs: 6000},
			},
		},
		{
			// Non-"run" records must not pollute per-hook stats.
			Type: "fix", HookType: "pre-commit", FixedCount: 1,
		},
	}

	stats := HookLeaderboard(records)
	if len(stats) != 2 {
		t.Fatalf("expected 2 hooks, got %d: %+v", len(stats), stats)
	}

	byName := map[string]HookStat{}
	for _, s := range stats {
		byName[s.Name] = s
	}

	gitleaks := byName["Gitleaks"]
	if gitleaks.TotalRuns != 2 || gitleaks.TotalFailures != 0 {
		t.Errorf("Gitleaks: expected 2 runs/0 failures, got %+v", gitleaks)
	}
	if gitleaks.FailureRate != 0 {
		t.Errorf("Gitleaks: expected 0%% failure rate, got %v", gitleaks.FailureRate)
	}
	if gitleaks.AvgDurationMs != 150 {
		t.Errorf("Gitleaks: expected avg duration 150ms, got %v", gitleaks.AvgDurationMs)
	}

	pytest := byName["Pytest"]
	if pytest.TotalRuns != 2 || pytest.TotalFailures != 1 {
		t.Errorf("Pytest: expected 2 runs/1 failure, got %+v", pytest)
	}
	if pytest.FailureRate != 50 {
		t.Errorf("Pytest: expected 50%% failure rate, got %v", pytest.FailureRate)
	}
	if pytest.AvgDurationMs != 5000 {
		t.Errorf("Pytest: expected avg duration 5000ms, got %v", pytest.AvgDurationMs)
	}
}

func TestHookLeaderboard_Empty(t *testing.T) {
	stats := HookLeaderboard(nil)
	if len(stats) != 0 {
		t.Errorf("expected an empty leaderboard for no records, got %+v", stats)
	}
}
