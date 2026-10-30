package service

import (
	"os"
	"path/filepath"
	"testing"
)

// writeFakeExecutable creates an executable file named `name` inside dir so
// exec.LookPath can find it when dir is the only entry on PATH.
func writeFakeExecutable(t *testing.T, dir, name string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte("#!/bin/sh\nexit 0\n"), 0755); err != nil {
		t.Fatal(err)
	}
}

func TestDetectRuntimes_InstalledBinary(t *testing.T) {
	dir := t.TempDir()
	writeFakeExecutable(t, dir, "go")
	t.Setenv("PATH", dir)

	structure := &ProjectStructure{Languages: []Language{LanguageGo}}
	checks := DetectRuntimes(structure)

	if len(checks) != 1 {
		t.Fatalf("expected 1 runtime check, got %d", len(checks))
	}
	if !checks[0].Installed {
		t.Errorf("expected go to be reported as installed")
	}
	if checks[0].Binary != "go" {
		t.Errorf("expected binary 'go', got %q", checks[0].Binary)
	}
}

func TestDetectRuntimes_MissingBinary(t *testing.T) {
	t.Setenv("PATH", t.TempDir())

	structure := &ProjectStructure{Languages: []Language{LanguagePython}}
	checks := DetectRuntimes(structure)

	if len(checks) != 1 {
		t.Fatalf("expected 1 runtime check, got %d", len(checks))
	}
	if checks[0].Installed {
		t.Errorf("expected python3 to be reported as missing")
	}
	if checks[0].InstallHint == "" {
		t.Errorf("expected a non-empty install hint for a missing runtime")
	}
}

func TestDetectRuntimes_FrameworksMapToBaseLanguage(t *testing.T) {
	t.Setenv("PATH", t.TempDir())

	structure := &ProjectStructure{Frameworks: []Language{LanguageReact}}
	checks := DetectRuntimes(structure)

	if len(checks) != 1 {
		t.Fatalf("expected 1 runtime check, got %d", len(checks))
	}
	if checks[0].Language != LanguageNode {
		t.Errorf("expected React to resolve to Node runtime, got %q", checks[0].Language)
	}
}

func TestDetectRuntimes_DedupesBaseLanguage(t *testing.T) {
	t.Setenv("PATH", t.TempDir())

	structure := &ProjectStructure{
		Languages:  []Language{LanguageNode, LanguageTypeScript},
		Frameworks: []Language{LanguageReact},
	}
	checks := DetectRuntimes(structure)

	if len(checks) != 1 {
		t.Fatalf("expected Node/TypeScript/React to dedupe to 1 runtime check, got %d", len(checks))
	}
	if checks[0].Language != LanguageNode {
		t.Errorf("expected base language Node, got %q", checks[0].Language)
	}
}

func TestDetectRuntimes_UnknownLanguageIsSkipped(t *testing.T) {
	t.Setenv("PATH", t.TempDir())

	structure := &ProjectStructure{Languages: []Language{LanguageDocker}}
	checks := DetectRuntimes(structure)

	if len(checks) != 0 {
		t.Fatalf("expected no runtime checks for a language with no runtime mapping, got %d", len(checks))
	}
}

func TestInstallHintForOS_Darwin(t *testing.T) {
	info := baseLanguageRuntimes[LanguageGo]
	hint := installHintForOS(info, "darwin")
	if hint != info.darwinHint {
		t.Errorf("expected darwin hint %q, got %q", info.darwinHint, hint)
	}
}

func TestInstallHintForOS_Windows(t *testing.T) {
	info := baseLanguageRuntimes[LanguageGo]
	hint := installHintForOS(info, "windows")
	if hint != info.windowsHint {
		t.Errorf("expected windows hint %q, got %q", info.windowsHint, hint)
	}
}

func TestInstallHintForOS_LinuxPicksFirstAvailablePackageManager(t *testing.T) {
	dir := t.TempDir()
	writeFakeExecutable(t, dir, "dnf")
	t.Setenv("PATH", dir)

	info := baseLanguageRuntimes[LanguageGo]
	hint := installHintForOS(info, "linux")

	want := "sudo dnf install -y golang"
	if hint != want {
		t.Errorf("expected %q, got %q", want, hint)
	}
}

func TestInstallHintForOS_LinuxPrefersEarlierManagerWhenBothPresent(t *testing.T) {
	dir := t.TempDir()
	writeFakeExecutable(t, dir, "apt-get")
	writeFakeExecutable(t, dir, "dnf")
	t.Setenv("PATH", dir)

	info := baseLanguageRuntimes[LanguageGo]
	hint := installHintForOS(info, "linux")

	want := "sudo apt-get install -y golang-go"
	if hint != want {
		t.Errorf("expected apt-get to take priority, got %q", hint)
	}
}

func TestInstallHintForOS_LinuxNoKnownPackageManager(t *testing.T) {
	t.Setenv("PATH", t.TempDir())

	info := baseLanguageRuntimes[LanguageGo]
	hint := installHintForOS(info, "linux")

	if hint == "" {
		t.Fatal("expected a non-empty fallback hint")
	}
	if hint == info.darwinHint || hint == info.windowsHint {
		t.Errorf("expected an honest fallback, not a guessed OS command: %q", hint)
	}
}

func TestInstallHintForOS_UnknownOS(t *testing.T) {
	info := baseLanguageRuntimes[LanguageGo]
	hint := installHintForOS(info, "plan9")

	if hint == "" {
		t.Fatal("expected a non-empty fallback hint for an unknown OS")
	}
}
