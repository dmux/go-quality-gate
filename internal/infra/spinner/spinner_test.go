package spinner

import "testing"

func TestConsoleSpinnerManager_Start_JSONModeIsNoop(t *testing.T) {
	m := NewConsoleSpinnerManager(true)

	m.Start("working...")

	if m.isActive {
		t.Error("expected Start to be a no-op in JSON mode")
	}
}

func TestConsoleSpinnerManager_StartStop(t *testing.T) {
	m := NewConsoleSpinnerManager(false)

	m.Start("working...")
	if !m.isActive {
		t.Fatal("expected isActive=true after Start")
	}
	if m.spinner.Suffix != " working..." {
		t.Errorf("expected spinner suffix to be set, got %q", m.spinner.Suffix)
	}

	m.Stop()
	if m.isActive {
		t.Error("expected isActive=false after Stop")
	}
}

func TestConsoleSpinnerManager_Stop_WhenNotActiveIsNoop(t *testing.T) {
	m := NewConsoleSpinnerManager(false)

	// Must not panic even though Start was never called.
	m.Stop()

	if m.isActive {
		t.Error("expected isActive to remain false")
	}
}

func TestConsoleSpinnerManager_UpdateMessage_JSONModeIsNoop(t *testing.T) {
	m := NewConsoleSpinnerManager(true)
	m.isActive = true // simulate an active spinner to isolate the jsonMode check

	m.UpdateMessage("new message")

	if m.spinner.Suffix != "" {
		t.Errorf("expected UpdateMessage to be a no-op in JSON mode, got suffix %q", m.spinner.Suffix)
	}
}

func TestConsoleSpinnerManager_UpdateMessage_WhenNotActiveIsNoop(t *testing.T) {
	m := NewConsoleSpinnerManager(false)

	m.UpdateMessage("new message")

	if m.spinner.Suffix != "" {
		t.Errorf("expected UpdateMessage to be a no-op when inactive, got suffix %q", m.spinner.Suffix)
	}
}

func TestConsoleSpinnerManager_UpdateMessage_WhenActive(t *testing.T) {
	m := NewConsoleSpinnerManager(false)
	m.Start("initial")
	defer m.Stop()

	m.UpdateMessage("updated")

	if m.spinner.Suffix != " updated" {
		t.Errorf("expected suffix to update, got %q", m.spinner.Suffix)
	}
}
