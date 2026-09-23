package service

import (
	"os"
	"path/filepath"
	"testing"
)

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	full := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func hasLang(structure *ProjectStructure, lang Language) bool {
	for _, l := range structure.Languages {
		if l == lang {
			return true
		}
	}
	return false
}

func hasFramework(structure *ProjectStructure, fw Language) bool {
	for _, f := range structure.Frameworks {
		if f == fw {
			return true
		}
	}
	return false
}

func hasTool(structure *ProjectStructure, tool string) bool {
	for _, tl := range structure.Tools {
		if tl == tool {
			return true
		}
	}
	return false
}

func TestDetectProjectStructure_EmptyDirectory(t *testing.T) {
	dir := t.TempDir()
	detector := NewLanguageDetector(dir)

	structure, err := detector.DetectProjectStructure()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(structure.Languages) != 0 || len(structure.Frameworks) != 0 || len(structure.Tools) != 0 {
		t.Errorf("expected an empty structure, got %+v", structure)
	}
}

func TestDetectProjectStructure_NonexistentPath(t *testing.T) {
	detector := NewLanguageDetector(filepath.Join(t.TempDir(), "does-not-exist"))

	_, err := detector.DetectProjectStructure()

	if err == nil {
		t.Fatal("expected an error for a nonexistent project path")
	}
}

func TestDetectProjectStructure_AllMarkerFiles(t *testing.T) {
	dir := t.TempDir()

	writeFile(t, dir, "go.mod", "module example.com/foo\n")
	writeFile(t, dir, "main.go", "package main\n")
	writeFile(t, dir, "Cargo.toml", "[package]\nname=\"foo\"\n")
	writeFile(t, dir, "lib.rs", "fn main() {}\n")
	writeFile(t, dir, "pom.xml", "<project></project>\n")
	writeFile(t, dir, "App.java", "class App {}\n")
	writeFile(t, dir, "Dockerfile", "FROM scratch\n")
	writeFile(t, dir, "app.ts", "export {}\n")
	writeFile(t, dir, "app.php", "<?php\n")

	writeFile(t, dir, "package.json", `{
		"dependencies": {"react": "^18.0.0"},
		"devDependencies": {"typescript": "^5.0.0", "eslint": "^9.0.0", "vitest": "^1.0.0"}
	}`)

	writeFile(t, dir, "requirements.txt", "Django==4.2\nruff>=0.1\n# a comment\n\npytest~=7.0\n")

	writeFile(t, dir, "composer.json", `{
		"require": {"laravel/framework": "^10.0"},
		"require-dev": {"phpunit/phpunit": "^10.0", "phpstan/phpstan": "^1.0"}
	}`)

	detector := NewLanguageDetector(dir)
	structure, err := detector.DetectProjectStructure()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, lang := range []Language{
		LanguageGo, LanguageNode, LanguagePython, LanguageRust,
		LanguagePHP, LanguageJava, LanguageDocker, LanguageTypeScript,
	} {
		if !hasLang(structure, lang) {
			t.Errorf("expected language %q to be detected, got %v", lang, structure.Languages)
		}
	}

	for _, fw := range []Language{LanguageReact, LanguageDjango, LanguageLaravel} {
		if !hasFramework(structure, fw) {
			t.Errorf("expected framework %q to be detected, got %v", fw, structure.Frameworks)
		}
	}

	for _, tool := range []string{"eslint", "vitest", "ruff", "pytest", "phpunit", "phpstan"} {
		if !hasTool(structure, tool) {
			t.Errorf("expected tool %q to be detected, got %v", tool, structure.Tools)
		}
	}

	if len(structure.Structure["go"]) == 0 {
		t.Error("expected go.mod to populate structure.Structure[\"go\"]")
	}
}

func TestDetectProjectStructure_LockFilesDetectNodeWithoutPackageJson(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "yarn.lock", "")

	structure, err := NewLanguageDetector(dir).DetectProjectStructure()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !hasLang(structure, LanguageNode) {
		t.Errorf("expected yarn.lock alone to detect Node, got %v", structure.Languages)
	}
}

func TestDetectProjectStructure_JSExtensionSkippedWhenTypeScriptAlreadyDetected(t *testing.T) {
	dir := t.TempDir()
	// .ts is processed first alphabetically isn't guaranteed, so force order
	// by detecting TypeScript via package.json first, then adding a .js file.
	writeFile(t, dir, "package.json", `{"devDependencies": {"typescript": "^5.0.0"}}`)
	writeFile(t, dir, "index.ts", "export {}\n")
	writeFile(t, dir, "legacy.js", "module.exports = {};\n")

	structure, err := NewLanguageDetector(dir).DetectProjectStructure()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !hasLang(structure, LanguageTypeScript) {
		t.Error("expected TypeScript to be detected")
	}
	// Node.js is still detected via package.json itself, so this mainly
	// exercises the "already has TypeScript" branch in analyzeFile without
	// asserting Node's absence (package.json unconditionally adds it).
}

func TestDetectProjectStructure_PlainJSFileDetectsNode(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "index.js", "module.exports = {};\n")

	structure, err := NewLanguageDetector(dir).DetectProjectStructure()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !hasLang(structure, LanguageNode) {
		t.Errorf("expected a plain .js file (no TypeScript present) to detect Node, got %v", structure.Languages)
	}
}

func TestDetectProjectStructure_PlainPyFileDetectsPython(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "script.py", "print('hi')\n")

	structure, err := NewLanguageDetector(dir).DetectProjectStructure()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !hasLang(structure, LanguagePython) {
		t.Errorf("expected a .py file to detect Python, got %v", structure.Languages)
	}
}

func TestDetectProjectStructure_VueAndAngular(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "package.json", `{
		"dependencies": {"vue": "^3.0.0", "@angular/core": "^17.0.0"}
	}`)

	structure, err := NewLanguageDetector(dir).DetectProjectStructure()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !hasFramework(structure, LanguageVue) {
		t.Errorf("expected Vue to be detected, got %v", structure.Frameworks)
	}
	if !hasFramework(structure, LanguageAngular) {
		t.Errorf("expected Angular to be detected, got %v", structure.Frameworks)
	}
}

func TestDetectProjectStructure_DedupesRepeatedFrameworksAndTools(t *testing.T) {
	dir := t.TempDir()
	// Two lines matching "django" and two matching "pytest" exercise the
	// "already added" early-return branches in addFrameworkIfNotExists and
	// addToolIfNotExists.
	writeFile(t, dir, "requirements.txt", "django==4.2\ndjango-extensions==3.2\npytest==7.0\npytest-cov==4.0\n")

	structure, err := NewLanguageDetector(dir).DetectProjectStructure()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	frameworkCount := 0
	for _, f := range structure.Frameworks {
		if f == LanguageDjango {
			frameworkCount++
		}
	}
	if frameworkCount != 1 {
		t.Errorf("expected Django to be added exactly once despite 2 matching lines, got %d", frameworkCount)
	}

	toolCount := 0
	for _, tl := range structure.Tools {
		if tl == "pytest" {
			toolCount++
		}
	}
	if toolCount != 1 {
		t.Errorf("expected pytest to be added exactly once despite 2 matching lines, got %d", toolCount)
	}
}

func TestAnalyzePythonRequirements_SeparatorOnlyLineIsSkipped(t *testing.T) {
	dir := t.TempDir()
	// A line with nothing but version-separator characters splits into zero
	// fields via strings.FieldsFunc, exercising the "len(parts) == 0"
	// continue branch.
	writeFile(t, dir, "requirements.txt", "===\ndjango==4.2\n")

	structure, err := NewLanguageDetector(dir).DetectProjectStructure()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !hasFramework(structure, LanguageDjango) {
		t.Error("expected Django to still be detected after skipping the separator-only line")
	}
}

func TestDetectProjectStructure_SkipsIgnoredDirectories(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "node_modules/some-pkg/index.js", "module.exports = {};\n")
	writeFile(t, dir, ".git/HEAD", "ref: refs/heads/main\n")
	writeFile(t, dir, "vendor/pkg/file.php", "<?php\n")
	writeFile(t, dir, ".hidden-dir/file.go", "package hidden\n")

	structure, err := NewLanguageDetector(dir).DetectProjectStructure()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(structure.Languages) != 0 {
		t.Errorf("expected everything under skipped directories to be ignored, got %v", structure.Languages)
	}
}

func TestAnalyzePackageJson_MalformedJSONIsIgnored(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "package.json", "{not valid json")

	structure, err := NewLanguageDetector(dir).DetectProjectStructure()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Node is still added (from the filename switch itself), but no
	// framework/tool should come from the unparsable dependency analysis.
	if len(structure.Frameworks) != 0 {
		t.Errorf("expected no frameworks from malformed package.json, got %v", structure.Frameworks)
	}
}

func TestAnalyzeComposerJson_MalformedJSONIsIgnored(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "composer.json", "{not valid json")

	structure, err := NewLanguageDetector(dir).DetectProjectStructure()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(structure.Frameworks) != 0 {
		t.Errorf("expected no frameworks from malformed composer.json, got %v", structure.Frameworks)
	}
}

func TestAnalyzePackageJson_UnreadableFileIsIgnored(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root: permission checks don't apply")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "package.json")
	writeFile(t, dir, "package.json", `{"dependencies": {"react": "^18.0.0"}}`)
	if err := os.Chmod(path, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(path, 0644) })

	structure, err := NewLanguageDetector(dir).DetectProjectStructure()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if hasFramework(structure, LanguageReact) {
		t.Error("expected an unreadable package.json to be silently skipped")
	}
}

func TestAnalyzeComposerJson_UnreadableFileIsIgnored(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root: permission checks don't apply")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "composer.json")
	writeFile(t, dir, "composer.json", `{"require": {"laravel/framework": "^10.0"}}`)
	if err := os.Chmod(path, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(path, 0644) })

	structure, err := NewLanguageDetector(dir).DetectProjectStructure()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if hasFramework(structure, LanguageLaravel) {
		t.Error("expected an unreadable composer.json to be silently skipped")
	}
}

func TestAnalyzePythonRequirements_UnreadableFileIsIgnored(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root: permission checks don't apply")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "requirements.txt")
	writeFile(t, dir, "requirements.txt", "django==4.2\n")
	if err := os.Chmod(path, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(path, 0644) })

	structure, err := NewLanguageDetector(dir).DetectProjectStructure()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if hasFramework(structure, LanguageDjango) {
		t.Error("expected an unreadable requirements.txt to be silently skipped")
	}
}

func TestAnalyzePythonRequirements_FastAPIAndFlask(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "requirements.txt", "fastapi==0.100\nflask==2.0\nmypy\nblack\nisort\nflake8\n")

	structure, err := NewLanguageDetector(dir).DetectProjectStructure()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !hasFramework(structure, LanguageFastAPI) {
		t.Error("expected FastAPI to be detected")
	}
	if !hasFramework(structure, LanguageFlask) {
		t.Error("expected Flask to be detected")
	}
	for _, tool := range []string{"mypy", "black", "isort", "flake8"} {
		if !hasTool(structure, tool) {
			t.Errorf("expected tool %q to be detected", tool)
		}
	}
}

func TestShouldSkipDirectory(t *testing.T) {
	cases := map[string]bool{
		".git":         true,
		"node_modules": true,
		"vendor":       true,
		".venv":        true,
		"dist":         true,
		".idea":        true,
		".anything":    true,
		"src":          false,
		"internal":     false,
	}
	for dir, want := range cases {
		if got := shouldSkipDirectory(dir); got != want {
			t.Errorf("shouldSkipDirectory(%q) = %v, want %v", dir, got, want)
		}
	}
}
