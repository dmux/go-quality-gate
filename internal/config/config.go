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
}

// ToolsPolicyInstall and ToolsPolicyRecommend are the only valid values for
// Settings.ToolsPolicy.
const (
	ToolsPolicyInstall   = "install"
	ToolsPolicyRecommend = "recommend"
)

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

type Hook struct {
	Name        string      `yaml:"name"`
	Command     string      `yaml:"command"`
	FixCommand  string      `yaml:"fix_command,omitempty"`
	OutputRules OutputRules `yaml:"output_rules,omitempty"`
}

type OutputRules struct {
	ShowOn           string `yaml:"show_on,omitempty"`
	OnFailureMessage string `yaml:"on_failure_message,omitempty"`
}
