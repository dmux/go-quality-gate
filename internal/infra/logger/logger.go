package logger

import (
	"fmt"
	"io"
	"os"

	"github.com/dmux/go-quality-gate/internal/infra/spinner"
)

// Logger interface for controlling output destination
type Logger interface {
	Print(format string, args ...interface{})
	Println(msg string)
	StartSpinner(message string)
	StopSpinner()
	UpdateSpinner(message string)
}

// ConsoleLogger logs to stdout or stderr depending on JSON mode
type ConsoleLogger struct {
	jsonMode       bool
	spinnerManager spinner.SpinnerManager
}

// NewConsoleLogger creates a new console logger
func NewConsoleLogger(jsonMode bool) *ConsoleLogger {
	return &ConsoleLogger{
		jsonMode:       jsonMode,
		spinnerManager: spinner.NewConsoleSpinnerManager(jsonMode),
	}
}

// Print prints formatted output to the appropriate stream
func (l *ConsoleLogger) Print(format string, args ...interface{}) {
	if l.jsonMode {
		fmt.Fprintf(os.Stderr, format, args...)
	} else {
		fmt.Printf(format, args...)
	}
}

// Println prints a line to the appropriate stream
func (l *ConsoleLogger) Println(msg string) {
	var output io.Writer = os.Stdout
	if l.jsonMode {
		output = os.Stderr
	}
	fmt.Fprintln(output, msg)
}

// StartSpinner starts a spinner with the given message
func (l *ConsoleLogger) StartSpinner(message string) {
	l.spinnerManager.Start(message)
}

// StopSpinner stops the current spinner
func (l *ConsoleLogger) StopSpinner() {
	l.spinnerManager.Stop()
}

// UpdateSpinner updates the spinner message
func (l *ConsoleLogger) UpdateSpinner(message string) {
	l.spinnerManager.UpdateMessage(message)
}