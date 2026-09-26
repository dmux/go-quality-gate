package service

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

type fakeHookInspector struct {
	dir string
	err error
}

func (f *fakeHookInspector) HooksDir() (string, error) {
	return f.dir, f.err
}

func findCheck(checks []DoctorCheck, name string) (DoctorCheck, bool) {
	for _, c := range checks {
		if c.Name == name {
			return c, true
		}
	}
	return DoctorCheck{}, false
}

func writeQualityYML(t *testing.T, dir, content string) string {
	t.Helper()
	path := filepath.Join(dir, "quality.yml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestDoctorService_Run_HooksDirUnavailable(t *testing.T) {
	dir := t.TempDir()
	writeQualityYML(t, dir, "tools: []\nhooks: {}\n")

	svc := NewDoctorService(&fakeHookInspector{err: errors.New("not a git repo")}, filepath.Join(dir, "quality.yml"))
	checks := svc.Run()

	check, ok := findCheck(checks, "hooks directory")
	if !ok {
		t.Fatal("expected a 'hooks directory' check")
	}
	if check.OK {
		t.Error("expected hooks directory check to fail")
	}
}

func TestDoctorService_Run_HooksNotInstalled(t *testing.T) {
	dir := t.TempDir()
	writeQualityYML(t, dir, "tools: []\nhooks: {}\n")

	svc := NewDoctorService(&fakeHookInspector{dir: t.TempDir()}, filepath.Join(dir, "quality.yml"))
	checks := svc.Run()

	check, ok := findCheck(checks, "pre-commit hook")
	if !ok {
		t.Fatal("expected a 'pre-commit hook' check")
	}
	if check.OK {
		t.Error("expected pre-commit hook to be reported as not installed")
	}
}

func TestDoctorService_Run_HooksInstalledAndValid(t *testing.T) {
	dir := t.TempDir()
	writeQualityYML(t, dir, "tools: []\nhooks: {}\n")

	binary := filepath.Join(t.TempDir(), "quality-gate")
	if err := os.WriteFile(binary, []byte("#!/bin/sh\n"), 0755); err != nil {
		t.Fatal(err)
	}

	hooksDir := t.TempDir()
	for _, hook := range ManagedHooks {
		content := HookContent(hook, binary)
		if err := os.WriteFile(filepath.Join(hooksDir, hook), []byte(content), 0755); err != nil {
			t.Fatal(err)
		}
	}

	svc := NewDoctorService(&fakeHookInspector{dir: hooksDir}, filepath.Join(dir, "quality.yml"))
	checks := svc.Run()

	for _, hook := range ManagedHooks {
		check, ok := findCheck(checks, hook+" hook")
		if !ok {
			t.Fatalf("expected a %q check", hook+" hook")
		}
		if !check.OK {
			t.Errorf("expected %s hook to be OK, got detail: %s", hook, check.Detail)
		}
	}
}

func TestDoctorService_Run_ConfigMissing(t *testing.T) {
	dir := t.TempDir()

	svc := NewDoctorService(&fakeHookInspector{dir: t.TempDir()}, filepath.Join(dir, "quality.yml"))
	checks := svc.Run()

	check, ok := findCheck(checks, filepath.Join(dir, "quality.yml"))
	if !ok {
		t.Fatal("expected a config path check")
	}
	if check.OK {
		t.Error("expected config check to fail for a missing file")
	}

	if _, ok := findCheck(checks, "go runtime"); ok {
		t.Error("expected no runtime checks when config fails to load")
	}
}

func TestDoctorService_Run_ConfigInvalid(t *testing.T) {
	dir := t.TempDir()
	path := writeQualityYML(t, dir, "settings:\n  tools_policy: bogus\ntools: []\nhooks: {}\n")

	svc := NewDoctorService(&fakeHookInspector{dir: t.TempDir()}, path)
	checks := svc.Run()

	check, ok := findCheck(checks, path)
	if !ok {
		t.Fatal("expected a config path check")
	}
	if check.OK {
		t.Error("expected config check to fail for an invalid tools_policy")
	}

	if _, ok := findCheck(checks, "go runtime"); ok {
		t.Error("expected no runtime checks when config is invalid")
	}
}

func TestDoctorService_Run_ToolAvailability(t *testing.T) {
	dir := t.TempDir()
	writeQualityYML(t, dir, `tools:
  - name: "Fake Tool"
    check_command: "fake-tool-binary --version"
    install_command: "install fake-tool-binary"
hooks: {}
`)
	// validateFileSystem's readability check stats "quality.yml" relative to
	// the process cwd rather than the configured path, so cwd must match.
	t.Chdir(dir)

	binDir := t.TempDir()
	t.Setenv("PATH", binDir)

	svc := NewDoctorService(&fakeHookInspector{dir: t.TempDir()}, "quality.yml")
	checks := svc.Run()

	check, ok := findCheck(checks, "Fake Tool tool")
	if !ok {
		t.Fatal("expected a 'Fake Tool tool' check")
	}
	if check.OK {
		t.Error("expected tool check to fail when the binary is missing")
	}

	writeFakeExecutable(t, binDir, "fake-tool-binary")

	checks = svc.Run()
	check, ok = findCheck(checks, "Fake Tool tool")
	if !ok {
		t.Fatal("expected a 'Fake Tool tool' check")
	}
	if !check.OK {
		t.Errorf("expected tool check to pass once the binary is on PATH, got detail: %s", check.Detail)
	}
}

func TestDoctorService_Run_RuntimeAvailability(t *testing.T) {
	dir := t.TempDir()
	writeQualityYML(t, dir, "tools: []\nhooks: {}\n")
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/x\n"), 0644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)

	t.Setenv("PATH", t.TempDir())

	svc := NewDoctorService(&fakeHookInspector{dir: t.TempDir()}, "quality.yml")
	checks := svc.Run()

	check, ok := findCheck(checks, "go runtime")
	if !ok {
		t.Fatal("expected a 'go runtime' check")
	}
	if check.OK {
		t.Error("expected go runtime check to fail when go is missing from PATH")
	}
}
