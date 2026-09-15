package shell

import (
	"errors"
	"os"
	"strings"
	"testing"
)

func TestRealShellRunner_Run_Success(t *testing.T) {
	runner := &RealShellRunner{}

	output, err := runner.Run("echo hello")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(output, "hello") {
		t.Errorf("expected output to contain %q, got %q", "hello", output)
	}
}

func TestRealShellRunner_Run_Failure(t *testing.T) {
	runner := &RealShellRunner{}

	output, err := runner.Run("exit 1")

	if err == nil {
		t.Fatal("expected an error for a failing command")
	}
	_ = output // combined output, may be empty for `exit 1`
}

func TestRealShellRunner_Run_CombinesStdoutAndStderr(t *testing.T) {
	runner := &RealShellRunner{}

	output, err := runner.Run("echo out && echo err 1>&2")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(output, "out") || !strings.Contains(output, "err") {
		t.Errorf("expected combined stdout+stderr output, got %q", output)
	}
}

func TestGetPreferredShell_UsesSHELLEnvVar(t *testing.T) {
	t.Setenv("SHELL", "/custom/shell")

	if got := getPreferredShell(); got != "/custom/shell" {
		t.Errorf("expected $SHELL to take precedence, got %q", got)
	}
}

func TestGetPreferredShell_FallsBackToKnownShell(t *testing.T) {
	t.Setenv("SHELL", "")

	orig := statFile
	statFile = func(name string) (os.FileInfo, error) {
		if name == "/bin/bash" {
			return nil, nil
		}
		return nil, errors.New("not found")
	}
	t.Cleanup(func() { statFile = orig })

	if got := getPreferredShell(); got != "/bin/bash" {
		t.Errorf("expected fallback to /bin/bash, got %q", got)
	}
}

func TestGetPreferredShell_DefaultsToBashWhenNoneFound(t *testing.T) {
	t.Setenv("SHELL", "")

	orig := statFile
	statFile = func(name string) (os.FileInfo, error) {
		return nil, errors.New("not found")
	}
	t.Cleanup(func() { statFile = orig })

	if got := getPreferredShell(); got != "bash" {
		t.Errorf("expected the final \"bash\" fallback, got %q", got)
	}
}
