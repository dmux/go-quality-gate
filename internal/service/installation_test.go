package service

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// MockGitRepository is a mock implementation of the GitRepository interface.

type MockGitRepository struct {
	InstalledHooks map[string]string
	Err            error
	// FailOnHookType, if set, makes InstallHook fail only for that specific
	// hookType, succeeding for any other — used to test that a later call
	// (e.g. pre-push) still propagates its own error.
	FailOnHookType string
}

// InstallHook implements the GitRepository interface.

func (r *MockGitRepository) InstallHook(hookType string, content string) error {
	if r.Err != nil {
		return r.Err
	}
	if r.FailOnHookType != "" && hookType == r.FailOnHookType {
		return errors.New("simulated failure for " + hookType)
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

func TestInstallationService_InstallHooks_PropagatesPrePushError(t *testing.T) {
	mockRepo := &MockGitRepository{FailOnHookType: "pre-push"}
	service := NewInstallationService(mockRepo)

	err := service.InstallHooks()

	if err == nil {
		t.Fatal("expected InstallHooks to return an error when installing pre-push fails")
	}
	if _, ok := mockRepo.InstalledHooks["pre-commit"]; !ok {
		t.Error("expected pre-commit to have been installed before pre-push failed")
	}
}

func TestInstallationService_InstallHooks_ExecutablePathError(t *testing.T) {
	origExecutable := osExecutable
	osExecutable = func() (string, error) { return "", errors.New("cannot resolve executable") }
	t.Cleanup(func() { osExecutable = origExecutable })

	service := NewInstallationService(&MockGitRepository{})

	if err := service.InstallHooks(); err == nil {
		t.Fatal("expected an error when os.Executable fails")
	}
}

func TestInstallationService_InstallHooks_SymlinkResolutionError(t *testing.T) {
	origEvalSymlinks := evalSymlinks
	evalSymlinks = func(path string) (string, error) { return "", errors.New("cannot resolve symlink") }
	t.Cleanup(func() { evalSymlinks = origEvalSymlinks })

	service := NewInstallationService(&MockGitRepository{})

	if err := service.InstallHooks(); err == nil {
		t.Fatal("expected an error when symlink resolution fails")
	}
}
