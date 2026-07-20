package service

import (
	"errors"
	"testing"

	"github.com/dmux/go-quality-gate/internal/domain"
)

type countingShellRunner struct {
	commands map[string]error
	calls    map[string]int
}

func (r *countingShellRunner) Run(command string) (string, error) {
	r.calls[command]++
	return "", r.commands[command]
}

type memoryToolState struct {
	hash      string
	loadErr   error
	saveErr   error
	saveCalls int
}

func (s *memoryToolState) LoadToolsHash() (string, error) {
	return s.hash, s.loadErr
}

func (s *memoryToolState) SaveToolsHash(hash string) error {
	s.saveCalls++
	if s.saveErr == nil {
		s.hash = hash
	}
	return s.saveErr
}

func TestToolManagerService_CachesSuccessfulValidation(t *testing.T) {
	runner := &countingShellRunner{
		commands: map[string]error{"tool --version": nil},
		calls:    make(map[string]int),
	}
	state := &memoryToolState{}
	manager := NewCachingToolManagerService(runner, &MockLogger{}, state)
	tools := []domain.Tool{{
		Name:           "Tool",
		CheckCommand:   "tool --version",
		InstallCommand: "install tool",
	}}

	if err := manager.EnsureToolsInstalled(tools); err != nil {
		t.Fatalf("first validation failed: %v", err)
	}
	if err := manager.EnsureToolsInstalled(tools); err != nil {
		t.Fatalf("cached validation failed: %v", err)
	}

	if got := runner.calls["tool --version"]; got != 1 {
		t.Fatalf("check command ran %d times, want 1", got)
	}
	if state.saveCalls != 1 {
		t.Fatalf("cache was saved %d times, want 1", state.saveCalls)
	}
}

func TestToolManagerService_InvalidatesCacheWhenToolsChange(t *testing.T) {
	runner := &countingShellRunner{
		commands: map[string]error{
			"tool --version":     nil,
			"new-tool --version": nil,
		},
		calls: make(map[string]int),
	}
	state := &memoryToolState{}
	manager := NewCachingToolManagerService(runner, &MockLogger{}, state)
	original := []domain.Tool{{Name: "Tool", CheckCommand: "tool --version", InstallCommand: "install tool"}}
	changed := append(original, domain.Tool{Name: "New Tool", CheckCommand: "new-tool --version", InstallCommand: "install new-tool"})

	if err := manager.EnsureToolsInstalled(original); err != nil {
		t.Fatalf("initial validation failed: %v", err)
	}
	if err := manager.EnsureToolsInstalled(changed); err != nil {
		t.Fatalf("validation after configuration change failed: %v", err)
	}

	if got := runner.calls["tool --version"]; got != 2 {
		t.Errorf("existing tool check ran %d times, want 2", got)
	}
	if got := runner.calls["new-tool --version"]; got != 1 {
		t.Errorf("new tool check ran %d times, want 1", got)
	}
}

func TestToolManagerService_DoesNotCacheFailedInstallation(t *testing.T) {
	installErr := errors.New("installation failed")
	runner := &countingShellRunner{
		commands: map[string]error{
			"tool --version": errors.New("not installed"),
			"install tool":   installErr,
		},
		calls: make(map[string]int),
	}
	state := &memoryToolState{}
	manager := NewCachingToolManagerService(runner, &MockLogger{}, state)
	tools := []domain.Tool{{Name: "Tool", CheckCommand: "tool --version", InstallCommand: "install tool"}}

	if err := manager.EnsureToolsInstalled(tools); !errors.Is(err, installErr) {
		t.Fatalf("got error %v, want %v", err, installErr)
	}
	if state.saveCalls != 0 {
		t.Fatalf("failed validation was cached %d times", state.saveCalls)
	}
}

func TestToolManagerService_CacheErrorsDoNotBreakValidation(t *testing.T) {
	runner := &countingShellRunner{
		commands: map[string]error{"tool --version": nil},
		calls:    make(map[string]int),
	}
	state := &memoryToolState{
		loadErr: errors.New("cannot read cache"),
		saveErr: errors.New("cannot write cache"),
	}
	manager := NewCachingToolManagerService(runner, &MockLogger{}, state)
	tools := []domain.Tool{{Name: "Tool", CheckCommand: "tool --version", InstallCommand: "install tool"}}

	if err := manager.EnsureToolsInstalled(tools); err != nil {
		t.Fatalf("cache errors broke validation: %v", err)
	}
	if got := runner.calls["tool --version"]; got != 1 {
		t.Fatalf("check command ran %d times, want 1", got)
	}
}
