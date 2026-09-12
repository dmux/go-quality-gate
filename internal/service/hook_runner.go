package service

import (
	"fmt"
	"sync"
	"time"

	"github.com/dmux/go-quality-gate/internal/domain"
	"github.com/dmux/go-quality-gate/internal/infra/logger"
	"github.com/dmux/go-quality-gate/internal/repository"
)

// maxParallelHooks caps the number of hooks RunHooksParallel runs concurrently.
const maxParallelHooks = 5

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
		s.logResult(result)
	}

	return results
}

// logResult logs the pass/fail outcome for one hook, honoring OutputRules.
// Must only be called from the calling goroutine, never from a worker
// goroutine — StartSpinner/StopSpinner/Print/Println are not goroutine-safe
// (see internal/infra/spinner/spinner.go: isActive/spinner fields have no lock).
func (s *HookRunnerService) logResult(result domain.ExecutionResult) {
	if !result.Success {
		s.logger.Print("❌ %s failed (%v)\n", result.Hook.Name, result.Duration.Round(time.Millisecond))
		if result.Hook.OutputRules.OnFailureMessage != "" {
			s.logger.Println(result.Hook.OutputRules.OnFailureMessage)
		}
		if result.Hook.OutputRules.ShowOn == "failure" || result.Hook.OutputRules.ShowOn == "always" {
			s.logger.Println(result.Output)
		}
	} else {
		s.logger.Print("✅ %s passed (%v)\n", result.Hook.Name, result.Duration.Round(time.Millisecond))
		if result.Hook.OutputRules.ShowOn == "always" {
			s.logger.Println(result.Output)
		}
	}
}

// RunHooksParallel runs hooks concurrently, bounded by maxParallelHooks, and
// returns results in input order regardless of completion order. No
// logger/spinner calls happen inside worker goroutines — all logging happens
// after wg.Wait(), in the calling goroutine, in input order.
func (s *HookRunnerService) RunHooksParallel(hooks []domain.Hook) []domain.ExecutionResult {
	results := make([]domain.ExecutionResult, len(hooks))

	if len(hooks) > 0 {
		s.logger.StartSpinner(fmt.Sprintf("Running %d hooks in parallel...", len(hooks)))

		var wg sync.WaitGroup
		sem := make(chan struct{}, maxParallelHooks)

		for i, hook := range hooks {
			wg.Add(1)
			go func(i int, hook domain.Hook) {
				defer wg.Done()
				sem <- struct{}{}
				defer func() { <-sem }()

				startTime := time.Now()
				output, err := s.shellRunner.Run(hook.Command)
				duration := time.Since(startTime)

				results[i] = domain.ExecutionResult{
					Hook:     hook,
					Success:  err == nil,
					Output:   output,
					Duration: duration,
				}
			}(i, hook)
		}

		wg.Wait()
		s.logger.StopSpinner()
	}

	for _, result := range results {
		s.logResult(result)
	}

	return results
}
