package service

import (
	"fmt"
	"strings"

	"github.com/dmux/go-quality-gate/internal/config"
	"github.com/dmux/go-quality-gate/internal/domain"
	"github.com/dmux/go-quality-gate/internal/repository"
)

// QualityGateService is the main service that orchestrates the quality gate process.

type QualityGateService struct {
	toolManager repository.ToolManager
	hookRunner  repository.HookRunner
	files       repository.FileLister
}

// RunOptions controls how Run executes the hooks.
type RunOptions struct {
	// Parallel runs independent hooks concurrently.
	Parallel bool
	// Scope selects the files passed to {files}/{dirs} and matched against
	// globs. The zero value means domain.ScopeChanged.
	Scope domain.Scope
}

// NewQualityGateService creates a new QualityGateService.

func NewQualityGateService(toolManager repository.ToolManager, hookRunner repository.HookRunner) *QualityGateService {
	return &QualityGateService{toolManager: toolManager, hookRunner: hookRunner}
}

// WithFileLister sets where the files of a run come from. Without one, hooks
// that use {files}, {dirs} or a glob fail to run.
func (s *QualityGateService) WithFileLister(files repository.FileLister) *QualityGateService {
	s.files = files
	return s
}

// Run executes the quality gate process for a given hook type (e.g., "pre-commit").

func (s *QualityGateService) Run(cfg *config.Config, hookType string, opts RunOptions) ([]domain.ExecutionResult, error) {
	// 1. Ensure all tools are installed.
	if err := s.toolManager.EnsureToolsInstalled(s.configToolsToDomain(cfg.Tools), cfg.Settings.InstallPolicy()); err != nil {
		return nil, fmt.Errorf("failed to ensure tools are installed: %w", err)
	}

	// 2. Scope the hooks to the files of the run, skipping those no file
	// matches.
	planned, err := s.planHooks(s.getHooksToRun(cfg.Hooks, hookType), hookType, normalizeScope(opts.Scope), false)
	if err != nil {
		return nil, err
	}
	// 3. Run the remaining hooks.
	var results []domain.ExecutionResult
	if opts.Parallel {
		results = s.runParallel(planned)
	} else {
		results = s.runSequential(planned)
	}

	// 4. Check the results and exit if any hook failed.
	for _, result := range results {
		if !result.Success {
			return results, fmt.Errorf("one or more hooks failed")
		}
	}

	return results, nil
}

// runSequential runs the hooks in config order, logging skipped hooks in
// place: runnable hooks between two skipped ones go to the runner as one
// batch.
func (s *QualityGateService) runSequential(planned []scopedHook) []domain.ExecutionResult {
	var results []domain.ExecutionResult
	var batch []domain.Hook
	flush := func() {
		if len(batch) > 0 {
			results = append(results, s.hookRunner.RunHooks(batch)...)
			batch = nil
		}
	}

	if !anySkipped(planned) {
		for _, p := range planned {
			batch = append(batch, p.hook)
		}
		return s.hookRunner.RunHooks(batch)
	}
	for _, p := range planned {
		if p.skipped {
			flush()
			results = append(results, s.skippedResult(p))
			continue
		}
		batch = append(batch, p.hook)
	}
	flush()
	return results
}

// runParallel runs every runnable hook concurrently. Skipped hooks are
// logged first, then merged back into the results in config order.
func (s *QualityGateService) runParallel(planned []scopedHook) []domain.ExecutionResult {
	var hooks []domain.Hook
	for _, p := range planned {
		if !p.skipped {
			hooks = append(hooks, p.hook)
		}
	}
	if !anySkipped(planned) {
		return s.hookRunner.RunHooksParallel(hooks)
	}

	skipped := make(map[int]domain.ExecutionResult)
	for i, p := range planned {
		if p.skipped {
			skipped[i] = s.skippedResult(p)
		}
	}
	ran := s.hookRunner.RunHooksParallel(hooks)

	merged := make([]domain.ExecutionResult, 0, len(planned))
	next := 0
	for i := range planned {
		if r, ok := skipped[i]; ok {
			merged = append(merged, r)
		} else if next < len(ran) {
			merged = append(merged, ran[next])
			next++
		}
	}
	return append(merged, ran[next:]...)
}

// skippedResult logs and returns the result of a skipped hook, which counts
// as passed.
func (s *QualityGateService) skippedResult(p scopedHook) domain.ExecutionResult {
	result := domain.ExecutionResult{Hook: p.hook, Success: true, Skipped: true, SkipReason: p.skipReason}
	s.hookRunner.LogResult(result)
	return result
}

func anySkipped(planned []scopedHook) bool {
	for _, p := range planned {
		if p.skipped {
			return true
		}
	}
	return false
}

// normalizeScope maps the zero value to the default scope.
func normalizeScope(scope domain.Scope) domain.Scope {
	if scope == "" {
		return domain.ScopeChanged
	}
	return scope
}

// Fix executes the fix commands for all fixable hooks.

func (s *QualityGateService) Fix(cfg *config.Config, hookType string, scope domain.Scope) error {
	var fixable []domain.Hook
	for _, hook := range s.getHooksToRun(cfg.Hooks, hookType) {
		if hook.FixCommand != "" {
			fixable = append(fixable, hook)
		}
	}
	planned, err := s.planHooks(fixable, hookType, normalizeScope(scope), true)
	if err != nil {
		return err
	}

	var fixErrors []string
	for _, p := range planned {
		hook := p.hook
		if p.skipped {
			continue
		}
		if _, err := s.hookRunner.RunFixCommand(hook); err != nil {
			// A non-zero exit from a fix command (e.g. `ruff check --fix`
			// still reporting unfixable errors) must not skip the
			// remaining fixes — collect and keep going.
			fixErrors = append(fixErrors, fmt.Sprintf("fix command for hook %s failed: %s", hook.Name, err))
		}
	}
	if len(fixErrors) > 0 {
		return fmt.Errorf("one or more fix commands failed:\n%s", strings.Join(fixErrors, "\n"))
	}

	return nil
}

func (s *QualityGateService) configToolsToDomain(configTools []config.Tool) []domain.Tool {
	var domainTools []domain.Tool
	for _, t := range configTools {
		domainTools = append(domainTools, domain.Tool{
			Name:           t.Name,
			CheckCommand:   t.CheckCommand,
			InstallCommand: t.InstallCommand,
		})
	}
	return domainTools
}

func (s *QualityGateService) getHooksToRun(configHooks config.Hooks, hookType string) []domain.Hook {
	var domainHooks []domain.Hook
	for _, group := range configHooks {
		if hooks, ok := group[hookType]; ok {
			for _, h := range hooks {
				domainHooks = append(domainHooks, domain.Hook{
					Name:       h.Name,
					Command:    h.Command,
					FixCommand: h.FixCommand,
					Glob:       h.Glob,
					OutputRules: domain.OutputRules{
						ShowOn:           h.OutputRules.ShowOn,
						OnFailureMessage: h.OutputRules.OnFailureMessage,
					},
				})
			}
		}
	}
	return domainHooks
}
