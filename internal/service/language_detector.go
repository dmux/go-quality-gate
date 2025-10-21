package service

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// Language represents a detected programming language/framework
type Language string

const (
	LanguageGo         Language = "go"
	LanguageNode       Language = "node"
	LanguagePython     Language = "python"
	LanguageRust       Language = "rust"
	LanguagePHP        Language = "php"
	LanguageJava       Language = "java"
	LanguageDocker     Language = "docker"
	LanguageTypeScript Language = "typescript"
	LanguageReact      Language = "react"
	LanguageVue        Language = "vue"
	LanguageAngular    Language = "angular"
	LanguageDjango     Language = "django"
	LanguageFastAPI    Language = "fastapi"
	LanguageFlask      Language = "flask"
	LanguageLaravel    Language = "laravel"
)

// ProjectStructure holds information about detected languages and frameworks
type ProjectStructure struct {
	Languages  []Language          `json:"languages"`
	Frameworks []Language          `json:"frameworks"`
	Tools      []string            `json:"tools"`
	Structure  map[string][]string `json:"structure"`
}

// LanguageDetector is responsible for analyzing project structure
type LanguageDetector struct {
	projectPath string
}

// NewLanguageDetector creates a new language detector
func NewLanguageDetector(projectPath string) *LanguageDetector {
	return &LanguageDetector{
		projectPath: projectPath,
	}
}

// DetectProjectStructure analyzes the project and returns detected languages/frameworks
func (d *LanguageDetector) DetectProjectStructure() (*ProjectStructure, error) {
	structure := &ProjectStructure{
		Languages:  []Language{},
		Frameworks: []Language{},
		Tools:      []string{},
		Structure:  make(map[string][]string),
	}

	// Walk through the project directory
	err := filepath.Walk(d.projectPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip hidden directories and common build/vendor directories
		if info.IsDir() && shouldSkipDirectory(info.Name()) {
			return filepath.SkipDir
		}

		if !info.IsDir() {
			d.analyzeFile(path, info.Name(), structure)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Post-process to detect frameworks based on dependencies
	d.detectFrameworks(structure)

	return structure, nil
}

// analyzeFile analyzes individual files to detect languages and tools
func (d *LanguageDetector) analyzeFile(fullPath, filename string, structure *ProjectStructure) {
	switch filename {
	// Go
	case "go.mod", "go.sum":
		d.addLanguageIfNotExists(LanguageGo, structure)
		structure.Structure["go"] = append(structure.Structure["go"], fullPath)

	// Node.js / JavaScript / TypeScript
	case "package.json":
		d.addLanguageIfNotExists(LanguageNode, structure)
		structure.Structure["node"] = append(structure.Structure["node"], fullPath)
		d.analyzePackageJson(fullPath, structure)
	case "package-lock.json", "yarn.lock", "pnpm-lock.yaml":
		d.addLanguageIfNotExists(LanguageNode, structure)

	// Python
	case "requirements.txt", "setup.py", "pyproject.toml", "Pipfile", "poetry.lock":
		d.addLanguageIfNotExists(LanguagePython, structure)
		structure.Structure["python"] = append(structure.Structure["python"], fullPath)
		if filename == "requirements.txt" {
			d.analyzePythonRequirements(fullPath, structure)
		}

	// Rust
	case "Cargo.toml", "Cargo.lock":
		d.addLanguageIfNotExists(LanguageRust, structure)
		structure.Structure["rust"] = append(structure.Structure["rust"], fullPath)

	// PHP
	case "composer.json", "composer.lock":
		d.addLanguageIfNotExists(LanguagePHP, structure)
		structure.Structure["php"] = append(structure.Structure["php"], fullPath)
		if filename == "composer.json" {
			d.analyzeComposerJson(fullPath, structure)
		}

	// Java
	case "pom.xml", "build.gradle", "gradle.properties":
		d.addLanguageIfNotExists(LanguageJava, structure)
		structure.Structure["java"] = append(structure.Structure["java"], fullPath)

	// Docker
	case "Dockerfile", "docker-compose.yml", "docker-compose.yaml":
		d.addLanguageIfNotExists(LanguageDocker, structure)
		structure.Structure["docker"] = append(structure.Structure["docker"], fullPath)
	}

	// File extensions
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".ts", ".tsx":
		d.addLanguageIfNotExists(LanguageTypeScript, structure)
	case ".js", ".jsx", ".mjs":
		if !d.hasLanguage(LanguageTypeScript, structure) {
			d.addLanguageIfNotExists(LanguageNode, structure)
		}
	case ".py":
		d.addLanguageIfNotExists(LanguagePython, structure)
	case ".go":
		d.addLanguageIfNotExists(LanguageGo, structure)
	case ".rs":
		d.addLanguageIfNotExists(LanguageRust, structure)
	case ".php":
		d.addLanguageIfNotExists(LanguagePHP, structure)
	case ".java", ".kt", ".scala":
		d.addLanguageIfNotExists(LanguageJava, structure)
	}
}

// analyzePackageJson analyzes package.json for framework detection
func (d *LanguageDetector) analyzePackageJson(path string, structure *ProjectStructure) {
	content, err := os.ReadFile(path)
	if err != nil {
		return
	}

	var packageJson struct {
		Dependencies    map[string]string `json:"dependencies"`
		DevDependencies map[string]string `json:"devDependencies"`
		Scripts         map[string]string `json:"scripts"`
	}

	if err := json.Unmarshal(content, &packageJson); err != nil {
		return
	}

	allDeps := make(map[string]string)
	for k, v := range packageJson.Dependencies {
		allDeps[k] = v
	}
	for k, v := range packageJson.DevDependencies {
		allDeps[k] = v
	}

	// Detect TypeScript
	if _, hasTS := allDeps["typescript"]; hasTS {
		d.addLanguageIfNotExists(LanguageTypeScript, structure)
	}

	// Detect React
	if _, hasReact := allDeps["react"]; hasReact {
		d.addFrameworkIfNotExists(LanguageReact, structure)
	}

	// Detect Vue
	if _, hasVue := allDeps["vue"]; hasVue {
		d.addFrameworkIfNotExists(LanguageVue, structure)
	}

	// Detect Angular
	if _, hasAngular := allDeps["@angular/core"]; hasAngular {
		d.addFrameworkIfNotExists(LanguageAngular, structure)
	}

	// Detect common tools
	tools := []string{"eslint", "prettier", "jest", "vitest", "cypress", "playwright"}
	for _, tool := range tools {
		if _, hasTool := allDeps[tool]; hasTool {
			d.addToolIfNotExists(tool, structure)
		}
	}
}

// analyzePythonRequirements analyzes requirements.txt for framework detection
func (d *LanguageDetector) analyzePythonRequirements(path string, structure *ProjectStructure) {
	content, err := os.ReadFile(path)
	if err != nil {
		return
	}

	contentStr := string(content)
	lines := strings.Split(contentStr, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(strings.ToLower(line))
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Extract package name (before ==, >=, etc.)
		parts := strings.FieldsFunc(line, func(r rune) bool {
			return r == '=' || r == '>' || r == '<' || r == '!' || r == '~'
		})
		if len(parts) == 0 {
			continue
		}

		packageName := strings.TrimSpace(parts[0])

		// Detect frameworks
		switch {
		case strings.Contains(packageName, "django"):
			d.addFrameworkIfNotExists(LanguageDjango, structure)
		case strings.Contains(packageName, "fastapi"):
			d.addFrameworkIfNotExists(LanguageFastAPI, structure)
		case strings.Contains(packageName, "flask"):
			d.addFrameworkIfNotExists(LanguageFlask, structure)
		}

		// Detect tools
		tools := []string{"black", "ruff", "flake8", "mypy", "pytest", "isort"}
		for _, tool := range tools {
			if strings.Contains(packageName, tool) {
				d.addToolIfNotExists(tool, structure)
			}
		}
	}
}

// analyzeComposerJson analyzes composer.json for framework detection
func (d *LanguageDetector) analyzeComposerJson(path string, structure *ProjectStructure) {
	content, err := os.ReadFile(path)
	if err != nil {
		return
	}

	var composerJson struct {
		Require    map[string]string `json:"require"`
		RequireDev map[string]string `json:"require-dev"`
	}

	if err := json.Unmarshal(content, &composerJson); err != nil {
		return
	}

	allDeps := make(map[string]string)
	for k, v := range composerJson.Require {
		allDeps[k] = v
	}
	for k, v := range composerJson.RequireDev {
		allDeps[k] = v
	}

	// Detect Laravel
	if _, hasLaravel := allDeps["laravel/framework"]; hasLaravel {
		d.addFrameworkIfNotExists(LanguageLaravel, structure)
	}

	// Detect tools
	tools := map[string]string{
		"phpunit/phpunit":           "phpunit",
		"squizlabs/php_codesniffer": "phpcs",
		"friendsofphp/php-cs-fixer": "php-cs-fixer",
		"phpstan/phpstan":           "phpstan",
		"psalm/phar":                "psalm",
	}

	for dep, tool := range tools {
		if _, hasTool := allDeps[dep]; hasTool {
			d.addToolIfNotExists(tool, structure)
		}
	}
}

// detectFrameworks performs post-processing to detect frameworks
func (d *LanguageDetector) detectFrameworks(structure *ProjectStructure) {
	// Additional framework detection logic based on file structure
	// This could be expanded to look for specific directory patterns, etc.
}

// Helper functions
func (d *LanguageDetector) addLanguageIfNotExists(lang Language, structure *ProjectStructure) {
	if !d.hasLanguage(lang, structure) {
		structure.Languages = append(structure.Languages, lang)
	}
}

func (d *LanguageDetector) addFrameworkIfNotExists(framework Language, structure *ProjectStructure) {
	for _, f := range structure.Frameworks {
		if f == framework {
			return
		}
	}
	structure.Frameworks = append(structure.Frameworks, framework)
}

func (d *LanguageDetector) addToolIfNotExists(tool string, structure *ProjectStructure) {
	for _, t := range structure.Tools {
		if t == tool {
			return
		}
	}
	structure.Tools = append(structure.Tools, tool)
}

func (d *LanguageDetector) hasLanguage(lang Language, structure *ProjectStructure) bool {
	for _, l := range structure.Languages {
		if l == lang {
			return true
		}
	}
	return false
}

func shouldSkipDirectory(dirname string) bool {
	skipDirs := []string{
		".git", ".svn", ".hg",
		"node_modules", "vendor", "target",
		".venv", "venv", "__pycache__",
		".next", ".nuxt", "dist", "build",
		".idea", ".vscode",
	}

	for _, skip := range skipDirs {
		if dirname == skip {
			return true
		}
	}

	return strings.HasPrefix(dirname, ".")
}