package service

import (
	"strings"
	"testing"
)

func TestTemplateGenerator_GenerateTemplate(t *testing.T) {
	generator := NewTemplateGenerator()

	t.Run("EmptyProject", func(t *testing.T) {
		structure := &ProjectStructure{
			Languages:  []Language{},
			Frameworks: []Language{},
			Tools:      []string{},
			Structure:  make(map[string][]string),
		}

		template := generator.GenerateTemplate(structure)

		// Should still include security tools
		if !strings.Contains(template, "Gitleaks") {
			t.Errorf("Expected Gitleaks to be included in empty project template")
		}

		if !strings.Contains(template, "security:") {
			t.Errorf("Expected security hooks to be included in empty project template")
		}
	})

	t.Run("GoProject", func(t *testing.T) {
		structure := &ProjectStructure{
			Languages:  []Language{LanguageGo},
			Frameworks: []Language{},
			Tools:      []string{},
			Structure:  make(map[string][]string),
		}

		template := generator.GenerateTemplate(structure)

		// Should include Go-specific tools
		expectedContent := []string{
			"Gofmt",
			"Golangci-lint",
			"gofmt -l .",
			"golangci-lint run ./...",
			"go test ./...",
			"go-backend:",
		}

		for _, content := range expectedContent {
			if !strings.Contains(template, content) {
				t.Errorf("Expected Go template to contain %q, but it was missing", content)
			}
		}
	})

	t.Run("PythonProject", func(t *testing.T) {
		structure := &ProjectStructure{
			Languages:  []Language{LanguagePython},
			Frameworks: []Language{},
			Tools:      []string{"ruff", "pytest"},
			Structure:  make(map[string][]string),
		}

		template := generator.GenerateTemplate(structure)

		// Should include Python-specific tools
		expectedContent := []string{
			"Ruff (Python Linter/Formatter)",
			"Black (Python Formatter)",
			"MyPy (Type Checker)",
			"ruff format",
			"ruff check",
			"pytest",
			"python-backend:",
		}

		for _, content := range expectedContent {
			if !strings.Contains(template, content) {
				t.Errorf("Expected Python template to contain %q, but it was missing", content)
			}
		}
	})

	t.Run("TypeScriptReactProject", func(t *testing.T) {
		structure := &ProjectStructure{
			Languages:  []Language{LanguageNode, LanguageTypeScript},
			Frameworks: []Language{LanguageReact},
			Tools:      []string{"prettier", "eslint"},
			Structure:  make(map[string][]string),
		}

		template := generator.GenerateTemplate(structure)

		// Should include Node.js/TypeScript-specific tools
		expectedContent := []string{
			"Prettier (Code Formatter)",
			"ESLint (Linter)",
			"npx prettier --check",
			"npx eslint",
			"npm test",
			"typescript-frontend:",
		}

		for _, content := range expectedContent {
			if !strings.Contains(template, content) {
				t.Errorf("Expected TypeScript React template to contain %q, but it was missing", content)
			}
		}

		// Should include TypeScript file patterns
		if !strings.Contains(template, "**/*.ts") || !strings.Contains(template, "**/*.tsx") {
			t.Errorf("Expected TypeScript file patterns to be included")
		}
	})

	t.Run("RustProject", func(t *testing.T) {
		structure := &ProjectStructure{
			Languages:  []Language{LanguageRust},
			Frameworks: []Language{},
			Tools:      []string{},
			Structure:  make(map[string][]string),
		}

		template := generator.GenerateTemplate(structure)

		// Should include Rust-specific tools and commands
		expectedContent := []string{
			"Rustfmt",
			"Clippy",
			"cargo fmt -- --check",
			"cargo clippy -- -D warnings",
			"cargo test",
			"rust-backend:",
		}

		for _, content := range expectedContent {
			if !strings.Contains(template, content) {
				t.Errorf("Expected Rust template to contain %q, but it was missing", content)
			}
		}
	})

	t.Run("PHPLaravelProject", func(t *testing.T) {
		structure := &ProjectStructure{
			Languages:  []Language{LanguagePHP},
			Frameworks: []Language{LanguageLaravel},
			Tools:      []string{"php-cs-fixer", "phpstan"},
			Structure:  make(map[string][]string),
		}

		template := generator.GenerateTemplate(structure)

		// Should include PHP-specific tools
		expectedContent := []string{
			"PHP CS Fixer",
			"PHPStan",
			"php-cs-fixer fix --dry-run",
			"phpstan analyse",
			"phpunit",
			"php-backend:",
		}

		for _, content := range expectedContent {
			if !strings.Contains(template, content) {
				t.Errorf("Expected PHP Laravel template to contain %q, but it was missing", content)
			}
		}

		// Should include Laravel-specific hooks
		if strings.Contains(template, "laravel-backend:") {
			if !strings.Contains(template, "Laravel Pint") {
				t.Errorf("Expected Laravel Pint to be included for Laravel projects")
			}
		}
	})
}

func TestTemplateGenerator_GetLanguageTools(t *testing.T) {
	generator := NewTemplateGenerator()

	t.Run("GoTools", func(t *testing.T) {
		tools := generator.getLanguageTools(LanguageGo)

		expectedTools := []string{"Gofmt", "Golangci-lint"}
		for _, expectedTool := range expectedTools {
			found := false
			for _, tool := range tools {
				if strings.Contains(tool.Name, expectedTool) {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Expected tool %q to be in Go tools", expectedTool)
			}
		}
	})

	t.Run("PythonTools", func(t *testing.T) {
		tools := generator.getLanguageTools(LanguagePython)

		expectedTools := []string{"Ruff", "Black", "MyPy"}
		for _, expectedTool := range expectedTools {
			found := false
			for _, tool := range tools {
				if strings.Contains(tool.Name, expectedTool) {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Expected tool %q to be in Python tools", expectedTool)
			}
		}
	})

	t.Run("NodeTools", func(t *testing.T) {
		tools := generator.getLanguageTools(LanguageNode)

		expectedTools := []string{"Prettier", "ESLint"}
		for _, expectedTool := range expectedTools {
			found := false
			for _, tool := range tools {
				if strings.Contains(tool.Name, expectedTool) {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Expected tool %q to be in Node tools", expectedTool)
			}
		}
	})

	t.Run("RustTools", func(t *testing.T) {
		tools := generator.getLanguageTools(LanguageRust)

		expectedTools := []string{"Rustfmt", "Clippy"}
		for _, expectedTool := range expectedTools {
			found := false
			for _, tool := range tools {
				if strings.Contains(tool.Name, expectedTool) {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Expected tool %q to be in Rust tools", expectedTool)
			}
		}
	})
}

func TestTemplateGenerator_GenerateSecurityHooks(t *testing.T) {
	generator := NewTemplateGenerator()

	securityHook := generator.generateSecurityHooks()

	// Check hook name and description
	if securityHook.Name != "security" {
		t.Errorf("Expected security hook name to be 'security', got: %s", securityHook.Name)
	}

	if len(securityHook.Commands) == 0 {
		t.Errorf("Expected security hooks to have at least one command")
	}

	// Check for Gitleaks command
	found := false
	for _, cmd := range securityHook.Commands {
		if strings.Contains(cmd.Command, "gitleaks") {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("Expected security hooks to contain gitleaks command")
	}
}

func TestTemplateGenerator_FormatToolsSection(t *testing.T) {
	generator := NewTemplateGenerator()

	tools := []ToolTemplate{
		{
			Name:           "Test Tool",
			CheckCommand:   "test --version",
			InstallCommand: "npm install -g test",
		},
		{
			Name:           "Another Tool",
			CheckCommand:   "another --help",
			InstallCommand: "pip install another",
		},
	}

	formatted := generator.formatToolsSection(tools)

	// Check structure
	expectedLines := []string{
		"tools:",
		`  - name: "Test Tool"`,
		`    check_command: "test --version"`,
		`    install_command: "npm install -g test"`,
		`  - name: "Another Tool"`,
		`    check_command: "another --help"`,
		`    install_command: "pip install another"`,
	}

	for _, expectedLine := range expectedLines {
		if !strings.Contains(formatted, expectedLine) {
			t.Errorf("Expected formatted tools to contain %q, but it was missing.\nActual output:\n%s", expectedLine, formatted)
		}
	}
}

func TestTemplateGenerator_HasLanguage(t *testing.T) {
	generator := NewTemplateGenerator()

	languages := []Language{LanguageGo, LanguagePython, LanguageTypeScript}

	testCases := []struct {
		target   Language
		expected bool
	}{
		{LanguageGo, true},
		{LanguagePython, true},
		{LanguageTypeScript, true},
		{LanguageRust, false},
		{LanguageJava, false},
	}

	for _, tc := range testCases {
		result := generator.hasLanguage(tc.target, languages)
		if result != tc.expected {
			t.Errorf("hasLanguage(%v, %v) = %v, want %v", tc.target, languages, result, tc.expected)
		}
	}
}

func TestTemplateGenerator_GenerateGoHooks(t *testing.T) {
	generator := NewTemplateGenerator()

	hook := generator.generateGoHooks()

	// Check hook name
	if hook.Name != "go-backend" {
		t.Errorf("Expected Go hook name to be 'go-backend', got: %s", hook.Name)
	}

	// Check for expected commands
	expectedCommands := []string{
		"gofmt -l .",
		"golangci-lint run ./...",
		"go test ./...",
	}

	for _, expectedCmd := range expectedCommands {
		found := false
		for _, cmd := range hook.Commands {
			if strings.Contains(cmd.Command, expectedCmd) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected Go hooks to contain command %q", expectedCmd)
		}
	}

	// Check for fix commands
	gofmtFixFound := false
	for _, cmd := range hook.Commands {
		if cmd.FixCommand != "" && strings.Contains(cmd.FixCommand, "gofmt -w") {
			gofmtFixFound = true
			break
		}
	}
	if !gofmtFixFound {
		t.Errorf("Expected Go hooks to contain gofmt fix command")
	}
}