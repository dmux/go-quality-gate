package service

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
)

// QualityTemplate represents a quality.yml template for a specific stack
type QualityTemplate struct {
	Tools []ToolTemplate `yaml:"tools"`
	Hooks []HookTemplate `yaml:"hooks"`
}

// ToolTemplate represents a tool configuration in the template
type ToolTemplate struct {
	Name           string `yaml:"name"`
	CheckCommand   string `yaml:"check_command"`
	InstallCommand string `yaml:"install_command"`
}

// HookTemplate represents a hook configuration in the template
type HookTemplate struct {
	Name        string                 `yaml:"name"`
	Commands    []CommandTemplate      `yaml:"commands"`
	Description string                 `yaml:"description,omitempty"`
	Context     map[string]interface{} `yaml:"context,omitempty"`
}

// CommandTemplate represents a command in a hook
type CommandTemplate struct {
	Name             string            `yaml:"name"`
	Command          string            `yaml:"command"`
	FixCommand       string            `yaml:"fix_command,omitempty"`
	OutputRules      map[string]string `yaml:"output_rules,omitempty"`
	WorkingDirectory string            `yaml:"working_directory,omitempty"`
	RequiredFiles    []string          `yaml:"required_files,omitempty"`
	// HookTypes lists the hook sections this command is emitted under
	// (e.g. pre-commit, pre-push). Empty means pre-commit only. This field
	// drives template grouping and is never written to the YAML itself.
	HookTypes []string `yaml:"-"`
}

// TemplateGenerator generates quality.yml content based on detected project structure
type TemplateGenerator struct {
	// projectPath is the absolute root the detector walked. It is used to
	// relativize file paths when building audit commands. Empty means paths
	// are already repo-relative (unit-test fixtures).
	projectPath string
}

// NewTemplateGenerator creates a new template generator
func NewTemplateGenerator() *TemplateGenerator {
	return &TemplateGenerator{}
}

// NewTemplateGeneratorWithRoot creates a generator that relativizes paths
// detected under projectPath (absolute) back to the repository root.
func NewTemplateGeneratorWithRoot(projectPath string) *TemplateGenerator {
	return &TemplateGenerator{projectPath: projectPath}
}

// GenerateTemplate creates a quality.yml template based on project structure
func (g *TemplateGenerator) GenerateTemplate(structure *ProjectStructure) string {
	var sections []string

	// Generate tools section
	tools := g.generateTools(structure)
	if len(tools) > 0 {
		sections = append(sections, g.formatToolsSection(tools))
	}

	// Generate hooks section
	hooks := g.generateHooks(structure)
	if len(hooks) > 0 {
		sections = append(sections, g.formatHooksSection(hooks))
	}

	return strings.Join(sections, "\n\n")
}

// generateTools creates tool configurations based on detected languages
func (g *TemplateGenerator) generateTools(structure *ProjectStructure) []ToolTemplate {
	var tools []ToolTemplate
	seen := make(map[string]bool)

	// Always include Gitleaks for security
	tools = append(tools, ToolTemplate{
		Name:           "Gitleaks",
		CheckCommand:   "gitleaks version",
		InstallCommand: "go install github.com/zricethezav/gitleaks/v8@latest",
	})
	seen["gitleaks"] = true

	// Language-specific tools
	for _, lang := range structure.Languages {
		langTools := g.getLanguageTools(lang)
		for _, tool := range langTools {
			if !seen[strings.ToLower(tool.Name)] {
				tools = append(tools, tool)
				seen[strings.ToLower(tool.Name)] = true
			}
		}
	}

	// Framework-specific tools
	for _, framework := range structure.Frameworks {
		frameworkTools := g.getFrameworkTools(framework)
		for _, tool := range frameworkTools {
			if !seen[strings.ToLower(tool.Name)] {
				tools = append(tools, tool)
				seen[strings.ToLower(tool.Name)] = true
			}
		}
	}

	return tools
}

// generateHooks creates hook configurations based on detected languages
func (g *TemplateGenerator) generateHooks(structure *ProjectStructure) []HookTemplate {
	var hooks []HookTemplate
	seen := make(map[string]bool)

	appendHook := func(hook HookTemplate) {
		if len(hook.Commands) > 0 && !seen[hook.Name] {
			hooks = append(hooks, hook)
			seen[hook.Name] = true
		}
	}

	// Security hooks (always included)
	appendHook(g.generateSecurityHooks())

	// Language-specific hooks
	for _, lang := range structure.Languages {
		appendHook(g.generateLanguageHooks(lang, structure))
	}

	// Framework-specific hooks
	for _, framework := range structure.Frameworks {
		appendHook(g.generateFrameworkHooks(framework, structure))
	}

	return hooks
}

// Language-specific tool definitions
func (g *TemplateGenerator) getLanguageTools(lang Language) []ToolTemplate {
	switch lang {
	case LanguageGo:
		return []ToolTemplate{
			{
				Name:           "Gofmt",
				CheckCommand:   "gofmt -h",
				InstallCommand: "# gofmt is included with Go installation",
			},
			{
				Name:           "Golangci-lint",
				CheckCommand:   "golangci-lint --version",
				InstallCommand: "go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest",
			},
		}
	case LanguagePython:
		return []ToolTemplate{
			{
				Name:           "Ruff (Python Linter/Formatter)",
				CheckCommand:   "ruff --version",
				InstallCommand: "pip install ruff",
			},
			{
				Name:           "Black (Python Formatter)",
				CheckCommand:   "black --version",
				InstallCommand: "pip install black",
			},
			{
				Name:           "MyPy (Type Checker)",
				CheckCommand:   "mypy --version",
				InstallCommand: "pip install mypy",
			},
			{
				Name:           "Pip-Audit (Python Dependency Audit)",
				CheckCommand:   "pip-audit --version",
				InstallCommand: "pip install pip-audit",
			},
		}
	case LanguageNode, LanguageTypeScript:
		return []ToolTemplate{
			{
				Name:           "Prettier (Code Formatter)",
				CheckCommand:   "npx prettier --version",
				InstallCommand: "npm install --save-dev prettier",
			},
			{
				Name:           "ESLint (Linter)",
				CheckCommand:   "npx eslint --version",
				InstallCommand: "npm install --save-dev eslint",
			},
		}
	case LanguageRust:
		return []ToolTemplate{
			{
				Name:           "Rustfmt",
				CheckCommand:   "rustfmt --version",
				InstallCommand: "rustup component add rustfmt",
			},
			{
				Name:           "Clippy",
				CheckCommand:   "cargo clippy --version",
				InstallCommand: "rustup component add clippy",
			},
		}
	case LanguagePHP:
		return []ToolTemplate{
			{
				Name:           "PHP CS Fixer",
				CheckCommand:   "php-cs-fixer --version",
				InstallCommand: "composer global require friendsofphp/php-cs-fixer",
			},
			{
				Name:           "PHPStan",
				CheckCommand:   "phpstan --version",
				InstallCommand: "composer require --dev phpstan/phpstan",
			},
		}
	case LanguageJava:
		return []ToolTemplate{
			{
				Name:           "Checkstyle",
				CheckCommand:   "checkstyle --version",
				InstallCommand: "# Install via package manager or Maven/Gradle plugin",
			},
		}
	default:
		return []ToolTemplate{}
	}
}

// Framework-specific tool definitions
func (g *TemplateGenerator) getFrameworkTools(framework Language) []ToolTemplate {
	switch framework {
	case LanguageReact:
		return []ToolTemplate{
			{
				Name:           "React DevTools Lint",
				CheckCommand:   "npx eslint-plugin-react --version",
				InstallCommand: "npm install --save-dev eslint-plugin-react",
			},
		}
	case LanguageDjango:
		return []ToolTemplate{
			{
				Name:           "Django Check",
				CheckCommand:   "python manage.py check --help",
				InstallCommand: "# Django check is built-in",
			},
		}
	case LanguageLaravel:
		return []ToolTemplate{
			{
				Name:           "Laravel Pint",
				CheckCommand:   "pint --version",
				InstallCommand: "composer require laravel/pint --dev",
			},
		}
	default:
		return []ToolTemplate{}
	}
}

// generateSecurityHooks creates security-related hooks
func (g *TemplateGenerator) generateSecurityHooks() HookTemplate {
	return HookTemplate{
		Name:        "security",
		Description: "Security checks for all projects",
		Commands: []CommandTemplate{
			{
				Name:    "🔒 Secret Detection (Gitleaks)",
				Command: "gitleaks detect --no-git --source . --verbose",
				OutputRules: map[string]string{
					"on_failure_message": "⚠️  Secret leak detected! Review your code before committing.",
				},
			},
		},
	}
}

// generateLanguageHooks creates language-specific hooks
func (g *TemplateGenerator) generateLanguageHooks(lang Language, structure *ProjectStructure) HookTemplate {
	switch lang {
	case LanguageGo:
		return g.generateGoHooks()
	case LanguagePython:
		return g.generatePythonHooks(structure)
	case LanguageNode, LanguageTypeScript:
		return g.generateNodeHooks(structure)
	case LanguageRust:
		return g.generateRustHooks()
	case LanguagePHP:
		return g.generatePHPHooks(structure)
	default:
		return HookTemplate{}
	}
}

// generateFrameworkHooks creates framework-specific hooks
func (g *TemplateGenerator) generateFrameworkHooks(framework Language, structure *ProjectStructure) HookTemplate {
	switch framework {
	case LanguageReact:
		return g.generateReactHooks()
	case LanguageDjango:
		return g.generateDjangoHooks()
	case LanguageLaravel:
		return g.generateLaravelHooks()
	default:
		return HookTemplate{}
	}
}

// Language-specific hook implementations
func (g *TemplateGenerator) generateGoHooks() HookTemplate {
	return HookTemplate{
		Name:        "go-backend",
		Description: "Quality checks for Go projects",
		Commands: []CommandTemplate{
			{
				Name:       "🎨 Format Check (gofmt)",
				Command:    "gofmt -l .",
				FixCommand: "gofmt -w .",
				OutputRules: map[string]string{
					"show_on":            "failure",
					"on_failure_message": "Code formatting issues detected. Run './quality-gate --fix' to format.",
				},
			},
			{
				Name:    "🔍 Lint (golangci-lint)",
				Command: "golangci-lint run ./...",
				OutputRules: map[string]string{
					"show_on": "failure",
				},
			},
			{
				Name:    "🧪 Tests",
				Command: "go test ./...",
				OutputRules: map[string]string{
					"show_on": "always",
				},
			},
		},
	}
}

func (g *TemplateGenerator) generatePythonHooks(structure *ProjectStructure) HookTemplate {
	commands := []CommandTemplate{}

	// Check for specific directories
	pythonDirs := []string{".", "src", "app", "backend"}
	for dir := range structure.Structure {
		if strings.Contains(dir, "python") {
			pythonDirs = append(pythonDirs, dir)
		}
	}

	// Use the first existing directory or default to "."
	targetDir := "."
	for _, dir := range pythonDirs {
		// In a real implementation, you'd check if the directory exists
		targetDir = dir
		break
	}

	auditSource := g.detectPythonAuditSource(structure)
	auditCommand := "pip-audit"
	auditFix := "pip-audit --fix"
	if auditSource != "" {
		auditCommand += " " + auditSource
		auditFix += " " + auditSource
	}
	auditCommand += " --aliases"

	commands = append(commands,
		CommandTemplate{
			Name:       "🎨 Format Check (Ruff)",
			Command:    fmt.Sprintf("ruff format %s --check", targetDir),
			FixCommand: fmt.Sprintf("ruff format %s", targetDir),
			OutputRules: map[string]string{
				"show_on":            "failure",
				"on_failure_message": "Code formatting issues detected. Run './quality-gate --fix' to format.",
			},
		},
		CommandTemplate{
			Name:       "🔍 Lint (Ruff)",
			Command:    fmt.Sprintf("ruff check %s", targetDir),
			FixCommand: fmt.Sprintf("ruff check %s --fix", targetDir),
			OutputRules: map[string]string{
				"show_on":            "failure",
				"on_failure_message": "Lint issues detected. Run './quality-gate --fix' to apply autofixes.",
			},
		},
		CommandTemplate{
			Name:    "🧪 Tests (pytest)",
			Command: "pytest",
			OutputRules: map[string]string{
				"show_on": "always",
			},
		},
		CommandTemplate{
			Name:       "🛡️ Dependency Audit (pip-audit)",
			Command:    auditCommand,
			FixCommand: auditFix,
			HookTypes:  []string{"pre-commit", "pre-push"},
			OutputRules: map[string]string{
				"show_on":            "failure",
				"on_failure_message": "Vulnerable dependencies found! Run './quality-gate --fix' or upgrade manually.",
			},
		},
	)

	return HookTemplate{
		Name:        "python-backend",
		Description: "Quality checks for Python projects",
		Commands:    commands,
	}
}

func (g *TemplateGenerator) generateNodeHooks(structure *ProjectStructure) HookTemplate {
	commands := []CommandTemplate{}

	// Determine file patterns based on detected languages
	patterns := []string{"'**/*.js'"}
	if g.hasLanguage(LanguageTypeScript, structure.Languages) {
		patterns = append(patterns, "'**/*.ts'", "'**/*.tsx'")
	}
	if g.hasFramework(LanguageReact, structure.Frameworks) {
		patterns = append(patterns, "'**/*.jsx'")
	}

	patternStr := strings.Join(patterns, " ")

	commands = append(commands,
		CommandTemplate{
			Name:       "🎨 Format Check (Prettier)",
			Command:    fmt.Sprintf("npx prettier --check %s", patternStr),
			FixCommand: fmt.Sprintf("npx prettier --write %s", patternStr),
			OutputRules: map[string]string{
				"show_on":            "failure",
				"on_failure_message": "Code formatting issues detected. Run './quality-gate --fix' to format.",
			},
		},
		CommandTemplate{
			Name:    "🔍 Lint (ESLint)",
			Command: fmt.Sprintf("npx eslint %s", patternStr),
			OutputRules: map[string]string{
				"show_on": "failure",
			},
		},
		CommandTemplate{
			Name:    "🧪 Tests",
			Command: "npm test",
			OutputRules: map[string]string{
				"show_on": "always",
			},
		},
	)

	return HookTemplate{
		Name:        "typescript-frontend",
		Description: "Quality checks for Node.js/TypeScript projects",
		Commands:    commands,
	}
}

func (g *TemplateGenerator) generateRustHooks() HookTemplate {
	return HookTemplate{
		Name:        "rust-backend",
		Description: "Quality checks for Rust projects",
		Commands: []CommandTemplate{
			{
				Name:       "🎨 Format Check (rustfmt)",
				Command:    "cargo fmt -- --check",
				FixCommand: "cargo fmt",
				OutputRules: map[string]string{
					"show_on":            "failure",
					"on_failure_message": "Code formatting issues detected. Run './quality-gate --fix' to format.",
				},
			},
			{
				Name:    "🔍 Lint (Clippy)",
				Command: "cargo clippy -- -D warnings",
				OutputRules: map[string]string{
					"show_on": "failure",
				},
			},
			{
				Name:    "🧪 Tests",
				Command: "cargo test",
				OutputRules: map[string]string{
					"show_on": "always",
				},
			},
		},
	}
}

func (g *TemplateGenerator) generatePHPHooks(structure *ProjectStructure) HookTemplate {
	return HookTemplate{
		Name:        "php-backend",
		Description: "Quality checks for PHP projects",
		Commands: []CommandTemplate{
			{
				Name:       "🎨 Format Check (PHP CS Fixer)",
				Command:    "php-cs-fixer fix --dry-run --diff",
				FixCommand: "php-cs-fixer fix",
				OutputRules: map[string]string{
					"show_on":            "failure",
					"on_failure_message": "Code formatting issues detected. Run './quality-gate --fix' to format.",
				},
			},
			{
				Name:    "🔍 Static Analysis (PHPStan)",
				Command: "phpstan analyse",
				OutputRules: map[string]string{
					"show_on": "failure",
				},
			},
			{
				Name:    "🧪 Tests (PHPUnit)",
				Command: "phpunit",
				OutputRules: map[string]string{
					"show_on": "always",
				},
			},
		},
	}
}

// Framework-specific hook implementations
func (g *TemplateGenerator) generateReactHooks() HookTemplate {
	return HookTemplate{
		Name:        "react-frontend",
		Description: "Additional quality checks for React projects",
		Commands: []CommandTemplate{
			{
				Name:    "⚛️ React Lint",
				Command: "npx eslint --ext .jsx,.tsx .",
				OutputRules: map[string]string{
					"show_on": "failure",
				},
			},
		},
	}
}

func (g *TemplateGenerator) generateDjangoHooks() HookTemplate {
	return HookTemplate{
		Name:        "django-backend",
		Description: "Additional quality checks for Django projects",
		Commands: []CommandTemplate{
			{
				Name:    "🔍 Django Check",
				Command: "python manage.py check",
				OutputRules: map[string]string{
					"show_on": "always",
				},
			},
			{
				Name:    "🗄️ Migration Check",
				Command: "python manage.py makemigrations --dry-run --check",
				OutputRules: map[string]string{
					"show_on": "failure",
				},
			},
		},
	}
}

func (g *TemplateGenerator) generateLaravelHooks() HookTemplate {
	return HookTemplate{
		Name:        "laravel-backend",
		Description: "Additional quality checks for Laravel projects",
		Commands: []CommandTemplate{
			{
				Name:       "🎨 Format Check (Laravel Pint)",
				Command:    "pint --test",
				FixCommand: "pint",
				OutputRules: map[string]string{
					"show_on":            "failure",
					"on_failure_message": "Code formatting issues detected. Run './quality-gate --fix' to format.",
				},
			},
		},
	}
}

// Formatting functions
func (g *TemplateGenerator) formatToolsSection(tools []ToolTemplate) string {
	var lines []string
	lines = append(lines, "tools:")

	for _, tool := range tools {
		lines = append(lines, fmt.Sprintf("  - name: \"%s\"", tool.Name))
		lines = append(lines, fmt.Sprintf("    check_command: \"%s\"", tool.CheckCommand))
		lines = append(lines, fmt.Sprintf("    install_command: \"%s\"", tool.InstallCommand))
		lines = append(lines, "")
	}

	return strings.Join(lines, "\n")
}

func (g *TemplateGenerator) formatHooksSection(hooks []HookTemplate) string {
	var lines []string
	lines = append(lines, "hooks:")

	for _, hook := range hooks {
		lines = append(lines, fmt.Sprintf("  %s:", hook.Name))
		if hook.Description != "" {
			lines = append(lines, fmt.Sprintf("    # %s", hook.Description))
		}

		// Group commands by hook type, preserving first-appearance order of
		// both types and commands within each type.
		var hookTypes []string
		byType := make(map[string][]CommandTemplate)
		for _, cmd := range hook.Commands {
			for _, hookType := range commandHookTypes(cmd) {
				if _, seen := byType[hookType]; !seen {
					hookTypes = append(hookTypes, hookType)
				}
				byType[hookType] = append(byType[hookType], cmd)
			}
		}

		for _, hookType := range hookTypes {
			lines = append(lines, fmt.Sprintf("    %s:", hookType))

			for _, cmd := range byType[hookType] {
				lines = append(lines, fmt.Sprintf("      - name: \"%s\"", cmd.Name))
				lines = append(lines, fmt.Sprintf("        command: \"%s\"", cmd.Command))

				if cmd.FixCommand != "" {
					lines = append(lines, fmt.Sprintf("        fix_command: \"%s\"", cmd.FixCommand))
				}

				if len(cmd.OutputRules) > 0 {
					lines = append(lines, "        output_rules:")
					for key, value := range cmd.OutputRules {
						lines = append(lines, fmt.Sprintf("          %s: \"%s\"", key, value))
					}
				}

				lines = append(lines, "")
			}
		}
	}

	return strings.Join(lines, "\n")
}

// commandHookTypes returns the hook sections a command is emitted under,
// defaulting to pre-commit only when no explicit types are set.
func commandHookTypes(cmd CommandTemplate) []string {
	if len(cmd.HookTypes) == 0 {
		return []string{"pre-commit"}
	}
	return cmd.HookTypes
}

// Helper functions

// detectPythonAuditSource resolves the pip-audit source argument from the
// detected Python files: a requirements.txt wins (-r <relpath>), then a
// pyproject.toml (audit its directory), then fall back to auditing the
// current environment (empty source).
func (g *TemplateGenerator) detectPythonAuditSource(structure *ProjectStructure) string {
	var requirements, pyproject []string
	for _, file := range structure.Structure["python"] {
		switch filepath.Base(file) {
		case "requirements.txt":
			requirements = append(requirements, file)
		case "pyproject.toml":
			pyproject = append(pyproject, file)
		}
	}

	if len(requirements) > 0 {
		sort.Strings(requirements)
		return "-r " + g.relPath(requirements[0])
	}
	if len(pyproject) > 0 {
		sort.Strings(pyproject)
		dir := filepath.Dir(pyproject[0])
		if dir == "" || dir == "." {
			return "."
		}
		return g.relPath(dir)
	}
	return ""
}

// relPath converts a detected file path to a repository-relative path when
// it is absolute under the generator's project root; already-relative paths
// pass through unchanged.
func (g *TemplateGenerator) relPath(path string) string {
	if g.projectPath != "" && filepath.IsAbs(path) {
		if rel, err := filepath.Rel(g.projectPath, path); err == nil {
			return rel
		}
	}
	return filepath.ToSlash(path)
}

func (g *TemplateGenerator) hasLanguage(target Language, languages []Language) bool {
	for _, lang := range languages {
		if lang == target {
			return true
		}
	}
	return false
}

func (g *TemplateGenerator) hasFramework(target Language, frameworks []Language) bool {
	for _, framework := range frameworks {
		if framework == target {
			return true
		}
	}
	return false
}
