package service

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/dmux/go-quality-gate/internal/config"
	"github.com/dmux/go-quality-gate/internal/domain"
)

func TestMatchGlob(t *testing.T) {
	cases := []struct {
		pattern, file string
		want          bool
	}{
		{"*.go", "main.go", true},
		{"*.go", "cmd/quality-gate/main.go", true},
		{"*.go", "main.go.txt", false},
		{"cmd/*.go", "cmd/main.go", true},
		{"cmd/*.go", "cmd/sub/main.go", false},
		{"web/**/*.ts", "web/a.ts", true},
		{"web/**/*.ts", "web/src/app/page.ts", true},
		{"web/**/*.ts", "other/a.ts", false},
		{"**/*.py", "a/b/c.py", true},
		{"web/**", "web/src/a.ts", true},
		{"Dockerfile", "build/Dockerfile", true},
	}
	for _, c := range cases {
		if got := matchGlob(c.pattern, c.file); got != c.want {
			t.Errorf("matchGlob(%q, %q) = %v, want %v", c.pattern, c.file, got, c.want)
		}
	}
}

func TestFilterGlob(t *testing.T) {
	files := []string{"a.go", "b.py", "web/c.ts", "README.md"}

	if got := filterGlob(files, nil); !reflect.DeepEqual(got, files) {
		t.Errorf("no patterns should keep every file, got %q", got)
	}
	got := filterGlob(files, []string{"*.go", "*.ts"})
	if want := []string{"a.go", "web/c.ts"}; !reflect.DeepEqual(got, want) {
		t.Errorf("filterGlob = %q, want %q", got, want)
	}
}

func TestExpandCommand(t *testing.T) {
	files := []string{"main.go", "dir/with space.go", "it's.go", "-flag.go", "dir/b.go", "x/y/z.go"}

	got := expandCommand("gofmt -l {files}", files)
	want := `gofmt -l main.go 'dir/with space.go' 'it'\''s.go' ./-flag.go dir/b.go x/y/z.go`
	if got != want {
		t.Errorf("expandCommand {files}:\n got %s\nwant %s", got, want)
	}

	got = expandCommand("go vet {dirs}", files)
	if want := "go vet . ./dir ./x/y"; got != want {
		t.Errorf("expandCommand {dirs} = %q, want %q", got, want)
	}

	if got := expandCommand("go test ./...", files); got != "go test ./..." {
		t.Errorf("a command without placeholders must not change, got %q", got)
	}
}

func scopedConfig() *config.Config {
	return &config.Config{
		Hooks: config.Hooks{
			"backend": {
				"pre-commit": []config.Hook{
					{Name: "Format", Command: "gofmt -l {files}", FixCommand: "gofmt -w {files}", Glob: []string{"*.go"}},
					{Name: "Vet", Command: "go vet {dirs}", Glob: []string{"*.go"}},
					{Name: "Python", Command: "ruff check {files}", FixCommand: "ruff check --fix {files}", Glob: []string{"*.py"}},
					{Name: "Tests", Command: "go test ./..."},
				},
				"pre-push": []config.Hook{
					{Name: "Lint", Command: "lint {files}"},
				},
			},
		},
	}
}

// echoRunner returns one passing result per hook it receives, like the real
// runner, so merged results can be checked.
type echoRunner struct {
	MockHookRunner
	batches int
}

func (r *echoRunner) RunHooks(hooks []domain.Hook) []domain.ExecutionResult {
	r.ReceivedHooks = append(r.ReceivedHooks, hooks...)
	r.batches++
	results := make([]domain.ExecutionResult, len(hooks))
	for i, h := range hooks {
		results[i] = domain.ExecutionResult{Hook: h, Success: true}
	}
	return results
}

func (r *echoRunner) RunHooksParallel(hooks []domain.Hook) []domain.ExecutionResult {
	return r.RunHooks(hooks)
}

func TestQualityGateService_Run_ChangedScope(t *testing.T) {
	runner := &echoRunner{}
	files := &MockFileLister{Staged: []string{"cmd/main.go", "README.md"}}
	svc := NewQualityGateService(&MockToolManager{}, runner).WithFileLister(files)

	results, err := svc.Run(scopedConfig(), "pre-commit", RunOptions{})
	if err != nil {
		t.Fatal(err)
	}

	var commands []string
	for _, h := range runner.ReceivedHooks {
		commands = append(commands, h.Command)
	}
	wantCommands := []string{"gofmt -l cmd/main.go", "go vet ./cmd", "go test ./..."}
	if !reflect.DeepEqual(commands, wantCommands) {
		t.Errorf("commands run = %q, want %q", commands, wantCommands)
	}

	// Results keep the config order, with the skipped hook in place.
	var names []string
	for _, r := range results {
		names = append(names, r.Hook.Name)
	}
	if want := []string{"Format", "Vet", "Python", "Tests"}; !reflect.DeepEqual(names, want) {
		t.Fatalf("result order = %q, want %q", names, want)
	}
	if !results[2].Skipped || !results[2].Success || results[2].SkipReason == "" {
		t.Errorf("expected Python to be skipped as passed, got %+v", results[2])
	}
	if len(runner.LoggedResults) != 1 || runner.LoggedResults[0].Hook.Name != "Python" {
		t.Errorf("expected the skipped hook to be logged, got %+v", runner.LoggedResults)
	}
	if !reflect.DeepEqual(files.Calls, []string{"staged"}) {
		t.Errorf("expected one staged-files lookup, got %q", files.Calls)
	}
}

func TestQualityGateService_Run_ParallelKeepsOrderWithSkipped(t *testing.T) {
	runner := &echoRunner{}
	files := &MockFileLister{Staged: []string{"a.go"}}
	svc := NewQualityGateService(&MockToolManager{}, runner).WithFileLister(files)

	results, err := svc.Run(scopedConfig(), "pre-commit", RunOptions{Parallel: true})
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, r := range results {
		names = append(names, r.Hook.Name)
	}
	if want := []string{"Format", "Vet", "Python", "Tests"}; !reflect.DeepEqual(names, want) {
		t.Fatalf("result order = %q, want %q", names, want)
	}
	if runner.batches != 1 {
		t.Errorf("parallel runs must reach the runner in one batch, got %d", runner.batches)
	}
}

func TestQualityGateService_Run_AllScopeUsesTrackedFiles(t *testing.T) {
	runner := &echoRunner{}
	files := &MockFileLister{Tracked: []string{"a.go", "b.py"}}
	svc := NewQualityGateService(&MockToolManager{}, runner).WithFileLister(files)

	results, err := svc.Run(scopedConfig(), "pre-commit", RunOptions{Scope: domain.ScopeAll, Parallel: true})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(files.Calls, []string{"tracked"}) {
		t.Errorf("expected tracked files to be listed, got %q", files.Calls)
	}
	for _, r := range results {
		if r.Skipped {
			t.Errorf("no hook should be skipped in a full run, got %s skipped", r.Hook.Name)
		}
	}
}

func TestQualityGateService_Run_PrePushUsesPushFiles(t *testing.T) {
	runner := &echoRunner{}
	files := &MockFileLister{Push: []string{"x.go"}}
	svc := NewQualityGateService(&MockToolManager{}, runner).WithFileLister(files)

	if _, err := svc.Run(scopedConfig(), "pre-push", RunOptions{}); err != nil {
		t.Fatal(err)
	}
	if got := runner.ReceivedHooks[0].Command; got != "lint x.go" {
		t.Errorf("command = %q, want %q", got, "lint x.go")
	}
}

func TestQualityGateService_Run_NothingChangedSkipsScopedHooks(t *testing.T) {
	runner := &echoRunner{}
	svc := NewQualityGateService(&MockToolManager{}, runner).WithFileLister(&MockFileLister{})

	results, err := svc.Run(scopedConfig(), "pre-commit", RunOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(runner.ReceivedHooks) != 1 || runner.ReceivedHooks[0].Name != "Tests" {
		t.Errorf("only the unscoped hook should run, got %+v", runner.ReceivedHooks)
	}
	if len(results) != 4 {
		t.Errorf("expected 4 results, got %d", len(results))
	}
}

func TestQualityGateService_Run_UnscopedConfigNeverListsFiles(t *testing.T) {
	files := &MockFileLister{Err: errors.New("not a git repository")}
	runner := &echoRunner{}
	svc := NewQualityGateService(&MockToolManager{}, runner).WithFileLister(files)

	if _, err := svc.Run(sampleConfig(), "pre-commit", RunOptions{}); err != nil {
		t.Fatalf("a config without placeholders or globs must not depend on git: %v", err)
	}
	if len(files.Calls) != 0 {
		t.Errorf("expected no file lookups, got %q", files.Calls)
	}
}

func TestQualityGateService_Run_FileListingFails(t *testing.T) {
	files := &MockFileLister{Err: errors.New("boom")}
	svc := NewQualityGateService(&MockToolManager{}, &echoRunner{}).WithFileLister(files)

	_, err := svc.Run(scopedConfig(), "pre-commit", RunOptions{})
	if err == nil || !strings.Contains(err.Error(), "boom") {
		t.Fatalf("expected the listing error, got %v", err)
	}
}

func TestQualityGateService_Run_NoFileLister(t *testing.T) {
	svc := NewQualityGateService(&MockToolManager{}, &echoRunner{})

	if _, err := svc.Run(scopedConfig(), "pre-commit", RunOptions{}); err == nil {
		t.Fatal("expected an error when scoped hooks run without a file lister")
	}
}

func TestQualityGateService_Fix_Scoped(t *testing.T) {
	runner := &MockHookRunner{}
	files := &MockFileLister{Staged: []string{"a.go"}}
	svc := NewQualityGateService(&MockToolManager{}, runner).WithFileLister(files)

	if err := svc.Fix(scopedConfig(), "pre-commit", ""); err != nil {
		t.Fatal(err)
	}
	if len(runner.FixedHooks) != 1 {
		t.Fatalf("expected only the Go fix to run, got %+v", runner.FixedHooks)
	}
	if got := runner.FixedHooks[0].FixCommand; got != "gofmt -w a.go" {
		t.Errorf("fix command = %q, want %q", got, "gofmt -w a.go")
	}
}

func TestQualityGateService_Fix_FileListingFails(t *testing.T) {
	files := &MockFileLister{Err: errors.New("boom")}
	svc := NewQualityGateService(&MockToolManager{}, &MockHookRunner{}).WithFileLister(files)

	if err := svc.Fix(scopedConfig(), "pre-commit", ""); err == nil {
		t.Fatal("expected the listing error")
	}
}

func TestQualityGateService_DescribeScope(t *testing.T) {
	files := &MockFileLister{Staged: []string{"a.go"}, Tracked: []string{"a.go", "b.go"}}
	svc := NewQualityGateService(&MockToolManager{}, &MockHookRunner{}).WithFileLister(files)

	if got, ok := svc.DescribeScope(scopedConfig(), "pre-commit", domain.ScopeChanged); !ok || !strings.HasPrefix(got, "1 changed file ") {
		t.Errorf("changed summary = %q, %v", got, ok)
	}
	if got, ok := svc.DescribeScope(scopedConfig(), "pre-commit", domain.ScopeAll); !ok || got != "full (2 tracked files)" {
		t.Errorf("full summary = %q, %v", got, ok)
	}
	if _, ok := svc.DescribeScope(sampleConfig(), "pre-commit", domain.ScopeChanged); ok {
		t.Error("an unscoped config has no scope to describe")
	}

	failing := NewQualityGateService(&MockToolManager{}, &MockHookRunner{}).WithFileLister(&MockFileLister{Err: errors.New("boom")})
	if _, ok := failing.DescribeScope(scopedConfig(), "pre-commit", domain.ScopeChanged); ok {
		t.Error("a listing error should hide the summary")
	}
}
