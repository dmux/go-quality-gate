package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/dmux/go-quality-gate/internal/config"
	"github.com/dmux/go-quality-gate/internal/domain"
	"github.com/dmux/go-quality-gate/internal/infra/git"
	"github.com/dmux/go-quality-gate/internal/infra/logger"
	"github.com/dmux/go-quality-gate/internal/infra/shell"
	"github.com/dmux/go-quality-gate/internal/mcp"
	"github.com/dmux/go-quality-gate/internal/service"
)

func main() {
	installFlag := flag.Bool("install", false, "Install git hooks")
	initFlag := flag.Bool("init", false, "Initialize quality.yml")
	fixFlag := flag.Bool("fix", false, "Fix fixable issues")
	versionFlag := flag.Bool("version", false, "Show version information")
	versionFlagShort := flag.Bool("v", false, "Show version information (shorthand)")
	outputFlag := flag.String("output", "", "Output format (e.g., json)")
	globalFlag := flag.Bool("global", false, "With --install, install hooks for every repository of this user")

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
	isMCP := len(args) > 0 && args[0] == "mcp"
	isJsonOutput := *outputFlag == "json" || isMCP

	// Helper function to print to the correct output stream
	logPrint := func(format string, args ...interface{}) {
		if isJsonOutput {
			fmt.Fprintf(os.Stderr, format, args...)
		} else {
			fmt.Printf(format, args...)
		}
	}

	logPrintln := func(msg string) {
		if isJsonOutput {
			fmt.Fprintln(os.Stderr, msg)
		} else {
			fmt.Println(msg)
		}
	}

	if *installFlag {
		if *globalFlag {
			if err := installGlobalHooks(); err != nil {
				logPrint("Error installing global git hooks: %v\n", err)
				os.Exit(1)
			}
			logPrintln("Global git hooks installed; every repository with a quality.yml is now gated.")
			return
		}
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
		logPrintln("  commit-msg    Watermark a commit message (called by the git hook)")
		logPrintln("  verify        Verify commit watermarks (e.g. in CI)")
		logPrintln("  doctor        Check that hooks and configuration are installed")
		logPrintln("  mcp           Start Model Context Protocol (MCP) server")
		logPrintln("")
		logPrintln("Options:")
		logPrintln("  --install     Install git hooks in the current repository")
		logPrintln("  --global      With --install, gate every repository of this user")
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
		logPrintln("  quality-gate verify --range origin/main..HEAD  # Check watermarks")
		logPrintln("  QG_SKIP=\"reason\" git commit      # Skip checks, leaving an audit trailer")
		logPrintln("  quality-gate --version           # Show version")
		os.Exit(1)
	}

	hookType := args[0]
	gitRepo := &git.RealGitRepository{}
	attestation := service.NewAttestationService(gitRepo, gitRepo, Version)

	switch hookType {
	case "commit-msg":
		if len(args) < 2 {
			logPrintln("Usage: quality-gate commit-msg <message-file>")
			os.Exit(1)
		}
		runCommitMsg(attestation, args[1])
		return
	case "verify":
		os.Exit(runVerify(attestation, args[1:], *outputFlag))
	case "doctor":
		os.Exit(runDoctor(gitRepo, *outputFlag))
	}

	if skipReason := os.Getenv(service.SkipEnvVar); skipReason != "" && (hookType == "pre-commit" || hookType == "pre-push") {
		logPrint("⚠️  Quality gate skipped (%s=%q); the commit will carry a %s trailer.\n", service.SkipEnvVar, skipReason, domain.SkipTrailer)
		return
	}

	cfg, err := config.LoadConfig("quality.yml")
	if err != nil {
		logPrint("Error loading quality.yml: %v\n", err)
		os.Exit(1)
	}

	shellRunner := &shell.RealShellRunner{}
	consoleLogger := logger.NewConsoleLogger(isJsonOutput)
	toolManager := service.NewCachingToolManagerService(shellRunner, consoleLogger, gitRepo)
	hookRunner := service.NewHookRunnerService(shellRunner, consoleLogger)
	qualityGate := service.NewQualityGateService(toolManager, hookRunner)

	if hookType == "mcp" {
		mcpServer := mcp.NewMCPServer(qualityGate, cfg)
		mcpServer.SetVersion(Version)
		mcpServer.OnPass(func(hookType string, results []domain.ExecutionResult) error {
			return recordAttestation(attestation, hookType, results)
		})
		if err := mcpServer.Start(); err != nil {
			logPrint("Error starting MCP server: %v\n", err)
			os.Exit(1)
		}
		return
	}

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
	} else if attestErr := recordAttestation(attestation, hookType, results); attestErr != nil {
		logPrint("⚠️  Could not record quality gate attestation: %v\n", attestErr)
	}

	if isJsonOutput && !isMCP {
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
		jsonBytes, marshalErr := json.MarshalIndent(jsonOutput, "", "  ")
		if marshalErr != nil {
			logPrint("Error marshaling JSON: %v\n", marshalErr)
			os.Exit(1)
		}
		fmt.Println(string(jsonBytes)) // JSON output always goes to stdout
		if overallStatus == "failure" {
			os.Exit(1)
		}
	} else if !isMCP {
		if overallStatus == "success" {
			logPrintln("Quality gate passed successfully.")
		} else {
			// Error already logged above, just exit
			os.Exit(1)
		}
	}
}

// recordAttestation stores the watermark for the commit being created. Only
// pre-commit results describe the staged content, so other hooks are ignored.
func recordAttestation(attestation *service.AttestationService, hookType string, results []domain.ExecutionResult) error {
	if hookType != "pre-commit" {
		return nil
	}
	configContent, err := os.ReadFile("quality.yml")
	if err != nil {
		return err
	}
	return attestation.Record(results, configContent)
}

// runCommitMsg watermarks the commit message. It never blocks the commit:
// enforcement happens in CI through `quality-gate verify`.
func runCommitMsg(attestation *service.AttestationService, messageFile string) {
	configContent, _ := os.ReadFile("quality.yml")
	outcome, err := attestation.Stamp(messageFile, configContent, os.Getenv(service.SkipEnvVar))
	if err != nil {
		fmt.Fprintf(os.Stderr, "⚠️  Could not watermark commit: %v\n", err)
		return
	}

	switch outcome {
	case service.StampAdded:
		fmt.Fprintln(os.Stderr, "🔏 Commit watermarked by quality-gate.")
	case service.StampSkipTrailerAdded:
		fmt.Fprintf(os.Stderr, "⚠️  Quality gate skipped; recorded in the %s trailer.\n", domain.SkipTrailer)
	case service.StampNoAttestation:
		fmt.Fprintln(os.Stderr, "⚠️  No passing quality gate run found for this commit; it will not be watermarked and CI may reject it.")
	case service.StampStale:
		fmt.Fprintln(os.Stderr, "⚠️  Staged content or quality.yml changed after the checks ran; commit not watermarked. Commit again to re-run the checks.")
	}
}

func runVerify(attestation *service.AttestationService, args []string, output string) int {
	fs := flag.NewFlagSet("verify", flag.ExitOnError)
	rangeSpec := fs.String("range", "HEAD", "Commit or revision range to verify (e.g. origin/main..HEAD)")
	policy := fs.String("policy", "strict", "strict: every commit must be attested; allow-skip: also accept "+domain.SkipTrailer)
	configPath := fs.String("config", "quality.yml", "Path of quality.yml relative to the repository root")
	fs.StringVar(&output, "output", output, "Output format (e.g., json)")
	_ = fs.Parse(args)

	if *policy != "strict" && *policy != "allow-skip" {
		fmt.Fprintf(os.Stderr, "Unknown policy %q (use strict or allow-skip)\n", *policy)
		return 1
	}

	verifications, err := attestation.Verify(*rangeSpec, *configPath, *policy == "allow-skip")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error verifying commits: %v\n", err)
		return 1
	}

	passed := true
	for _, v := range verifications {
		passed = passed && v.Accepted
	}

	if output == "json" {
		status := "success"
		if !passed {
			status = "failure"
		}
		jsonBytes, _ := json.MarshalIndent(struct {
			Status  string                      `json:"status"`
			Range   string                      `json:"range"`
			Policy  string                      `json:"policy"`
			Commits []domain.CommitVerification `json:"commits"`
		}{status, *rangeSpec, *policy, verifications}, "", "  ")
		fmt.Println(string(jsonBytes))
	} else {
		for _, v := range verifications {
			icon := "✅"
			if !v.Accepted {
				icon = "❌"
			}
			line := fmt.Sprintf("%s %.12s %s", icon, v.Commit, v.Status)
			if v.SkipReason != "" {
				line += fmt.Sprintf(" (reason: %s)", v.SkipReason)
			}
			if v.Detail != "" {
				line += ": " + v.Detail
			}
			fmt.Println(line)
		}
		if len(verifications) == 0 {
			fmt.Println("No commits to verify.")
		} else if passed {
			fmt.Printf("All %d commit(s) passed the quality gate.\n", len(verifications))
		} else {
			fmt.Println("Some commits did not pass the quality gate (were hooks skipped with --no-verify?).")
		}
	}

	if !passed {
		return 1
	}
	return 0
}

func runDoctor(gitRepo *git.RealGitRepository, output string) int {
	checks := service.NewDoctorService(gitRepo, exec.LookPath, "quality.yml").Run()

	healthy := true
	for _, c := range checks {
		healthy = healthy && c.OK
	}

	if output == "json" {
		jsonBytes, _ := json.MarshalIndent(struct {
			Healthy bool                  `json:"healthy"`
			Checks  []service.DoctorCheck `json:"checks"`
		}{healthy, checks}, "", "  ")
		fmt.Println(string(jsonBytes))
	} else {
		for _, c := range checks {
			icon := "✅"
			if !c.OK {
				icon = "❌"
			}
			if c.Detail != "" {
				fmt.Printf("%s %s: %s\n", icon, c.Name, c.Detail)
			} else {
				fmt.Printf("%s %s\n", icon, c.Name)
			}
		}
	}

	if !healthy {
		return 1
	}
	return 0
}

func installGlobalHooks() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	dir := filepath.Join(home, ".quality-gate", "hooks")
	if err := service.NewInstallationService(&git.DirHookRepository{Dir: dir}).InstallGlobalHooks(); err != nil {
		return err
	}
	return git.SetGlobalHooksPath(dir)
}
