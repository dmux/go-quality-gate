package config

type Config struct {
	Settings Settings `yaml:"settings,omitempty"`
	Tools    Tools    `yaml:"tools"`
	Hooks    Hooks    `yaml:"hooks"`
}

// Settings holds optional, repo-wide behavior toggles for quality-gate.
type Settings struct {
	// ToolsPolicy controls what happens when a configured tool is missing:
	// "install" (default) auto-installs it, "recommend" only prints the
	// install command and leaves installation to the user.
	ToolsPolicy string `yaml:"tools_policy,omitempty"`

	// Watermark turns on the commit watermark: passing pre-commit runs are
	// attested and the commit-msg hook appends a Quality-Gate trailer that
	// `quality-gate verify` checks in CI. Disabled unless set to true.
	Watermark bool `yaml:"watermark,omitempty"`

	// Scope selects which files hooks analyze by default: "changed"
	// (default) passes only the files changed by the commit or push to the
	// {files} and {dirs} placeholders, "all" passes every tracked file. The
	// --all flag and QG_SCOPE=all override it per run.
	Scope string `yaml:"scope,omitempty"`
}

// ScopeChanged and ScopeAll are the only valid values for Settings.Scope.
const (
	ScopeChanged = "changed"
	ScopeAll     = "all"
)

// DefaultScope returns the effective scope, defaulting to ScopeChanged when
// unset.
func (s Settings) DefaultScope() string {
	if s.Scope == "" {
		return ScopeChanged
	}
	return s.Scope
}

// ToolsPolicyInstall and ToolsPolicyRecommend are the only valid values for
// Settings.ToolsPolicy.
const (
	ToolsPolicyInstall   = "install"
	ToolsPolicyRecommend = "recommend"
)

// WatermarkEnabled reports whether commits are watermarked. It is false unless
// the config opts in with `settings.watermark: true`.
func (s Settings) WatermarkEnabled() bool {
	return s.Watermark
}

// InstallPolicy returns the effective tools policy, defaulting to
// ToolsPolicyInstall when unset so existing configs keep today's behavior.
func (s Settings) InstallPolicy() string {
	if s.ToolsPolicy == "" {
		return ToolsPolicyInstall
	}
	return s.ToolsPolicy
}

type Tools []Tool

type Tool struct {
	Name           string `yaml:"name"`
	CheckCommand   string `yaml:"check_command"`
	InstallCommand string `yaml:"install_command"`
}

type Hooks map[string]map[string][]Hook

const HookScopeStaged = "staged"

type Hook struct {
	Name       string `yaml:"name"`
	Command    string `yaml:"command"`
	FixCommand string `yaml:"fix_command,omitempty"`
	// Glob limits the hook to changed files matching one of the patterns
	// (e.g. "*.go"); the hook is skipped when none match.
	Glob        []string    `yaml:"glob,omitempty"`
	// Scope overrides settings.scope for this hook. "staged" always checks the commit index.
	Scope       string      `yaml:"scope,omitempty"`
	OutputRules OutputRules `yaml:"output_rules,omitempty"`
}

type OutputRules struct {
	ShowOn           string `yaml:"show_on,omitempty"`
	OnFailureMessage string `yaml:"on_failure_message,omitempty"`
}
