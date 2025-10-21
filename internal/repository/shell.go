package repository

// ShellRunner defines the interface for running shell commands.

type ShellRunner interface {
	Run(command string) (string, error)
}
