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
	commitMsg, ok := mockRepo.InstalledHooks["commit-msg"]
	if !ok {
		t.Fatal("expected a commit-msg hook to be installed")
	}
	prePush, ok := mockRepo.InstalledHooks["pre-push"]
	if !ok {
		t.Fatal("expected a pre-push hook to be installed")
	}

	wantPreCommit := "#!/bin/sh\n" + HookMarker + "\nexec " + wantPath + " pre-commit\n"
	wantCommitMsg := "#!/bin/sh\n" + HookMarker + "\nexec " + wantPath + " commit-msg \"$1\"\n"
	wantPrePush := "#!/bin/sh\n" + HookMarker + "\nexec " + wantPath + " pre-push\n"

	if preCommit != wantPreCommit {
		t.Errorf("pre-commit hook content = %q, want %q", preCommit, wantPreCommit)
	}
	if commitMsg != wantCommitMsg {
		t.Errorf("commit-msg hook content = %q, want %q", commitMsg, wantCommitMsg)
	}
	if prePush != wantPrePush {
		t.Errorf("pre-push hook content = %q, want %q", prePush, wantPrePush)
	}
	if got := HookBinary(preCommit); got != wantPath {
		t.Errorf("HookBinary = %q, want %q", got, wantPath)
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

type dirHooks struct{ dir string }

func (d *dirHooks) InstallHook(hookType, content string) error {
	return os.WriteFile(filepath.Join(d.dir, hookType), []byte(content), 0755)
}

func (d *dirHooks) HooksDir() (string, error) { return d.dir, nil }

func TestInstallHooks_WritesManagedHooks(t *testing.T) {
	hooks := &dirHooks{dir: t.TempDir()}
	if err := NewInstallationService(hooks).InstallHooks(); err != nil {
		t.Fatal(err)
	}

	for _, hook := range ManagedHooks {
		content, err := os.ReadFile(filepath.Join(hooks.dir, hook))
		if err != nil {
			t.Fatalf("%s not installed: %v", hook, err)
		}
		if !IsManagedHook(string(content)) {
			t.Errorf("%s lacks the managed marker", hook)
		}
	}
}

func TestInstallGlobalHooks_OnlyActsWithConfig(t *testing.T) {
	hooks := &dirHooks{dir: t.TempDir()}
	if err := NewInstallationService(hooks).InstallGlobalHooks(); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(filepath.Join(hooks.dir, "pre-commit"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), "quality.yml") || !IsManagedHook(string(content)) {
		t.Fatalf("global hook must be managed and guarded by quality.yml: %q", content)
	}
	if !filepath.IsAbs(HookBinary(string(content))) {
		t.Fatalf("global hook must call the binary by absolute path: %q", content)
	}
}

func TestDoctor_ReportsMissingTamperedAndBrokenHooks(t *testing.T) {
	hooks := &dirHooks{dir: t.TempDir()}
	binary := filepath.Join(t.TempDir(), "quality-gate")
	if err := os.WriteFile(binary, []byte("#!/bin/sh\n"), 0755); err != nil {
		t.Fatal(err)
	}
	write := func(name, content string) {
		if err := os.WriteFile(filepath.Join(hooks.dir, name), []byte(content), 0755); err != nil {
			t.Fatal(err)
		}
	}
	write("pre-commit", HookContent("pre-commit", binary))
	write("commit-msg", "#!/bin/sh\nexit 0\n")
	write("pre-push", HookContent("pre-push", filepath.Join(t.TempDir(), "moved-away")))

	checks := NewDoctorService(hooks, filepath.Join(t.TempDir(), "quality.yml")).Run()

	got := map[string]bool{}
	for _, c := range checks {
		got[c.Name] = c.OK
	}
	want := map[string]bool{
		"pre-commit hook": true,
		"commit-msg hook": false, // not managed by quality-gate
		"pre-push hook":   false, // binary no longer exists
	}
	for name, ok := range want {
		if got[name] != ok {
			t.Errorf("%s: ok=%v, want %v", name, got[name], ok)
		}
	}
	if last := checks[len(checks)-1]; last.OK {
		t.Error("missing quality.yml must fail the config check")
	}
}
