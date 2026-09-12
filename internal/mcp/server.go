package mcp

import (
	"context"
	"fmt"

	"github.com/dmux/go-quality-gate/internal/config"
	"github.com/dmux/go-quality-gate/internal/service"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type MCPServer struct {
	qgService *service.QualityGateService
	cfg       *config.Config
}

func NewMCPServer(qgService *service.QualityGateService, cfg *config.Config) *MCPServer {
	return &MCPServer{
		qgService: qgService,
		cfg:       cfg,
	}
}

func (s *MCPServer) Start() error {
	mcpServer := server.NewMCPServer(
		"go-quality-gate",
		"1.2.0", // TODO: inject version
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

	results, err := s.qgService.Run(s.cfg, hookType, false)

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
