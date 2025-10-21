package service

// MockLogger is a mock implementation of the Logger interface.
type MockLogger struct {
	Messages []string
}

// Print implements the Logger interface.
func (m *MockLogger) Print(format string, args ...interface{}) {
	// For testing, we just ignore the output
}

// Println implements the Logger interface.
func (m *MockLogger) Println(msg string) {
	m.Messages = append(m.Messages, msg)
}

// StartSpinner implements the Logger interface.
func (m *MockLogger) StartSpinner(message string) {
	// For testing, we just ignore the output
}

// StopSpinner implements the Logger interface.
func (m *MockLogger) StopSpinner() {
	// For testing, we just ignore the output
}

// UpdateSpinner implements the Logger interface.
func (m *MockLogger) UpdateSpinner(message string) {
	// For testing, we just ignore the output
}