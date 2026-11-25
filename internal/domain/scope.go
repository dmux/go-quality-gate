package domain

// Scope selects which files the hooks of a run analyze.
type Scope string

const (
	// ScopeChanged analyzes only the files changed by the commit or push.
	ScopeChanged Scope = "changed"
	// ScopeAll analyzes every tracked file.
	ScopeAll Scope = "all"
)
