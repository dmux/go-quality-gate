package service

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/dmux/go-quality-gate/internal/domain"
)

func TestHookRunnerService_RunHooks(t *testing.T) {
	mockRunner := &MockShellRunner{
		Commands: make(map[string]struct {
			Output string
			Err    error
		}),
	}

	mockLogger := &MockLogger{}
	service := NewHookRunnerService(mockRunner, mockLogger)

	hooks := []domain.Hook{
		{
			Name:    "Hook 1",
			Command: "run_hook_1",
		},
		{
			Name:    "Hook 2",
			Command: "run_hook_2",
			OutputRules: domain.OutputRules{
				ShowOn: "failure",
			},
		},
	}

	// Scenario 1: Both hooks pass
	mockRunner.Commands["run_hook_1"] = struct {
		Output string
		Err    error
	}{"success", nil}
	mockRunner.Commands["run_hook_2"] = struct {
		Output string
		Err    error
	}{"success", nil}

	results := service.RunHooks(hooks)

	if len(results) != 2 {
		t.Fatalf("Expected 2 results, but got %d", len(results))
	}

	if !results[0].Success {
		t.Error("Expected hook 1 to succeed, but it failed")
	}

	if !results[1].Success {
		t.Error("Expected hook 2 to succeed, but it failed")
	}

	// Scenario 2: One hook fails
	mockRunner.Commands["run_hook_1"] = struct {
		Output string
		Err    error
	}{"success", nil}
	mockRunner.Commands["run_hook_2"] = struct {
		Output string
		Err    error
	}{"failure", errors.New("hook failed")}

	results = service.RunHooks(hooks)

	if len(results) != 2 {
		t.Fatalf("Expected 2 results, but got %d", len(results))
	}

	if !results[0].Success {
		t.Error("Expected hook 1 to succeed, but it failed")
	}

	if results[1].Success {
		t.Error("Expected hook 2 to fail, but it succeeded")
	}
}

func TestHookRunnerService_RunHooksParallel_MatchesSequential(t *testing.T) {
	hooks := []domain.Hook{
		{Name: "Hook 1", Command: "cmd1"},
		{Name: "Hook 2", Command: "cmd2"},
		{Name: "Hook 3", Command: "cmd3"},
	}
	newRunner := func() *MockShellRunner {
		return &MockShellRunner{Commands: map[string]struct {
			Output string
			Err    error
		}{
			"cmd1": {"out1", nil},
			"cmd2": {"", errors.New("boom")},
			"cmd3": {"out3", nil},
		}}
	}

	seq := NewHookRunnerService(newRunner(), &MockLogger{}).RunHooks(hooks)
	par := NewHookRunnerService(newRunner(), &MockLogger{}).RunHooksParallel(hooks)

	if len(seq) != len(par) {
		t.Fatalf("length mismatch: seq=%d par=%d", len(seq), len(par))
	}
	for i := range seq {
		if seq[i].Success != par[i].Success || seq[i].Output != par[i].Output || seq[i].Hook.Name != par[i].Hook.Name {
			t.Errorf("result %d mismatch: seq=%+v par=%+v", i, seq[i], par[i])
		}
	}
}

func TestHookRunnerService_RunHooksParallel_PreservesOrder(t *testing.T) {
	hooks := []domain.Hook{
		{Name: "Slow", Command: "slow"},
		{Name: "Medium", Command: "medium"},
		{Name: "Fast", Command: "fast"},
	}
	mockRunner := &MockShellRunner{
		Commands: map[string]struct {
			Output string
			Err    error
		}{
			"slow": {"s", nil}, "medium": {"m", nil}, "fast": {"f", nil},
		},
		Delays: map[string]time.Duration{
			"slow": 30 * time.Millisecond, "medium": 15 * time.Millisecond, "fast": 0,
		},
	}
	service := NewHookRunnerService(mockRunner, &MockLogger{})

	results := service.RunHooksParallel(hooks)

	want := []string{"Slow", "Medium", "Fast"}
	for i, name := range want {
		if results[i].Hook.Name != name {
			t.Errorf("index %d: expected %s, got %s", i, name, results[i].Hook.Name)
		}
	}
}

func TestHookRunnerService_RunHooksParallel_RespectsMaxConcurrency(t *testing.T) {
	const hookCount = 8 // > maxParallelHooks (5)
	hooks := make([]domain.Hook, hookCount)
	commands := make(map[string]struct {
		Output string
		Err    error
	})
	delays := make(map[string]time.Duration)
	for i := range hookCount {
		cmd := fmt.Sprintf("cmd%d", i)
		hooks[i] = domain.Hook{Name: fmt.Sprintf("Hook %d", i), Command: cmd}
		commands[cmd] = struct {
			Output string
			Err    error
		}{"ok", nil}
		delays[cmd] = 20 * time.Millisecond // overlap window so peak concurrency is observable
	}
	mockRunner := &MockShellRunner{Commands: commands, Delays: delays}
	service := NewHookRunnerService(mockRunner, &MockLogger{})

	service.RunHooksParallel(hooks)

	if peak := mockRunner.PeakConcurrency(); peak > 5 {
		t.Errorf("expected peak concurrency <= 5, got %d", peak)
	} else if peak < 2 {
		t.Errorf("expected hooks to actually run concurrently, got peak concurrency %d", peak)
	}
}

func TestHookRunnerService_RunHooksParallel_MixedResults(t *testing.T) {
	hooks := []domain.Hook{
		{Name: "Pass1", Command: "pass1"},
		{Name: "Fail1", Command: "fail1", OutputRules: domain.OutputRules{ShowOn: "failure", OnFailureMessage: "check it"}},
		{Name: "Pass2", Command: "pass2"},
	}
	mockRunner := &MockShellRunner{Commands: map[string]struct {
		Output string
		Err    error
	}{
		"pass1": {"ok1", nil},
		"fail1": {"bad", errors.New("failed")},
		"pass2": {"ok2", nil},
	}}
	mockLogger := &MockLogger{}
	service := NewHookRunnerService(mockRunner, mockLogger)

	results := service.RunHooksParallel(hooks)

	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}
	if !results[0].Success || results[1].Success || !results[2].Success {
		t.Errorf("unexpected success flags: %v %v %v", results[0].Success, results[1].Success, results[2].Success)
	}
	found := false
	for _, m := range mockLogger.Messages {
		if m == "check it" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected OnFailureMessage to be logged, messages=%v", mockLogger.Messages)
	}
}
