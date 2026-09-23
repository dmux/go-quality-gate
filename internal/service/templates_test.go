package service

import (
	"strings"
	"testing"
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
		"Ruff", "Black", "MyPy", // Python
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

	if len(hook.Commands) != 3 {
		t.Fatalf("expected 3 python hook commands, got %d", len(hook.Commands))
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
