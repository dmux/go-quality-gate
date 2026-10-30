package service

import (
	"errors"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// scriptedShellRunner returns canned output/errors for commands. A command
// matches an entry when it has the entry's key as a prefix, so callers can key
// on the meaningful part (e.g. "go install") without spelling out the full
// argument string.
type scriptedShellRunner struct {
	responses []scriptedResponse
	commands  []string
}

type scriptedResponse struct {
	match  string
	output string
	err    error
}

func (r *scriptedShellRunner) Run(command string) (string, error) {
	r.commands = append(r.commands, command)
	for _, resp := range r.responses {
		if strings.HasPrefix(command, resp.match) {
			return resp.output, resp.err
		}
	}
	return "", nil
}

const goVersionModOutput = "/home/user/go/bin/quality-gate: go1.25.5\n" +
	"\tpath\tgithub.com/dmux/go-quality-gate/cmd/quality-gate\n" +
	"\tmod\tgithub.com/dmux/go-quality-gate\tv1.5.0\th1:abc=\n"

func TestUpdate_Success_NewVersion(t *testing.T) {
	runner := &scriptedShellRunner{responses: []scriptedResponse{
		{match: "go version -m", output: goVersionModOutput},
		{match: "go env GOBIN", output: ""},
		{match: "go env GOPATH", output: "/home/user/go"},
	}}
	svc := NewUpdateService(runner, "v1.4.0")

	result, err := svc.Update()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.PreviousVersion != "v1.4.0" {
		t.Errorf("expected previous v1.4.0, got %q", result.PreviousVersion)
	}
	if result.NewVersion != "v1.5.0" {
		t.Errorf("expected new v1.5.0, got %q", result.NewVersion)
	}
	if !result.Updated {
		t.Error("expected Updated to be true")
	}

	// It must actually run the documented install command.
	var ranInstall bool
	for _, c := range runner.commands {
		if strings.HasPrefix(c, "go install "+UpdateInstallTarget) {
			ranInstall = true
		}
	}
	if !ranInstall {
		t.Errorf("expected `go install %s` to be run, got %v", UpdateInstallTarget, runner.commands)
	}
}

func TestUpdate_AlreadyUpToDate(t *testing.T) {
	runner := &scriptedShellRunner{responses: []scriptedResponse{
		{match: "go version -m", output: goVersionModOutput},
		{match: "go env GOPATH", output: "/home/user/go"},
	}}
	svc := NewUpdateService(runner, "v1.5.0")

	result, err := svc.Update()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Updated {
		t.Error("expected Updated to be false when versions match")
	}
	if result.NewVersion != "v1.5.0" {
		t.Errorf("expected new v1.5.0, got %q", result.NewVersion)
	}
}

func TestUpdate_UsesGOBINWhenSet(t *testing.T) {
	var versionCmd string
	runner := &scriptedShellRunner{responses: []scriptedResponse{
		{match: "go env GOBIN", output: "/opt/bin"},
		{match: "go version -m", output: goVersionModOutput},
	}}
	svc := NewUpdateService(runner, "v1.4.0")

	if _, err := svc.Update(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, c := range runner.commands {
		if strings.HasPrefix(c, "go version -m") {
			versionCmd = c
		}
	}
	name := updateBinaryName
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	want := fmt.Sprintf("%q", filepath.Join("/opt/bin", name))
	if !strings.Contains(versionCmd, want) {
		t.Errorf("expected version command to use GOBIN path %q, got %q", want, versionCmd)
	}
}

func TestUpdate_ToolchainMissing(t *testing.T) {
	runner := &scriptedShellRunner{responses: []scriptedResponse{
		{match: "go version", err: errors.New("command not found: go")},
	}}
	svc := NewUpdateService(runner, "v1.4.0")

	_, err := svc.Update()
	if err == nil {
		t.Fatal("expected an error when the Go toolchain is unavailable")
	}
	if !strings.Contains(err.Error(), "Go toolchain") {
		t.Errorf("expected a toolchain error, got %v", err)
	}
}

func TestUpdate_InstallFails(t *testing.T) {
	runner := &scriptedShellRunner{responses: []scriptedResponse{
		{match: "go install", output: "network is unreachable", err: errors.New("exit status 1")},
	}}
	svc := NewUpdateService(runner, "v1.4.0")

	_, err := svc.Update()
	if err == nil {
		t.Fatal("expected an error when `go install` fails")
	}
	if !strings.Contains(err.Error(), "go install") {
		t.Errorf("expected an install error, got %v", err)
	}
}

func TestUpdate_NoGoPathNoGoBin(t *testing.T) {
	runner := &scriptedShellRunner{} // all env lookups return ""
	svc := NewUpdateService(runner, "v1.4.0")

	_, err := svc.Update()
	if err == nil {
		t.Fatal("expected an error when neither GOBIN nor GOPATH is set")
	}
	if !strings.Contains(err.Error(), "GOBIN") {
		t.Errorf("expected a GOBIN/GOPATH error, got %v", err)
	}
}

func TestParseModuleVersion(t *testing.T) {
	if got := parseModuleVersion(goVersionModOutput); got != "v1.5.0" {
		t.Errorf("expected v1.5.0, got %q", got)
	}
	if got := parseModuleVersion("no module info here"); got != "" {
		t.Errorf("expected empty string for unrelated output, got %q", got)
	}
}
