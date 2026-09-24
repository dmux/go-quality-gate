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
}

// NewQualityGateService creates a new QualityGateService.

func NewQualityGateService(toolManager repository.ToolManager, hookRunner repository.HookRunner) *QualityGateService {
	return &QualityGateService{toolManager: toolManager, hookRunner: hookRunner}
}

// Run executes the quality gate process for a given hook type (e.g., "pre-commit").

func (s *QualityGateService) Run(cfg *config.Config, hookType string, parallel bool) ([]domain.ExecutionResult, error) {
	// 1. Ensure all tools are installed.
	if err := s.toolManager.EnsureToolsInstalled(s.configToolsToDomain(cfg.Tools)); err != nil {
		return nil, fmt.Errorf("failed to ensure tools are installed: %w", err)
	}

	// 2. Run the hooks for the given hook type.
	hooksToRun := s.getHooksToRun(cfg.Hooks, hookType)
	var results []domain.ExecutionResult
	if parallel {
		results = s.hookRunner.RunHooksParallel(hooksToRun)
	} else {
		results = s.hookRunner.RunHooks(hooksToRun)
	}

	// 3. Check the results and exit if any hook failed.
	for _, result := range results {
		if !result.Success {
			return results, fmt.Errorf("one or more hooks failed")
		}
	}

	return results, nil
}

// Fix executes the fix commands for all fixable hooks.

func (s *QualityGateService) Fix(cfg *config.Config, hookType string) error {
	hooksToFix := s.getHooksToRun(cfg.Hooks, hookType)
	var fixErrors []string
	for _, hook := range hooksToFix {
		if hook.FixCommand == "" {
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
