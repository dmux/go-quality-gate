package git

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const toolsHashRelativePath = "quality-gate/tools.sha256"

// RealGitRepository is a real implementation of the GitRepository interface.

type RealGitRepository struct{}

// InstallHook implements the GitRepository interface.

func (r *RealGitRepository) InstallHook(hookType string, content string) error {
	gitDir, err := findGitDir()
	if err != nil {
		return err
	}

	hookPath := filepath.Join(gitDir, "hooks", hookType)

	f, err := os.Create(hookPath)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.WriteString(content)
	if err != nil {
		return err
	}

	return os.Chmod(hookPath, 0755)
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

	tempFile, err := os.CreateTemp(stateDir, "tools-*.tmp")
	if err != nil {
		return err
	}
	tempPath := tempFile.Name()
	defer os.Remove(tempPath)

	if _, err := tempFile.WriteString(hash + "\n"); err != nil {
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

func findGitDir() (string, error) {
	path, err := os.Getwd()
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
