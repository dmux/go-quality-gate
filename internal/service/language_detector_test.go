package service

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLanguageDetector_DetectProjectStructure(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "quality-gate-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Test Go project detection
	t.Run("DetectGoProject", func(t *testing.T) {
		// Create go.mod file
		goModPath := filepath.Join(tmpDir, "go.mod")
		err := os.WriteFile(goModPath, []byte("module test\n\ngo 1.21\n"), 0644)
		if err != nil {
			t.Fatalf("Failed to create go.mod: %v", err)
		}

		// Create main.go file
		mainGoPath := filepath.Join(tmpDir, "main.go")
		err = os.WriteFile(mainGoPath, []byte("package main\n\nfunc main() {}\n"), 0644)
		if err != nil {
			t.Fatalf("Failed to create main.go: %v", err)
		}

		detector := NewLanguageDetector(tmpDir)
		structure, err := detector.DetectProjectStructure()
		if err != nil {
			t.Fatalf("DetectProjectStructure failed: %v", err)
		}

		// Check if Go is detected
		if !containsLanguage(structure.Languages, LanguageGo) {
			t.Errorf("Expected Go language to be detected, got: %v", structure.Languages)
		}

		// Check if go files are tracked
		if len(structure.Structure["go"]) == 0 {
			t.Errorf("Expected Go files to be tracked in structure")
		}
	})

	// Test Python project detection
	t.Run("DetectPythonProject", func(t *testing.T) {
		pythonDir := filepath.Join(tmpDir, "python-project")
		err := os.MkdirAll(pythonDir, 0755)
		if err != nil {
			t.Fatalf("Failed to create python dir: %v", err)
		}

		// Create requirements.txt
		reqPath := filepath.Join(pythonDir, "requirements.txt")
		reqContent := `django==4.2.0
pytest==7.4.0
black==23.0.0
ruff==0.0.292
`
		err = os.WriteFile(reqPath, []byte(reqContent), 0644)
		if err != nil {
			t.Fatalf("Failed to create requirements.txt: %v", err)
		}

		// Create main.py
		mainPyPath := filepath.Join(pythonDir, "main.py")
		err = os.WriteFile(mainPyPath, []byte("print('hello')\n"), 0644)
		if err != nil {
			t.Fatalf("Failed to create main.py: %v", err)
		}

		detector := NewLanguageDetector(pythonDir)
		structure, err := detector.DetectProjectStructure()
		if err != nil {
			t.Fatalf("DetectProjectStructure failed: %v", err)
		}

		// Check if Python is detected
		if !containsLanguage(structure.Languages, LanguagePython) {
			t.Errorf("Expected Python language to be detected, got: %v", structure.Languages)
		}

		// Check if Django framework is detected
		if !containsLanguage(structure.Frameworks, LanguageDjango) {
			t.Errorf("Expected Django framework to be detected, got: %v", structure.Frameworks)
		}

		// Check if tools are detected
		expectedTools := []string{"pytest", "black", "ruff"}
		for _, tool := range expectedTools {
			if !containsString(structure.Tools, tool) {
				t.Errorf("Expected tool %s to be detected, got: %v", tool, structure.Tools)
			}
		}
	})

	// Test Node.js project detection
	t.Run("DetectNodeProject", func(t *testing.T) {
		nodeDir := filepath.Join(tmpDir, "node-project")
		err := os.MkdirAll(nodeDir, 0755)
		if err != nil {
			t.Fatalf("Failed to create node dir: %v", err)
		}

		// Create package.json with React
		packagePath := filepath.Join(nodeDir, "package.json")
		packageContent := `{
  "name": "test-project",
  "version": "1.0.0",
  "dependencies": {
    "react": "^18.0.0",
    "react-dom": "^18.0.0"
  },
  "devDependencies": {
    "typescript": "^5.0.0",
    "prettier": "^3.0.0",
    "eslint": "^8.0.0"
  }
}`
		err = os.WriteFile(packagePath, []byte(packageContent), 0644)
		if err != nil {
			t.Fatalf("Failed to create package.json: %v", err)
		}

		// Create TypeScript files
		indexPath := filepath.Join(nodeDir, "index.tsx")
		err = os.WriteFile(indexPath, []byte("import React from 'react';\n"), 0644)
		if err != nil {
			t.Fatalf("Failed to create index.tsx: %v", err)
		}

		detector := NewLanguageDetector(nodeDir)
		structure, err := detector.DetectProjectStructure()
		if err != nil {
			t.Fatalf("DetectProjectStructure failed: %v", err)
		}

		// Check if Node and TypeScript are detected
		if !containsLanguage(structure.Languages, LanguageNode) {
			t.Errorf("Expected Node language to be detected, got: %v", structure.Languages)
		}

		if !containsLanguage(structure.Languages, LanguageTypeScript) {
			t.Errorf("Expected TypeScript language to be detected, got: %v", structure.Languages)
		}

		// Check if React framework is detected
		if !containsLanguage(structure.Frameworks, LanguageReact) {
			t.Errorf("Expected React framework to be detected, got: %v", structure.Frameworks)
		}

		// Check if tools are detected
		expectedTools := []string{"prettier", "eslint"}
		for _, tool := range expectedTools {
			if !containsString(structure.Tools, tool) {
				t.Errorf("Expected tool %s to be detected, got: %v", tool, structure.Tools)
			}
		}
	})
}

func TestLanguageDetector_ShouldSkipDirectory(t *testing.T) {
	testCases := []struct {
		dirname  string
		expected bool
	}{
		{"node_modules", true},
		{".git", true},
		{"vendor", true},
		{".venv", true},
		{"__pycache__", true},
		{"dist", true},
		{"build", true},
		{".idea", true},
		{".hidden", true},
		{"src", false},
		{"lib", false},
		{"test", false},
		{"docs", false},
	}

	for _, tc := range testCases {
		t.Run(tc.dirname, func(t *testing.T) {
			result := shouldSkipDirectory(tc.dirname)
			if result != tc.expected {
				t.Errorf("shouldSkipDirectory(%q) = %v, want %v", tc.dirname, result, tc.expected)
			}
		})
	}
}

func TestLanguageDetector_AnalyzePackageJson(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "quality-gate-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Test with Angular project
	angularPackage := `{
  "name": "angular-app",
  "dependencies": {
    "@angular/core": "^16.0.0",
    "@angular/common": "^16.0.0"
  },
  "devDependencies": {
    "jest": "^29.0.0"
  }
}`

	packagePath := filepath.Join(tmpDir, "package.json")
	err = os.WriteFile(packagePath, []byte(angularPackage), 0644)
	if err != nil {
		t.Fatalf("Failed to create package.json: %v", err)
	}

	detector := NewLanguageDetector(tmpDir)
	structure := &ProjectStructure{
		Languages:  []Language{},
		Frameworks: []Language{},
		Tools:      []string{},
		Structure:  make(map[string][]string),
	}

	detector.analyzePackageJson(packagePath, structure)

	// Check if Angular is detected
	if !containsLanguage(structure.Frameworks, LanguageAngular) {
		t.Errorf("Expected Angular framework to be detected, got: %v", structure.Frameworks)
	}

	// Check if jest is detected as tool
	if !containsString(structure.Tools, "jest") {
		t.Errorf("Expected jest tool to be detected, got: %v", structure.Tools)
	}
}

// Helper functions for tests
func containsLanguage(languages []Language, target Language) bool {
	for _, lang := range languages {
		if lang == target {
			return true
		}
	}
	return false
}

func containsString(strings []string, target string) bool {
	for _, str := range strings {
		if str == target {
			return true
		}
	}
	return false
}