package git

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const toolsHashRelativePath = "quality-gate/tools.sha256"

// RealGitRepository is a real implementation of the GitRepository interface.

type RealGitRepository struct{}

// createHookFile is os.Create, swappable in tests to exercise the
// write-failure path in InstallHook, which a real OS-level write()
// failure isn't practical to reproduce portably. *os.File satisfies
// io.WriteCloser.
var createHookFile = func(path string) (io.WriteCloser, error) {
	return os.Create(path)
}

// chmodFile is os.Chmod, swappable in tests to exercise InstallHook's final
// chmod-failure path without depending on OS-specific permission tricks.
var chmodFile = os.Chmod

// InstallHook implements the GitRepository interface.

func (r *RealGitRepository) InstallHook(hookType string, content string) error {
	gitDir, err := findGitDir()
	if err != nil {
		return err
	}

	hookPath := filepath.Join(gitDir, "hooks", hookType)

	f, err := createHookFile(hookPath)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.Write([]byte(content))
	if err != nil {
		return err
	}

	return chmodFile(hookPath, 0755)
}

// LoadToolsHash returns the hash of the tools configuration that was last
// successfully validated. A missing state file is treated as an empty cache.
func (r *RealGitRepository) LoadToolsHash() (string, error) {
	gitDir, err := findGitDir()
	if err != nil {
		return "", err
	}

	content, err := os.ReadFile(filepath.Join(gitDir, toolsHashRelativePath))
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(content)), nil
}

// tempWriteCloser is the subset of *os.File's methods SaveToolsHash needs
// for its create-write-chmod-close-rename sequence, extracted so tests can
// substitute a fake that fails on a specific step.
type tempWriteCloser interface {
	io.Writer
	Chmod(mode os.FileMode) error
	Close() error
	Name() string
}

// createTempFile is os.CreateTemp, swappable in tests to exercise
// SaveToolsHash's write/chmod/close failure paths, which real OS-level
// failures on an already-created temp file aren't practical to reproduce
// portably.
var createTempFile = func(dir, pattern string) (tempWriteCloser, error) {
	return os.CreateTemp(dir, pattern)
}

// SaveToolsHash atomically records a successfully validated tools
// configuration, avoiding a partially written cache after interruption.
func (r *RealGitRepository) SaveToolsHash(hash string) error {
	gitDir, err := findGitDir()
	if err != nil {
		return err
	}

	statePath := filepath.Join(gitDir, toolsHashRelativePath)
	stateDir := filepath.Dir(statePath)
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		return err
	}

	tempFile, err := createTempFile(stateDir, "tools-*.tmp")
	if err != nil {
		return err
	}
	tempPath := tempFile.Name()
	defer os.Remove(tempPath)

	if _, err := tempFile.Write([]byte(hash + "\n")); err != nil {
		tempFile.Close()
		return err
	}
	if err := tempFile.Chmod(0644); err != nil {
		tempFile.Close()
		return err
	}
	if err := tempFile.Close(); err != nil {
		return err
	}

	return os.Rename(tempPath, statePath)
}

// getwd is os.Getwd, swappable in tests to exercise findGitDir's error path
// (os.Getwd failing is otherwise not reliably reproducible across
// platforms — e.g. macOS still resolves a deleted cwd).
var getwd = os.Getwd

func findGitDir() (string, error) {
	path, err := getwd()
	if err != nil {
		return "", err
	}

	for {
		gitDir := filepath.Join(path, ".git")
		if info, err := os.Stat(gitDir); err == nil && info.IsDir() {
			return gitDir, nil
		}

		parent := filepath.Dir(path)
		if parent == path {
			return "", fmt.Errorf(".git directory not found")
		}
		path = parent
	}
}
