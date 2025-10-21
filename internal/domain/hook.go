package domain

// Hook represents a command to be executed as part of a git hook.

type Hook struct {
	Name        string
	Command     string
	FixCommand  string
	OutputRules OutputRules
}

// OutputRules defines how the output of a hook should be handled.

type OutputRules struct {
	ShowOn           string
	OnFailureMessage string
}
