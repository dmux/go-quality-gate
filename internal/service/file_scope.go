package service

import (
	"fmt"
	"path"
	"regexp"
	"sort"
	"strings"

	"github.com/dmux/go-quality-gate/internal/config"
	"github.com/dmux/go-quality-gate/internal/domain"
	"github.com/dmux/go-quality-gate/internal/repository"
)

const (
	// FilesPlaceholder is replaced by the shell-quoted files a hook analyzes.
	FilesPlaceholder = "{files}"
	// DirsPlaceholder is replaced by the shell-quoted parent directories of
	// those files, as ./path, for package-based tools such as `go vet`.
	DirsPlaceholder = "{dirs}"

	// ScopeEnvVar overrides the configured scope for one run, e.g.
	// QG_SCOPE=all git commit.
	ScopeEnvVar = "QG_SCOPE"
)

// usesFiles reports whether a hook depends on the files of the run: through
// a placeholder in the given command or a glob filter. Hooks that don't keep
// running unchanged, whatever the scope.
func usesFiles(hook domain.Hook, command string) bool {
	return len(hook.Glob) > 0 || strings.Contains(command, FilesPlaceholder) || strings.Contains(command, DirsPlaceholder)
}

// resolveFiles returns the files a run analyzes: every tracked file for
// ScopeAll, otherwise what the push (pre-push) or the commit (any other hook
// type) changes.
func resolveFiles(lister repository.FileLister, hookType string, scope domain.Scope) ([]string, error) {
	if lister == nil {
		return nil, fmt.Errorf("file scoping is not available")
	}
	switch {
	case scope == domain.ScopeAll:
		return lister.TrackedFiles()
	case hookType == "pre-push":
		return lister.PushFiles()
	default:
		return lister.StagedFiles()
	}
}

// filterGlob returns the files matching at least one pattern, or every file
// when there are no patterns.
func filterGlob(files, patterns []string) []string {
	if len(patterns) == 0 {
		return files
	}
	var matched []string
	for _, f := range files {
		for _, p := range patterns {
			if matchGlob(p, f) {
				matched = append(matched, f)
				break
			}
		}
	}
	return matched
}

// matchGlob matches a root-relative path against a pattern. A pattern without
// a slash matches the file name in any directory ("*.go" matches
// "cmd/main.go"); otherwise it matches the whole path, where a "**" segment
// matches any number of directories ("web/**/*.ts").
func matchGlob(pattern, file string) bool {
	if !strings.Contains(pattern, "/") {
		ok, _ := path.Match(pattern, path.Base(file))
		return ok
	}
	return matchSegments(strings.Split(pattern, "/"), strings.Split(file, "/"))
}

func matchSegments(pattern, parts []string) bool {
	for len(pattern) > 0 {
		if pattern[0] == "**" {
			for i := 0; i <= len(parts); i++ {
				if matchSegments(pattern[1:], parts[i:]) {
					return true
				}
			}
			return false
		}
		if len(parts) == 0 {
			return false
		}
		if ok, _ := path.Match(pattern[0], parts[0]); !ok {
			return false
		}
		pattern, parts = pattern[1:], parts[1:]
	}
	return len(parts) == 0
}

// expandCommand replaces the {files} and {dirs} placeholders with the given
// files and their unique parent directories, quoted for a POSIX shell.
func expandCommand(command string, files []string) string {
	if strings.Contains(command, FilesPlaceholder) {
		quoted := make([]string, len(files))
		for i, f := range files {
			quoted[i] = shellQuote(argPath(f))
		}
		command = strings.ReplaceAll(command, FilesPlaceholder, strings.Join(quoted, " "))
	}
	if strings.Contains(command, DirsPlaceholder) {
		command = strings.ReplaceAll(command, DirsPlaceholder, strings.Join(parentDirs(files), " "))
	}
	return command
}

// parentDirs returns the sorted, unique, shell-quoted parent directories of
// files as ./path (the root is ".").
func parentDirs(files []string) []string {
	seen := map[string]bool{}
	var dirs []string
	for _, f := range files {
		dir := path.Dir(f)
		if dir != "." {
			dir = "./" + dir
		}
		if !seen[dir] {
			seen[dir] = true
			dirs = append(dirs, dir)
		}
	}
	sort.Strings(dirs)
	for i, d := range dirs {
		dirs[i] = shellQuote(d)
	}
	return dirs
}

// argPath keeps a path starting with "-" from being read as a flag.
func argPath(file string) string {
	if strings.HasPrefix(file, "-") {
		return "./" + file
	}
	return file
}

var shellSafe = regexp.MustCompile(`^[A-Za-z0-9_./@%+=:,-]+$`)

// shellQuote quotes s for a POSIX shell, leaving plain paths untouched so
// expanded commands stay readable.
func shellQuote(s string) string {
	if shellSafe.MatchString(s) {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// scopedHook is a hook planned for a run: its command expanded for the run's
// files, or skipped when no file matched it.
type scopedHook struct {
	hook       domain.Hook
	skipped    bool
	skipReason string
}

// planHooks expands each hook's command (fix command when fix is true) for
// the files of the run. Files are only resolved when a hook needs them, so
// configs without placeholders or globs never call git.
func (s *QualityGateService) planHooks(hooks []domain.Hook, hookType string, scope domain.Scope, fix bool) ([]scopedHook, error) {
	var files []string
	resolved := false
	planned := make([]scopedHook, len(hooks))

	for i, hook := range hooks {
		command := hook.Command
		if fix {
			command = hook.FixCommand
		}
		planned[i] = scopedHook{hook: hook}
		if !usesFiles(hook, command) {
			continue
		}

		if !resolved {
			var err error
			if files, err = resolveFiles(s.files, hookType, scope); err != nil {
				return nil, fmt.Errorf("failed to list the files to check: %w", err)
			}
			resolved = true
		}

		matched := filterGlob(files, hook.Glob)
		if len(matched) == 0 {
			planned[i].skipped = true
			planned[i].skipReason = "no matching changed files"
			if scope == domain.ScopeAll {
				planned[i].skipReason = "no matching files"
			}
			continue
		}

		if fix {
			planned[i].hook.FixCommand = expandCommand(command, matched)
		} else {
			planned[i].hook.Command = expandCommand(command, matched)
		}
	}
	return planned, nil
}

// DescribeScope summarizes the files a run of hookType analyzes, e.g.
// "3 changed files". ok is false when no hook of the run depends on files,
// so there is nothing worth reporting.
func (s *QualityGateService) DescribeScope(cfg *config.Config, hookType string, scope domain.Scope) (summary string, ok bool) {
	needed := false
	for _, h := range s.getHooksToRun(cfg.Hooks, hookType) {
		needed = needed || usesFiles(h, h.Command)
	}
	if !needed {
		return "", false
	}
	files, err := resolveFiles(s.files, hookType, scope)
	if err != nil {
		return "", false
	}
	noun := "files"
	if len(files) == 1 {
		noun = "file"
	}
	if scope == domain.ScopeAll {
		return fmt.Sprintf("full (%d tracked %s)", len(files), noun), true
	}
	return fmt.Sprintf("%d changed %s (use --all for a full analysis)", len(files), noun), true
}
