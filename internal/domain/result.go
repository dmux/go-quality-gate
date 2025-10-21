package domain

import "time"

// ExecutionResult represents the result of a hook execution.

type ExecutionResult struct {
	Hook     Hook
	Success  bool
	Output   string
	Duration time.Duration
}
