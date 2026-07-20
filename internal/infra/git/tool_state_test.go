package git

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRealGitRepository_ToolStateRoundTrip(t *testing.T) {
	repositoryRoot := t.TempDir()
	if err := os.Mkdir(filepath.Join(repositoryRoot, ".git"), 0755); err != nil {
		t.Fatal(err)
	}
	nestedDir := filepath.Join(repositoryRoot, "nested", "directory")
	if err := os.MkdirAll(nestedDir, 0755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(nestedDir)

	repository := &RealGitRepository{}
	if hash, err := repository.LoadToolsHash(); err != nil || hash != "" {
		t.Fatalf("missing cache returned hash %q and error %v", hash, err)
	}
	if err := repository.SaveToolsHash("expected-hash"); err != nil {
		t.Fatalf("saving cache: %v", err)
	}
	if hash, err := repository.LoadToolsHash(); err != nil || hash != "expected-hash" {
		t.Fatalf("loaded hash %q and error %v, want expected-hash and nil", hash, err)
	}

	statePath := filepath.Join(repositoryRoot, ".git", toolsHashRelativePath)
	info, err := os.Stat(statePath)
	if err != nil {
		t.Fatalf("stat cache file: %v", err)
	}
	if got := info.Mode().Perm(); got != 0644 {
		t.Fatalf("cache permissions are %o, want 644", got)
	}
}

func TestRealGitRepository_CorruptToolStateIsReturnedAsCacheMiss(t *testing.T) {
	repositoryRoot := t.TempDir()
	statePath := filepath.Join(repositoryRoot, ".git", toolsHashRelativePath)
	if err := os.MkdirAll(filepath.Dir(statePath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(statePath, []byte("incomplete-state\n"), 0644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(repositoryRoot)

	repository := &RealGitRepository{}
	hash, err := repository.LoadToolsHash()
	if err != nil {
		t.Fatalf("loading corrupt state: %v", err)
	}
	if hash != "incomplete-state" {
		t.Fatalf("loaded hash %q, want incomplete-state", hash)
	}
}
