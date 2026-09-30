package service

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/dmux/go-quality-gate/internal/config"
	"github.com/dmux/go-quality-gate/internal/repository"
)

// DoctorCheck is the outcome of one installation health check. Skipped marks
// a check that could not be performed at all: it is not a failure, but
// reporting it as a plain pass would claim verification that never happened.
type DoctorCheck struct {
	Name    string `json:"name"`
	OK      bool   `json:"ok"`
	Skipped bool   `json:"skipped,omitempty"`
	Detail  string `json:"detail,omitempty"`
}

// DoctorService reports whether quality-gate is correctly wired into the
// current repository, so a missing or tampered hook is caught early.
type DoctorService struct {
	hooks      repository.HookInspector
	configPath string
}

// NewDoctorService creates a new DoctorService.
func NewDoctorService(hooks repository.HookInspector, configPath string) *DoctorService {
	return &DoctorService{hooks: hooks, configPath: configPath}
}

// Run executes every check. It never stops at the first failure.
func (s *DoctorService) Run() []DoctorCheck {
	var checks []DoctorCheck

	hooksDir, err := s.hooks.HooksDir()
	if err != nil {
		checks = append(checks, DoctorCheck{Name: "hooks directory", Detail: err.Error()})
	} else {
		for _, hook := range ManagedHooks {
			checks = append(checks, checkHook(filepath.Join(hooksDir, hook), hook))
		}
	}

	cfg, err := config.LoadConfig(s.configPath)
	if err != nil {
		return append(checks, DoctorCheck{Name: s.configPath, Detail: err.Error()})
	}

	result := config.NewConfigValidator(cfg).Validate()
	if !result.Valid {
		checks = append(checks, DoctorCheck{Name: s.configPath, Detail: fmt.Sprintf("invalid:\n%s", result.GetFormattedErrors())})
		return checks
	}
	checks = append(checks, DoctorCheck{Name: s.configPath, OK: true})

	for _, tool := range cfg.Tools {
		checks = append(checks, checkToolAvailability(tool))
	}

	// filepath.Dir("quality.yml") is ".", so resolve it to keep the reported
	// paths absolute and unambiguous.
	projectDir, err := filepath.Abs(filepath.Dir(s.configPath))
	if err != nil {
		projectDir = filepath.Dir(s.configPath)
	}
	structure, err := NewLanguageDetector(projectDir).DetectProjectStructure()
	if err != nil {
		checks = append(checks, DoctorCheck{Name: "project runtimes", Detail: err.Error()})
		return checks
	}
	for _, rc := range DetectRuntimes(structure) {
		checks = append(checks, checkRuntimeAvailability(rc))
	}

	return checks
}

// checkToolAvailability reports whether a configured tool's binary is on
// PATH. Commands piped/chained/qualified with a path are skipped, the same
// exclusion config.ConfigValidator.validateToolAvailability applies, since
// running a fragment of such a command wouldn't be a meaningful check.
func checkToolAvailability(tool config.Tool) DoctorCheck {
	name := tool.Name + " tool"

	parts := strings.Fields(tool.CheckCommand)
	if len(parts) == 0 {
		return DoctorCheck{Name: name, Detail: "no check_command configured"}
	}
	cmdName := parts[0]
	if strings.ContainsAny(cmdName, "/|&") {
		return DoctorCheck{Name: name, OK: true, Skipped: true, Detail: "not verified (complex check_command)"}
	}

	if _, err := exec.LookPath(cmdName); err != nil {
		return DoctorCheck{Name: name, Detail: fmt.Sprintf("%q not found in PATH; install with: %s", cmdName, tool.InstallCommand)}
	}
	return DoctorCheck{Name: name, OK: true, Detail: cmdName}
}

// checkRuntimeAvailability reports whether a detected language's base
// runtime is on PATH, with an OS-specific install hint when it's not.
func checkRuntimeAvailability(rc RuntimeCheck) DoctorCheck {
	name := string(rc.Language) + " runtime"
	if !rc.Installed {
		return DoctorCheck{Name: name, Detail: fmt.Sprintf("%q not found in PATH; install with: %s", rc.Binary, rc.InstallHint)}
	}
	return DoctorCheck{Name: name, OK: true, Detail: rc.Binary}
}

func checkHook(path, hook string) DoctorCheck {
	name := hook + " hook"
	info, err := os.Stat(path)
	if err != nil {
		return DoctorCheck{Name: name, Detail: "not installed; run `quality-gate --install`"}
	}
	if info.Mode().Perm()&0111 == 0 {
		return DoctorCheck{Name: name, Detail: path + " is not executable"}
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return DoctorCheck{Name: name, Detail: err.Error()}
	}
	if !IsManagedHook(string(content)) {
		return DoctorCheck{Name: name, Detail: path + " was not written by quality-gate; run `quality-gate --install`"}
	}
	// Hooks call the binary by absolute path, so a moved or deleted binary
	// silently breaks them.
	binary := HookBinary(string(content))
	if info, err := os.Stat(binary); err != nil || info.IsDir() || info.Mode().Perm()&0111 == 0 {
		return DoctorCheck{Name: name, Detail: fmt.Sprintf("runs %q, which is missing or not executable; run `quality-gate --install`", binary)}
	}
	return DoctorCheck{Name: name, OK: true, Detail: path}
}
