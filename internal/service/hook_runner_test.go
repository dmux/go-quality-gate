package service

import (
	"errors"
	"testing"

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
