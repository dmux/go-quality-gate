package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/dmux/go-quality-gate/internal/config"
	"github.com/dmux/go-quality-gate/internal/domain"
	"github.com/dmux/go-quality-gate/internal/infra/git"
	"github.com/dmux/go-quality-gate/internal/infra/history"
	"github.com/dmux/go-quality-gate/internal/infra/logger"
	"github.com/dmux/go-quality-gate/internal/infra/shell"
	"github.com/dmux/go-quality-gate/internal/infra/webui"
	"github.com/dmux/go-quality-gate/internal/mcp"
	"github.com/dmux/go-quality-gate/internal/service"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// marshalJSONIndent is json.MarshalIndent, swappable in tests to exercise
// this file's marshal-error branches — the values passed to it (version
// info, validation results, hook results) are all plain-typed and can't
// realistically fail to marshal, so a real failure isn't reproducible.
var marshalJSONIndent = json.MarshalIndent

// run implements the CLI end to end, returning a process exit code instead
// of calling os.Exit directly, and writing to the given streams instead of
// hardcoding os.Stdout/os.Stderr, so it's fully exercisable from tests.
func run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("quality-gate", flag.ContinueOnError)
	fs.SetOutput(stderr)

	installFlag := fs.Bool("install", false, "Install git hooks")
	initFlag := fs.Bool("init", false, "Initialize quality.yml")
	fixFlag := fs.Bool("fix", false, "Fix fixable issues")
	versionFlag := fs.Bool("version", false, "Show version information")
	versionFlagShort := fs.Bool("v", false, "Show version information (shorthand)")
	outputFlag := fs.String("output", "", "Output format (e.g., json)")
	parallelFlag := fs.Bool("parallel", false, "Run independent hooks concurrently")
	portFlag := fs.Int("port", 4173, "Port for the 'ui' web dashboard")

	if err := fs.Parse(args); err != nil {
		return 2
	}

	// Handle version flag first, before any other operations
	if *versionFlag || *versionFlagShort {
		if *outputFlag == "json" {
			versionJSON := struct {
				Version   string `json:"version"`
				BuildDate string `json:"build_date"`
				GitCommit string `json:"git_commit"`
			}{
				Version:   Version,
				BuildDate: BuildDate,
				GitCommit: GitCommit,
			}
			jsonBytes, err := marshalJSONIndent(versionJSON, "", "  ")
			if err != nil {
				fmt.Fprintf(stdout, "Error marshaling version JSON: %v\n", err)
				return 1
			}
			fmt.Fprintln(stdout, string(jsonBytes))
		} else {
			fmt.Fprintln(stdout, VersionInfo())
		}
		return 0
	}

	cmdArgs := fs.Args()
	isMCP := len(cmdArgs) > 0 && cmdArgs[0] == "mcp"
	isJsonOutput := *outputFlag == "json" || isMCP

	logPrint := func(format string, args ...interface{}) {
		if isJsonOutput {
			fmt.Fprintf(stderr, format, args...)
		} else {
			fmt.Fprintf(stdout, format, args...)
		}
	}

	logPrintln := func(msg string) {
		if isJsonOutput {
			fmt.Fprintln(stderr, msg)
		} else {
			fmt.Fprintln(stdout, msg)
		}
	}

	if *installFlag {
		logPrintln("Installing git hooks...")
		gitRepo := &git.RealGitRepository{}
		installationService := service.NewInstallationService(gitRepo)
		if err := installationService.InstallHooks(); err != nil {
			logPrint("Error installing git hooks: %v\n", err)
			return 1
		}
		logPrintln("Git hooks installed successfully.")
		return 0
	}

	if *initFlag {
		logPrintln("Initializing quality.yml...")
		initService := service.NewInitService()
		if err := initService.Init(); err != nil {
			logPrint("Error initializing quality.yml: %v\n", err)
			return 1
		}
		logPrintln("quality.yml initialized successfully.")
		return 0
	}

	if len(cmdArgs) == 0 {
		logPrintln("Usage: quality-gate [OPTIONS] [HOOK_TYPE]")
		logPrintln("")
		logPrintln("Hook Types:")
		logPrintln("  pre-commit    Run pre-commit quality checks")
		logPrintln("  pre-push      Run pre-push quality checks")
		logPrintln("  mcp           Start Model Context Protocol (MCP) server")
		logPrintln("  stats         Show streaks, achievements, and time saved")
		logPrintln("  ui            Open the local web dashboard")
		logPrintln("")
		logPrintln("Options:")
		logPrintln("  --install     Install git hooks in the current repository")
		logPrintln("  --init        Initialize quality.yml with intelligent analysis")
		logPrintln("  --fix         Automatically fix detected issues")
		logPrintln("  --version, -v Show version information")
		logPrintln("  --output json Output results in JSON format")
		logPrintln("  --parallel    Run independent hooks concurrently")
		logPrintln("  --port N      Port for 'ui' web dashboard (default 4173)")
		logPrintln("")
		logPrintln("Examples:")
		logPrintln("  quality-gate --init              # Create quality.yml for your project")
		logPrintln("  quality-gate --install           # Install git hooks")
		logPrintln("  quality-gate pre-commit          # Run pre-commit checks")
		logPrintln("  quality-gate --fix pre-commit    # Fix issues and run checks")
		logPrintln("  quality-gate --version           # Show version")
		return 1
	}

	hookType := cmdArgs[0]

	if hookType == "stats" {
		runStatsCommand(logPrintln)
		return 0
	}

	if hookType == "ui" {
		gitDir, err := history.FindGitDir()
		if err != nil {
			logPrint("Not inside a git repository: %v\n", err)
			return 1
		}
		if err := webui.Serve(gitDir, *portFlag); err != nil {
			logPrint("Error running dashboard server: %v\n", err)
			return 1
		}
		return 0
	}

	cfg, err := config.LoadConfig("quality.yml")
	if err != nil {
		logPrint("Error loading quality.yml: %v\n", err)
		return 1
	}

	validationResult := config.NewConfigValidator(cfg).Validate()
	if len(validationResult.Errors) > 0 {
		if !validationResult.Valid {
			// Critical/Error severity: block execution before any hook can run.
			// stdout is the MCP stdio transport, so never write there for the
			// mcp subcommand — only stderr, regardless of JSON mode.
			if isJsonOutput && !isMCP {
				jsonBytes, marshalErr := marshalJSONIndent(validationResult, "", "  ")
				if marshalErr != nil {
					logPrint("Error marshaling validation JSON: %v\n", marshalErr)
					return 1
				}
				fmt.Fprintln(stdout, string(jsonBytes))
			} else {
				logPrint("%s\n", validationResult.GetFormattedErrors())
			}
			return 1
		}
		// Warning-only: display but continue. logPrintln already routes to
		// stderr in JSON/MCP mode, so this never corrupts a JSON stdout
		// payload or the MCP stdio transport.
		logPrintln(validationResult.GetFormattedErrors())
	}

	shellRunner := &shell.RealShellRunner{}
	consoleLogger := logger.NewConsoleLogger(isJsonOutput)
	gitRepo := &git.RealGitRepository{}
	toolManager := service.NewCachingToolManagerService(shellRunner, consoleLogger, gitRepo)
	hookRunner := service.NewHookRunnerService(shellRunner, consoleLogger)
	qualityGate := service.NewQualityGateService(toolManager, hookRunner)

	if hookType == "mcp" {
		mcpServer := mcp.NewMCPServer(qualityGate, cfg)
		if err := mcpServer.Start(); err != nil {
			logPrint("Error starting MCP server: %v\n", err)
			return 1
		}
		return 0
	}

	if *fixFlag {
		logPrintln("Fixing fixable issues...")
		err = qualityGate.Fix(cfg, hookType)
		if err != nil {
			logPrint("Error fixing issues: %v\n", err)
			return 1
		}
		logPrintln("Fixable issues fixed successfully.")
		recordFixHistory(cfg, hookType)
		return 0
	}

	runStart := time.Now()
	results, err := qualityGate.Run(cfg, hookType, *parallelFlag)
	recordRunHistory(hookType, results, err == nil, time.Since(runStart))

	overallStatus := "success"
	if err != nil {
		overallStatus = "failure"
		logPrint("Quality gate failed: %v\n", err)
		if *outputFlag != "json" {
			return 1
		}
		// outputFlag == "json": fall through to report the failure as JSON
		// below instead of exiting immediately.
	}

	// isMCP is always false below this point: the hookType == "mcp" branch
	// above already returned whenever it was true.
	if isJsonOutput {
		// Convert results to include duration in a more readable format
		type JSONResult struct {
			Hook         domain.Hook `json:"hook"`
			Success      bool        `json:"success"`
			Output       string      `json:"output"`
			DurationMs   int64       `json:"duration_ms"`
			DurationText string      `json:"duration"`
		}

		var jsonResults []JSONResult
		for _, result := range results {
			jsonResults = append(jsonResults, JSONResult{
				Hook:         result.Hook,
				Success:      result.Success,
				Output:       result.Output,
				DurationMs:   result.Duration.Milliseconds(),
				DurationText: result.Duration.Round(time.Millisecond).String(),
			})
		}

		jsonOutput := struct {
			Status  string       `json:"status"`
			Results []JSONResult `json:"results"`
		}{
			Status:  overallStatus,
			Results: jsonResults,
		}
		jsonBytes, marshalErr := marshalJSONIndent(jsonOutput, "", "  ")
		if marshalErr != nil {
			logPrint("Error marshaling JSON: %v\n", marshalErr)
			return 1
		}
		fmt.Fprintln(stdout, string(jsonBytes)) // JSON output always goes to stdout
		if overallStatus == "failure" {
			return 1
		}
		return 0
	}

	// Reaching here in text mode guarantees overallStatus == "success": a
	// text-mode failure already returned above.
	logPrintln("Quality gate passed successfully.")
	return 0
}
