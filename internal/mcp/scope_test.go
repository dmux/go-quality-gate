package mcp

import (
	"context"
	"strings"
	"testing"

	"github.com/dmux/go-quality-gate/internal/config"
	"github.com/dmux/go-quality-gate/internal/infra/git"
	"github.com/dmux/go-quality-gate/internal/infra/logger"
	"github.com/dmux/go-quality-gate/internal/infra/shell"
	"github.com/dmux/go-quality-gate/internal/service"
	"github.com/mark3labs/mcp-go/mcp"
)

// fakeFiles is a fixed repository.FileLister.
type fakeFiles struct{ staged, tracked []string }

func (f fakeFiles) StagedFiles() ([]string, error)  { return f.staged, nil }
func (f fakeFiles) PushFiles() ([]string, error)    { return f.staged, nil }
func (f fakeFiles) TrackedFiles() ([]string, error) { return f.tracked, nil }

func newScopedTestServer(files fakeFiles) *MCPServer {
	shellRunner := &shell.RealShellRunner{}
	consoleLogger := logger.NewConsoleLogger(true)
	gitRepo := &git.RealGitRepository{}
	qualityGate := service.NewQualityGateService(
		service.NewCachingToolManagerService(shellRunner, consoleLogger, gitRepo),
		service.NewHookRunnerService(shellRunner, consoleLogger),
	).WithFileLister(files)

	cfg := &config.Config{Hooks: config.Hooks{"g": {"pre-commit": []config.Hook{
		{Name: "Go files", Command: "echo checked {files}", FixCommand: "echo fixed {files}", Glob: []string{"*.go"}},
	}}}}
	return NewMCPServer(qualityGate, cfg, service.NewDoctorService(gitRepo, "quality.yml"), "test")
}

func callTool(t *testing.T, handler func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error), args map[string]interface{}) *mcp.CallToolResult {
	t.Helper()
	res, err := handler(context.Background(), mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: args}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	return res
}

func TestMCPServer_RunQualityChecks_Scope(t *testing.T) {
	server := newScopedTestServer(fakeFiles{staged: []string{"README.md"}, tracked: []string{"a.go"}})

	res := callTool(t, server.handleRunQualityChecks, map[string]interface{}{"hookType": "pre-commit"})
	text := res.Content[0].(mcp.TextContent).Text
	if !strings.Contains(text, "SKIPPED") {
		t.Errorf("expected the Go hook to be skipped for a README-only change, got:\n%s", text)
	}

	res = callTool(t, server.handleRunQualityChecks, map[string]interface{}{"hookType": "pre-commit", "scope": "all"})
	text = res.Content[0].(mcp.TextContent).Text
	if !strings.Contains(text, "checked a.go") {
		t.Errorf("expected a full run over tracked files, got:\n%s", text)
	}
}

func TestMCPServer_InvalidScope(t *testing.T) {
	server := newScopedTestServer(fakeFiles{})

	for name, handler := range map[string]func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error){
		"run": server.handleRunQualityChecks,
		"fix": server.handleRunAutoFix,
	} {
		res := callTool(t, handler, map[string]interface{}{"hookType": "pre-commit", "scope": "everything"})
		if !res.IsError {
			t.Errorf("%s: expected an error for an invalid scope", name)
		}
	}
}

func TestMCPServer_RunAutoFix_Scope(t *testing.T) {
	server := newScopedTestServer(fakeFiles{tracked: []string{"a.go"}})

	res := callTool(t, server.handleRunAutoFix, map[string]interface{}{"hookType": "pre-commit", "scope": "all"})
	if res.IsError || !strings.Contains(res.Content[0].(mcp.TextContent).Text, "successfully") {
		t.Errorf("expected the scoped fix to succeed, got %+v", res)
	}
}
