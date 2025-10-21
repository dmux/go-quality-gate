package shell

import (
	"os"
	"os/exec"
)

// RealShellRunner is a real implementation of the ShellRunner interface.

type RealShellRunner struct{}

// Run implements the ShellRunner interface.

func (r *RealShellRunner) Run(command string) (string, error) {
	shell := getPreferredShell()
	cmd := exec.Command(shell, "-c", command)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

// getPreferredShell returns the preferred shell to use, falling back to bash if not found.

func getPreferredShell() string {
	if shell := os.Getenv("SHELL"); shell != "" {
		return shell
	}
	
	// Try common shells in order of preference
	shells := []string{"/bin/zsh", "/bin/bash", "/bin/sh"}
	for _, shell := range shells {
		if _, err := os.Stat(shell); err == nil {
			return shell
		}
	}
	
	// Default fallback
	return "bash"
}
