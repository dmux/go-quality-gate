package git

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	toolsHashRelativePath   = "quality-gate/tools.sha256"
	attestationRelativePath = "quality-gate/attestation.json"
)

// RealGitRepository is a real implementation of the GitRepository interface.

type RealGitRepository struct{}

// createHookFile is os.Create, swappable in tests to exercise the
// write-failure path in InstallHook, which a real OS-level write()
// failure isn't practical to reproduce portably. *os.File satisfies
// io.WriteCloser.
var createHookFile = func(path string) (io.WriteCloser, error) {
	return os.Create(path)
}

// chmodFile is os.Chmod, swappable in tests to exercise InstallHook's final
// chmod-failure path without depending on OS-specific permission tricks.
var chmodFile = os.Chmod

// InstallHook implements the GitRepository interface.

func (r *RealGitRepository) InstallHook(hookType string, content string) error {
	hooksDir, err := r.repositoryHooksDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(hooksDir, 0755); err != nil {
		return err
	}

	hookPath := filepath.Join(hooksDir, hookType)

	f, err := createHookFile(hookPath)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.Write([]byte(content))
	if err != nil {
		return err
	}

	return chmodFile(hookPath, 0755)
}

// HooksDir returns the directory git reads hooks from, honouring
// core.hooksPath and linked worktrees.
func (r *RealGitRepository) HooksDir() (string, error) {
	if out, err := runGit("rev-parse", "--path-format=absolute", "--git-path", "hooks"); err == nil && out != "" {
		return out, nil
	}

	gitDir, err := findGitDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(gitDir, "hooks"), nil
}

// repositoryHooksDir returns where a per-repository install writes hooks.
// A core.hooksPath set outside the repository (e.g. by --install --global)
// is shared by every repository, so it is never written to from here.
func (r *RealGitRepository) repositoryHooksDir() (string, error) {
	if _, err := runGit("config", "--local", "--get", "core.hooksPath"); err == nil {
		return r.HooksDir()
	}
	if out, err := runGit("rev-parse", "--path-format=absolute", "--git-common-dir"); err == nil && out != "" {
		return filepath.Join(out, "hooks"), nil
	}

	gitDir, err := findGitDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(gitDir, "hooks"), nil
}

// LoadToolsHash returns the hash of the tools configuration that was last
// successfully validated. A missing state file is treated as an empty cache.
func (r *RealGitRepository) LoadToolsHash() (string, error) {
	content, err := readState(toolsHashRelativePath)
	if err != nil || content == nil {
		return "", err
	}
	return strings.TrimSpace(string(content)), nil
}

// SaveToolsHash atomically records a successfully validated tools
// configuration, avoiding a partially written cache after interruption.
func (r *RealGitRepository) SaveToolsHash(hash string) error {
	return writeStateAtomic(toolsHashRelativePath, []byte(hash+"\n"))
}

// LoadAttestation returns the pending attestation, or nil when none exists.
func (r *RealGitRepository) LoadAttestation() ([]byte, error) {
	return readState(attestationRelativePath)
}

// SaveAttestation atomically records the attestation of a passing run.
func (r *RealGitRepository) SaveAttestation(content []byte) error {
	return writeStateAtomic(attestationRelativePath, content)
}

// DeleteAttestation removes the pending attestation so it is used only once.
func (r *RealGitRepository) DeleteAttestation() error {
	gitDir, err := findGitDir()
	if err != nil {
		return err
	}
	err = os.Remove(filepath.Join(gitDir, attestationRelativePath))
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

// WriteTree implements the CommitRepository interface.
func (r *RealGitRepository) WriteTree() (string, error) {
	return runGit("write-tree")
}

// CommitTree implements the CommitRepository interface.
func (r *RealGitRepository) CommitTree(rev string) (string, error) {
	return runGit("rev-parse", "--verify", rev+"^{tree}")
}

// CommitMessage implements the CommitRepository interface.
func (r *RealGitRepository) CommitMessage(rev string) (string, error) {
	return runGit("log", "-1", "--format=%B", rev)
}

// ShowFile implements the CommitRepository interface.
func (r *RealGitRepository) ShowFile(rev, path string) ([]byte, error) {
	if _, err := runGit("cat-file", "-e", rev+":"+path); err != nil {
		return nil, nil
	}
	cmd := exec.Command("git", "show", rev+":"+path)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git show %s:%s: %w: %s", rev, path, err, strings.TrimSpace(stderr.String()))
	}
	return out, nil
}

// RevList implements the CommitRepository interface. A spec without a range
// operator selects that single commit.
func (r *RealGitRepository) RevList(rangeSpec string) ([]string, error) {
	args := []string{"rev-list", "--no-merges"}
	if !strings.Contains(rangeSpec, "..") && !strings.HasSuffix(rangeSpec, "^!") {
		args = append(args, "--max-count=1")
	}
	out, err := runGit(append(args, rangeSpec)...)
	if err != nil {
		return nil, err
	}
	if out == "" {
		return nil, nil
	}
	return strings.Split(out, "\n"), nil
}

// AddTrailer implements the CommitRepository interface.
func (r *RealGitRepository) AddTrailer(messageFile, trailer string) error {
	_, err := runGit("interpret-trailers", "--in-place", "--if-exists", "replace", "--trailer", trailer, messageFile)
	return err
}

func runGit(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
	}
	return strings.TrimSpace(string(out)), nil
}

func readState(relativePath string) ([]byte, error) {
	gitDir, err := findGitDir()
	if err != nil {
		return nil, err
	}

	content, err := os.ReadFile(filepath.Join(gitDir, relativePath))
	if os.IsNotExist(err) {
		return nil, nil
	}
	return content, err
}

// tempWriteCloser is the subset of *os.File's methods writeStateAtomic needs
// for its create-write-chmod-close-rename sequence, extracted so tests can
// substitute a fake that fails on a specific step.
type tempWriteCloser interface {
	io.Writer
	Chmod(mode os.FileMode) error
	Close() error
	Name() string
}

// createTempFile is os.CreateTemp, swappable in tests to exercise
// writeStateAtomic's write/chmod/close failure paths, which real OS-level
// failures on an already-created temp file aren't practical to reproduce
// portably.
var createTempFile = func(dir, pattern string) (tempWriteCloser, error) {
	return os.CreateTemp(dir, pattern)
}

func writeStateAtomic(relativePath string, content []byte) error {
	gitDir, err := findGitDir()
	if err != nil {
		return err
	}

	statePath := filepath.Join(gitDir, relativePath)
	stateDir := filepath.Dir(statePath)
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		return err
	}

	tempFile, err := createTempFile(stateDir, "state-*.tmp")
	if err != nil {
		return err
	}
	tempPath := tempFile.Name()
	defer os.Remove(tempPath)

	if _, err := tempFile.Write(content); err != nil {
		tempFile.Close()
		return err
	}
	if err := tempFile.Chmod(0644); err != nil {
		tempFile.Close()
		return err
	}
	if err := tempFile.Close(); err != nil {
		return err
	}

	return os.Rename(tempPath, statePath)
}

// getwd is os.Getwd, swappable in tests to exercise findGitDir's error path
// (os.Getwd failing is otherwise not reliably reproducible across
// platforms — e.g. macOS still resolves a deleted cwd).
var getwd = os.Getwd

// findGitDir locates the git directory. In linked worktrees, where .git is a
// file, git itself resolves the real directory.
func findGitDir() (string, error) {
	path, err := getwd()
	if err != nil {
		return "", err
	}

	for {
		gitPath := filepath.Join(path, ".git")
		if info, err := os.Stat(gitPath); err == nil {
			if info.IsDir() {
				return gitPath, nil
			}
			if out, err := runGit("rev-parse", "--absolute-git-dir"); err == nil && out != "" {
				return out, nil
			}
		}

		parent := filepath.Dir(path)
		if parent == path {
			return "", fmt.Errorf(".git directory not found")
		}
		path = parent
	}
}

// DirHookRepository installs hooks into a fixed directory, used for a global
// core.hooksPath shared by every repository on the machine.
type DirHookRepository struct {
	Dir string
}

// InstallHook implements the GitRepository interface.
func (r *DirHookRepository) InstallHook(hookType string, content string) error {
	if err := os.MkdirAll(r.Dir, 0755); err != nil {
		return err
	}
	hookPath := filepath.Join(r.Dir, hookType)
	if err := os.WriteFile(hookPath, []byte(content), 0755); err != nil {
		return err
	}
	return os.Chmod(hookPath, 0755)
}

// SetGlobalHooksPath points the user's git configuration at dir.
func SetGlobalHooksPath(dir string) error {
	_, err := runGit("config", "--global", "core.hooksPath", dir)
	return err
}
