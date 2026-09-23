package repository

// GitRepository defines the interface for interacting with a git repository.

type GitRepository interface {
	InstallHook(hookType string, content string) error
}

// HookInspector reads installed hooks so their integrity can be checked.
type HookInspector interface {
	HooksDir() (string, error)
}

// ToolStateRepository persists the tools configuration hash that was last
// successfully validated in the current repository.
type ToolStateRepository interface {
	LoadToolsHash() (string, error)
	SaveToolsHash(hash string) error
}

// CommitRepository exposes the commit data needed to create and verify
// quality gate attestations.
type CommitRepository interface {
	// WriteTree returns the tree hash of the current index, i.e. the content
	// the next commit will record.
	WriteTree() (string, error)
	// CommitTree returns the tree hash recorded by a commit.
	CommitTree(rev string) (string, error)
	// CommitMessage returns the full message of a commit.
	CommitMessage(rev string) (string, error)
	// ShowFile returns the content of a file at a given revision. A missing
	// file is returned as nil content and a nil error.
	ShowFile(rev, path string) ([]byte, error)
	// RevList returns the non-merge commits of a revision range, newest first.
	RevList(rangeSpec string) ([]string, error)
	// AddTrailer appends (or replaces) a trailer in a commit message file.
	AddTrailer(messageFile, trailer string) error
}

// AttestationStateRepository persists the attestation produced by a passing
// pre-commit run until the commit-msg hook consumes it.
type AttestationStateRepository interface {
	LoadAttestation() ([]byte, error)
	SaveAttestation(content []byte) error
	DeleteAttestation() error
}
