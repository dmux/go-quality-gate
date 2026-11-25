package domain

import "time"

// ExecutionResult represents the result of a hook execution.

type ExecutionResult struct {
	Hook     Hook
	Success  bool
	Output   string
	Duration time.Duration
	// Skipped is set when the hook did not run because no changed file
	// matched it. A skipped hook counts as passed.
	Skipped    bool
	SkipReason string
}
