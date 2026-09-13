package service

import (
	"errors"
	"strings"
	"testing"

	"github.com/dmux/go-quality-gate/internal/config"
	"github.com/dmux/go-quality-gate/internal/domain"
)

func sampleConfig() *config.Config {
	return &config.Config{
		Tools: config.Tools{
			{Name: "Gitleaks", CheckCommand: "gitleaks version", InstallCommand: "install gitleaks"},
		},
		Hooks: config.Hooks{
			"security": {
				"pre-commit": []config.Hook{
					{Name: "Secrets", Command: "gitleaks detect"},
				},
			},
			"backend": {
				"pre-commit": []config.Hook{
					{Name: "Format", Command: "ruff format --check", FixCommand: "ruff format"},
					{Name: "Tests", Command: "pytest"},
				},
				"pre-push": []config.Hook{
					{Name: "Slow tests", Command: "pytest --slow"},
				},
			},
		},
	}
}

// T1: all hooks pass — Run returns results with no error.
func TestQualityGateService_Run_AllHooksPass(t *testing.T) {
	toolManager := &MockToolManager{}
	hookRunner := &MockHookRunner{
		RunHooksResults: []domain.ExecutionResult{
			{Hook: domain.Hook{Name: "Secrets"}, Success: true},
			{Hook: domain.Hook{Name: "Format"}, Success: true},
			{Hook: domain.Hook{Name: "Tests"}, Success: true},
		},
	}
	service := NewQualityGateService(toolManager, hookRunner)

	results, err := service.Run(sampleConfig(), "pre-commit", false)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}
	if !toolManager.EnsureCalled {
		t.Error("expected EnsureToolsInstalled to be called")
	}
}

// T2: one hook fails — Run returns results plus an error.
func TestQualityGateService_Run_OneHookFails(t *testing.T) {
	toolManager := &MockToolManager{}
	hookRunner := &MockHookRunner{
		RunHooksResults: []domain.ExecutionResult{
			{Hook: domain.Hook{Name: "Secrets"}, Success: true},
			{Hook: domain.Hook{Name: "Format"}, Success: false},
		},
	}
	service := NewQualityGateService(toolManager, hookRunner)

	results, err := service.Run(sampleConfig(), "pre-commit", false)

	if err == nil {
		t.Fatal("expected an error when a hook fails")
	}
	if len(results) != 2 {
		t.Fatalf("expected results to still be returned, got %d", len(results))
	}
}

// T3: EnsureToolsInstalled fails — Run returns an error immediately, without
// ever invoking the hook runner.
func TestQualityGateService_Run_ToolInstallFails(t *testing.T) {
	toolManager := &MockToolManager{Err: errors.New("tool install failed")}
	hookRunner := &MockHookRunner{
		RunHooksResults: []domain.ExecutionResult{
			{Hook: domain.Hook{Name: "Secrets"}, Success: true},
		},
	}
	service := NewQualityGateService(toolManager, hookRunner)

	results, err := service.Run(sampleConfig(), "pre-commit", false)

	if err == nil {
		t.Fatal("expected an error when tool installation fails")
	}
	if results != nil {
		t.Errorf("expected nil results, got %v", results)
	}
	if hookRunner.ReceivedHooks != nil {
		t.Error("expected RunHooks to never be called when tool installation fails")
	}
}

// T8: getHooksToRun filters correctly by hookType, across multiple hook
// groups, and only passes the matching hooks through to the hook runner.
func TestQualityGateService_Run_FiltersHooksByHookType(t *testing.T) {
	toolManager := &MockToolManager{}
	hookRunner := &MockHookRunner{}
	service := NewQualityGateService(toolManager, hookRunner)

	if _, err := service.Run(sampleConfig(), "pre-push", false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(hookRunner.ReceivedHooks) != 1 {
		t.Fatalf("expected 1 hook for pre-push, got %d: %+v", len(hookRunner.ReceivedHooks), hookRunner.ReceivedHooks)
	}
	if hookRunner.ReceivedHooks[0].Name != "Slow tests" {
		t.Errorf("expected 'Slow tests' hook, got %q", hookRunner.ReceivedHooks[0].Name)
	}
}

func TestQualityGateService_Run_UnknownHookTypeYieldsNoHooks(t *testing.T) {
	toolManager := &MockToolManager{}
	hookRunner := &MockHookRunner{}
	service := NewQualityGateService(toolManager, hookRunner)

	results, err := service.Run(sampleConfig(), "post-checkout", false)

	if err != nil {
		t.Fatalf("expected no error for a hook type with no configured hooks, got %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected no results, got %d", len(results))
	}
	if hookRunner.ReceivedHooks != nil {
		t.Errorf("expected RunHooks to receive no hooks, got %v", hookRunner.ReceivedHooks)
	}
}

func TestQualityGateService_Run_ParallelFlagRoutesToRunHooksParallel(t *testing.T) {
	toolManager := &MockToolManager{}
	hookRunner := &MockHookRunner{
		RunHooksParallelResults: []domain.ExecutionResult{
			{Hook: domain.Hook{Name: "Secrets"}, Success: true},
		},
	}
	service := NewQualityGateService(toolManager, hookRunner)

	results, err := service.Run(sampleConfig(), "pre-commit", true)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result from RunHooksParallel, got %d", len(results))
	}
	if hookRunner.ReceivedParallelHooks == nil {
		t.Error("expected RunHooksParallel to be called")
	}
	if hookRunner.ReceivedHooks != nil {
		t.Error("expected RunHooks (sequential) to not be called when parallel=true")
	}
}

// T5: fix command executes successfully.
func TestQualityGateService_Fix_Success(t *testing.T) {
	toolManager := &MockToolManager{}
	hookRunner := &MockHookRunner{}
	service := NewQualityGateService(toolManager, hookRunner)

	if err := service.Fix(sampleConfig(), "pre-commit"); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(hookRunner.FixedHooks) != 1 {
		t.Fatalf("expected exactly 1 hook to be fixed (only 'Format' has a fix_command), got %d: %+v", len(hookRunner.FixedHooks), hookRunner.FixedHooks)
	}
	if hookRunner.FixedHooks[0].Name != "Format" {
		t.Errorf("expected 'Format' to be fixed, got %q", hookRunner.FixedHooks[0].Name)
	}
}

// T7: fix command fails — Fix returns an error naming the hook.
func TestQualityGateService_Fix_CommandFails(t *testing.T) {
	toolManager := &MockToolManager{}
	hookRunner := &MockHookRunner{
		FixResults: map[string]struct {
			Output string
			Err    error
		}{
			"Format": {Err: errors.New("fix failed")},
		},
	}
	service := NewQualityGateService(toolManager, hookRunner)

	err := service.Fix(sampleConfig(), "pre-commit")

	if err == nil {
		t.Fatal("expected an error when the fix command fails")
	}
	if !strings.Contains(err.Error(), "Format") {
		t.Errorf("expected error to mention hook name %q, got %q", "Format", err.Error())
	}
}

func TestQualityGateService_Fix_NoFixCommandsConfigured(t *testing.T) {
	toolManager := &MockToolManager{}
	hookRunner := &MockHookRunner{}
	service := NewQualityGateService(toolManager, hookRunner)

	// "Secrets" has no fix_command, so Fix should be a no-op for pre-commit's
	// security group alone.
	cfg := &config.Config{
		Hooks: config.Hooks{
			"security": {
				"pre-commit": []config.Hook{
					{Name: "Secrets", Command: "gitleaks detect"},
				},
			},
		},
	}

	if err := service.Fix(cfg, "pre-commit"); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if hookRunner.FixedHooks != nil {
		t.Errorf("expected no fix commands to run, got %v", hookRunner.FixedHooks)
	}
}
