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
