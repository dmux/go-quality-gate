package webui

import (
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/dmux/go-quality-gate/internal/infra/history"
)

func TestStatsHandler_EmptyHistory(t *testing.T) {
	store := history.NewStore(t.TempDir())

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/stats", nil)
	statsHandler(store)(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected application/json content type, got %q", ct)
	}

	var resp apiResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON response: %v", err)
	}
	if resp.Stats.TotalRuns != 0 {
		t.Errorf("expected 0 total runs for empty history, got %d", resp.Stats.TotalRuns)
	}
	if resp.RecentRuns == nil {
		t.Error("expected recent_runs to be an empty array, not null")
	}
	if len(resp.Daily) != dailySummaryWindowDays {
		t.Errorf("expected %d daily entries, got %d", dailySummaryWindowDays, len(resp.Daily))
	}
}

func TestStatsHandler_WithHistory(t *testing.T) {
	dir := t.TempDir()
	store := history.NewStore(dir)

	if err := store.Append(history.Record{
		Timestamp: time.Now(), Type: "run", HookType: "pre-commit", Success: true, DurationMs: 42,
		Hooks: []history.HookOutcome{{Name: "Gitleaks", Success: true, DurationMs: 42}},
	}); err != nil {
		t.Fatal(err)
	}
	if err := store.Append(history.Record{
		Timestamp: time.Now(), Type: "fix", HookType: "pre-commit", FixedCount: 1,
	}); err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/stats", nil)
	statsHandler(store)(rr, req)

	var resp apiResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON response: %v", err)
	}
	if resp.Stats.TotalRuns != 1 {
		t.Errorf("expected 1 total run, got %d", resp.Stats.TotalRuns)
	}
	if resp.Stats.TotalAutoFixes != 1 {
		t.Errorf("expected 1 total auto-fix, got %d", resp.Stats.TotalAutoFixes)
	}
	if len(resp.RecentRuns) != 1 || resp.RecentRuns[0].HookType != "pre-commit" {
		t.Errorf("expected 1 recent run for pre-commit, got %+v", resp.RecentRuns)
	}
	if len(resp.HookStats) != 1 || resp.HookStats[0].Name != "Gitleaks" {
		t.Errorf("expected 1 hook stat for Gitleaks, got %+v", resp.HookStats)
	}
	if len(resp.Achievements) == 0 {
		t.Error("expected a non-empty achievements list")
	}
}

func TestStatsHandler_LoadError(t *testing.T) {
	// Reuse the same failure trick as the history package: an unreadable
	// log file (chmod 0000) makes Load() return an error.
	dir := t.TempDir()
	store := history.NewStore(dir)
	if err := store.Append(history.Record{Type: "run"}); err != nil {
		t.Fatal(err)
	}

	// Make the log unreadable. Skipped when running as root, where
	// permission bits don't block reads.
	logPath := dir + "/quality-gate/history.jsonl"
	if err := chmodOrSkip(t, logPath); err != nil {
		t.Skip(err)
	}

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/stats", nil)
	statsHandler(store)(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 when history fails to load, got %d", rr.Code)
	}
}

type failingResponseWriter struct {
	header http.Header
}

func (f *failingResponseWriter) Header() http.Header {
	if f.header == nil {
		f.header = make(http.Header)
	}
	return f.header
}
func (f *failingResponseWriter) Write(p []byte) (int, error) { return 0, errors.New("write failed") }
func (f *failingResponseWriter) WriteHeader(statusCode int)  {}

func TestStatsHandler_EncodeError(t *testing.T) {
	store := history.NewStore(t.TempDir())
	w := &failingResponseWriter{}
	req := httptest.NewRequest(http.MethodGet, "/api/stats", nil)

	// Must not panic even though every Write (including http.Error's own
	// attempt to report the encode failure) fails.
	statsHandler(store)(w, req)
}

func TestNewMux_ServesAPIAndStaticAssets(t *testing.T) {
	store := history.NewStore(t.TempDir())
	mux := newMux(store)

	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/stats", nil))
	if rr.Code != http.StatusOK {
		t.Errorf("expected /api/stats to return 200, got %d", rr.Code)
	}

	rr2 := httptest.NewRecorder()
	mux.ServeHTTP(rr2, httptest.NewRequest(http.MethodGet, "/", nil))
	if rr2.Code != http.StatusOK {
		t.Errorf("expected / to serve the embedded dashboard with 200, got %d", rr2.Code)
	}
	if !strings.Contains(rr2.Body.String(), "<html") {
		t.Errorf("expected the embedded index.html to be served, got: %s", rr2.Body.String()[:min(200, rr2.Body.Len())])
	}
}

func TestServeFn_HandlesRequestsUntilListenerCloses(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}

	store := history.NewStore(t.TempDir())
	done := make(chan error, 1)
	go func() { done <- serve(listener, store) }()

	resp, err := http.Get("http://" + listener.Addr().String() + "/api/stats")
	if err != nil {
		t.Fatalf("request to live server failed: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	if err := listener.Close(); err != nil {
		t.Fatal(err)
	}

	select {
	case err := <-done:
		if err == nil {
			t.Error("expected serve to return an error once its listener is closed")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("serve did not return after its listener was closed")
	}
}

func TestServe_BindFailure(t *testing.T) {
	// Occupy a port first so Serve's own net.Listen fails.
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()

	port := listener.Addr().(*net.TCPAddr).Port

	origStart := startCmd
	startCmd = func(cmd *exec.Cmd) error { return nil } // don't actually spawn a browser
	t.Cleanup(func() { startCmd = origStart })

	if err := Serve(t.TempDir(), port); err == nil {
		t.Fatal("expected an error when the port is already in use")
	}
}

func TestServe_SuccessPathStartsServing(t *testing.T) {
	// Serve's goroutine (via serveOn -> go openBrowser) reads startCmd
	// concurrently with this test. Restoring startCmd via t.Cleanup as soon
	// as the test function returns would race with that read unless we
	// first confirm the one-and-only call into startCmd has already
	// happened — the channel receive below is that synchronization point.
	browserOpened := make(chan struct{}, 1)
	origStart := startCmd
	startCmd = func(cmd *exec.Cmd) error {
		select {
		case browserOpened <- struct{}{}:
		default:
		}
		return nil
	}
	t.Cleanup(func() { startCmd = origStart })

	errCh := make(chan error, 1)
	// Port 0: the OS picks a free port, so this can't collide with anything
	// else. The listener/goroutine are intentionally never torn down — Go's
	// net.Listen has no external handle to close here, and the process
	// exiting at the end of the test binary reclaims it.
	go func() { errCh <- Serve(t.TempDir(), 0) }()

	select {
	case err := <-errCh:
		t.Fatalf("Serve returned early: %v", err)
	case <-browserOpened:
		// startCmd has been called (and returned): bind + serveOn's setup
		// succeeded, and no further reads of startCmd will occur.
	case <-time.After(2 * time.Second):
		t.Fatal("Serve did not attempt to open the browser in time")
	}
}

func TestStartCmd_Default(t *testing.T) {
	// Exercises the real (non-overridden) startCmd with a harmless command,
	// instead of an actual browser opener, so this doesn't pop open a
	// browser during `go test`.
	if err := startCmd(exec.Command("true")); err != nil {
		t.Errorf("expected the real startCmd to start a trivial command, got %v", err)
	}
}

func TestRecentRuns_LimitAndOrder(t *testing.T) {
	records := []history.Record{
		{Type: "run", HookType: "pre-commit", Success: true},
		{Type: "fix", HookType: "pre-commit", FixedCount: 1}, // must be skipped
		{Type: "run", HookType: "pre-commit", Success: false},
		{Type: "run", HookType: "pre-push", Success: true},
	}

	runs := recentRuns(records, 2)

	if len(runs) != 2 {
		t.Fatalf("expected the limit to cap results at 2, got %d", len(runs))
	}
	// Most recent first: index 3 (pre-push) then index 2 (pre-commit, failed).
	if runs[0].HookType != "pre-push" || runs[1].Success != false {
		t.Errorf("expected newest-first order, got %+v", runs)
	}
}

func TestRecentRuns_EmptyIsNotNil(t *testing.T) {
	runs := recentRuns(nil, 10)
	if runs == nil {
		t.Error("expected an empty slice, not nil, so it serializes as [] not null")
	}
}

func TestBrowserCommand(t *testing.T) {
	cases := map[string]string{
		"darwin":  "open",
		"windows": "cmd",
		"linux":   "xdg-open",
		"freebsd": "xdg-open",
	}
	for goos, wantProgram := range cases {
		cmd := browserCommand("http://127.0.0.1:4173", goos)
		if !strings.HasSuffix(cmd.Path, wantProgram) && !strings.Contains(cmd.Path, wantProgram) {
			t.Errorf("goos=%s: expected command program %q, got %q", goos, wantProgram, cmd.Path)
		}
	}
}

func TestOpenBrowser_InvokesStartCmd(t *testing.T) {
	orig := startCmd
	var gotURL string
	startCmd = func(cmd *exec.Cmd) error {
		gotURL = cmd.Args[len(cmd.Args)-1]
		return errors.New("simulated failure, intentionally ignored by openBrowser")
	}
	t.Cleanup(func() { startCmd = orig })

	openBrowser("http://127.0.0.1:4173")

	if gotURL != "http://127.0.0.1:4173" {
		t.Errorf("expected the browser command to target the dashboard URL, got %q", gotURL)
	}
}

// chmodOrSkip makes path unreadable, returning an error the caller should
// treat as "skip this test" (e.g. running as root, where permission bits
// don't block reads, so the intended failure never happens).
func chmodOrSkip(t *testing.T, path string) error {
	t.Helper()
	if err := os.Chmod(path, 0); err != nil {
		return err
	}
	t.Cleanup(func() { os.Chmod(path, 0644) })
	if _, err := os.ReadFile(path); err == nil {
		return errors.New("file is still readable after chmod 0 — likely running as root")
	}
	return nil
}
