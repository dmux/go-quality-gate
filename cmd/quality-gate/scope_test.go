package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/dmux/go-quality-gate/internal/config"
	"github.com/dmux/go-quality-gate/internal/domain"
)

const scopedQualityYML = `hooks:
  backend:
    pre-commit:
      - name: "Go files"
        command: "echo checked {files}"
        glob: ["*.go"]
      - name: "Always"
        command: "echo always"
`

// initScopeRepo runs a real git init in a temp dir and chdirs into it, so
// staged and tracked files can be listed.
func initScopeRepo(t *testing.T) func(args ...string) {
	t.Helper()
	t.Chdir(t.TempDir())
	gitCmd := func(args ...string) {
		t.Helper()
		if out, err := exec.Command("git", args...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	gitCmd("init", "-q")
	return gitCmd
}

func TestRun_Scope_ChangedByDefault(t *testing.T) {
	gitCmd := initScopeRepo(t)
	writeQualityYML(t, scopedQualityYML)
	if err := os.WriteFile("main.go", []byte("package main\n"), 0644); err != nil {
		t.Fatal(err)
	}
	gitCmd("add", "quality.yml")

	code, stdout, _ := runCLI("--output", "json", "pre-commit")
	if code != 0 {
		t.Fatalf("expected exit 0, got %d: %s", code, stdout)
	}
	var out struct {
		Scope   string `json:"scope"`
		Results []struct {
			Skipped bool   `json:"skipped"`
			Output  string `json:"output"`
		} `json:"results"`
	}
	if err := json.Unmarshal([]byte(stdout), &out); err != nil {
		t.Fatalf("invalid JSON %q: %v", stdout, err)
	}
	if out.Scope != "changed" || len(out.Results) != 2 || !out.Results[0].Skipped || out.Results[1].Skipped {
		t.Errorf("unexpected changed-scope result: %+v", out)
	}

	// Staging the Go file makes the hook run on it alone.
	gitCmd("add", "main.go")
	code, stdout, _ = runCLI("pre-commit")
	if code != 0 || !strings.Contains(stdout, "Scope: 2 changed files") {
		t.Errorf("expected the scope banner, got %d:\n%s", code, stdout)
	}
}

func TestRun_Scope_AllFlag(t *testing.T) {
	gitCmd := initScopeRepo(t)
	writeQualityYML(t, scopedQualityYML)
	if err := os.WriteFile("main.go", []byte("package main\n"), 0644); err != nil {
		t.Fatal(err)
	}
	gitCmd("add", ".")
	gitCmd("-c", "user.name=t", "-c", "user.email=t@example.com", "commit", "-q", "-m", "init")

	code, stdout, _ := runCLI("--all", "--output", "json", "pre-commit")
	if code != 0 {
		t.Fatalf("expected exit 0, got %d: %s", code, stdout)
	}
	if !strings.Contains(stdout, `"scope": "all"`) || !strings.Contains(stdout, "checked main.go") {
		t.Errorf("expected a full run over main.go, got:\n%s", stdout)
	}
}

func TestEffectiveScope(t *testing.T) {
	cfgAll := &config.Config{Settings: config.Settings{Scope: config.ScopeAll}}
	cfgDefault := &config.Config{}

	cases := []struct {
		name string
		cfg  *config.Config
		all  bool
		env  string
		want domain.Scope
	}{
		{"default", cfgDefault, false, "", domain.ScopeChanged},
		{"flag", cfgDefault, true, "", domain.ScopeAll},
		{"env all", cfgDefault, false, "all", domain.ScopeAll},
		{"env changed overrides settings", cfgAll, false, "changed", domain.ScopeChanged},
		{"settings", cfgAll, false, "", domain.ScopeAll},
		{"flag beats env", cfgDefault, true, "changed", domain.ScopeAll},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Setenv("QG_SCOPE", c.env)
			if got := effectiveScope(c.cfg, c.all); got != c.want {
				t.Errorf("effectiveScope = %q, want %q", got, c.want)
			}
		})
	}
}
