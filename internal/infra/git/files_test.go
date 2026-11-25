package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"
)

// newFilesRepo creates a repository in a temp dir, chdirs into it and
// returns a helper that runs git there.
func newFilesRepo(t *testing.T) func(args ...string) {
	t.Helper()
	root := t.TempDir()
	t.Chdir(root)
	gitCmd := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@example.com",
			"GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@example.com",
			"GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_NOSYSTEM=1")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	gitCmd("init", "-q", "-b", "main")
	return gitCmd
}

func writeFile(t *testing.T, name, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(name), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(name, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestRealGitRepository_StagedFiles(t *testing.T) {
	gitCmd := newFilesRepo(t)
	writeFile(t, "keep.go", "package a\n")
	writeFile(t, "gone.go", "package a\n")
	gitCmd("add", ".")
	gitCmd("commit", "-q", "-m", "init")

	writeFile(t, "dir/with space.go", "package b\n")
	writeFile(t, "keep.go", "package a // changed\n")
	writeFile(t, "unstaged.go", "package a\n")
	gitCmd("add", "dir/with space.go", "keep.go")
	gitCmd("rm", "-q", "gone.go")

	files, err := (&RealGitRepository{}).StagedFiles()
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"dir/with space.go", "keep.go"}
	if !reflect.DeepEqual(files, want) {
		t.Errorf("StagedFiles() = %q, want %q", files, want)
	}
}

func TestRealGitRepository_StagedFiles_Empty(t *testing.T) {
	newFilesRepo(t)

	files, err := (&RealGitRepository{}).StagedFiles()
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 0 {
		t.Errorf("expected no staged files, got %q", files)
	}
}

func TestRealGitRepository_TrackedFiles_FromSubdirectory(t *testing.T) {
	gitCmd := newFilesRepo(t)
	writeFile(t, "a.go", "package a\n")
	writeFile(t, "sub/b.go", "package b\n")
	gitCmd("add", ".")

	if err := os.Chdir("sub"); err != nil {
		t.Fatal(err)
	}
	files, err := (&RealGitRepository{}).TrackedFiles()
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"a.go", "sub/b.go"}
	if !reflect.DeepEqual(files, want) {
		t.Errorf("TrackedFiles() = %q, want %q", files, want)
	}
}

func TestRealGitRepository_PushFiles(t *testing.T) {
	gitCmd := newFilesRepo(t)
	writeFile(t, "base.go", "package a\n")
	gitCmd("add", ".")
	gitCmd("commit", "-q", "-m", "init")

	// Without an upstream every tracked file is returned.
	files, err := (&RealGitRepository{}).PushFiles()
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"base.go"}; !reflect.DeepEqual(files, want) {
		t.Errorf("PushFiles() without upstream = %q, want %q", files, want)
	}

	// Fake an upstream at the current commit, then commit on top of it.
	gitCmd("remote", "add", "origin", "https://example.invalid/repo.git")
	gitCmd("update-ref", "refs/remotes/origin/main", "HEAD")
	gitCmd("config", "branch.main.remote", "origin")
	gitCmd("config", "branch.main.merge", "refs/heads/main")
	writeFile(t, "new.go", "package a\n")
	gitCmd("add", "new.go")
	gitCmd("commit", "-q", "-m", "new")

	files, err = (&RealGitRepository{}).PushFiles()
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"new.go"}; !reflect.DeepEqual(files, want) {
		t.Errorf("PushFiles() = %q, want %q", files, want)
	}
}

func TestListFiles_GitError(t *testing.T) {
	t.Chdir(t.TempDir())
	t.Setenv("GIT_CEILING_DIRECTORIES", filepath.Dir(t.TempDir()))

	if _, err := (&RealGitRepository{}).StagedFiles(); err == nil {
		t.Fatal("expected an error outside a git repository")
	}
}
