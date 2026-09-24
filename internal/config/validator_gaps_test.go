package config

import (
	"os"
	"path/filepath"
	"testing"
)

func findSeverity(errs []ValidationError, field string) (ValidationError, bool) {
	for _, e := range errs {
		if e.Field == field {
			return e, true
		}
	}
	return ValidationError{}, false
}

func TestValidateTools_EmptyInstallCommand(t *testing.T) {
	cfg := &Config{Tools: Tools{{Name: "Foo", CheckCommand: "foo --version"}}}
	validator := NewConfigValidator(cfg)
	result := &ValidationResult{Valid: true}

	validator.validateTools(result)

	if _, ok := findSeverity(result.Errors, "tools[0].install_command"); !ok {
		t.Errorf("expected a warning for empty install_command, got %+v", result.Errors)
	}
}

func TestValidateHooks_EmptyHookGroupName(t *testing.T) {
	cfg := &Config{Hooks: Hooks{"": {"pre-commit": []Hook{{Name: "x", Command: "echo hi"}}}}}
	validator := NewConfigValidator(cfg)
	result := &ValidationResult{Valid: true}

	validator.validateHooks(result)

	found := false
	for _, e := range result.Errors {
		if e.Issue == "Hook group name is empty" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected an error for an empty hook group name, got %+v", result.Errors)
	}
}

func TestValidateHooks_NoHookTypesConfigured(t *testing.T) {
	cfg := &Config{Hooks: Hooks{"security": {"pre-commit": []Hook{}}}}
	validator := NewConfigValidator(cfg)
	result := &ValidationResult{Valid: true}

	validator.validateHooks(result)

	found := false
	for _, e := range result.Errors {
		if e.Issue == "No hook types configured (pre-commit, pre-push)" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected a warning when every hook type has zero commands, got %+v", result.Errors)
	}
}

func TestValidateCommands_EmptySlice(t *testing.T) {
	validator := NewConfigValidator(&Config{})
	result := &ValidationResult{Valid: true}

	validator.validateCommands([]Hook{}, "hooks.security.pre-commit", result)

	found := false
	for _, e := range result.Errors {
		if e.Issue == "No commands configured for this hook" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected a warning for an empty commands slice, got %+v", result.Errors)
	}
}

func TestValidateToolNames_DetectsRealTypo(t *testing.T) {
	validator := NewConfigValidator(&Config{})
	result := &ValidationResult{Valid: true}

	validator.validateToolNames("esslint .", "test.command", result)

	found := false
	for _, e := range result.Errors {
		if e.Issue == "Possible typo: 'esslint' should be 'eslint'" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected a typo warning for 'esslint', got %+v", result.Errors)
	}
}

func TestValidateToolAvailability_SkipsComplexCommands(t *testing.T) {
	validator := NewConfigValidator(&Config{})
	result := &ValidationResult{Valid: true}

	validator.validateToolAvailability(Tool{Name: "Foo", CheckCommand: "/usr/bin/foo --version"}, "tools[0]", result)

	if len(result.Errors) != 0 {
		t.Errorf("expected an absolute-path check command to be skipped, got %+v", result.Errors)
	}
}

func TestCheckCommandToolReferences_NpxPrefix(t *testing.T) {
	validator := NewConfigValidator(&Config{})
	result := &ValidationResult{Valid: true}

	commands := []Hook{{Name: "Lint", Command: "npx eslint ."}}
	validator.checkCommandToolReferences(commands, map[string]bool{}, "hooks.frontend.pre-commit", result)

	found := false
	for _, e := range result.Errors {
		if e.Issue == "Command uses 'eslint' but no tool configuration found" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected npx's actual tool name (eslint) to be checked, got %+v", result.Errors)
	}
}

func TestCheckCommandToolReferences_MissingToolConfig(t *testing.T) {
	validator := NewConfigValidator(&Config{})
	result := &ValidationResult{Valid: true}

	commands := []Hook{{Name: "Tests", Command: "pytest"}}
	validator.checkCommandToolReferences(commands, map[string]bool{}, "hooks.backend.pre-commit", result)

	if len(result.Errors) != 1 {
		t.Fatalf("expected 1 warning for an unconfigured common tool, got %+v", result.Errors)
	}
}

func TestCheckCommandToolReferences_PipAuditMissingToolConfig(t *testing.T) {
	validator := NewConfigValidator(&Config{})
	result := &ValidationResult{Valid: true}

	commands := []Hook{{Name: "Audit", Command: "pip-audit -r requirements.txt --aliases"}}
	validator.checkCommandToolReferences(commands, map[string]bool{}, "hooks.python-backend.pre-push", result)

	found := false
	for _, e := range result.Errors {
		if e.Issue == "Command uses 'pip-audit' but no tool configuration found" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected a warning for an unconfigured pip-audit tool, got %+v", result.Errors)
	}
}

func TestCheckCommandToolReferences_PipAuditConfiguredIsSilent(t *testing.T) {
	validator := NewConfigValidator(&Config{})
	result := &ValidationResult{Valid: true}

	commands := []Hook{{Name: "Audit", Command: "pip-audit --aliases"}}
	validator.checkCommandToolReferences(commands, map[string]bool{"pip-audit": true}, "hooks.python-backend.pre-push", result)

	if len(result.Errors) != 0 {
		t.Errorf("expected no warning when pip-audit is configured, got %+v", result.Errors)
	}
}

func TestValidateEssentialHooks_SecurityHookPresent(t *testing.T) {
	cfg := &Config{Hooks: Hooks{"Security": {}}}
	validator := NewConfigValidator(cfg)
	result := &ValidationResult{Valid: true}

	validator.validateEssentialHooks(result)

	if len(result.Errors) != 0 {
		t.Errorf("expected no warning when a security hook group exists, got %+v", result.Errors)
	}
}

func TestValidateFileSystem_ReadableFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "quality.yml"), []byte("tools: []\n"), 0644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)

	validator := NewConfigValidator(&Config{})
	result := &ValidationResult{Valid: true}
	validator.validateFileSystem(result)

	if len(result.Errors) != 0 {
		t.Errorf("expected no file-system errors for a readable quality.yml, got %+v", result.Errors)
	}
}

func TestValidateFileSystem_UnreadableFile(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root: permission checks don't apply")
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "quality.yml")
	if err := os.WriteFile(path, []byte("tools: []\n"), 0000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(path, 0644) })
	t.Chdir(dir)

	validator := NewConfigValidator(&Config{})
	result := &ValidationResult{Valid: true}
	validator.validateFileSystem(result)

	found := false
	for _, e := range result.Errors {
		if e.Issue == "quality.yml is not readable" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected an error for an unreadable quality.yml, got %+v", result.Errors)
	}
}

func TestValidateFileSystem_MissingFile(t *testing.T) {
	t.Chdir(t.TempDir())

	validator := NewConfigValidator(&Config{})
	result := &ValidationResult{Valid: true}
	validator.validateFileSystem(result)

	found := false
	for _, e := range result.Errors {
		if e.Severity == SeverityCritical && e.Issue == "Cannot access quality.yml file" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected a critical error when quality.yml doesn't exist, got %+v", result.Errors)
	}
}

func TestHasCriticalOrErrorSeverity_WarningsOnlyReturnsFalse(t *testing.T) {
	validator := NewConfigValidator(&Config{})

	got := validator.hasCriticalOrErrorSeverity([]ValidationError{
		{Severity: SeverityWarning},
		{Severity: SeverityWarning},
	})

	if got {
		t.Error("expected false when only warnings are present")
	}
}

func TestGetFormattedErrors_NoErrors(t *testing.T) {
	result := &ValidationResult{Valid: true, Errors: []ValidationError{}}

	if got := result.GetFormattedErrors(); got != "✅ No validation errors found" {
		t.Errorf("unexpected message for zero errors: %q", got)
	}
}
