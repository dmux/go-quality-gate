package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	config, err := LoadConfig("testdata/quality.yml")
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if len(config.Tools) != 1 {
		t.Errorf("Expected 1 tool, got %d", len(config.Tools))
	}

	if config.Tools[0].Name != "Gitleaks" {
		t.Errorf("Expected tool name 'Gitleaks', got '%s'", config.Tools[0].Name)
	}

	if len(config.Hooks) != 1 {
		t.Errorf("Expected 1 hook group, got %d", len(config.Hooks))
	}

	hookGroup, ok := config.Hooks["security"]
	if !ok {
		t.Fatal("Expected to find 'security' hook group")
	}

	preCommitHooks, ok := hookGroup["pre-commit"]
	if !ok {
		t.Fatal("Expected to find 'pre-commit' hooks")
	}

	if len(preCommitHooks) != 1 {
		t.Errorf("Expected 1 pre-commit hook, got %d", len(preCommitHooks))
	}

	if preCommitHooks[0].Name != "🔒 Verificação de Segredos (Gitleaks)" {
		t.Errorf("Expected hook name '🔒 Verificação de Segredos (Gitleaks)', got '%s'", preCommitHooks[0].Name)
	}
}

func TestLoadConfig_FileNotFound(t *testing.T) {
	_, err := LoadConfig(filepath.Join(t.TempDir(), "does-not-exist.yml"))

	if err == nil {
		t.Fatal("expected an error for a missing config file")
	}
}

func TestLoadConfig_InvalidYAML(t *testing.T) {
	path := filepath.Join(t.TempDir(), "quality.yml")
	if err := os.WriteFile(path, []byte("tools: [this is not: valid: yaml"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadConfig(path)

	if err == nil {
		t.Fatal("expected an error for malformed YAML")
	}
}
