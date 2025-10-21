package git

import (
	"fmt"
	"os"
	"path/filepath"
)

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
