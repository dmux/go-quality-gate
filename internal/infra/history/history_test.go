package history

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestFindGitDir_Found(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, ".git"), 0755); err != nil {
		t.Fatal(err)
	}
	nested := filepath.Join(root, "a", "b", "c")
	if err := os.MkdirAll(nested, 0755); err != nil {
		t.Fatal(err)
	}

	t.Chdir(nested)

	got, err := FindGitDir()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want, _ := filepath.EvalSymlinks(filepath.Join(root, ".git"))
	gotResolved, _ := filepath.EvalSymlinks(got)
	if gotResolved != want {
		t.Errorf("expected %q, got %q", want, gotResolved)
	}
}

func TestFindGitDir_NotFound(t *testing.T) {
	// A fresh temp dir has no .git anywhere in its ancestry.
	t.Chdir(t.TempDir())

	_, err := FindGitDir()
	if err == nil {
		t.Fatal("expected an error when no .git directory exists")
	}
}

func TestFindGitDir_GetwdFails(t *testing.T) {
	orig := getwd
	getwd = func() (string, error) { return "", errors.New("boom") }
	t.Cleanup(func() { getwd = orig })

	if _, err := FindGitDir(); err == nil {
		t.Fatal("expected an error when os.Getwd fails")
	}
}

type failingWriter struct{}

func (failingWriter) Write(p []byte) (int, error) { return 0, errors.New("disk full") }
func (failingWriter) Close() error                { return nil }

func TestStore_Append_WriteFails(t *testing.T) {
	orig := openLogFileForAppend
	openLogFileForAppend = func(path string) (io.WriteCloser, error) { return failingWriter{}, nil }
	t.Cleanup(func() { openLogFileForAppend = orig })

	store := NewStore(t.TempDir())
	if err := store.Append(Record{Type: "run"}); err == nil {
		t.Fatal("expected an error when the write fails")
	}
}

func TestStore_Append_MkdirAllFails(t *testing.T) {
	dir := t.TempDir()
	// A regular file where Append needs to create a directory forces
	// MkdirAll to fail (ENOTDIR).
	blocker := filepath.Join(dir, "blocker")
	if err := os.WriteFile(blocker, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}

	store := NewStore(blocker)
	if err := store.Append(Record{Type: "run"}); err == nil {
		t.Fatal("expected an error when the log directory can't be created")
	}
}

func TestStore_Append_OpenFileFails(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)

	// Pre-create the log file's exact path as a directory instead of a
	// file, so OpenFile for writing fails (EISDIR).
	logPath := filepath.Join(dir, relativeLogPath)
	if err := os.MkdirAll(logPath, 0755); err != nil {
		t.Fatal(err)
	}

	if err := store.Append(Record{Type: "run"}); err == nil {
		t.Fatal("expected an error when the log path is a directory")
	}
}

func TestStore_Append_MarshalFails(t *testing.T) {
	store := NewStore(t.TempDir())

	// time.Time.MarshalJSON errors for years outside [0,9999], which is
	// otherwise unreachable for a Timestamp we always set from time.Now().
	badTime := time.Date(10000, 1, 1, 0, 0, 0, 0, time.UTC)

	if err := store.Append(Record{Timestamp: badTime, Type: "run"}); err == nil {
		t.Fatal("expected a marshal error for an out-of-range timestamp")
	}
}

func TestStore_Load_OpenFails(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root: permission checks don't apply")
	}

	dir := t.TempDir()
	store := NewStore(dir)
	logPath := filepath.Join(dir, relativeLogPath)
	if err := os.MkdirAll(filepath.Dir(logPath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(logPath, []byte("{}\n"), 0000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(logPath, 0644) })

	_, err := store.Load()
	if err == nil {
		t.Fatal("expected an error opening an unreadable log file")
	}
}

func TestStore_Load_SkipsMalformedAndBlankLines(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)
	logPath := filepath.Join(dir, relativeLogPath)
	if err := os.MkdirAll(filepath.Dir(logPath), 0755); err != nil {
		t.Fatal(err)
	}

	content := `{"type":"run","hook_type":"pre-commit","success":true}
not valid json

{"type":"fix","hook_type":"pre-commit","fixed_count":2}
`
	if err := os.WriteFile(logPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	records, err := store.Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("expected 2 valid records (malformed/blank lines skipped), got %d: %+v", len(records), records)
	}
	if records[0].Type != "run" || records[1].Type != "fix" {
		t.Errorf("unexpected record contents: %+v", records)
	}
}

func TestStore_Load_ScannerTooLong(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)
	logPath := filepath.Join(dir, relativeLogPath)
	if err := os.MkdirAll(filepath.Dir(logPath), 0755); err != nil {
		t.Fatal(err)
	}

	// A single line larger than the scanner's 4MB max buffer trips
	// bufio.ErrTooLong via scanner.Err().
	huge := strings.Repeat("a", 5*1024*1024)
	if err := os.WriteFile(logPath, []byte(huge+"\n"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := store.Load()
	if err == nil {
		t.Fatal("expected a scanner error for an oversized line")
	}
}

func TestStore_Load_MissingFileIsEmptyHistory(t *testing.T) {
	store := NewStore(t.TempDir())

	records, err := store.Load()
	if err != nil {
		t.Fatalf("expected no error for a missing log, got %v", err)
	}
	if records != nil {
		t.Errorf("expected nil records for a missing log, got %v", records)
	}
}
