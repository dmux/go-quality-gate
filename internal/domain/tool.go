package domain

// Tool represents a command-line tool that needs to be installed.

type Tool struct {
	Name           string
	CheckCommand   string
	InstallCommand string
}
