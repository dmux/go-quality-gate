package mcp

import (
	"context"
	"fmt"

	"github.com/dmux/go-quality-gate/internal/config"
	"github.com/dmux/go-quality-gate/internal/domain"
	"github.com/dmux/go-quality-gate/internal/service"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type MCPServer struct {
	qgService *service.QualityGateService
	cfg       *config.Config
	onPass    func(hookType string, results []domain.ExecutionResult) error
	version   string
}

func NewMCPServer(qgService *service.QualityGateService, cfg *config.Config) *MCPServer {
	return &MCPServer{
		qgService: qgService,
		cfg:       cfg,
		version:   "dev",
	}
}

// SetVersion sets the version reported to MCP clients.
func (s *MCPServer) SetVersion(version string) {
	s.version = version
}

// OnPass registers a callback run after a fully passing check, used to record
// the commit attestation so agent-made commits are watermarked too.
func (s *MCPServer) OnPass(fn func(hookType string, results []domain.ExecutionResult) error) {
	s.onPass = fn
}

func (s *MCPServer) Start() error {
	mcpServer := server.NewMCPServer(
		"go-quality-gate",
		s.version,
	)

	runQualityChecksTool := mcp.NewTool("run_quality_checks",
		mcp.WithDescription("Run quality gate checks (e.g., linters, formatters, security checks) for a specific hook type."),
		mcp.WithString("hookType",
			mcp.Required(),
			mcp.Description("The type of hook to run (e.g., 'pre-commit', 'pre-push')."),
		),
	)

	runAutoFixTool := mcp.NewTool("run_auto_fix",
		mcp.WithDescription("Run automatic fixes for the specified hook type."),
		mcp.WithString("hookType",
			mcp.Required(),
			mcp.Description("The type of hook to fix (e.g., 'pre-commit', 'pre-push')."),
		),
	)

	mcpServer.AddTool(runQualityChecksTool, s.handleRunQualityChecks)
	mcpServer.AddTool(runAutoFixTool, s.handleRunAutoFix)

	return server.ServeStdio(mcpServer)
}

func (s *MCPServer) handleRunQualityChecks(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, ok := request.Params.Arguments.(map[string]interface{})
	if !ok {
		return mcp.NewToolResultError("invalid arguments"), nil
	}
	hookType, ok := args["hookType"].(string)
	if !ok {
		return mcp.NewToolResultError("hookType must be a string"), nil
	}

	results, err := s.qgService.Run(s.cfg, hookType)

	// Create a text result with details of each check
	var textOutput string
	overallSuccess := true

	for _, result := range results {
		status := "SUCCESS"
		if !result.Success {
			status = "FAILURE"
			overallSuccess = false
		}

		textOutput += fmt.Sprintf("Hook: %s\nCommand: %s\nStatus: %s\nDuration: %v\nOutput:\n%s\n\n",
			result.Hook.Name,
			result.Hook.Command,
			status,
			result.Duration,
			result.Output,
		)
	}

	if err != nil {
		textOutput += fmt.Sprintf("\nExecution Error: %v\n", err)
	}

	if overallSuccess && err == nil && s.onPass != nil {
		if passErr := s.onPass(hookType, results); passErr != nil {
			textOutput += fmt.Sprintf("\nWarning: could not record attestation: %v\n", passErr)
		}
	}

	if overallSuccess {
		textOutput = "All quality checks passed successfully.\n\n" + textOutput
	} else {
		textOutput = "Some quality checks failed.\n\n" + textOutput
	}

	return mcp.NewToolResultText(textOutput), nil
}

func (s *MCPServer) handleRunAutoFix(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args, ok := request.Params.Arguments.(map[string]interface{})
	if !ok {
		return mcp.NewToolResultError("invalid arguments"), nil
	}
	hookType, ok := args["hookType"].(string)
	if !ok {
		return mcp.NewToolResultError("hookType must be a string"), nil
	}

	err := s.qgService.Fix(s.cfg, hookType)
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("Auto-fix encountered an error: %v", err)), nil
	}

	return mcp.NewToolResultText("Auto-fix commands executed successfully."), nil
}
