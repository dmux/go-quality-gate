package service

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type dirHooks struct{ dir string }

func (d *dirHooks) InstallHook(hookType, content string) error {
	return os.WriteFile(filepath.Join(d.dir, hookType), []byte(content), 0755)
}

func (d *dirHooks) HooksDir() (string, error) { return d.dir, nil }

func TestInstallHooks_WritesManagedHooks(t *testing.T) {
	hooks := &dirHooks{dir: t.TempDir()}
	if err := NewInstallationService(hooks).InstallHooks(); err != nil {
		t.Fatal(err)
	}

	for _, hook := range ManagedHooks {
		content, err := os.ReadFile(filepath.Join(hooks.dir, hook))
		if err != nil {
			t.Fatalf("%s not installed: %v", hook, err)
		}
		if !IsManagedHook(string(content)) {
			t.Errorf("%s lacks the managed marker", hook)
		}
	}
	msgHook, _ := os.ReadFile(filepath.Join(hooks.dir, "commit-msg"))
	if !strings.Contains(string(msgHook), `quality-gate commit-msg "$1"`) {
		t.Errorf("commit-msg hook does not pass the message file: %q", msgHook)
	}
}

func TestGlobalHookContent_OnlyActsWithConfig(t *testing.T) {
	content := GlobalHookContent("pre-commit")
	if !strings.Contains(content, "quality.yml") || !IsManagedHook(content) {
		t.Fatalf("global hook must be managed and guarded by quality.yml: %q", content)
	}
}

func TestDoctor_ReportsMissingAndTamperedHooks(t *testing.T) {
	hooks := &dirHooks{dir: t.TempDir()}
	if err := os.WriteFile(filepath.Join(hooks.dir, "pre-commit"), []byte(HookContent("pre-commit")), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(hooks.dir, "commit-msg"), []byte("#!/bin/sh\nexit 0\n"), 0755); err != nil {
		t.Fatal(err)
	}

	notFound := func(string) (string, error) { return "", errors.New("not found") }
	checks := NewDoctorService(hooks, notFound, filepath.Join(t.TempDir(), "quality.yml")).Run()

	got := map[string]bool{}
	for _, c := range checks {
		got[c.Name] = c.OK
	}
	want := map[string]bool{
		"binary on PATH":  false,
		"pre-commit hook": true,
		"commit-msg hook": false,
		"pre-push hook":   false,
	}
	for name, ok := range want {
		if got[name] != ok {
			t.Errorf("%s: ok=%v, want %v", name, got[name], ok)
		}
	}
	if last := checks[len(checks)-1]; last.OK {
		t.Error("missing quality.yml must fail the config check")
	}
}
