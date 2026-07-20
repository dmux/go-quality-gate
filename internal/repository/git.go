package repository

// GitRepository defines the interface for interacting with a git repository.

type GitRepository interface {
	InstallHook(hookType string, content string) error
}

// ToolStateRepository persists the tools configuration hash that was last
// successfully validated in the current repository.
type ToolStateRepository interface {
	LoadToolsHash() (string, error)
	SaveToolsHash(hash string) error
}
