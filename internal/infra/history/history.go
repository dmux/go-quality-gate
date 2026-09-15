// Package history persists a local, append-only log of quality-gate
// executions (JSONL, one JSON object per line) used to power the "stats"
// command's streaks, achievements, and time-saved estimates.
package history

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

const relativeLogPath = "quality-gate/history.jsonl"

// getwd is os.Getwd, swappable in tests to exercise FindGitDir's error path
// (os.Getwd failing is otherwise not reliably reproducible across platforms —
// e.g. macOS still resolves a deleted cwd).
var getwd = os.Getwd

// FindGitDir walks up from the current working directory to locate the
// nearest .git directory, mirroring the lookup used elsewhere for the
// tools-hash state file.
func FindGitDir() (string, error) {
	path, err := getwd()
	if err != nil {
		return "", err
	}

	for {
		gitDir := filepath.Join(path, ".git")
		if info, err := os.Stat(gitDir); err == nil && info.IsDir() {
			return gitDir, nil
		}

		parent := filepath.Dir(path)
		if parent == path {
			return "", fmt.Errorf(".git directory not found")
		}
		path = parent
	}
}

// HookOutcome is the per-hook result recorded for a "run" entry.
type HookOutcome struct {
	Name       string `json:"name"`
	Success    bool   `json:"success"`
	DurationMs int64  `json:"duration_ms"`
}

// Record is a single JSONL entry in the history log.
type Record struct {
	Timestamp time.Time `json:"timestamp"`
	// Type is "run" (a pre-commit/pre-push execution) or "fix" (a --fix invocation).
	Type     string `json:"type"`
	HookType string `json:"hook_type"`

	// Success and DurationMs apply to "run" records (overall outcome).
	Success    bool          `json:"success"`
	DurationMs int64         `json:"duration_ms"`
	Hooks      []HookOutcome `json:"hooks,omitempty"`

	// FixedCount applies to "fix" records: how many hooks with a
	// fix_command were fixed in that invocation.
	FixedCount int `json:"fixed_count,omitempty"`
}

// Store reads and appends to the JSONL history log.
type Store struct {
	path string
}

// NewStore creates a Store rooted at the given .git directory, e.g. the
// value returned by resolving the repository's .git directory.
func NewStore(gitDir string) *Store {
	return &Store{path: filepath.Join(gitDir, relativeLogPath)}
}

// openLogFileForAppend is swappable in tests to exercise the write-failure
// path in Append, which a real OS-level write() failure isn't practical to
// reproduce portably. *os.File satisfies io.WriteCloser.
var openLogFileForAppend = func(path string) (io.WriteCloser, error) {
	return os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
}

// Append writes one record to the end of the log, creating the log file
// and its parent directory if needed.
func (s *Store) Append(rec Record) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0755); err != nil {
		return fmt.Errorf("failed to create history directory: %w", err)
	}

	f, err := openLogFileForAppend(s.path)
	if err != nil {
		return fmt.Errorf("failed to open history log: %w", err)
	}
	defer f.Close()

	data, err := json.Marshal(rec)
	if err != nil {
		return fmt.Errorf("failed to marshal history record: %w", err)
	}

	if _, err := f.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("failed to write history record: %w", err)
	}

	return nil
}

// Load reads every record from the log. A missing log file is treated as an
// empty history rather than an error. Malformed lines (e.g. from an
// interrupted write) are skipped rather than failing the whole read, since
// this is a best-effort local log, not a source of truth.
func (s *Store) Load() ([]Record, error) {
	f, err := os.Open(s.path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to open history log: %w", err)
	}
	defer f.Close()

	var records []Record
	scanner := bufio.NewScanner(f)
	// History lines (results/output snippets) can exceed bufio's 64KB
	// default token size; allow generously large lines.
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var rec Record
		if err := json.Unmarshal(line, &rec); err != nil {
			continue
		}
		records = append(records, rec)
	}
	if err := scanner.Err(); err != nil {
		return records, fmt.Errorf("failed to read history log: %w", err)
	}

	return records, nil
}
