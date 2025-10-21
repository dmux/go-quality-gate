package repository

// GitRepository defines the interface for interacting with a git repository.

type GitRepository interface {
	InstallHook(hookType string, content string) error
}
