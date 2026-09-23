package service

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dmux/go-quality-gate/internal/repository"
)

// HookMarker identifies hooks written by quality-gate so tampering or
// replacement can be detected.
const HookMarker = "# quality-gate-managed"

// ManagedHooks lists the hooks quality-gate installs, in install order.
var ManagedHooks = []string{"pre-commit", "commit-msg", "pre-push"}

var hookArgs = map[string]string{
	"pre-commit": `pre-commit`,
	"commit-msg": `commit-msg "$1"`,
	"pre-push":   `pre-push`,
}

// HookContent returns the script installed in a repository for a hook. The
// hook invokes the quality-gate binary by its absolute path rather than by
// name, so it isn't subject to PATH lookup at commit/push time.
func HookContent(hookType, binary string) string {
	return fmt.Sprintf("#!/bin/sh\n%s\nexec %s %s\n", HookMarker, binary, hookArgs[hookType])
}

// GlobalHookContent returns the script installed in a global core.hooksPath.
// It only acts in repositories that have a quality.yml, and still runs any
// repository-local hook that quality-gate does not manage, since a global
// hooks path disables .git/hooks.
func GlobalHookContent(hookType, binary string) string {
	return fmt.Sprintf(`#!/bin/sh
%s global
local_hook="$(git rev-parse --git-common-dir 2>/dev/null)/hooks/%s"
if [ -x "$local_hook" ] && ! grep -q "%s" "$local_hook"; then
	"$local_hook" "$@" || exit $?
fi
[ -f "$(git rev-parse --show-toplevel)/quality.yml" ] || exit 0
exec %s %s
`, HookMarker, hookType, HookMarker, binary, hookArgs[hookType])
}

// IsManagedHook reports whether a hook script was written by quality-gate.
func IsManagedHook(content string) bool {
	return strings.Contains(content, HookMarker)
}

// HookBinary returns the binary a managed hook executes, or "" if none.
func HookBinary(content string) string {
	for _, line := range strings.Split(content, "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[0] == "exec" {
			return fields[1]
		}
	}
	return ""
}

// InstallationService is responsible for installing the git hooks.

type InstallationService struct {
	gitRepo repository.GitRepository
}

// NewInstallationService creates a new InstallationService.

func NewInstallationService(gitRepo repository.GitRepository) *InstallationService {
	return &InstallationService{gitRepo: gitRepo}
}

// osExecutable and evalSymlinks are os.Executable/filepath.EvalSymlinks,
// swappable in tests to exercise InstallHooks' error paths — neither
// realistically fails for the running test binary itself.
var (
	osExecutable = os.Executable
	evalSymlinks = filepath.EvalSymlinks
)

// InstallHooks installs the pre-commit, commit-msg and pre-push git hooks.
func (s *InstallationService) InstallHooks() error {
	return s.install(HookContent)
}

// InstallGlobalHooks installs the hooks into a global hooks directory.
func (s *InstallationService) InstallGlobalHooks() error {
	return s.install(GlobalHookContent)
}

func (s *InstallationService) install(content func(hookType, binary string) string) error {
	execPath, err := osExecutable()
	if err != nil {
		return fmt.Errorf("failed to resolve executable path: %w", err)
	}

	realPath, err := evalSymlinks(execPath)
	if err != nil {
		return fmt.Errorf("failed to resolve symlink: %w", err)
	}

	for _, hook := range ManagedHooks {
		if err := s.gitRepo.InstallHook(hook, content(hook, realPath)); err != nil {
			return fmt.Errorf("failed to install %s hook: %w", hook, err)
		}
	}
	return nil
}
