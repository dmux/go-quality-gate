package config

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

// ValidationError represents a configuration validation error
type ValidationError struct {
	Field       string
	Value       string
	Issue       string
	Suggestion  string
	Severity    ValidationSeverity
}

// ValidationSeverity indicates the severity of a validation issue
type ValidationSeverity int

const (
	SeverityWarning ValidationSeverity = iota
	SeverityError
	SeverityCritical
)

func (s ValidationSeverity) String() string {
	switch s {
	case SeverityWarning:
		return "WARNING"
	case SeverityError:
		return "ERROR"
	case SeverityCritical:
		return "CRITICAL"
	default:
		return "UNKNOWN"
	}
}

// ValidationResult holds the result of configuration validation
type ValidationResult struct {
	Valid  bool
	Errors []ValidationError
}

// ConfigValidator validates quality.yml configurations
type ConfigValidator struct {
	config *Config
}

// NewConfigValidator creates a new configuration validator
func NewConfigValidator(config *Config) *ConfigValidator {
	return &ConfigValidator{
		config: config,
	}
}

// Validate performs comprehensive validation of the quality.yml configuration
func (v *ConfigValidator) Validate() *ValidationResult {
	result := &ValidationResult{
		Valid:  true,
		Errors: []ValidationError{},
	}

	// Validate tools section
	v.validateTools(result)

	// Validate hooks section
	v.validateHooks(result)

	// Validate cross-references between tools and hooks
	v.validateToolReferences(result)

	// Check for common configuration issues
	v.validateCommonIssues(result)

	// Set overall validity
	result.Valid = !v.hasCriticalOrErrorSeverity(result.Errors)

	return result
}

// validateTools validates the tools section of the configuration
func (v *ConfigValidator) validateTools(result *ValidationResult) {
	if len(v.config.Tools) == 0 {
		result.Errors = append(result.Errors, ValidationError{
			Field:      "tools",
			Value:      "empty",
			Issue:      "No tools configured",
			Suggestion: "Add at least one tool configuration for quality checks",
			Severity:   SeverityWarning,
		})
		return
	}

	for i, tool := range v.config.Tools {
		fieldPrefix := fmt.Sprintf("tools[%d]", i)

		// Validate tool name
		if strings.TrimSpace(tool.Name) == "" {
			result.Errors = append(result.Errors, ValidationError{
				Field:      fieldPrefix + ".name",
				Value:      tool.Name,
				Issue:      "Tool name is empty",
				Suggestion: "Provide a descriptive name for the tool",
				Severity:   SeverityError,
			})
		}

		// Validate check command
		if strings.TrimSpace(tool.CheckCommand) == "" {
			result.Errors = append(result.Errors, ValidationError{
				Field:      fieldPrefix + ".check_command",
				Value:      tool.CheckCommand,
				Issue:      "Check command is empty",
				Suggestion: "Provide a command to check if the tool is installed (e.g., 'tool --version')",
				Severity:   SeverityError,
			})
		} else {
			v.validateCommand(tool.CheckCommand, fieldPrefix+".check_command", result)
		}

		// Validate install command
		if strings.TrimSpace(tool.InstallCommand) == "" {
			result.Errors = append(result.Errors, ValidationError{
				Field:      fieldPrefix + ".install_command",
				Value:      tool.InstallCommand,
				Issue:      "Install command is empty",
				Suggestion: "Provide a command to install the tool",
				Severity:   SeverityWarning,
			})
		} else {
			v.validateCommand(tool.InstallCommand, fieldPrefix+".install_command", result)
		}

		// Test if tool is available (optional check)
		v.validateToolAvailability(tool, fieldPrefix, result)
	}
}

// validateHooks validates the hooks section of the configuration
func (v *ConfigValidator) validateHooks(result *ValidationResult) {
	if len(v.config.Hooks) == 0 {
		result.Errors = append(result.Errors, ValidationError{
			Field:      "hooks",
			Value:      "empty",
			Issue:      "No hooks configured",
			Suggestion: "Add at least one hook configuration (pre-commit, pre-push, etc.)",
			Severity:   SeverityWarning,
		})
		return
	}

	for hookName, hookGroup := range v.config.Hooks {
		fieldPrefix := fmt.Sprintf("hooks.%s", hookName)

		// Validate hook group name
		if strings.TrimSpace(hookName) == "" {
			result.Errors = append(result.Errors, ValidationError{
				Field:      fieldPrefix,
				Value:      hookName,
				Issue:      "Hook group name is empty",
				Suggestion: "Use descriptive names like 'security', 'backend', 'frontend'",
				Severity:   SeverityError,
			})
		}

		// Validate each hook type in the group
		hasAnyHooks := false
		for hookType, commands := range hookGroup {
			if len(commands) > 0 {
				hasAnyHooks = true
				v.validateCommands(commands, fmt.Sprintf("%s.%s", fieldPrefix, hookType), result)
			}
		}

		// Ensure at least one hook type is configured
		if !hasAnyHooks {
			result.Errors = append(result.Errors, ValidationError{
				Field:      fieldPrefix,
				Value:      "no hooks",
				Issue:      "No hook types configured (pre-commit, pre-push)",
				Suggestion: "Add at least one hook type with commands",
				Severity:   SeverityWarning,
			})
		}
	}
}

// validateCommands validates individual command configurations
func (v *ConfigValidator) validateCommands(commands []Hook, fieldPrefix string, result *ValidationResult) {
	if len(commands) == 0 {
		result.Errors = append(result.Errors, ValidationError{
			Field:      fieldPrefix,
			Value:      "empty",
			Issue:      "No commands configured for this hook",
			Suggestion: "Add at least one command for this hook",
			Severity:   SeverityWarning,
		})
		return
	}

	for i, cmd := range commands {
		cmdFieldPrefix := fmt.Sprintf("%s[%d]", fieldPrefix, i)

		// Validate command name
		if strings.TrimSpace(cmd.Name) == "" {
			result.Errors = append(result.Errors, ValidationError{
				Field:      cmdFieldPrefix + ".name",
				Value:      cmd.Name,
				Issue:      "Command name is empty",
				Suggestion: "Provide a descriptive name with emoji (e.g., '🎨 Format Check')",
				Severity:   SeverityError,
			})
		}

		// Validate main command
		if strings.TrimSpace(cmd.Command) == "" {
			result.Errors = append(result.Errors, ValidationError{
				Field:      cmdFieldPrefix + ".command",
				Value:      cmd.Command,
				Issue:      "Command is empty",
				Suggestion: "Provide the command to execute",
				Severity:   SeverityCritical,
			})
		} else {
			v.validateCommand(cmd.Command, cmdFieldPrefix+".command", result)
		}

		// Validate fix command (if present)
		if cmd.FixCommand != "" {
			v.validateCommand(cmd.FixCommand, cmdFieldPrefix+".fix_command", result)
		}

		// Validate output rules
		v.validateOutputRules(cmd.OutputRules, cmdFieldPrefix+".output_rules", result)
	}
}

// validateCommand validates individual command syntax and security
func (v *ConfigValidator) validateCommand(command, fieldPath string, result *ValidationResult) {
	// Check for potentially dangerous commands
	dangerousPatterns := []string{
		`rm\s+-rf\s+/`,           // Dangerous rm commands
		`rm\s+-rf\s+\*`,          // Wildcard deletion
		`sudo\s+rm`,              // Sudo deletion
		`>\s*/dev/sd[a-z]`,       // Writing to disk devices
		`dd\s+.*of=/dev`,         // DD to devices
		`curl.*\|\s*sh`,          // Piping curl to shell
		`wget.*\|\s*sh`,          // Piping wget to shell
		`eval\s+\$\(.*curl`,      // Eval with curl
		`:\(\)\{.*;\}:`,          // Fork bomb pattern
	}

	for _, pattern := range dangerousPatterns {
		if matched, _ := regexp.MatchString(pattern, command); matched {
			result.Errors = append(result.Errors, ValidationError{
				Field:      fieldPath,
				Value:      command,
				Issue:      "Potentially dangerous command detected",
				Suggestion: "Review the command for security implications",
				Severity:   SeverityCritical,
			})
			break
		}
	}

	// Check for common command issues
	v.validateCommandSyntax(command, fieldPath, result)
}

// validateCommandSyntax checks for common command syntax issues
func (v *ConfigValidator) validateCommandSyntax(command, fieldPath string, result *ValidationResult) {
	// Check for unmatched quotes
	singleQuotes := strings.Count(command, "'")
	doubleQuotes := strings.Count(command, "\"")

	if singleQuotes%2 != 0 {
		result.Errors = append(result.Errors, ValidationError{
			Field:      fieldPath,
			Value:      command,
			Issue:      "Unmatched single quotes in command",
			Suggestion: "Ensure all single quotes are properly paired",
			Severity:   SeverityError,
		})
	}

	if doubleQuotes%2 != 0 {
		result.Errors = append(result.Errors, ValidationError{
			Field:      fieldPath,
			Value:      command,
			Issue:      "Unmatched double quotes in command",
			Suggestion: "Ensure all double quotes are properly paired",
			Severity:   SeverityError,
		})
	}

	// Check for common typos in popular tools
	v.validateToolNames(command, fieldPath, result)
}

// validateToolNames checks for common typos in tool names
func (v *ConfigValidator) validateToolNames(command, fieldPath string, result *ValidationResult) {
	commonTypos := map[string]string{
		"prettier":     "prettier",
		"pretier":      "prettier",
		"pretter":      "prettier",
		"eslint":       "eslint",
		"esslint":      "eslint",
		"eslinter":     "eslint",
		"pytest":       "pytest",
		"py.test":      "pytest",
		"ruf":          "ruff",
		"ruff ":        "ruff",
		"gofmt ":       "gofmt",
		"go fmt":       "gofmt",
		"golangci":     "golangci-lint",
		"golangci-lint": "golangci-lint",
	}

	cmdLower := strings.ToLower(command)
	for typo, correct := range commonTypos {
		if strings.Contains(cmdLower, typo) && typo != correct {
			result.Errors = append(result.Errors, ValidationError{
				Field:      fieldPath,
				Value:      command,
				Issue:      fmt.Sprintf("Possible typo: '%s' should be '%s'", typo, correct),
				Suggestion: fmt.Sprintf("Check if you meant '%s' instead of '%s'", correct, typo),
				Severity:   SeverityWarning,
			})
		}
	}
}

// validateOutputRules validates output rule configurations
func (v *ConfigValidator) validateOutputRules(rules OutputRules, fieldPath string, result *ValidationResult) {
	if rules.ShowOn != "" {
		validShowOnValues := []string{"always", "failure", "success"}
		if !contains(validShowOnValues, rules.ShowOn) {
			result.Errors = append(result.Errors, ValidationError{
				Field:      fieldPath + ".show_on",
				Value:      rules.ShowOn,
				Issue:      "Invalid show_on value",
				Suggestion: "Use 'always', 'failure', or 'success'",
				Severity:   SeverityError,
			})
		}
	}

	// Validate message templates
	if rules.OnFailureMessage != "" {
		v.validateMessageTemplate(rules.OnFailureMessage, fieldPath+".on_failure_message", result)
	}
}

// validateMessageTemplate validates message template syntax
func (v *ConfigValidator) validateMessageTemplate(message, fieldPath string, result *ValidationResult) {
	// Check for template variable syntax (basic validation)
	if strings.Contains(message, "{{") && !strings.Contains(message, "}}") {
		result.Errors = append(result.Errors, ValidationError{
			Field:      fieldPath,
			Value:      message,
			Issue:      "Unclosed template variable in message",
			Suggestion: "Ensure all {{ variables }} are properly closed",
			Severity:   SeverityWarning,
		})
	}
}

// validateToolAvailability checks if tools are actually available
func (v *ConfigValidator) validateToolAvailability(tool Tool, fieldPrefix string, result *ValidationResult) {
	// Extract command name from check command
	parts := strings.Fields(tool.CheckCommand)
	if len(parts) == 0 {
		return
	}

	cmdName := parts[0]

	// Skip shell built-ins and complex commands
	if strings.Contains(cmdName, "/") || strings.Contains(cmdName, "|") || strings.Contains(cmdName, "&") {
		return
	}

	// Check if command exists
	if _, err := exec.LookPath(cmdName); err != nil {
		result.Errors = append(result.Errors, ValidationError{
			Field:      fieldPrefix + ".check_command",
			Value:      tool.CheckCommand,
			Issue:      fmt.Sprintf("Tool '%s' not found in PATH", cmdName),
			Suggestion: fmt.Sprintf("Install '%s' or check the installation command: %s", tool.Name, tool.InstallCommand),
			Severity:   SeverityWarning,
		})
	}
}

// validateToolReferences ensures commands reference existing tools
func (v *ConfigValidator) validateToolReferences(result *ValidationResult) {
	// Create a map of available tools by checking their command names
	availableTools := make(map[string]bool)
	for _, tool := range v.config.Tools {
		parts := strings.Fields(tool.CheckCommand)
		if len(parts) > 0 {
			cmdName := parts[0]
			availableTools[cmdName] = true
		}
	}

	// Check if hook commands reference available tools
	for hookName, hookGroup := range v.config.Hooks {
		for hookType, commands := range hookGroup {
			if len(commands) > 0 {
				v.checkCommandToolReferences(commands, availableTools, fmt.Sprintf("hooks.%s.%s", hookName, hookType), result)
			}
		}
	}
}

// checkCommandToolReferences checks if commands reference configured tools
func (v *ConfigValidator) checkCommandToolReferences(commands []Hook, availableTools map[string]bool, fieldPrefix string, result *ValidationResult) {
	commonTools := []string{
		"prettier", "eslint", "ruff", "black", "pytest", "gofmt", "golangci-lint",
		"rustfmt", "cargo", "php-cs-fixer", "phpstan", "gitleaks",
	}

	for i, cmd := range commands {
		cmdFieldPrefix := fmt.Sprintf("%s[%d]", fieldPrefix, i)
		
		// Extract command name
		parts := strings.Fields(cmd.Command)
		if len(parts) == 0 {
			continue
		}

		cmdName := parts[0]
		if strings.HasPrefix(cmdName, "npx") && len(parts) > 1 {
			cmdName = parts[1] // For npx commands, get the actual tool name
		}

		// Check if it's a common tool that should be configured
		if contains(commonTools, cmdName) && !availableTools[cmdName] {
			result.Errors = append(result.Errors, ValidationError{
				Field:      cmdFieldPrefix + ".command",
				Value:      cmd.Command,
				Issue:      fmt.Sprintf("Command uses '%s' but no tool configuration found", cmdName),
				Suggestion: fmt.Sprintf("Add a tool configuration for '%s' in the tools section", cmdName),
				Severity:   SeverityWarning,
			})
		}
	}
}

// validateCommonIssues checks for common configuration problems
func (v *ConfigValidator) validateCommonIssues(result *ValidationResult) {
	// Check for duplicate tool names
	v.validateDuplicateToolNames(result)

	// Check for duplicate hook names
	v.validateDuplicateHookGroups(result)

	// Check for missing essential hooks
	v.validateEssentialHooks(result)

	// Check file permissions and existence
	v.validateFileSystem(result)
}

// validateDuplicateToolNames checks for duplicate tool names
func (v *ConfigValidator) validateDuplicateToolNames(result *ValidationResult) {
	seen := make(map[string]int)
	for i, tool := range v.config.Tools {
		if prevIndex, exists := seen[tool.Name]; exists {
			result.Errors = append(result.Errors, ValidationError{
				Field:      fmt.Sprintf("tools[%d].name", i),
				Value:      tool.Name,
				Issue:      fmt.Sprintf("Duplicate tool name (also defined at tools[%d])", prevIndex),
				Suggestion: "Use unique names for each tool or merge configurations",
				Severity:   SeverityError,
			})
		}
		seen[tool.Name] = i
	}
}

// validateDuplicateHookGroups checks for duplicate hook group names
func (v *ConfigValidator) validateDuplicateHookGroups(result *ValidationResult) {
	// Hook groups are stored in a map, so duplicates are automatically prevented
	// This validation could be extended for other duplicate checks
}

// validateEssentialHooks suggests essential hooks that might be missing
func (v *ConfigValidator) validateEssentialHooks(result *ValidationResult) {
	hasSecurityHooks := false
	
	for hookName := range v.config.Hooks {
		if strings.Contains(strings.ToLower(hookName), "security") {
			hasSecurityHooks = true
			break
		}
	}

	if !hasSecurityHooks {
		result.Errors = append(result.Errors, ValidationError{
			Field:      "hooks",
			Value:      "missing security",
			Issue:      "No security hooks configured",
			Suggestion: "Consider adding a security hook group with tools like gitleaks for secret detection",
			Severity:   SeverityWarning,
		})
	}
}

// validateFileSystem checks file system related issues
func (v *ConfigValidator) validateFileSystem(result *ValidationResult) {
	// Check if quality.yml is readable
	if info, err := os.Stat("quality.yml"); err != nil {
		result.Errors = append(result.Errors, ValidationError{
			Field:      "file",
			Value:      "quality.yml",
			Issue:      "Cannot access quality.yml file",
			Suggestion: "Ensure quality.yml exists and is readable",
			Severity:   SeverityCritical,
		})
	} else {
		// Check file permissions
		if info.Mode().Perm()&0044 == 0 {
			result.Errors = append(result.Errors, ValidationError{
				Field:      "file",
				Value:      "quality.yml",
				Issue:      "quality.yml is not readable",
				Suggestion: "Fix file permissions: chmod 644 quality.yml",
				Severity:   SeverityError,
			})
		}
	}
}

// Helper functions

func (v *ConfigValidator) hasCriticalOrErrorSeverity(errors []ValidationError) bool {
	for _, err := range errors {
		if err.Severity == SeverityCritical || err.Severity == SeverityError {
			return true
		}
	}
	return false
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// GetFormattedErrors returns a human-readable string of all validation errors
func (r *ValidationResult) GetFormattedErrors() string {
	if len(r.Errors) == 0 {
		return "✅ No validation errors found"
	}

	var lines []string
	lines = append(lines, fmt.Sprintf("❌ Found %d validation issues:", len(r.Errors)))

	for _, err := range r.Errors {
		icon := "⚠️"
		if err.Severity == SeverityError {
			icon = "❌"
		} else if err.Severity == SeverityCritical {
			icon = "🚨"
		}

		lines = append(lines, fmt.Sprintf("  %s [%s] %s: %s", icon, err.Severity, err.Field, err.Issue))
		if err.Suggestion != "" {
			lines = append(lines, fmt.Sprintf("     💡 %s", err.Suggestion))
		}
	}

	return strings.Join(lines, "\n")
}

// GetErrorsBySeverity returns errors grouped by severity
func (r *ValidationResult) GetErrorsBySeverity() map[ValidationSeverity][]ValidationError {
	result := make(map[ValidationSeverity][]ValidationError)
	
	for _, err := range r.Errors {
		result[err.Severity] = append(result[err.Severity], err)
	}
	
	return result
}