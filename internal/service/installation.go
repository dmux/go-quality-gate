package service

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/dmux/go-quality-gate/internal/repository"
)

// InstallationService is responsible for installing the git hooks.

type InstallationService struct {
	gitRepo repository.GitRepository
}

// NewInstallationService creates a new InstallationService.

func NewInstallationService(gitRepo repository.GitRepository) *InstallationService {
	return &InstallationService{gitRepo: gitRepo}
}

// osExecutable and evalSymlinks are os.Executable/filepath.EvalSymlinks,
// swappable in tests to exercise InstallHooks' error paths — neither
// realistically fails for the running test binary itself.
var (
	osExecutable = os.Executable
	evalSymlinks = filepath.EvalSymlinks
)

// InstallHooks installs the pre-commit and pre-push git hooks. The hooks
// invoke the quality-gate binary by its resolved absolute path rather than
// by name, so they aren't subject to PATH lookup at commit/push time.
func (s *InstallationService) InstallHooks() error {
	execPath, err := osExecutable()
	if err != nil {
		return fmt.Errorf("failed to resolve executable path: %w", err)
	}

	realPath, err := evalSymlinks(execPath)
	if err != nil {
		return fmt.Errorf("failed to resolve symlink: %w", err)
	}

	preCommitContent := fmt.Sprintf("#!/bin/sh\nexec %s pre-commit\n", realPath)
	prePushContent := fmt.Sprintf("#!/bin/sh\nexec %s pre-push\n", realPath)

	if err := s.gitRepo.InstallHook("pre-commit", preCommitContent); err != nil {
		return fmt.Errorf("failed to install pre-commit hook: %w", err)
	}

	if err := s.gitRepo.InstallHook("pre-push", prePushContent); err != nil {
		return fmt.Errorf("failed to install pre-push hook: %w", err)
	}

	return nil
}
