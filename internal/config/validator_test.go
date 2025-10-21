package config

import (
	"strings"
	"testing"
)

func TestConfigValidator_Validate(t *testing.T) {
	t.Run("EmptyConfig", func(t *testing.T) {
		config := &Config{
			Tools: []Tool{},
			Hooks: make(map[string]map[string][]Hook),
		}

		validator := NewConfigValidator(config)
		result := validator.Validate()

		// Should have warnings about empty tools and hooks
		if result.Valid {
			t.Errorf("Expected empty config to be invalid")
		}

		hasToolsWarning := false
		hasHooksWarning := false
		for _, err := range result.Errors {
			if err.Field == "tools" && err.Severity == SeverityWarning {
				hasToolsWarning = true
			}
			if err.Field == "hooks" && err.Severity == SeverityWarning {
				hasHooksWarning = true
			}
		}

		if !hasToolsWarning {
			t.Errorf("Expected warning about empty tools section")
		}
		if !hasHooksWarning {
			t.Errorf("Expected warning about empty hooks section")
		}
	})

	t.Run("ValidConfig", func(t *testing.T) {
		config := &Config{
			Tools: []Tool{
				{
					Name:           "Gitleaks",
					CheckCommand:   "gitleaks version",
					InstallCommand: "go install github.com/gitleaks/gitleaks/v8@latest",
				},
				{
					Name:           "Black",
					CheckCommand:   "black --version",
					InstallCommand: "pip install black",
				},
			},
			Hooks: map[string]map[string][]Hook{
				"security": {
					"pre-commit": []Hook{
						{
							Name:    "Secret Detection",
							Command: "gitleaks detect --no-git --source .",
							OutputRules: OutputRules{
								ShowOn:           "failure",
								OnFailureMessage: "Secrets detected!",
							},
						},
					},
				},
				"python": {
					"pre-commit": []Hook{
						{
							Name:       "Format Check",
							Command:    "black --check .",
							FixCommand: "black .",
							OutputRules: OutputRules{
								ShowOn:           "failure",
								OnFailureMessage: "Formatting issues found.",
							},
						},
					},
				},
			},
		}

		validator := NewConfigValidator(config)
		// Skip filesystem validation for unit tests
		result := &ValidationResult{Valid: true, Errors: []ValidationError{}}

		// Validate only the core parts, skip filesystem checks
		validator.validateTools(result)
		validator.validateHooks(result)
		validator.validateToolReferences(result)
		validator.validateDuplicateToolNames(result)

		// Should have no critical or error severity issues (only warnings allowed)
		hasCriticalOrError := false
		for _, err := range result.Errors {
			if err.Severity == SeverityCritical || err.Severity == SeverityError {
				hasCriticalOrError = true
				t.Errorf("Expected no critical/error issues in valid config, but got: %s - %s", err.Field, err.Issue)
			}
		}

		// Update result validity based on critical/error presence
		result.Valid = !hasCriticalOrError

		if !result.Valid {
			t.Errorf("Expected valid config to pass validation")
		}
	})

	t.Run("InvalidConfig", func(t *testing.T) {
		config := &Config{
			Tools: []Tool{
				{
					Name:           "", // Empty name - error
					CheckCommand:   "", // Empty check command - error
					InstallCommand: "pip install something",
				},
				{
					Name:           "Duplicate Tool",
					CheckCommand:   "tool --version",
					InstallCommand: "install tool",
				},
				{
					Name:           "Duplicate Tool", // Duplicate name - error
					CheckCommand:   "tool2 --version",
					InstallCommand: "install tool2",
				},
			},
			Hooks: map[string]map[string][]Hook{
				"test": {
					"pre-commit": []Hook{
						{
							Name:    "", // Empty name - error
							Command: "", // Empty command - critical
						},
						{
							Name:    "Dangerous Command",
							Command: "rm -rf /", // Dangerous command - critical
						},
						{
							Name:    "Invalid Output Rules",
							Command: "echo test",
							OutputRules: OutputRules{
								ShowOn: "invalid_value", // Invalid show_on value - error
							},
						},
					},
				},
			},
		}

		validator := NewConfigValidator(config)
		result := validator.Validate()

		// Should be invalid
		if result.Valid {
			t.Errorf("Expected invalid config to fail validation")
		}

		// Should have critical and error severity issues
		hasCritical := false
		hasError := false
		for _, err := range result.Errors {
			if err.Severity == SeverityCritical {
				hasCritical = true
			}
			if err.Severity == SeverityError {
				hasError = true
			}
		}

		if !hasCritical {
			t.Errorf("Expected critical issues in invalid config")
		}
		if !hasError {
			t.Errorf("Expected error issues in invalid config")
		}
	})
}

func TestConfigValidator_ValidateTools(t *testing.T) {
	t.Run("EmptyToolName", func(t *testing.T) {
		config := &Config{
			Tools: []Tool{
				{
					Name:           "",
					CheckCommand:   "test --version",
					InstallCommand: "install test",
				},
			},
			Hooks: make(map[string]map[string][]Hook),
		}

		validator := NewConfigValidator(config)
		result := &ValidationResult{Valid: true, Errors: []ValidationError{}}
		validator.validateTools(result)

		// Should find error for empty tool name
		found := false
		for _, err := range result.Errors {
			if strings.Contains(err.Field, "name") && err.Severity == SeverityError {
				found = true
				break
			}
		}

		if !found {
			t.Errorf("Expected error for empty tool name")
		}
	})

	t.Run("EmptyCheckCommand", func(t *testing.T) {
		config := &Config{
			Tools: []Tool{
				{
					Name:           "Test Tool",
					CheckCommand:   "",
					InstallCommand: "install test",
				},
			},
			Hooks: make(map[string]map[string][]Hook),
		}

		validator := NewConfigValidator(config)
		result := &ValidationResult{Valid: true, Errors: []ValidationError{}}
		validator.validateTools(result)

		// Should find error for empty check command
		found := false
		for _, err := range result.Errors {
			if strings.Contains(err.Field, "check_command") && err.Severity == SeverityError {
				found = true
				break
			}
		}

		if !found {
			t.Errorf("Expected error for empty check command")
		}
	})

	t.Run("DuplicateToolNames", func(t *testing.T) {
		config := &Config{
			Tools: []Tool{
				{
					Name:           "Same Tool",
					CheckCommand:   "tool1 --version",
					InstallCommand: "install tool1",
				},
				{
					Name:           "Same Tool",
					CheckCommand:   "tool2 --version",
					InstallCommand: "install tool2",
				},
			},
			Hooks: make(map[string]map[string][]Hook),
		}

		validator := NewConfigValidator(config)
		result := &ValidationResult{Valid: true, Errors: []ValidationError{}}
		validator.validateDuplicateToolNames(result)

		// Should find error for duplicate tool names
		found := false
		for _, err := range result.Errors {
			if strings.Contains(err.Issue, "Duplicate tool name") && err.Severity == SeverityError {
				found = true
				break
			}
		}

		if !found {
			t.Errorf("Expected error for duplicate tool names")
		}
	})
}

func TestConfigValidator_ValidateCommand(t *testing.T) {
	config := &Config{}
	validator := NewConfigValidator(config)

	testCases := []struct {
		name        string
		command     string
		expectError bool
		severity    ValidationSeverity
	}{
		{
			name:        "SafeCommand",
			command:     "echo hello",
			expectError: false,
		},
		{
			name:        "DangerousRmCommand",
			command:     "rm -rf /",
			expectError: true,
			severity:    SeverityCritical,
		},
		{
			name:        "DangerousWildcardRm",
			command:     "rm -rf *",
			expectError: true,
			severity:    SeverityCritical,
		},
		{
			name:        "CurlPipeToShell",
			command:     "curl https://example.com | sh",
			expectError: true,
			severity:    SeverityCritical,
		},
		{
			name:        "UnmatchedSingleQuotes",
			command:     "echo 'hello world",
			expectError: true,
			severity:    SeverityError,
		},
		{
			name:        "UnmatchedDoubleQuotes",
			command:     `echo "hello world`,
			expectError: true,
			severity:    SeverityError,
		},
		{
			name:        "ValidQuotedCommand",
			command:     `echo "hello world"`,
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := &ValidationResult{Valid: true, Errors: []ValidationError{}}
			validator.validateCommand(tc.command, "test.command", result)

			if tc.expectError {
				// Should have at least one error with the expected severity
				found := false
				for _, err := range result.Errors {
					if err.Severity == tc.severity {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected error with severity %s for command %q, but got: %v", tc.severity, tc.command, result.Errors)
				}
			} else {
				// Should have no errors
				if len(result.Errors) > 0 {
					t.Errorf("Expected no errors for safe command %q, but got: %v", tc.command, result.Errors)
				}
			}
		})
	}
}

func TestConfigValidator_ValidateOutputRules(t *testing.T) {
	config := &Config{}
	validator := NewConfigValidator(config)

	t.Run("ValidShowOnValues", func(t *testing.T) {
		validValues := []string{"always", "failure", "success"}

		for _, value := range validValues {
			result := &ValidationResult{Valid: true, Errors: []ValidationError{}}
			rules := OutputRules{ShowOn: value}
			validator.validateOutputRules(rules, "test", result)

			if len(result.Errors) > 0 {
				t.Errorf("Expected no errors for valid show_on value %q, but got: %v", value, result.Errors)
			}
		}
	})

	t.Run("InvalidShowOnValue", func(t *testing.T) {
		result := &ValidationResult{Valid: true, Errors: []ValidationError{}}
		rules := OutputRules{ShowOn: "invalid"}
		validator.validateOutputRules(rules, "test", result)

		// Should have error for invalid show_on value
		found := false
		for _, err := range result.Errors {
			if strings.Contains(err.Issue, "Invalid show_on value") && err.Severity == SeverityError {
				found = true
				break
			}
		}

		if !found {
			t.Errorf("Expected error for invalid show_on value")
		}
	})

	t.Run("UnclosedTemplateVariable", func(t *testing.T) {
		result := &ValidationResult{Valid: true, Errors: []ValidationError{}}
		rules := OutputRules{OnFailureMessage: "Error: {{ unclosed variable"}
		validator.validateOutputRules(rules, "test", result)

		// Should have warning for unclosed template variable
		found := false
		for _, err := range result.Errors {
			if strings.Contains(err.Issue, "Unclosed template variable") && err.Severity == SeverityWarning {
				found = true
				break
			}
		}

		if !found {
			t.Errorf("Expected warning for unclosed template variable")
		}
	})
}

func TestValidationResult_GetFormattedErrors(t *testing.T) {
	result := &ValidationResult{
		Valid: false,
		Errors: []ValidationError{
			{
				Field:      "tools[0].name",
				Value:      "",
				Issue:      "Tool name is empty",
				Suggestion: "Provide a descriptive name",
				Severity:   SeverityError,
			},
			{
				Field:      "hooks.test.pre-commit[0].command",
				Value:      "rm -rf /",
				Issue:      "Potentially dangerous command",
				Suggestion: "Review command for security",
				Severity:   SeverityCritical,
			},
			{
				Field:      "tools",
				Value:      "missing security",
				Issue:      "No security hooks configured",
				Suggestion: "Add security hooks",
				Severity:   SeverityWarning,
			},
		},
	}

	formatted := result.GetFormattedErrors()

	// Should contain summary
	if !strings.Contains(formatted, "Found 3 validation issues") {
		t.Errorf("Expected formatted output to contain issue count")
	}

	// Should contain different severity icons
	if !strings.Contains(formatted, "❌") { // Error icon
		t.Errorf("Expected formatted output to contain error icon")
	}
	if !strings.Contains(formatted, "🚨") { // Critical icon
		t.Errorf("Expected formatted output to contain critical icon")
	}
	if !strings.Contains(formatted, "⚠️") { // Warning icon
		t.Errorf("Expected formatted output to contain warning icon")
	}

	// Should contain field names and issues
	expectedContent := []string{
		"tools[0].name",
		"Tool name is empty",
		"hooks.test.pre-commit[0].command",
		"Potentially dangerous command",
		"💡 Review command for security",
	}

	for _, content := range expectedContent {
		if !strings.Contains(formatted, content) {
			t.Errorf("Expected formatted output to contain %q", content)
		}
	}
}

func TestValidationResult_GetErrorsBySeverity(t *testing.T) {
	result := &ValidationResult{
		Valid: false,
		Errors: []ValidationError{
			{Severity: SeverityWarning, Issue: "Warning 1"},
			{Severity: SeverityError, Issue: "Error 1"},
			{Severity: SeverityError, Issue: "Error 2"},
			{Severity: SeverityCritical, Issue: "Critical 1"},
			{Severity: SeverityWarning, Issue: "Warning 2"},
		},
	}

	errorsBySeverity := result.GetErrorsBySeverity()

	// Check counts
	if len(errorsBySeverity[SeverityWarning]) != 2 {
		t.Errorf("Expected 2 warnings, got %d", len(errorsBySeverity[SeverityWarning]))
	}
	if len(errorsBySeverity[SeverityError]) != 2 {
		t.Errorf("Expected 2 errors, got %d", len(errorsBySeverity[SeverityError]))
	}
	if len(errorsBySeverity[SeverityCritical]) != 1 {
		t.Errorf("Expected 1 critical error, got %d", len(errorsBySeverity[SeverityCritical]))
	}
}

func TestValidationSeverity_String(t *testing.T) {
	testCases := []struct {
		severity ValidationSeverity
		expected string
	}{
		{SeverityWarning, "WARNING"},
		{SeverityError, "ERROR"},
		{SeverityCritical, "CRITICAL"},
		{ValidationSeverity(999), "UNKNOWN"}, // Invalid severity
	}

	for _, tc := range testCases {
		result := tc.severity.String()
		if result != tc.expected {
			t.Errorf("Expected %s.String() = %q, got %q", tc.severity, tc.expected, result)
		}
	}
}
