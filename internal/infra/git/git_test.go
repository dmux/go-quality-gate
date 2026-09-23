package git

import (
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestFindGitDir_NotFound(t *testing.T) {
	t.Chdir(t.TempDir())

	if _, err := findGitDir(); err == nil {
		t.Fatal("expected an error when no .git directory exists")
	}
}

func TestFindGitDir_GetwdFails(t *testing.T) {
	orig := getwd
	getwd = func() (string, error) { return "", errors.New("boom") }
	t.Cleanup(func() { getwd = orig })

	if _, err := findGitDir(); err == nil {
		t.Fatal("expected an error when os.Getwd fails")
	}
}

func TestRealGitRepository_InstallHook_Success(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".git", "hooks"), 0755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(root)

	repo := &RealGitRepository{}
	if err := repo.InstallHook("pre-commit", "#!/bin/sh\necho hi\n"); err != nil {
		t.Fatalf("InstallHook failed: %v", err)
	}

	hookPath := filepath.Join(root, ".git", "hooks", "pre-commit")
	content, err := os.ReadFile(hookPath)
	if err != nil {
		t.Fatalf("expected hook file to exist: %v", err)
	}
	if string(content) != "#!/bin/sh\necho hi\n" {
		t.Errorf("unexpected hook content: %q", content)
	}

	info, err := os.Stat(hookPath)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0755 {
		t.Errorf("expected hook to be executable (0755), got %o", info.Mode().Perm())
	}
}

func TestRealGitRepository_InstallHook_NotAGitRepo(t *testing.T) {
	t.Chdir(t.TempDir())

	repo := &RealGitRepository{}
	if err := repo.InstallHook("pre-commit", "content"); err == nil {
		t.Fatal("expected an error when not inside a git repository")
	}
}

func TestRealGitRepository_InstallHook_CreatesMissingHooksDir(t *testing.T) {
	root := t.TempDir()
	// .git exists but .git/hooks does not; InstallHook creates it, since
	// core.hooksPath may point at a directory that doesn't exist yet.
	if err := os.Mkdir(filepath.Join(root, ".git"), 0755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(root)

	repo := &RealGitRepository{}
	if err := repo.InstallHook("pre-commit", "content"); err != nil {
		t.Fatalf("expected the hooks directory to be created, got %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, ".git", "hooks", "pre-commit")); err != nil {
		t.Fatalf("hook not written: %v", err)
	}
}

func TestRealGitRepository_InstallHook_HooksDirIsAFile(t *testing.T) {
	root := t.TempDir()
	// A regular file where the hooks directory should be makes MkdirAll fail.
	if err := os.Mkdir(filepath.Join(root, ".git"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".git", "hooks"), []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(root)

	repo := &RealGitRepository{}
	if err := repo.InstallHook("pre-commit", "content"); err == nil {
		t.Fatal("expected an error when the hooks directory can't be created")
	}
}

type failingWriteCloser struct{}

func (failingWriteCloser) Write(p []byte) (int, error) { return 0, errors.New("disk full") }
func (failingWriteCloser) Close() error                { return nil }

func TestRealGitRepository_InstallHook_WriteFails(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".git", "hooks"), 0755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(root)

	orig := createHookFile
	createHookFile = func(path string) (io.WriteCloser, error) { return failingWriteCloser{}, nil }
	t.Cleanup(func() { createHookFile = orig })

	repo := &RealGitRepository{}
	if err := repo.InstallHook("pre-commit", "content"); err == nil {
		t.Fatal("expected an error when writing the hook fails")
	}
}

func TestRealGitRepository_InstallHook_ChmodFails(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".git", "hooks"), 0755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(root)

	orig := chmodFile
	chmodFile = func(path string, mode os.FileMode) error { return errors.New("chmod denied") }
	t.Cleanup(func() { chmodFile = orig })

	repo := &RealGitRepository{}
	if err := repo.InstallHook("pre-commit", "content"); err == nil {
		t.Fatal("expected an error when chmod fails")
	}
}

func TestRealGitRepository_LoadToolsHash_NotAGitRepo(t *testing.T) {
	t.Chdir(t.TempDir())

	repo := &RealGitRepository{}
	if _, err := repo.LoadToolsHash(); err == nil {
		t.Fatal("expected an error when not inside a git repository")
	}
}

func TestRealGitRepository_LoadToolsHash_UnreadableFile(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root: permission checks don't apply")
	}

	root := t.TempDir()
	statePath := filepath.Join(root, ".git", toolsHashRelativePath)
	if err := os.MkdirAll(filepath.Dir(statePath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(statePath, []byte("hash"), 0000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(statePath, 0644) })
	t.Chdir(root)

	repo := &RealGitRepository{}
	if _, err := repo.LoadToolsHash(); err == nil {
		t.Fatal("expected an error reading an unreadable state file")
	}
}

func TestRealGitRepository_SaveToolsHash_NotAGitRepo(t *testing.T) {
	t.Chdir(t.TempDir())

	repo := &RealGitRepository{}
	if err := repo.SaveToolsHash("hash"); err == nil {
		t.Fatal("expected an error when not inside a git repository")
	}
}

func TestRealGitRepository_SaveToolsHash_MkdirAllFails(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, ".git"), 0755); err != nil {
		t.Fatal(err)
	}
	// A regular file where SaveToolsHash needs to create "quality-gate/"
	// forces MkdirAll to fail (ENOTDIR).
	if err := os.WriteFile(filepath.Join(root, ".git", "quality-gate"), []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(root)

	repo := &RealGitRepository{}
	if err := repo.SaveToolsHash("hash"); err == nil {
		t.Fatal("expected an error when the state directory can't be created")
	}
}

func TestRealGitRepository_SaveToolsHash_CreateTempFails(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".git"), 0755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(root)

	orig := createTempFile
	createTempFile = func(dir, pattern string) (tempWriteCloser, error) {
		return nil, errors.New("cannot create temp file")
	}
	t.Cleanup(func() { createTempFile = orig })

	repo := &RealGitRepository{}
	if err := repo.SaveToolsHash("hash"); err == nil {
		t.Fatal("expected an error when creating the temp file fails")
	}
}

// fakeTempFile lets tests fail a specific step of SaveToolsHash's
// write/chmod/close sequence while still producing a real, removable file
// on disk (so the deferred os.Remove(tempPath) in SaveToolsHash succeeds).
type fakeTempFile struct {
	*os.File
	failWrite, failChmod, failClose bool
}

func (f *fakeTempFile) Write(p []byte) (int, error) {
	if f.failWrite {
		return 0, errors.New("write failed")
	}
	return f.File.Write(p)
}

func (f *fakeTempFile) Chmod(mode os.FileMode) error {
	if f.failChmod {
		return errors.New("chmod failed")
	}
	return f.File.Chmod(mode)
}

func (f *fakeTempFile) Close() error {
	if f.failClose {
		f.File.Close()
		return errors.New("close failed")
	}
	return f.File.Close()
}

func newFakeTempFileFactory(t *testing.T, failWrite, failChmod, failClose bool) func(dir, pattern string) (tempWriteCloser, error) {
	t.Helper()
	return func(dir, pattern string) (tempWriteCloser, error) {
		f, err := os.CreateTemp(dir, pattern)
		if err != nil {
			return nil, err
		}
		return &fakeTempFile{File: f, failWrite: failWrite, failChmod: failChmod, failClose: failClose}, nil
	}
}

func TestRealGitRepository_SaveToolsHash_WriteFails(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".git"), 0755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(root)

	orig := createTempFile
	createTempFile = newFakeTempFileFactory(t, true, false, false)
	t.Cleanup(func() { createTempFile = orig })

	repo := &RealGitRepository{}
	if err := repo.SaveToolsHash("hash"); err == nil {
		t.Fatal("expected an error when writing the temp file fails")
	}
}

func TestRealGitRepository_SaveToolsHash_ChmodFails(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".git"), 0755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(root)

	orig := createTempFile
	createTempFile = newFakeTempFileFactory(t, false, true, false)
	t.Cleanup(func() { createTempFile = orig })

	repo := &RealGitRepository{}
	if err := repo.SaveToolsHash("hash"); err == nil {
		t.Fatal("expected an error when chmod on the temp file fails")
	}
}

func TestRealGitRepository_SaveToolsHash_CloseFails(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".git"), 0755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(root)

	orig := createTempFile
	createTempFile = newFakeTempFileFactory(t, false, false, true)
	t.Cleanup(func() { createTempFile = orig })

	repo := &RealGitRepository{}
	if err := repo.SaveToolsHash("hash"); err == nil {
		t.Fatal("expected an error when closing the temp file fails")
	}
}

func TestRealGitRepository_InstallHook_IgnoresGlobalHooksPath(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	// Resolve symlinks (macOS /var -> /private/var) so paths compare equal.
	globalHooks, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	globalConfig := filepath.Join(t.TempDir(), "gitconfig")
	if err := os.WriteFile(globalConfig, []byte("[core]\n\thooksPath = "+globalHooks+"\n"), 0644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GIT_CONFIG_GLOBAL", globalConfig)

	root := t.TempDir()
	t.Chdir(root)
	if out, err := exec.Command("git", "init", "-q").CombinedOutput(); err != nil {
		t.Fatalf("git init: %v: %s", err, out)
	}

	repo := &RealGitRepository{}
	if err := repo.InstallHook("pre-commit", "content"); err != nil {
		t.Fatal(err)
	}

	// A per-repository install must never overwrite hooks shared by every
	// repository through a global core.hooksPath.
	if _, err := os.Stat(filepath.Join(globalHooks, "pre-commit")); !os.IsNotExist(err) {
		t.Fatalf("hook was written to the global hooks path (stat err: %v)", err)
	}
	if _, err := os.Stat(filepath.Join(root, ".git", "hooks", "pre-commit")); err != nil {
		t.Fatalf("hook not written to the repository: %v", err)
	}
	if dir, err := repo.HooksDir(); err != nil || filepath.Clean(dir) != filepath.Clean(globalHooks) {
		t.Fatalf("HooksDir = %q, %v; want the effective global path %q", dir, err, globalHooks)
	}
}
