package service

import (
	"fmt"

	"github.com/dmux/go-quality-gate/internal/repository"
)

const (
	preCommitHookContent = "#!/bin/sh\nexec quality-gate pre-commit\n"
	prePushHookContent   = "#!/bin/sh\nexec quality-gate pre-push\n"
)

// InstallationService is responsible for installing the git hooks.

type InstallationService struct {
	gitRepo repository.GitRepository
}

// NewInstallationService creates a new InstallationService.

func NewInstallationService(gitRepo repository.GitRepository) *InstallationService {
	return &InstallationService{gitRepo: gitRepo}
}

// InstallHooks installs the pre-commit and pre-push git hooks.

func (s *InstallationService) InstallHooks() error {
	if err := s.gitRepo.InstallHook("pre-commit", preCommitHookContent); err != nil {
		return fmt.Errorf("failed to install pre-commit hook: %w", err)
	}

	if err := s.gitRepo.InstallHook("pre-push", prePushHookContent); err != nil {
		return fmt.Errorf("failed to install pre-push hook: %w", err)
	}

	return nil
}
