package service

import (
	"errors"
	"testing"

	"github.com/dmux/go-quality-gate/internal/domain"
)

// MockShellRunner is a mock implementation of the ShellRunner interface.

type MockShellRunner struct {
	Commands map[string]struct {
		Output string
		Err    error
	}
}

// Run implements the ShellRunner interface.

func (r *MockShellRunner) Run(command string) (string, error) {
	if cmd, ok := r.Commands[command]; ok {
		return cmd.Output, cmd.Err
	}
	return "", errors.New("command not found")
}

func TestToolManagerService_EnsureToolsInstalled(t *testing.T) {
	mockRunner := &MockShellRunner{
		Commands: make(map[string]struct {
			Output string
			Err    error
		}),
	}

	mockLogger := &MockLogger{}
	service := NewToolManagerService(mockRunner, mockLogger)

	tools := []domain.Tool{
		{
			Name:           "Tool 1",
			CheckCommand:   "check_tool_1",
			InstallCommand: "install_tool_1",
		},
		{
			Name:           "Tool 2",
			CheckCommand:   "check_tool_2",
			InstallCommand: "install_tool_2",
		},
	}

	// Scenario 1: Both tools are not installed
	mockRunner.Commands["check_tool_1"] = struct {
		Output string
		Err    error
	}{"", errors.New("not installed")}
	mockRunner.Commands["install_tool_1"] = struct {
		Output string
		Err    error
	}{"installed", nil}
	mockRunner.Commands["check_tool_2"] = struct {
		Output string
		Err    error
	}{"", errors.New("not installed")}
	mockRunner.Commands["install_tool_2"] = struct {
		Output string
		Err    error
	}{"installed", nil}

	err := service.EnsureToolsInstalled(tools)
	if err != nil {
		t.Errorf("Expected no error, but got: %v", err)
	}

	// Scenario 2: One tool is installed, the other is not
	mockRunner.Commands["check_tool_1"] = struct {
		Output string
		Err    error
	}{"installed", nil}
	mockRunner.Commands["check_tool_2"] = struct {
		Output string
		Err    error
	}{"", errors.New("not installed")}
	mockRunner.Commands["install_tool_2"] = struct {
		Output string
		Err    error
	}{"installed", nil}

	err = service.EnsureToolsInstalled(tools)
	if err != nil {
		t.Errorf("Expected no error, but got: %v", err)
	}

	// Scenario 3: Installation fails
	mockRunner.Commands["check_tool_1"] = struct {
		Output string
		Err    error
	}{"", errors.New("not installed")}
	mockRunner.Commands["install_tool_1"] = struct {
		Output string
		Err    error
	}{"", errors.New("installation failed")}

	err = service.EnsureToolsInstalled(tools)
	if err == nil {
		t.Error("Expected an error, but got none")
	}
}
