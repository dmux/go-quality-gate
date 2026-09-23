package service

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/dmux/go-quality-gate/internal/config"
	"github.com/dmux/go-quality-gate/internal/repository"
)

// DoctorCheck is the outcome of one installation health check.
type DoctorCheck struct {
	Name   string `json:"name"`
	OK     bool   `json:"ok"`
	Detail string `json:"detail,omitempty"`
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

	return append(checks, s.checkConfig())
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

func (s *DoctorService) checkConfig() DoctorCheck {
	cfg, err := config.LoadConfig(s.configPath)
	if err != nil {
		return DoctorCheck{Name: s.configPath, Detail: err.Error()}
	}
	result := config.NewConfigValidator(cfg).Validate()
	if !result.Valid {
		return DoctorCheck{Name: s.configPath, Detail: fmt.Sprintf("invalid:\n%s", result.GetFormattedErrors())}
	}
	return DoctorCheck{Name: s.configPath, OK: true}
}
