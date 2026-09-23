package service

import "github.com/dmux/go-quality-gate/internal/domain"

// MockToolManager is a mock implementation of the repository.ToolManager interface.
type MockToolManager struct {
	Err           error
	EnsureCalled  bool
	ReceivedTools []domain.Tool
}

// EnsureToolsInstalled implements the repository.ToolManager interface.
func (m *MockToolManager) EnsureToolsInstalled(tools []domain.Tool) error {
	m.EnsureCalled = true
	m.ReceivedTools = tools
	return m.Err
}

// MockHookRunner is a mock implementation of the repository.HookRunner interface.
type MockHookRunner struct {
	RunHooksResults         []domain.ExecutionResult
	RunHooksParallelResults []domain.ExecutionResult
	// FixResults, keyed by hook name, controls RunFixCommand's return value
	// per hook. A hook name with no entry succeeds with an empty output.
	FixResults map[string]struct {
		Output string
		Err    error
	}

	ReceivedHooks         []domain.Hook
	ReceivedParallelHooks []domain.Hook
	FixedHooks            []domain.Hook
}

// RunHooks implements the repository.HookRunner interface.
func (m *MockHookRunner) RunHooks(hooks []domain.Hook) []domain.ExecutionResult {
	m.ReceivedHooks = hooks
	return m.RunHooksResults
}

// RunHooksParallel implements the repository.HookRunner interface.
func (m *MockHookRunner) RunHooksParallel(hooks []domain.Hook) []domain.ExecutionResult {
	m.ReceivedParallelHooks = hooks
	return m.RunHooksParallelResults
}

// RunFixCommand implements the repository.HookRunner interface.
func (m *MockHookRunner) RunFixCommand(hook domain.Hook) (string, error) {
	m.FixedHooks = append(m.FixedHooks, hook)
	if result, ok := m.FixResults[hook.Name]; ok {
		return result.Output, result.Err
	}
	return "", nil
}

// MockLogger is a mock implementation of the Logger interface.
type MockLogger struct {
	Messages []string
}

// Print implements the Logger interface.
func (m *MockLogger) Print(format string, args ...interface{}) {
	// For testing, we just ignore the output
}

// Println implements the Logger interface.
func (m *MockLogger) Println(msg string) {
	m.Messages = append(m.Messages, msg)
}

// StartSpinner implements the Logger interface.
func (m *MockLogger) StartSpinner(message string) {
	// For testing, we just ignore the output
}

// StopSpinner implements the Logger interface.
func (m *MockLogger) StopSpinner() {
	// For testing, we just ignore the output
}

// UpdateSpinner implements the Logger interface.
func (m *MockLogger) UpdateSpinner(message string) {
	// For testing, we just ignore the output
}
