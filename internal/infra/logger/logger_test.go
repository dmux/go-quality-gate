package logger

import (
	"bytes"
	"io"
	"os"
	"testing"
)

// captureStd redirects os.Stdout and os.Stderr for the duration of fn,
// returning what was written to each.
func captureStd(t *testing.T, fn func()) (stdout, stderr string) {
	t.Helper()

	origOut, origErr := os.Stdout, os.Stderr
	outR, outW, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	errR, errW, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout, os.Stderr = outW, errW
	t.Cleanup(func() { os.Stdout, os.Stderr = origOut, origErr })

	fn()

	outW.Close()
	errW.Close()

	var outBuf, errBuf bytes.Buffer
	io.Copy(&outBuf, outR)
	io.Copy(&errBuf, errR)

	return outBuf.String(), errBuf.String()
}

// mockSpinnerManager records calls instead of driving a real terminal
// spinner, so Print/Println tests don't need one running.
type mockSpinnerManager struct {
	started, stopped bool
	lastMessage      string
}

func (m *mockSpinnerManager) Start(message string)         { m.started = true; m.lastMessage = message }
func (m *mockSpinnerManager) Stop()                        { m.stopped = true }
func (m *mockSpinnerManager) UpdateMessage(message string) { m.lastMessage = message }

func TestConsoleLogger_Print_TextMode(t *testing.T) {
	l := &ConsoleLogger{jsonMode: false}

	stdout, stderr := captureStd(t, func() {
		l.Print("hello %s", "world")
	})

	if stdout != "hello world" {
		t.Errorf("expected %q on stdout, got %q", "hello world", stdout)
	}
	if stderr != "" {
		t.Errorf("expected nothing on stderr, got %q", stderr)
	}
}

func TestConsoleLogger_Print_JSONMode(t *testing.T) {
	l := &ConsoleLogger{jsonMode: true}

	stdout, stderr := captureStd(t, func() {
		l.Print("hello %s", "world")
	})

	if stderr != "hello world" {
		t.Errorf("expected %q on stderr, got %q", "hello world", stderr)
	}
	if stdout != "" {
		t.Errorf("expected nothing on stdout, got %q", stdout)
	}
}

func TestConsoleLogger_Println_TextMode(t *testing.T) {
	l := &ConsoleLogger{jsonMode: false}

	stdout, stderr := captureStd(t, func() {
		l.Println("a line")
	})

	if stdout != "a line\n" {
		t.Errorf("expected %q on stdout, got %q", "a line\n", stdout)
	}
	if stderr != "" {
		t.Errorf("expected nothing on stderr, got %q", stderr)
	}
}

func TestConsoleLogger_Println_JSONMode(t *testing.T) {
	l := &ConsoleLogger{jsonMode: true}

	stdout, stderr := captureStd(t, func() {
		l.Println("a line")
	})

	if stderr != "a line\n" {
		t.Errorf("expected %q on stderr, got %q", "a line\n", stderr)
	}
	if stdout != "" {
		t.Errorf("expected nothing on stdout, got %q", stdout)
	}
}

func TestConsoleLogger_SpinnerDelegation(t *testing.T) {
	mock := &mockSpinnerManager{}
	l := &ConsoleLogger{spinnerManager: mock}

	l.StartSpinner("working")
	if !mock.started || mock.lastMessage != "working" {
		t.Errorf("expected StartSpinner to delegate, got %+v", mock)
	}

	l.UpdateSpinner("still working")
	if mock.lastMessage != "still working" {
		t.Errorf("expected UpdateSpinner to delegate, got %+v", mock)
	}

	l.StopSpinner()
	if !mock.stopped {
		t.Errorf("expected StopSpinner to delegate, got %+v", mock)
	}
}

func TestNewConsoleLogger(t *testing.T) {
	l := NewConsoleLogger(true)

	if l.jsonMode != true {
		t.Error("expected jsonMode to be set from the constructor argument")
	}
	if l.spinnerManager == nil {
		t.Error("expected a spinner manager to be constructed")
	}
}
