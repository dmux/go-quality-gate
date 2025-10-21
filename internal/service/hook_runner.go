package service

import (
	"fmt"
	"time"

	"github.com/dmux/go-quality-gate/internal/domain"
	"github.com/dmux/go-quality-gate/internal/infra/logger"
	"github.com/dmux/go-quality-gate/internal/repository"
)

// HookRunnerService is responsible for running hooks.

type HookRunnerService struct {
	shellRunner repository.ShellRunner
	logger      logger.Logger
}

// NewHookRunnerService creates a new HookRunnerService.

func NewHookRunnerService(shellRunner repository.ShellRunner, logger logger.Logger) *HookRunnerService {
	return &HookRunnerService{shellRunner: shellRunner, logger: logger}
}

func (s *HookRunnerService) RunFixCommand(hook domain.Hook) (string, error) {
	if hook.FixCommand == "" {
		return "", fmt.Errorf("no fix command defined for hook: %s", hook.Name)
	}

	s.logger.StartSpinner(fmt.Sprintf("Running fix command for %s...", hook.Name))
	output, err := s.shellRunner.Run(hook.FixCommand)
	s.logger.StopSpinner()

	if err != nil {
		return output, fmt.Errorf("failed to run fix command for %s: %w\n%s", hook.Name, err, output)
	}
	s.logger.Print("🔧 Fix command for %s completed.\n", hook.Name)
	return output, nil
}

// RunHooks runs the given hooks and returns the execution results.

func (s *HookRunnerService) RunHooks(hooks []domain.Hook) []domain.ExecutionResult {
	var results []domain.ExecutionResult

	for _, hook := range hooks {
		s.logger.StartSpinner(fmt.Sprintf("Running %s...", hook.Name))

		startTime := time.Now()
		output, err := s.shellRunner.Run(hook.Command)
		duration := time.Since(startTime)

		s.logger.StopSpinner()

		result := domain.ExecutionResult{
			Hook:     hook,
			Success:  err == nil,
			Output:   output,
			Duration: duration,
		}

		results = append(results, result)

		if !result.Success {
			s.logger.Print("❌ %s failed (%v)\n", hook.Name, duration.Round(time.Millisecond))
			if hook.OutputRules.OnFailureMessage != "" {
				s.logger.Println(hook.OutputRules.OnFailureMessage)
			}
			if hook.OutputRules.ShowOn == "failure" || hook.OutputRules.ShowOn == "always" {
				s.logger.Println(output)
			}
		} else {
			s.logger.Print("✅ %s passed (%v)\n", hook.Name, duration.Round(time.Millisecond))
			if hook.OutputRules.ShowOn == "always" {
				s.logger.Println(output)
			}
		}
	}

	return results
}
