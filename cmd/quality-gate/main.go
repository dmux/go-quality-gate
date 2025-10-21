package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/dmux/go-quality-gate/internal/config"
	"github.com/dmux/go-quality-gate/internal/domain"
	"github.com/dmux/go-quality-gate/internal/infra/git"
	"github.com/dmux/go-quality-gate/internal/infra/logger"
	"github.com/dmux/go-quality-gate/internal/infra/shell"
	"github.com/dmux/go-quality-gate/internal/service"
)

func main() {
	installFlag := flag.Bool("install", false, "Install git hooks")
	initFlag := flag.Bool("init", false, "Initialize quality.yml")
	fixFlag := flag.Bool("fix", false, "Fix fixable issues")
	versionFlag := flag.Bool("version", false, "Show version information")
	versionFlagShort := flag.Bool("v", false, "Show version information (shorthand)")
	outputFlag := flag.String("output", "", "Output format (e.g., json)")

	flag.Parse()

	// Handle version flag first, before any other operations
	if *versionFlag || *versionFlagShort {
		if *outputFlag == "json" {
			// Output version information as JSON
			versionJSON := struct {
				Version   string `json:"version"`
				BuildDate string `json:"build_date"`
				GitCommit string `json:"git_commit"`
			}{
				Version:   Version,
				BuildDate: BuildDate,
				GitCommit: GitCommit,
			}
			jsonBytes, err := json.MarshalIndent(versionJSON, "", "  ")
			if err != nil {
				fmt.Printf("Error marshaling version JSON: %v\n", err)
				os.Exit(1)
			}
			fmt.Println(string(jsonBytes))
		} else {
			fmt.Println(VersionInfo())
		}
		return
	}

	args := flag.Args()

	// Helper function to print to the correct output stream
	logPrint := func(format string, args ...interface{}) {
		if *outputFlag == "json" {
			fmt.Fprintf(os.Stderr, format, args...)
		} else {
			fmt.Printf(format, args...)
		}
	}

	logPrintln := func(msg string) {
		if *outputFlag == "json" {
			fmt.Fprintln(os.Stderr, msg)
		} else {
			fmt.Println(msg)
		}
	}

	if *installFlag {
		logPrintln("Installing git hooks...")
		gitRepo := &git.RealGitRepository{}
		installationService := service.NewInstallationService(gitRepo)
		if err := installationService.InstallHooks(); err != nil {
			logPrint("Error installing git hooks: %v\n", err)
			os.Exit(1)
		}
		logPrintln("Git hooks installed successfully.")
		return
	}

	if *initFlag {
		logPrintln("Initializing quality.yml...")
		initService := service.NewInitService()
		if err := initService.Init(); err != nil {
			logPrint("Error initializing quality.yml: %v\n", err)
			os.Exit(1)
		}
		logPrintln("quality.yml initialized successfully.")
		return
	}

	if len(args) == 0 {
		logPrintln("Usage: quality-gate [OPTIONS] [HOOK_TYPE]")
		logPrintln("")
		logPrintln("Hook Types:")
		logPrintln("  pre-commit    Run pre-commit quality checks")
		logPrintln("  pre-push      Run pre-push quality checks")
		logPrintln("")
		logPrintln("Options:")
		logPrintln("  --install     Install git hooks in the current repository")
		logPrintln("  --init        Initialize quality.yml with intelligent analysis")
		logPrintln("  --fix         Automatically fix detected issues")
		logPrintln("  --version, -v Show version information")
		logPrintln("  --output json Output results in JSON format")
		logPrintln("")
		logPrintln("Examples:")
		logPrintln("  quality-gate --init              # Create quality.yml for your project")
		logPrintln("  quality-gate --install           # Install git hooks")
		logPrintln("  quality-gate pre-commit          # Run pre-commit checks")
		logPrintln("  quality-gate --fix pre-commit    # Fix issues and run checks")
		logPrintln("  quality-gate --version           # Show version")
		os.Exit(1)
	}

	hookType := args[0]

	cfg, err := config.LoadConfig("quality.yml")
	if err != nil {
		logPrint("Error loading quality.yml: %v\n", err)
		os.Exit(1)
	}

	shellRunner := &shell.RealShellRunner{}
	consoleLogger := logger.NewConsoleLogger(*outputFlag == "json")
	toolManager := service.NewToolManagerService(shellRunner, consoleLogger)
	hookRunner := service.NewHookRunnerService(shellRunner, consoleLogger)
	qualityGate := service.NewQualityGateService(toolManager, hookRunner)

	if *fixFlag {
		logPrintln("Fixing fixable issues...")
		err = qualityGate.Fix(cfg, hookType)
		if err != nil {
			logPrint("Error fixing issues: %v\n", err)
			os.Exit(1)
		}
		logPrintln("Fixable issues fixed successfully.")
		return
	}

	results, err := qualityGate.Run(cfg, hookType)
	
	overallStatus := "success"
	if err != nil {
		overallStatus = "failure"
		logPrint("Quality gate failed: %v\n", err)
		if *outputFlag != "json" {
			os.Exit(1)
		}
	}

	if *outputFlag == "json" {
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
			Status: overallStatus,
			Results: jsonResults,
		}
		jsonBytes, marshalErr := json.MarshalIndent(jsonOutput, "", "  ")
		if marshalErr != nil {
			logPrint("Error marshaling JSON: %v\n", marshalErr)
			os.Exit(1)
		}
		fmt.Println(string(jsonBytes)) // JSON output always goes to stdout
		if overallStatus == "failure" {
			os.Exit(1)
		}
	} else {
		if overallStatus == "success" {
			logPrintln("Quality gate passed successfully.")
		} else {
			// Error already logged above, just exit
			os.Exit(1)
		}
	}
}