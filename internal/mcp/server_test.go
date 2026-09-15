package mcp

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/dmux/go-quality-gate/internal/config"
	"github.com/dmux/go-quality-gate/internal/infra/git"
	"github.com/dmux/go-quality-gate/internal/infra/logger"
	"github.com/dmux/go-quality-gate/internal/infra/shell"
	"github.com/dmux/go-quality-gate/internal/service"
	"github.com/mark3labs/mcp-go/mcp"
)

func newTestServer(hooks []config.Hook) *MCPServer {
	shellRunner := &shell.RealShellRunner{}
	consoleLogger := logger.NewConsoleLogger(true)
	gitRepo := &git.RealGitRepository{}

	toolManager := service.NewCachingToolManagerService(shellRunner, consoleLogger, gitRepo)
	hookRunner := service.NewHookRunnerService(shellRunner, consoleLogger)
	qualityGate := service.NewQualityGateService(toolManager, hookRunner)

	cfg := &config.Config{
		Tools: []config.Tool{},
		Hooks: config.Hooks{
			"test-group": {
				"test-hook": hooks,
			},
		},
	}

	return NewMCPServer(qualityGate, cfg)
}

func TestMCPServer_RunQualityChecks(t *testing.T) {
	// Create mock services
	shellRunner := &shell.RealShellRunner{}
	consoleLogger := logger.NewConsoleLogger(true)
	gitRepo := &git.RealGitRepository{}

	toolManager := service.NewCachingToolManagerService(shellRunner, consoleLogger, gitRepo)
	hookRunner := service.NewHookRunnerService(shellRunner, consoleLogger)
	qualityGate := service.NewQualityGateService(toolManager, hookRunner)

	cfg := &config.Config{
		Tools: []config.Tool{},
		Hooks: config.Hooks{
			"test-group": {
				"test-hook": []config.Hook{
					{
						Name:    "Test Hook",
						Command: "echo 'hello'",
					},
				},
			},
		},
	}

	server := NewMCPServer(qualityGate, cfg)

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "run_quality_checks",
			Arguments: map[string]interface{}{
				"hookType": "test-hook",
			},
		},
	}

	res, err := server.handleRunQualityChecks(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.IsError {
		t.Fatalf("expected no error, got: %v", res)
	}

	content := res.Content[0].(mcp.TextContent).Text
	if len(content) == 0 {
		t.Fatalf("expected text output, got empty")
	}

	if !strings.Contains(content, "Test Hook") {
		t.Fatalf("expected output to contain 'Test Hook', got %s", content)
	}
}

func TestMCPServer_RunAutoFix(t *testing.T) {
	shellRunner := &shell.RealShellRunner{}
	consoleLogger := logger.NewConsoleLogger(true)
	gitRepo := &git.RealGitRepository{}

	toolManager := service.NewCachingToolManagerService(shellRunner, consoleLogger, gitRepo)
	hookRunner := service.NewHookRunnerService(shellRunner, consoleLogger)
	qualityGate := service.NewQualityGateService(toolManager, hookRunner)

	cfg := &config.Config{
		Tools: []config.Tool{},
		Hooks: config.Hooks{
			"test-group": {
				"test-hook": []config.Hook{
					{
						Name:       "Test Hook",
						Command:    "echo 'hello'",
						FixCommand: "echo 'fixed'",
					},
				},
			},
		},
	}

	server := NewMCPServer(qualityGate, cfg)

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "run_auto_fix",
			Arguments: map[string]interface{}{
				"hookType": "test-hook",
			},
		},
	}

	res, err := server.handleRunAutoFix(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.IsError {
		t.Fatalf("expected no error, got: %v", res)
	}

	content := res.Content[0].(mcp.TextContent).Text
	if !strings.Contains(content, "Auto-fix commands executed successfully.") {
		t.Fatalf("expected success message, got %s", content)
	}
}

func TestMCPServer_RunQualityChecks_InvalidArguments(t *testing.T) {
	server := newTestServer([]config.Hook{{Name: "Test Hook", Command: "echo hello"}})

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{Name: "run_quality_checks", Arguments: "not a map"},
	}

	res, err := server.handleRunQualityChecks(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.IsError {
		t.Fatal("expected a tool error result for non-map arguments")
	}
}

func TestMCPServer_RunQualityChecks_MissingHookType(t *testing.T) {
	server := newTestServer([]config.Hook{{Name: "Test Hook", Command: "echo hello"}})

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "run_quality_checks",
			Arguments: map[string]interface{}{"hookType": 123}, // not a string
		},
	}

	res, err := server.handleRunQualityChecks(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.IsError {
		t.Fatal("expected a tool error result when hookType isn't a string")
	}
}

func TestMCPServer_RunQualityChecks_FailingHook(t *testing.T) {
	server := newTestServer([]config.Hook{{Name: "Failing Hook", Command: "exit 1"}})

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "run_quality_checks",
			Arguments: map[string]interface{}{"hookType": "test-hook"},
		},
	}

	res, err := server.handleRunQualityChecks(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content := res.Content[0].(mcp.TextContent).Text
	if !strings.Contains(content, "Some quality checks failed.") {
		t.Errorf("expected the failure summary, got %s", content)
	}
	if !strings.Contains(content, "Execution Error:") {
		t.Errorf("expected the execution error to be reported, got %s", content)
	}
	if !strings.Contains(content, "FAILURE") {
		t.Errorf("expected the failing hook's status, got %s", content)
	}
}

func TestMCPServer_RunAutoFix_InvalidArguments(t *testing.T) {
	server := newTestServer([]config.Hook{{Name: "Test Hook", Command: "echo hello", FixCommand: "echo fixed"}})

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{Name: "run_auto_fix", Arguments: "not a map"},
	}

	res, err := server.handleRunAutoFix(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.IsError {
		t.Fatal("expected a tool error result for non-map arguments")
	}
}

func TestMCPServer_RunAutoFix_MissingHookType(t *testing.T) {
	server := newTestServer([]config.Hook{{Name: "Test Hook", Command: "echo hello", FixCommand: "echo fixed"}})

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "run_auto_fix",
			Arguments: map[string]interface{}{"hookType": 123},
		},
	}

	res, err := server.handleRunAutoFix(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.IsError {
		t.Fatal("expected a tool error result when hookType isn't a string")
	}
}

func TestMCPServer_RunAutoFix_CommandFails(t *testing.T) {
	server := newTestServer([]config.Hook{{Name: "Failing Fix", Command: "echo hello", FixCommand: "exit 1"}})

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "run_auto_fix",
			Arguments: map[string]interface{}{"hookType": "test-hook"},
		},
	}

	res, err := server.handleRunAutoFix(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content := res.Content[0].(mcp.TextContent).Text
	if !strings.Contains(content, "Auto-fix encountered an error") {
		t.Errorf("expected the fix error to be reported, got %s", content)
	}
}

func TestMCPServer_Start_ReturnsWhenStdinCloses(t *testing.T) {
	origStdin, origStdout := os.Stdin, os.Stdout

	stdinR, stdinW, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	stdoutR, stdoutW, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdin, os.Stdout = stdinR, stdoutW
	t.Cleanup(func() { os.Stdin, os.Stdout = origStdin, origStdout })

	// Closing the write end immediately gives the server an EOF on its
	// first read, so Listen (and therefore Start) returns right away
	// instead of blocking on real stdin forever.
	stdinW.Close()

	server := newTestServer([]config.Hook{{Name: "Test Hook", Command: "echo hello"}})

	done := make(chan error, 1)
	go func() { done <- server.Start() }()

	select {
	case err := <-done:
		if err != nil {
			t.Errorf("expected Start to return nil on EOF, got %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Start did not return after stdin closed")
	}

	stdoutW.Close()
	stdoutR.Close()
}
