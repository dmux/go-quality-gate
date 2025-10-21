package spinner

import (
	"time"

	"github.com/briandowns/spinner"
)

// SpinnerManager manages spinner display
type SpinnerManager interface {
	Start(message string)
	Stop()
	UpdateMessage(message string)
}

// ConsoleSpinnerManager manages console spinners
type ConsoleSpinnerManager struct {
	spinner  *spinner.Spinner
	jsonMode bool
	isActive bool
}

// NewConsoleSpinnerManager creates a new spinner manager
func NewConsoleSpinnerManager(jsonMode bool) *ConsoleSpinnerManager {
	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	s.Color("cyan")

	return &ConsoleSpinnerManager{
		spinner:  s,
		jsonMode: jsonMode,
		isActive: false,
	}
}

// Start starts the spinner with a message
func (s *ConsoleSpinnerManager) Start(message string) {
	if s.jsonMode {
		return // Don't show spinners in JSON mode
	}

	s.spinner.Suffix = " " + message
	s.spinner.Start()
	s.isActive = true
}

// Stop stops the spinner
func (s *ConsoleSpinnerManager) Stop() {
	if s.isActive {
		s.spinner.Stop()
		s.isActive = false
	}
}

// UpdateMessage updates the spinner message without restarting
func (s *ConsoleSpinnerManager) UpdateMessage(message string) {
	if s.jsonMode || !s.isActive {
		return
	}

	s.spinner.Suffix = " " + message
}
