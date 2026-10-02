package service

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/dmux/go-quality-gate/internal/repository"
)

const (
	// UpdateModulePath is the Go module path of quality-gate.
	UpdateModulePath = "github.com/dmux/go-quality-gate"

	// UpdateInstallTarget is the package `--update` installs. It mirrors the
	// documented `go install ...@latest` command so the two stay in sync.
	UpdateInstallTarget = UpdateModulePath + "/cmd/quality-gate@latest"

	// updateBinaryName is the base name of the installed executable.
	updateBinaryName = "quality-gate"
)

// UpdateResult reports the outcome of a self-update.
type UpdateResult struct {
	PreviousVersion string `json:"previous_version"`
	NewVersion      string `json:"new_version"`
	Updated         bool   `json:"updated"`
}

// UpdateService upgrades the installed quality-gate binary by running
// `go install <UpdateInstallTarget>`, exactly as a user would by hand, and
// then reports the version that ended up on disk.
type UpdateService struct {
	runner         repository.ShellRunner
	currentVersion string
}

// NewUpdateService creates a new UpdateService. currentVersion is the version
// of the running binary, used as the "previous" version in the result.
func NewUpdateService(runner repository.ShellRunner, currentVersion string) *UpdateService {
	return &UpdateService{runner: runner, currentVersion: currentVersion}
}

// Update runs `go install <UpdateInstallTarget>` and reports the version now
// installed. It relies on the Go toolchain being available, since that is how
// quality-gate is installed via `go install` in the first place.
func (s UpdateService) Update() (UpdateResult, error) {
	result := UpdateResult{PreviousVersion: s.currentVersion}

	if _, err := s.runner.Run("go version"); err != nil {
		return result, fmt.Errorf("the Go toolchain is required to self-update; install Go or update manually with `go install %s`: %w", UpdateInstallTarget, err)
	}

	out, err := s.runner.Run("go install " + UpdateInstallTarget)
	if err != nil {
		return result, fmt.Errorf("failed to run `go install %s`: %w\n%s", UpdateInstallTarget, err, strings.TrimSpace(out))
	}

	newVersion, err := s.installedVersion()
	if err != nil {
		return result, err
	}
	result.NewVersion = newVersion
	result.Updated = newVersion != "" && newVersion != s.currentVersion
	return result, nil
}

// installedVersion reads the module version embedded in the freshly installed
// binary. It reads the build info rather than running `quality-gate --version`
// because `go install` does not inject the -ldflags the release build uses, so
// the binary's Version variable would still report the development default.
func (s UpdateService) installedVersion() (string, error) {
	binPath, err := s.binaryPath()
	if err != nil {
		return "", err
	}
	out, err := s.runner.Run(fmt.Sprintf("go version -m %q", binPath))
	if err != nil {
		return "", fmt.Errorf("installed quality-gate but could not read its version from %q: %w\n%s", binPath, err, strings.TrimSpace(out))
	}
	return parseModuleVersion(out), nil
}

// binaryPath resolves where `go install` writes the quality-gate binary,
// preferring GOBIN and falling back to the first GOPATH entry's bin directory.
func (s UpdateService) binaryPath() (string, error) {
	name := updateBinaryName
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	if gobin := s.goEnv("GOBIN"); gobin != "" {
		return filepath.Join(gobin, name), nil
	}
	gopath := s.goEnv("GOPATH")
	if gopath == "" {
		return "", fmt.Errorf("could not locate the installed binary: neither GOBIN nor GOPATH is set")
	}
	// GOPATH may list several paths; `go install` writes to the first one.
	first := strings.TrimSpace(strings.Split(gopath, string(os.PathListSeparator))[0])
	return filepath.Join(first, "bin", name), nil
}

// goEnv returns the value of a `go env` variable, or "" if it can't be read.
func (s UpdateService) goEnv(key string) string {
	out, err := s.runner.Run("go env " + key)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(out)
}

// parseModuleVersion extracts the module version from `go version -m` output,
// e.g. the "v1.4.0" in a line like:
//
//	mod	github.com/dmux/go-quality-gate	v1.4.0	h1:...
func parseModuleVersion(goVersionOutput string) string {
	for _, line := range strings.Split(goVersionOutput, "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 3 && fields[0] == "mod" && fields[1] == UpdateModulePath {
			return fields[2]
		}
	}
	return ""
}
