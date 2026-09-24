package service

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/dmux/go-quality-gate/internal/config"
)

func fullStructure() *ProjectStructure {
	return &ProjectStructure{
		Languages: []Language{
			LanguageGo, LanguagePython, LanguageNode, LanguageTypeScript,
			LanguageRust, LanguagePHP, LanguageJava, LanguageDocker,
		},
		Frameworks: []Language{LanguageReact, LanguageDjango, LanguageLaravel},
		Tools:      []string{},
		Structure: map[string][]string{
			"python": {"backend/requirements.txt"},
		},
	}
}

func TestGenerateTemplate_EmptyStructureStillIncludesSecurityBaseline(t *testing.T) {
	g := NewTemplateGenerator()

	out := g.GenerateTemplate(&ProjectStructure{})

	if !strings.Contains(out, "Gitleaks") {
		t.Errorf("expected Gitleaks to always be included, got:\n%s", out)
	}
	if !strings.Contains(out, "security") {
		t.Errorf("expected the security hook to always be included, got:\n%s", out)
	}
}

func TestGenerateTemplate_AllLanguagesAndFrameworks(t *testing.T) {
	g := NewTemplateGenerator()

	out := g.GenerateTemplate(fullStructure())

	// Language tool markers.
	for _, marker := range []string{
		"Gofmt", "Golangci-lint", // Go
		"Ruff", "Black", "MyPy", "Pip-Audit", // Python
		"Prettier", "ESLint", // Node/TypeScript
		"Rustfmt", "Clippy", // Rust
		"PHP CS Fixer", "PHPStan", // PHP
		"Checkstyle", // Java
	} {
		if !strings.Contains(out, marker) {
			t.Errorf("expected tool marker %q in generated template", marker)
		}
	}

	// Framework tool markers.
	for _, marker := range []string{"React DevTools Lint", "Django Check", "Laravel Pint"} {
		if !strings.Contains(out, marker) {
			t.Errorf("expected framework tool marker %q in generated template", marker)
		}
	}

	// Hook group markers.
	for _, marker := range []string{
		"go-backend", "python-backend", "typescript-frontend", "rust-backend", "php-backend",
		"react-frontend", "django-backend", "laravel-backend",
	} {
		if !strings.Contains(out, marker) {
			t.Errorf("expected hook group %q in generated template", marker)
		}
	}

	// Docker and Java have no dedicated hook generator (default empty
	// HookTemplate case) but must not break generation.
	if !strings.Contains(out, "tools:") || !strings.Contains(out, "hooks:") {
		t.Errorf("expected both tools: and hooks: sections, got:\n%s", out)
	}
}

func TestGenerateTemplate_DedupesToolsAcrossLanguagesAndFrameworks(t *testing.T) {
	g := NewTemplateGenerator()

	// Node + TypeScript both contribute Prettier/ESLint; ensure they appear
	// only once each (the `seen` map dedup branch).
	structure := &ProjectStructure{
		Languages: []Language{LanguageNode, LanguageTypeScript},
	}
	out := g.GenerateTemplate(structure)

	if count := strings.Count(out, "ESLint (Linter)"); count != 1 {
		t.Errorf("expected ESLint tool to appear exactly once, appeared %d times", count)
	}
}

func TestGetLanguageTools_UnknownLanguageReturnsEmpty(t *testing.T) {
	g := NewTemplateGenerator()

	tools := g.getLanguageTools(LanguageDocker)

	if len(tools) != 0 {
		t.Errorf("expected no tools for an unhandled language, got %+v", tools)
	}
}

func TestGetFrameworkTools_UnknownFrameworkReturnsEmpty(t *testing.T) {
	g := NewTemplateGenerator()

	tools := g.getFrameworkTools(LanguageVue)

	if len(tools) != 0 {
		t.Errorf("expected no tools for an unhandled framework, got %+v", tools)
	}
}

func TestGenerateLanguageHooks_UnknownLanguageReturnsEmptyTemplate(t *testing.T) {
	g := NewTemplateGenerator()

	hook := g.generateLanguageHooks(LanguageJava, &ProjectStructure{})

	if len(hook.Commands) != 0 {
		t.Errorf("expected no commands for a language with no hook generator, got %+v", hook)
	}
}

func TestGenerateFrameworkHooks_UnknownFrameworkReturnsEmptyTemplate(t *testing.T) {
	g := NewTemplateGenerator()

	hook := g.generateFrameworkHooks(LanguageVue, &ProjectStructure{})

	if len(hook.Commands) != 0 {
		t.Errorf("expected no commands for a framework with no hook generator, got %+v", hook)
	}
}

func TestGenerateNodeHooks_TypeScriptAndReactExpandPatterns(t *testing.T) {
	g := NewTemplateGenerator()
	structure := &ProjectStructure{
		Languages:  []Language{LanguageTypeScript},
		Frameworks: []Language{LanguageReact},
	}

	hook := g.generateNodeHooks(structure)

	formatCmd := hook.Commands[0].Command
	if !strings.Contains(formatCmd, "*.ts") || !strings.Contains(formatCmd, "*.tsx") {
		t.Errorf("expected TypeScript patterns in prettier command, got %q", formatCmd)
	}
	if !strings.Contains(formatCmd, "*.jsx") {
		t.Errorf("expected React's .jsx pattern in prettier command, got %q", formatCmd)
	}
}

func TestGenerateNodeHooks_PlainJavaScriptHasNoExtraPatterns(t *testing.T) {
	g := NewTemplateGenerator()
	structure := &ProjectStructure{Languages: []Language{LanguageNode}}

	hook := g.generateNodeHooks(structure)

	formatCmd := hook.Commands[0].Command
	if strings.Contains(formatCmd, "*.ts") || strings.Contains(formatCmd, "*.jsx") {
		t.Errorf("expected no TypeScript/React patterns for a plain Node project, got %q", formatCmd)
	}
}

func TestGeneratePythonHooks_UsesDetectedPythonDirectory(t *testing.T) {
	g := NewTemplateGenerator()
	structure := &ProjectStructure{
		Structure: map[string][]string{"python": {"backend/requirements.txt"}},
	}

	hook := g.generatePythonHooks(structure)

	if len(hook.Commands) != 4 {
		t.Fatalf("expected 4 python hook commands, got %d", len(hook.Commands))
	}

	audit := hook.Commands[3]
	if !strings.Contains(audit.Command, "pip-audit -r backend/requirements.txt --aliases") {
		t.Errorf("expected audit command with requirements source, got %q", audit.Command)
	}
	if !strings.Contains(audit.FixCommand, "pip-audit --fix -r backend/requirements.txt") {
		t.Errorf("expected fix command with requirements source, got %q", audit.FixCommand)
	}
	if len(audit.HookTypes) != 2 || audit.HookTypes[0] != "pre-commit" || audit.HookTypes[1] != "pre-push" {
		t.Errorf("expected audit on pre-commit and pre-push, got %v", audit.HookTypes)
	}

	lint := hook.Commands[1]
	if lint.FixCommand != "ruff check . --fix" {
		t.Errorf("expected lint autofix via ruff check --fix, got %q", lint.FixCommand)
	}
}

func TestDetectPythonAuditSource_RequirementsWins(t *testing.T) {
	g := NewTemplateGenerator()
	structure := &ProjectStructure{
		Structure: map[string][]string{
			"python": {"backend/pyproject.toml", "backend/requirements.txt"},
		},
	}

	if got := g.detectPythonAuditSource(structure); got != "-r backend/requirements.txt" {
		t.Errorf("expected requirements source, got %q", got)
	}
}

func TestDetectPythonAuditSource_PyprojectDirectory(t *testing.T) {
	g := NewTemplateGenerator()
	structure := &ProjectStructure{
		Structure: map[string][]string{"python": {"backend/pyproject.toml"}},
	}

	if got := g.detectPythonAuditSource(structure); got != "backend" {
		t.Errorf("expected pyproject directory source, got %q", got)
	}
}

func TestDetectPythonAuditSource_RootPyproject(t *testing.T) {
	g := NewTemplateGenerator()
	structure := &ProjectStructure{
		Structure: map[string][]string{"python": {"pyproject.toml"}},
	}

	if got := g.detectPythonAuditSource(structure); got != "." {
		t.Errorf("expected root project source, got %q", got)
	}
}

func TestDetectPythonAuditSource_NoSourceFallsBackToEnvironment(t *testing.T) {
	g := NewTemplateGenerator()
	structure := &ProjectStructure{
		Structure: map[string][]string{"python": {"setup.py"}},
	}

	if got := g.detectPythonAuditSource(structure); got != "" {
		t.Errorf("expected empty source for environment audit, got %q", got)
	}
}

func TestDetectPythonAuditSource_RelativizesAbsolutePaths(t *testing.T) {
	g := NewTemplateGeneratorWithRoot("/repo")
	structure := &ProjectStructure{
		Structure: map[string][]string{"python": {"/repo/backend/requirements.txt"}},
	}

	if got := g.detectPythonAuditSource(structure); got != "-r backend/requirements.txt" {
		t.Errorf("expected repo-relative requirements path, got %q", got)
	}
}

func TestFormatHooksSection_EmitsPrePushOnlyForAuditCommand(t *testing.T) {
	g := NewTemplateGenerator()
	hooks := []HookTemplate{
		{
			Name: "python-backend",
			Commands: []CommandTemplate{
				{Name: "Format", Command: "ruff format . --check"},
				{Name: "Audit", Command: "pip-audit --aliases", HookTypes: []string{"pre-commit", "pre-push"}},
			},
		},
	}

	out := g.formatHooksSection(hooks)

	if !strings.Contains(out, "pre-commit:") {
		t.Errorf("expected a pre-commit block, got:\n%s", out)
	}
	if !strings.Contains(out, "pre-push:") {
		t.Errorf("expected a pre-push block, got:\n%s", out)
	}
	if strings.Contains(out, "hook_types") {
		t.Errorf("HookTypes must not leak into the YAML output, got:\n%s", out)
	}

	// pre-push must contain the audit command only.
	prePushIdx := strings.Index(out, "pre-push:")
	if prePushIdx < 0 {
		t.Fatalf("missing pre-push block")
	}
	if !strings.Contains(out[prePushIdx:], "pip-audit") {
		t.Errorf("expected pip-audit in pre-push block, got:\n%s", out[prePushIdx:])
	}

	// Commands without explicit types stay pre-commit-only.
	preCommit := out[strings.Index(out, "pre-commit:"):prePushIdx]
	if !strings.Contains(preCommit, "ruff format") {
		t.Errorf("expected format command in pre-commit block, got:\n%s", preCommit)
	}
}

func TestFormatHooksSection_GroupWithoutPrePushHasNoPrePushBlock(t *testing.T) {
	g := NewTemplateGenerator()
	hooks := []HookTemplate{
		{
			Name:     "go-backend",
			Commands: []CommandTemplate{{Name: "Format", Command: "gofmt -l ."}},
		},
	}

	out := g.formatHooksSection(hooks)

	if strings.Contains(out, "pre-push:") {
		t.Errorf("expected no pre-push block for pre-commit-only commands, got:\n%s", out)
	}
}

func TestFormatHooksSection_OmitsFixCommandWhenAbsent(t *testing.T) {
	g := NewTemplateGenerator()
	hooks := []HookTemplate{
		{
			Name: "custom",
			Commands: []CommandTemplate{
				{Name: "Check", Command: "do-check"}, // no FixCommand, no OutputRules
			},
		},
	}

	out := g.formatHooksSection(hooks)

	if strings.Contains(out, "fix_command") {
		t.Errorf("expected no fix_command line when FixCommand is empty, got:\n%s", out)
	}
	if strings.Contains(out, "output_rules") {
		t.Errorf("expected no output_rules block when OutputRules is empty, got:\n%s", out)
	}
}

func TestGenerateTemplate_RoundTripParsesAndValidates(t *testing.T) {
	g := NewTemplateGenerator()
	out := g.GenerateTemplate(fullStructure())

	var cfg config.Config
	if err := yaml.Unmarshal([]byte(out), &cfg); err != nil {
		t.Fatalf("generated quality.yml does not parse: %v\n%s", err, out)
	}

	// validateFileSystem requires a quality.yml in the working directory.
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "quality.yml"), []byte(out), 0644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)

	result := config.NewConfigValidator(&cfg).Validate()
	if !result.Valid {
		t.Errorf("generated quality.yml fails validation:\n%s\n---\n%s", result.GetFormattedErrors(), out)
	}

	// The python group must expose the audit command in both hook types.
	py := cfg.Hooks["python-backend"]
	if len(py["pre-push"]) == 0 {
		t.Fatalf("expected python-backend.pre-push commands, hooks: %v", cfg.Hooks)
	}
	auditPush := py["pre-push"][0]
	if !strings.Contains(auditPush.Command, "pip-audit") || auditPush.FixCommand == "" {
		t.Errorf("expected pip-audit command with fix_command in pre-push, got %+v", auditPush)
	}
}
