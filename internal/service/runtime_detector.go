package service

import (
	"fmt"
	"os/exec"
	"runtime"
)

// RuntimeCheck is the outcome of checking whether a language runtime is
// present on the current machine, together with an OS-appropriate hint for
// installing it when it is missing.
type RuntimeCheck struct {
	Language    Language `json:"language"`
	Binary      string   `json:"binary"`
	Installed   bool     `json:"installed"`
	InstallHint string   `json:"install_hint,omitempty"`
}

// runtimeInfo describes how to detect and, per OS, how to install a base
// language runtime.
type runtimeInfo struct {
	binary      string
	darwinHint  string
	windowsHint string
	linuxHints  []linuxPackageManagerHint
}

// linuxPackageManagerHint maps a package manager's own binary (used to
// detect which distro family we're on) to the command that installs the
// runtime through it.
type linuxPackageManagerHint struct {
	managerBinary string
	installHint   string
}

// baseLanguageRuntimes maps every base language we can detect to how it's
// checked and installed. Framework languages resolve to one of these via
// frameworkToBaseLanguage before lookup.
var baseLanguageRuntimes = map[Language]runtimeInfo{
	LanguageGo: {
		binary:      "go",
		darwinHint:  "brew install go",
		windowsHint: "winget install -e --id GoLang.Go",
		linuxHints: []linuxPackageManagerHint{
			{"apt-get", "sudo apt-get install -y golang-go"},
			{"dnf", "sudo dnf install -y golang"},
			{"yum", "sudo yum install -y golang"},
			{"apk", "sudo apk add go"},
			{"pacman", "sudo pacman -S go"},
		},
	},
	LanguagePython: {
		binary:      "python3",
		darwinHint:  "brew install python3",
		windowsHint: "winget install -e --id Python.Python.3",
		linuxHints: []linuxPackageManagerHint{
			{"apt-get", "sudo apt-get install -y python3"},
			{"dnf", "sudo dnf install -y python3"},
			{"yum", "sudo yum install -y python3"},
			{"apk", "sudo apk add python3"},
			{"pacman", "sudo pacman -S python"},
		},
	},
	LanguageNode: {
		binary:      "node",
		darwinHint:  "brew install node",
		windowsHint: "winget install -e --id OpenJS.NodeJS",
		linuxHints: []linuxPackageManagerHint{
			{"apt-get", "sudo apt-get install -y nodejs npm"},
			{"dnf", "sudo dnf install -y nodejs"},
			{"yum", "sudo yum install -y nodejs"},
			{"apk", "sudo apk add nodejs npm"},
			{"pacman", "sudo pacman -S nodejs npm"},
		},
	},
	LanguageRust: {
		binary:      "cargo",
		darwinHint:  "brew install rust",
		windowsHint: "winget install -e --id Rustlang.Rustup",
		linuxHints: []linuxPackageManagerHint{
			{"apt-get", "sudo apt-get install -y cargo"},
			{"dnf", "sudo dnf install -y cargo"},
			{"yum", "sudo yum install -y cargo"},
			{"apk", "sudo apk add cargo"},
			{"pacman", "sudo pacman -S rust"},
		},
	},
	LanguagePHP: {
		binary:      "php",
		darwinHint:  "brew install php",
		windowsHint: "winget install -e --id PHP.PHP",
		linuxHints: []linuxPackageManagerHint{
			{"apt-get", "sudo apt-get install -y php"},
			{"dnf", "sudo dnf install -y php"},
			{"yum", "sudo yum install -y php"},
			{"apk", "sudo apk add php"},
			{"pacman", "sudo pacman -S php"},
		},
	},
	LanguageJava: {
		binary:      "java",
		darwinHint:  "brew install openjdk",
		windowsHint: "winget install -e --id EclipseAdoptium.Temurin.17.JDK",
		linuxHints: []linuxPackageManagerHint{
			{"apt-get", "sudo apt-get install -y default-jdk"},
			{"dnf", "sudo dnf install -y java-17-openjdk"},
			{"yum", "sudo yum install -y java-17-openjdk"},
			{"apk", "sudo apk add openjdk17"},
			{"pacman", "sudo pacman -S jdk-openjdk"},
		},
	},
}

// frameworkToBaseLanguage maps framework/superset languages to the base
// runtime they depend on, so e.g. React doesn't get reported separately
// from Node.
var frameworkToBaseLanguage = map[Language]Language{
	LanguageTypeScript: LanguageNode,
	LanguageReact:      LanguageNode,
	LanguageVue:        LanguageNode,
	LanguageAngular:    LanguageNode,
	LanguageDjango:     LanguagePython,
	LanguageFastAPI:    LanguagePython,
	LanguageFlask:      LanguagePython,
	LanguageLaravel:    LanguagePHP,
}

// DetectRuntimes reports, for every base language implied by the detected
// project structure, whether its runtime is present on PATH and how to
// install it if not.
func DetectRuntimes(structure *ProjectStructure) []RuntimeCheck {
	baseLanguages := collectBaseLanguages(structure)

	checks := make([]RuntimeCheck, 0, len(baseLanguages))
	for _, lang := range baseLanguages {
		info, ok := baseLanguageRuntimes[lang]
		if !ok {
			continue
		}

		_, err := exec.LookPath(info.binary)
		checks = append(checks, RuntimeCheck{
			Language:    lang,
			Binary:      info.binary,
			Installed:   err == nil,
			InstallHint: installHintForOS(info, runtime.GOOS),
		})
	}

	return checks
}

// collectBaseLanguages resolves every detected language/framework to its
// base runtime language, deduplicated and in a stable order.
func collectBaseLanguages(structure *ProjectStructure) []Language {
	seen := make(map[Language]bool)
	var ordered []Language

	add := func(lang Language) {
		base := lang
		if mapped, ok := frameworkToBaseLanguage[lang]; ok {
			base = mapped
		}
		if _, ok := baseLanguageRuntimes[base]; !ok {
			return
		}
		if !seen[base] {
			seen[base] = true
			ordered = append(ordered, base)
		}
	}

	for _, lang := range structure.Languages {
		add(lang)
	}
	for _, framework := range structure.Frameworks {
		add(framework)
	}

	return ordered
}

// installHintForOS returns the install command for the given OS, or an
// honest explanation when no known command exists rather than guessing.
// goos is threaded as a parameter (rather than always reading runtime.GOOS
// directly) so tests can exercise every branch regardless of which OS they
// actually run on.
func installHintForOS(info runtimeInfo, goos string) string {
	switch goos {
	case "darwin":
		return info.darwinHint
	case "windows":
		return info.windowsHint
	case "linux":
		for _, hint := range info.linuxHints {
			if _, err := exec.LookPath(hint.managerBinary); err == nil {
				return hint.installHint
			}
		}
		return fmt.Sprintf("no supported package manager found (checked apt-get, dnf, yum, apk, pacman); install %s manually", info.binary)
	default:
		return fmt.Sprintf("no known install command for OS %q; install %s manually", goos, info.binary)
	}
}
