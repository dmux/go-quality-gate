package service

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// MockGitRepository is a mock implementation of the GitRepository interface.

type MockGitRepository struct {
	InstalledHooks map[string]string
	Err            error
}

// InstallHook implements the GitRepository interface.

func (r *MockGitRepository) InstallHook(hookType string, content string) error {
	if r.Err != nil {
		return r.Err
	}
	if r.InstalledHooks == nil {
		r.InstalledHooks = make(map[string]string)
	}
	r.InstalledHooks[hookType] = content
	return nil
}

func TestInstallationService_InstallHooks_UsesAbsolutePath(t *testing.T) {
	mockRepo := &MockGitRepository{}
	service := NewInstallationService(mockRepo)

	if err := service.InstallHooks(); err != nil {
		t.Fatalf("InstallHooks failed: %v", err)
	}

	execPath, err := os.Executable()
	if err != nil {
		t.Fatalf("failed to resolve test executable path: %v", err)
	}
	wantPath, err := filepath.EvalSymlinks(execPath)
	if err != nil {
		t.Fatalf("failed to resolve symlinks for test executable: %v", err)
	}

	preCommit, ok := mockRepo.InstalledHooks["pre-commit"]
	if !ok {
		t.Fatal("expected a pre-commit hook to be installed")
	}
	prePush, ok := mockRepo.InstalledHooks["pre-push"]
	if !ok {
		t.Fatal("expected a pre-push hook to be installed")
	}

	wantPreCommit := "#!/bin/sh\nexec " + wantPath + " pre-commit\n"
	wantPrePush := "#!/bin/sh\nexec " + wantPath + " pre-push\n"

	if preCommit != wantPreCommit {
		t.Errorf("pre-commit hook content = %q, want %q", preCommit, wantPreCommit)
	}
	if prePush != wantPrePush {
		t.Errorf("pre-push hook content = %q, want %q", prePush, wantPrePush)
	}

	// Guard against regressing to a bare, PATH-resolved command name.
	if strings.Contains(preCommit, "exec quality-gate ") {
		t.Error("pre-commit hook must not invoke quality-gate via PATH lookup")
	}
	if !filepath.IsAbs(wantPath) {
		t.Fatalf("resolved executable path is not absolute: %s", wantPath)
	}
}

func TestInstallationService_InstallHooks_PropagatesGitRepoError(t *testing.T) {
	mockRepo := &MockGitRepository{Err: os.ErrPermission}
	service := NewInstallationService(mockRepo)

	if err := service.InstallHooks(); err == nil {
		t.Fatal("expected InstallHooks to return an error when the git repository fails")
	}
}
