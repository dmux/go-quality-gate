package service

import (
	"fmt"
	"strings"

	"github.com/dmux/go-quality-gate/internal/repository"
)

// HookMarker identifies hooks written by quality-gate so tampering or
// replacement can be detected.
const HookMarker = "# quality-gate-managed"

// ManagedHooks lists the hooks quality-gate installs, in install order.
var ManagedHooks = []string{"pre-commit", "commit-msg", "pre-push"}

var hookCommands = map[string]string{
	"pre-commit": `exec quality-gate pre-commit`,
	"commit-msg": `exec quality-gate commit-msg "$1"`,
	"pre-push":   `exec quality-gate pre-push`,
}

// HookContent returns the script installed in a repository for a hook.
func HookContent(hookType string) string {
	return fmt.Sprintf("#!/bin/sh\n%s\n%s\n", HookMarker, hookCommands[hookType])
}

// GlobalHookContent returns the script installed in a global core.hooksPath.
// It only acts in repositories that have a quality.yml, and still runs any
// repository-local hook that quality-gate does not manage, since a global
// hooks path disables .git/hooks.
func GlobalHookContent(hookType string) string {
	return fmt.Sprintf(`#!/bin/sh
%s global
local_hook="$(git rev-parse --git-common-dir 2>/dev/null)/hooks/%s"
if [ -x "$local_hook" ] && ! grep -q "%s" "$local_hook"; then
	"$local_hook" "$@" || exit $?
fi
[ -f "$(git rev-parse --show-toplevel)/quality.yml" ] || exit 0
%s
`, HookMarker, hookType, HookMarker, hookCommands[hookType])
}

// IsManagedHook reports whether a hook script was written by quality-gate.
func IsManagedHook(content string) bool {
	return strings.Contains(content, HookMarker)
}

// InstallationService is responsible for installing the git hooks.

type InstallationService struct {
	gitRepo repository.GitRepository
}

// NewInstallationService creates a new InstallationService.

func NewInstallationService(gitRepo repository.GitRepository) *InstallationService {
	return &InstallationService{gitRepo: gitRepo}
}

// InstallHooks installs the pre-commit, commit-msg and pre-push git hooks.

func (s *InstallationService) InstallHooks() error {
	return s.install(HookContent)
}

// InstallGlobalHooks installs the hooks into a global hooks directory.
func (s *InstallationService) InstallGlobalHooks() error {
	return s.install(GlobalHookContent)
}

func (s *InstallationService) install(content func(string) string) error {
	for _, hook := range ManagedHooks {
		if err := s.gitRepo.InstallHook(hook, content(hook)); err != nil {
			return fmt.Errorf("failed to install %s hook: %w", hook, err)
		}
	}
	return nil
}
