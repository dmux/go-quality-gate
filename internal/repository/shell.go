package repository

import "github.com/dmux/go-quality-gate/internal/domain"

// ShellRunner defines the interface for running shell commands.

type ShellRunner interface {
	Run(command string) (string, error)
}

// ToolManager defines the interface for tool management.
type ToolManager interface {
	EnsureToolsInstalled(tools []domain.Tool) error
}

// HookRunner defines the interface for hook execution.
type HookRunner interface {
	RunHooks(hooks []domain.Hook) []domain.ExecutionResult
	RunHooksParallel(hooks []domain.Hook) []domain.ExecutionResult
	RunFixCommand(hook domain.Hook) (string, error)
}
