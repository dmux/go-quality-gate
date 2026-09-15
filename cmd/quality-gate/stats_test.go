package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/dmux/go-quality-gate/internal/config"
	"github.com/dmux/go-quality-gate/internal/domain"
	"github.com/dmux/go-quality-gate/internal/infra/history"
)

func TestOpenHistoryStore_NotAGitRepo(t *testing.T) {
	t.Chdir(t.TempDir())

	_, ok := openHistoryStore()

	if ok {
		t.Error("expected ok=false outside a git repository")
	}
}

func TestRecordRunHistory_NotAGitRepoIsNoop(t *testing.T) {
	t.Chdir(t.TempDir())

	// Must not panic; there's nowhere to write, so it's silently skipped.
	recordRunHistory("pre-commit", nil, true, time.Millisecond)
}

func TestRecordRunHistory_WritesRecord(t *testing.T) {
	initGitRepo(t)

	results := []domain.ExecutionResult{
		{Hook: domain.Hook{Name: "Say hi"}, Success: true, Duration: 5 * time.Millisecond},
	}
	recordRunHistory("pre-commit", results, true, 10*time.Millisecond)

	data, err := os.ReadFile(".git/quality-gate/history.jsonl")
	if err != nil {
		t.Fatalf("expected a history record: %v", err)
	}
	if !strings.Contains(string(data), `"type":"run"`) || !strings.Contains(string(data), "Say hi") {
		t.Errorf("unexpected history content: %s", data)
	}
}

func TestRecordFixHistory_NotAGitRepoIsNoop(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	cfg := &config.Config{Hooks: config.Hooks{
		"security": {"pre-commit": []config.Hook{{Name: "x", Command: "echo hi", FixCommand: "echo fixed"}}},
	}}

	// Must not panic even without a .git directory.
	recordFixHistory(cfg, "pre-commit")
}

func TestRecordFixHistory_NoFixableHooksIsNoop(t *testing.T) {
	initGitRepo(t)

	cfg := &config.Config{Hooks: config.Hooks{
		"security": {"pre-commit": []config.Hook{{Name: "x", Command: "echo hi"}}}, // no FixCommand
	}}

	recordFixHistory(cfg, "pre-commit")

	if _, err := os.Stat(".git/quality-gate/history.jsonl"); !os.IsNotExist(err) {
		t.Errorf("expected no history file to be written, got err=%v", err)
	}
}

func TestRecordFixHistory_WritesRecord(t *testing.T) {
	initGitRepo(t)

	cfg := &config.Config{Hooks: config.Hooks{
		"security": {"pre-commit": []config.Hook{{Name: "x", Command: "echo hi", FixCommand: "echo fixed"}}},
	}}

	recordFixHistory(cfg, "pre-commit")

	data, err := os.ReadFile(".git/quality-gate/history.jsonl")
	if err != nil {
		t.Fatalf("expected a history record: %v", err)
	}
	if !strings.Contains(string(data), `"type":"fix"`) || !strings.Contains(string(data), `"fixed_count":1`) {
		t.Errorf("unexpected history content: %s", data)
	}
}

func writeHistoryRecords(t *testing.T, records ...history.Record) {
	t.Helper()
	gitDir, err := history.FindGitDir()
	if err != nil {
		t.Fatal(err)
	}
	store := history.NewStore(gitDir)
	for _, r := range records {
		if err := store.Append(r); err != nil {
			t.Fatal(err)
		}
	}
}

func TestRunStatsCommand_NotAGitRepo(t *testing.T) {
	t.Chdir(t.TempDir())

	var out []string
	runStatsCommand(func(s string) { out = append(out, s) })

	joined := strings.Join(out, "\n")
	if !strings.Contains(joined, "Not inside a git repository") {
		t.Errorf("expected the not-a-git-repo message, got %q", joined)
	}
}

func TestRunStatsCommand_LoadError(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root: permission checks don't apply")
	}
	initGitRepo(t)

	logPath := filepath.Join(".git", "quality-gate", "history.jsonl")
	if err := os.MkdirAll(filepath.Dir(logPath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(logPath, []byte("x"), 0000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(logPath, 0644) })

	var out []string
	runStatsCommand(func(s string) { out = append(out, s) })

	joined := strings.Join(out, "\n")
	if !strings.Contains(joined, "Error reading history") {
		t.Errorf("expected a read-error message, got %q", joined)
	}
}

func TestRunStatsCommand_NoHistory(t *testing.T) {
	initGitRepo(t)

	var out []string
	runStatsCommand(func(s string) { out = append(out, s) })

	joined := strings.Join(out, "\n")
	if !strings.Contains(joined, "No history yet") {
		t.Errorf("expected the no-history message, got %q", joined)
	}
}

func TestRunStatsCommand_StreakBrokenToday(t *testing.T) {
	initGitRepo(t)
	writeHistoryRecords(t, history.Record{
		Timestamp: time.Now(), Type: "run", HookType: "pre-commit", Success: false,
	})

	var out []string
	runStatsCommand(func(s string) { out = append(out, s) })

	joined := strings.Join(out, "\n")
	if !strings.Contains(joined, "Streak: 0 days") || !strings.Contains(joined, "failed") {
		t.Errorf("expected the streak-broken-today message, got %q", joined)
	}
}

func TestRunStatsCommand_ActiveStreak(t *testing.T) {
	initGitRepo(t)
	writeHistoryRecords(t, history.Record{
		Timestamp: time.Now().AddDate(0, 0, -5), Type: "run", HookType: "pre-commit", Success: true,
	})

	var out []string
	runStatsCommand(func(s string) { out = append(out, s) })

	joined := strings.Join(out, "\n")
	if !strings.Contains(joined, "5 days without breaking") {
		t.Errorf("expected an active-streak message, got %q", joined)
	}
}

func TestRunStatsCommand_StartingToday(t *testing.T) {
	initGitRepo(t)
	writeHistoryRecords(t, history.Record{
		Timestamp: time.Now(), Type: "run", HookType: "pre-commit", Success: true,
	})

	var out []string
	runStatsCommand(func(s string) { out = append(out, s) })

	joined := strings.Join(out, "\n")
	if !strings.Contains(joined, "starting today") {
		t.Errorf("expected the starting-today message, got %q", joined)
	}
}

func TestRunStatsCommand_UnlockedAndLockedAchievements(t *testing.T) {
	initGitRepo(t)
	// A single run unlocks "First Steps" (target 1) but leaves every other
	// achievement locked, exercising both branches of the achievements loop.
	writeHistoryRecords(t, history.Record{
		Timestamp: time.Now(), Type: "run", HookType: "pre-commit", Success: true,
	})

	var out []string
	runStatsCommand(func(s string) { out = append(out, s) })

	joined := strings.Join(out, "\n")
	if !strings.Contains(joined, "✅") {
		t.Errorf("expected at least one unlocked achievement, got %q", joined)
	}
	if !strings.Contains(joined, "🔒") {
		t.Errorf("expected at least one locked achievement, got %q", joined)
	}
	if !strings.Contains(joined, "Total runs: 1") {
		t.Errorf("expected the totals line, got %q", joined)
	}
}
