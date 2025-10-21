package config

type Config struct {
	Tools Tools `yaml:"tools"`
	Hooks Hooks `yaml:"hooks"`
}

type Tools []Tool

type Tool struct {
	Name           string `yaml:"name"`
	CheckCommand   string `yaml:"check_command"`
	InstallCommand string `yaml:"install_command"`
}

type Hooks map[string]map[string][]Hook

type Hook struct {
	Name          string       `yaml:"name"`
	Command       string       `yaml:"command"`
	FixCommand    string       `yaml:"fix_command,omitempty"`
	OutputRules   OutputRules  `yaml:"output_rules,omitempty"`
}

type OutputRules struct {
	ShowOn         string `yaml:"show_on,omitempty"`
	OnFailureMessage string `yaml:"on_failure_message,omitempty"`
}
