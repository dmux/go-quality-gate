package service

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// captureStdout redirects os.Stdout for the duration of fn and returns what
// was written to it, mirroring the technique already used for the logger
// package's tests.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	orig := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	t.Cleanup(func() { os.Stdout = orig })

	fn()

	w.Close()
	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}

func TestNewInitService_DefaultsToCurrentDirectory(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	svc := NewInitService()

	if svc.detector == nil || svc.generator == nil {
		t.Fatal("expected a fully constructed InitService")
	}
}

func TestInitService_Init_CreatesQualityYml(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "go.mod", "module example.com/foo\n")
	t.Chdir(dir)

	svc := NewInitServiceWithPath(dir)
	if err := svc.Init(); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	content, err := os.ReadFile("quality.yml")
	if err != nil {
		t.Fatalf("expected quality.yml to be created: %v", err)
	}
	if !strings.Contains(string(content), "Gitleaks") {
		t.Errorf("expected generated content to include Gitleaks, got:\n%s", content)
	}
}

func TestInitService_InitWithOptions_RefusesToOverwriteWithoutForce(t *testing.T) {
	dir := t.TempDir()
	outputPath := filepath.Join(dir, "quality.yml")
	if err := os.WriteFile(outputPath, []byte("existing: true\n"), 0644); err != nil {
		t.Fatal(err)
	}

	svc := NewInitServiceWithPath(dir)
	err := svc.InitWithOptions(InitOptions{OutputPath: outputPath})

	if err == nil {
		t.Fatal("expected an error when quality.yml already exists and Force is false")
	}

	content, _ := os.ReadFile(outputPath)
	if string(content) != "existing: true\n" {
		t.Error("expected the existing file to be left untouched")
	}
}

func TestInitService_InitWithOptions_ForceOverwrites(t *testing.T) {
	dir := t.TempDir()
	outputPath := filepath.Join(dir, "quality.yml")
	if err := os.WriteFile(outputPath, []byte("existing: true\n"), 0644); err != nil {
		t.Fatal(err)
	}

	svc := NewInitServiceWithPath(dir)
	if err := svc.InitWithOptions(InitOptions{OutputPath: outputPath, Force: true}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(content), "existing: true") {
		t.Error("expected the file to have been overwritten")
	}
}

func TestInitService_InitWithOptions_WriteFailure(t *testing.T) {
	dir := t.TempDir()
	svc := NewInitServiceWithPath(dir)

	// A path inside a nonexistent directory makes os.WriteFile fail.
	badPath := filepath.Join(dir, "nonexistent-subdir", "quality.yml")

	err := svc.InitWithOptions(InitOptions{OutputPath: badPath})

	if err == nil {
		t.Fatal("expected an error when the output path can't be written")
	}
}

func TestInitService_InitWithOptions_DetectionFailure(t *testing.T) {
	svc := NewInitServiceWithPath(filepath.Join(t.TempDir(), "does-not-exist"))

	err := svc.InitWithOptions(InitOptions{OutputPath: filepath.Join(t.TempDir(), "quality.yml")})

	if err == nil {
		t.Fatal("expected an error when project structure detection fails")
	}
}

func TestInitService_InitWithOptions_VerboseOutput(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "go.mod", "module example.com/foo\n")
	writeFile(t, dir, "main.go", "package main\n")

	// 4 package.json files (one with react+eslint) exercise: the
	// Frameworks line, the Tools line, and the ">3 files" ("and N more")
	// branch in printDetectedStructure all at once.
	writeFile(t, dir, "package.json", `{"dependencies": {"react": "^18.0.0"}, "devDependencies": {"eslint": "^9.0.0"}}`)
	writeFile(t, dir, "frontend/package.json", `{}`)
	writeFile(t, dir, "backend/package.json", `{}`)
	writeFile(t, dir, "mobile/package.json", `{}`)

	svc := NewInitServiceWithPath(dir)
	outputPath := filepath.Join(dir, "quality.yml")

	output := captureStdout(t, func() {
		if err := svc.InitWithOptions(InitOptions{OutputPath: outputPath, Verbose: true}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	for _, marker := range []string{
		"Analyzing project structure", "Detected project components",
		"Languages:", "Frameworks:", "Existing Tools:",
		"go files:", "node files:", "and 3 more",
		"Generating quality.yml template", "Successfully created", "Next steps",
	} {
		if !strings.Contains(output, marker) {
			t.Errorf("expected verbose output to contain %q, got:\n%s", marker, output)
		}
	}
}

func TestInitService_InitWithOptions_VerboseOutput_FewFilesListsEachOne(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "go.mod", "module example.com/foo\n")

	svc := NewInitServiceWithPath(dir)
	outputPath := filepath.Join(dir, "quality.yml")

	output := captureStdout(t, func() {
		if err := svc.InitWithOptions(InitOptions{OutputPath: outputPath, Verbose: true}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(output, "go.mod") {
		t.Errorf("expected the single detected file to be listed by name, got:\n%s", output)
	}
}

func TestInitService_GetProjectAnalysis(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "go.mod", "module example.com/foo\n")

	svc := NewInitServiceWithPath(dir)
	structure, err := svc.GetProjectAnalysis()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !hasLang(structure, LanguageGo) {
		t.Errorf("expected Go to be detected, got %v", structure.Languages)
	}
}

func TestInitService_GetProjectAnalysis_DetectionFailure(t *testing.T) {
	svc := NewInitServiceWithPath(filepath.Join(t.TempDir(), "does-not-exist"))

	_, err := svc.GetProjectAnalysis()

	if err == nil {
		t.Fatal("expected an error when project structure detection fails")
	}
}

func TestInitService_GeneratePreview(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "go.mod", "module example.com/foo\n")

	svc := NewInitServiceWithPath(dir)
	preview, err := svc.GeneratePreview()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(preview, "Gofmt") {
		t.Errorf("expected the preview to include Go tooling, got:\n%s", preview)
	}
}

func TestInitService_GeneratePreview_DetectionFailure(t *testing.T) {
	svc := NewInitServiceWithPath(filepath.Join(t.TempDir(), "does-not-exist"))

	_, err := svc.GeneratePreview()

	if err == nil {
		t.Fatal("expected an error when project structure detection fails")
	}
}

func TestInitService_GeneratePreview_PythonProjectIncludesPipAudit(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "requirements.txt", "django==4.2\n")
	writeFile(t, dir, "main.py", "print('hi')\n")

	svc := NewInitServiceWithPath(dir)
	preview, err := svc.GeneratePreview()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, marker := range []string{
		"Pip-Audit", "pip-audit -r requirements.txt --aliases",
		"pip-audit --fix -r requirements.txt", "pre-push:",
	} {
		if !strings.Contains(preview, marker) {
			t.Errorf("expected preview to contain %q, got:\n%s", marker, preview)
		}
	}
}
