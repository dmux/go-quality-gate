package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// runCLI invokes run() with the given args, capturing stdout/stderr into
// strings, and returns (exitCode, stdout, stderr).
func runCLI(args ...string) (int, string, string) {
	var stdout, stderr bytes.Buffer
	code := run(args, &stdout, &stderr)
	return code, stdout.String(), stderr.String()
}

// initGitRepo creates a temp dir with a .git directory (and hooks/
// subdirectory, as a real `git init` would), chdirs into it, and returns
// the path.
func initGitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".git", "hooks"), 0755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)
	return dir
}

func writeQualityYML(t *testing.T, content string) {
	t.Helper()
	if err := os.WriteFile("quality.yml", []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

const simpleQualityYML = `hooks:
  security:
    pre-commit:
      - name: "Say hi"
        command: "echo hi"
        fix_command: "echo fixed"
`

const failingQualityYML = `hooks:
  security:
    pre-commit:
      - name: "Fails"
        command: "exit 1"
`

// --- version ---

func TestRun_Version_Text(t *testing.T) {
	code, stdout, _ := runCLI("--version")
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
	if !strings.Contains(stdout, Version) {
		t.Errorf("expected version text in stdout, got %q", stdout)
	}
}

func TestRun_Version_Short(t *testing.T) {
	code, stdout, _ := runCLI("-v")
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
	if !strings.Contains(stdout, Version) {
		t.Errorf("expected version text in stdout, got %q", stdout)
	}
}

func TestRun_Version_JSON(t *testing.T) {
	code, stdout, _ := runCLI("--version", "--output=json")
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
	var payload map[string]string
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatalf("expected valid JSON, got %q: %v", stdout, err)
	}
	if payload["version"] != Version {
		t.Errorf("expected version %q, got %q", Version, payload["version"])
	}
}

func TestRun_Version_JSONMarshalError(t *testing.T) {
	orig := marshalJSONIndent
	marshalJSONIndent = func(v interface{}, prefix, indent string) ([]byte, error) {
		return nil, errors.New("boom")
	}
	t.Cleanup(func() { marshalJSONIndent = orig })

	code, stdout, _ := runCLI("--version", "--output=json")
	if code != 1 {
		t.Fatalf("expected exit 1, got %d", code)
	}
	if !strings.Contains(stdout, "Error marshaling version JSON") {
		t.Errorf("expected a marshal error message, got %q", stdout)
	}
}

// --- flag parsing ---

func TestRun_UnknownFlag(t *testing.T) {
	code, _, stderr := runCLI("--this-flag-does-not-exist")
	if code != 2 {
		t.Fatalf("expected exit 2 for a flag parse error, got %d", code)
	}
	if stderr == "" {
		t.Error("expected flag package to report the parse error on stderr")
	}
}

// --- usage ---

func TestRun_NoArgs_PrintsUsage(t *testing.T) {
	code, stdout, _ := runCLI()
	if code != 1 {
		t.Fatalf("expected exit 1, got %d", code)
	}
	if !strings.Contains(stdout, "Usage: quality-gate") {
		t.Errorf("expected usage text, got %q", stdout)
	}
}

// --- install ---

func TestRun_Install_Success(t *testing.T) {
	initGitRepo(t)

	code, stdout, _ := runCLI("--install")

	if code != 0 {
		t.Fatalf("expected exit 0, got %d, stdout=%q", code, stdout)
	}
	if !strings.Contains(stdout, "Git hooks installed successfully") {
		t.Errorf("expected success message, got %q", stdout)
	}
	if _, err := os.Stat(".git/hooks/pre-commit"); err != nil {
		t.Errorf("expected pre-commit hook to be installed: %v", err)
	}
}

func TestRun_Install_Failure(t *testing.T) {
	t.Chdir(t.TempDir()) // not a git repo

	code, stdout, _ := runCLI("--install")

	if code != 1 {
		t.Fatalf("expected exit 1, got %d", code)
	}
	if !strings.Contains(stdout, "Error installing git hooks") {
		t.Errorf("expected an error message, got %q", stdout)
	}
}

// --- init ---

func TestRun_Init_Success(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/foo\n"), 0644); err != nil {
		t.Fatal(err)
	}

	code, stdout, _ := runCLI("--init")

	if code != 0 {
		t.Fatalf("expected exit 0, got %d, stdout=%q", code, stdout)
	}
	if !strings.Contains(stdout, "quality.yml initialized successfully") {
		t.Errorf("expected success message, got %q", stdout)
	}
	if _, err := os.Stat("quality.yml"); err != nil {
		t.Errorf("expected quality.yml to be created: %v", err)
	}
}

func TestRun_Init_Failure(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	writeQualityYML(t, simpleQualityYML) // already exists, --init doesn't force

	code, stdout, _ := runCLI("--init")

	if code != 1 {
		t.Fatalf("expected exit 1, got %d", code)
	}
	if !strings.Contains(stdout, "Error initializing quality.yml") {
		t.Errorf("expected an error message, got %q", stdout)
	}
}

// --- stats ---

func TestRun_Stats_NoHistory(t *testing.T) {
	initGitRepo(t)

	code, stdout, _ := runCLI("stats")

	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
	if !strings.Contains(stdout, "No history yet") {
		t.Errorf("expected the no-history message, got %q", stdout)
	}
}

func TestRun_Stats_NotAGitRepo(t *testing.T) {
	t.Chdir(t.TempDir())

	code, stdout, _ := runCLI("stats")

	if code != 0 {
		t.Fatalf("expected exit 0 even outside a git repo, got %d", code)
	}
	if !strings.Contains(stdout, "Not inside a git repository") {
		t.Errorf("expected the not-a-git-repo message, got %q", stdout)
	}
}

// --- ui ---

func TestRun_UI_NotAGitRepo(t *testing.T) {
	t.Chdir(t.TempDir())

	code, stdout, _ := runCLI("ui")

	if code != 1 {
		t.Fatalf("expected exit 1, got %d", code)
	}
	if !strings.Contains(stdout, "Not inside a git repository") {
		t.Errorf("expected the not-a-git-repo message, got %q", stdout)
	}
}

func TestRun_UI_BindFailure(t *testing.T) {
	initGitRepo(t)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	port := listener.Addr().(*net.TCPAddr).Port

	code, stdout, _ := runCLI("--port", strconv.Itoa(port), "ui")

	if code != 1 {
		t.Fatalf("expected exit 1, got %d", code)
	}
	if !strings.Contains(stdout, "Error running dashboard server") {
		t.Errorf("expected a dashboard server error, got %q", stdout)
	}
}

// --- config loading / validation ---

func TestRun_MissingQualityYML(t *testing.T) {
	initGitRepo(t)

	code, stdout, _ := runCLI("pre-commit")

	if code != 1 {
		t.Fatalf("expected exit 1, got %d", code)
	}
	if !strings.Contains(stdout, "Error loading quality.yml") {
		t.Errorf("expected a load error, got %q", stdout)
	}
}

func TestRun_ValidationBlocks_Text(t *testing.T) {
	initGitRepo(t)
	writeQualityYML(t, `hooks:
  security:
    pre-commit:
      - name: "Dangerous"
        command: "rm -rf /"
`)

	code, stdout, _ := runCLI("pre-commit")

	if code != 1 {
		t.Fatalf("expected exit 1, got %d", code)
	}
	if !strings.Contains(stdout, "CRITICAL") {
		t.Errorf("expected the critical validation error to be printed, got %q", stdout)
	}
}

func TestRun_ValidationBlocks_JSON(t *testing.T) {
	initGitRepo(t)
	writeQualityYML(t, `hooks:
  security:
    pre-commit:
      - name: "Dangerous"
        command: "rm -rf /"
`)

	code, stdout, stderr := runCLI("--output=json", "pre-commit")

	if code != 1 {
		t.Fatalf("expected exit 1, got %d", code)
	}
	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatalf("expected a validation-result JSON payload on stdout, got %q (stderr=%q): %v", stdout, stderr, err)
	}
	if payload["valid"] != false {
		t.Errorf("expected valid=false in the JSON payload, got %+v", payload)
	}
}

func TestRun_ValidationBlocks_JSONMarshalError(t *testing.T) {
	initGitRepo(t)
	writeQualityYML(t, `hooks:
  security:
    pre-commit:
      - name: "Dangerous"
        command: "rm -rf /"
`)

	orig := marshalJSONIndent
	marshalJSONIndent = func(v interface{}, prefix, indent string) ([]byte, error) {
		return nil, errors.New("boom")
	}
	t.Cleanup(func() { marshalJSONIndent = orig })

	code, _, stderr := runCLI("--output=json", "pre-commit")

	if code != 1 {
		t.Fatalf("expected exit 1, got %d", code)
	}
	if !strings.Contains(stderr, "Error marshaling validation JSON") {
		t.Errorf("expected a marshal error message, got %q", stderr)
	}
}

func TestRun_ValidationWarnOnly_StillRuns(t *testing.T) {
	initGitRepo(t)
	// No security hook group -> warning only (not Critical/Error), so
	// execution must still proceed.
	writeQualityYML(t, `hooks:
  demo:
    pre-commit:
      - name: "Say hi"
        command: "echo hi"
`)

	code, stdout, _ := runCLI("pre-commit")

	if code != 0 {
		t.Fatalf("expected exit 0, got %d, stdout=%q", code, stdout)
	}
	if !strings.Contains(stdout, "WARNING") {
		t.Errorf("expected the warning to still be displayed, got %q", stdout)
	}
	if !strings.Contains(stdout, "Quality gate passed successfully") {
		t.Errorf("expected execution to continue past the warning, got %q", stdout)
	}
}

// --- fix ---

func TestRun_Fix_Success(t *testing.T) {
	initGitRepo(t)
	writeQualityYML(t, simpleQualityYML)

	code, stdout, _ := runCLI("--fix", "pre-commit")

	if code != 0 {
		t.Fatalf("expected exit 0, got %d, stdout=%q", code, stdout)
	}
	if !strings.Contains(stdout, "Fixable issues fixed successfully") {
		t.Errorf("expected success message, got %q", stdout)
	}

	data, err := os.ReadFile(".git/quality-gate/history.jsonl")
	if err != nil {
		t.Fatalf("expected a fix history record: %v", err)
	}
	if !strings.Contains(string(data), `"type":"fix"`) {
		t.Errorf("expected a fix-type history record, got %q", data)
	}
}

func TestRun_Fix_Failure(t *testing.T) {
	initGitRepo(t)
	writeQualityYML(t, `hooks:
  security:
    pre-commit:
      - name: "Bad fix"
        command: "echo hi"
        fix_command: "exit 1"
`)

	code, stdout, _ := runCLI("--fix", "pre-commit")

	if code != 1 {
		t.Fatalf("expected exit 1, got %d", code)
	}
	if !strings.Contains(stdout, "Error fixing issues") {
		t.Errorf("expected an error message, got %q", stdout)
	}
}

// --- normal run ---

func TestRun_Success_Text(t *testing.T) {
	initGitRepo(t)
	writeQualityYML(t, simpleQualityYML)

	code, stdout, _ := runCLI("pre-commit")

	if code != 0 {
		t.Fatalf("expected exit 0, got %d, stdout=%q", code, stdout)
	}
	if !strings.Contains(stdout, "Quality gate passed successfully") {
		t.Errorf("expected success message, got %q", stdout)
	}

	data, err := os.ReadFile(".git/quality-gate/history.jsonl")
	if err != nil {
		t.Fatalf("expected a run history record: %v", err)
	}
	if !strings.Contains(string(data), `"type":"run"`) {
		t.Errorf("expected a run-type history record, got %q", data)
	}
}

func TestRun_Failure_Text(t *testing.T) {
	initGitRepo(t)
	writeQualityYML(t, failingQualityYML)

	code, stdout, _ := runCLI("pre-commit")

	if code != 1 {
		t.Fatalf("expected exit 1, got %d", code)
	}
	if !strings.Contains(stdout, "Quality gate failed") {
		t.Errorf("expected a failure message, got %q", stdout)
	}
}

func TestRun_Success_JSON(t *testing.T) {
	initGitRepo(t)
	writeQualityYML(t, simpleQualityYML)

	code, stdout, _ := runCLI("--output=json", "pre-commit")

	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
	var payload struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatalf("expected valid JSON, got %q: %v", stdout, err)
	}
	if payload.Status != "success" {
		t.Errorf("expected status=success, got %q", payload.Status)
	}
}

func TestRun_Failure_JSON(t *testing.T) {
	initGitRepo(t)
	writeQualityYML(t, failingQualityYML)

	code, stdout, _ := runCLI("--output=json", "pre-commit")

	if code != 1 {
		t.Fatalf("expected exit 1, got %d", code)
	}
	var payload struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatalf("expected valid JSON even on failure, got %q: %v", stdout, err)
	}
	if payload.Status != "failure" {
		t.Errorf("expected status=failure, got %q", payload.Status)
	}
}

func TestRun_Success_JSONMarshalError(t *testing.T) {
	initGitRepo(t)
	writeQualityYML(t, simpleQualityYML)

	orig := marshalJSONIndent
	marshalJSONIndent = func(v interface{}, prefix, indent string) ([]byte, error) {
		return nil, errors.New("boom")
	}
	t.Cleanup(func() { marshalJSONIndent = orig })

	code, _, stderr := runCLI("--output=json", "pre-commit")

	if code != 1 {
		t.Fatalf("expected exit 1, got %d", code)
	}
	if !strings.Contains(stderr, "Error marshaling JSON") {
		t.Errorf("expected a marshal error message, got %q", stderr)
	}
}

func TestRun_Parallel(t *testing.T) {
	initGitRepo(t)
	writeQualityYML(t, simpleQualityYML)

	code, stdout, _ := runCLI("--parallel", "pre-commit")

	if code != 0 {
		t.Fatalf("expected exit 0, got %d, stdout=%q", code, stdout)
	}
}

// --- mcp ---

// withStdin temporarily replaces os.Stdin (and os.Stdout, since the mcp-go
// library writes directly to the real os.Stdout regardless of what run()
// was given) for the duration of fn.
func withRedirectedStdio(t *testing.T, stdinR *os.File) {
	t.Helper()
	origStdin, origStdout := os.Stdin, os.Stdout
	_, stdoutW, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdin, os.Stdout = stdinR, stdoutW
	t.Cleanup(func() {
		os.Stdin, os.Stdout = origStdin, origStdout
		stdoutW.Close()
	})
}

func TestRun_MCP_Success(t *testing.T) {
	initGitRepo(t)
	writeQualityYML(t, simpleQualityYML)

	stdinR, stdinW, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	withRedirectedStdio(t, stdinR)
	stdinW.Close() // EOF: mcp-go's Listen returns nil promptly.

	code, _, _ := runCLI("mcp")

	if code != 0 {
		t.Fatalf("expected exit 0 on stdin EOF, got %d", code)
	}
}

func TestRun_MCP_Error(t *testing.T) {
	initGitRepo(t)
	writeQualityYML(t, simpleQualityYML)

	stdinR, stdinW, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	withRedirectedStdio(t, stdinR)
	stdinR.Close() // Reading from a closed (not just EOF'd) file errors.
	stdinW.Close()

	code, _, stderr := runCLI("mcp")

	if code != 1 {
		t.Fatalf("expected exit 1, got %d", code)
	}
	if !strings.Contains(stderr, "Error starting MCP server") {
		t.Errorf("expected an MCP error message, got %q", stderr)
	}
}

func TestRun_MCP_ValidationBlocks(t *testing.T) {
	initGitRepo(t)
	writeQualityYML(t, `hooks:
  security:
    pre-commit:
      - name: "Dangerous"
        command: "rm -rf /"
`)

	code, _, stderr := runCLI("mcp")

	if code != 1 {
		t.Fatalf("expected exit 1, got %d", code)
	}
	// mcp mode must never write to stdout (it's the JSON-RPC transport);
	// the validation error must go to stderr as plain text, not JSON.
	if !strings.Contains(stderr, "CRITICAL") {
		t.Errorf("expected the validation error on stderr, got %q", stderr)
	}
}
