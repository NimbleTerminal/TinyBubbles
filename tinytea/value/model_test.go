package value

import (
	tea "nimbleterminal.dev/tinybubbles/tinytea"
	cells "nimbleterminal.dev/tinybubbles/widgets/value"
	"testing"
)

func TestApplyCancel(t *testing.T) {
	m, err := New(cells.Config{Min: -10, Max: 10, Step: 5})
	if err != nil {
		t.Fatal(err)
	}
	m.Update(tea.WindowSizeMsg{Width: 21, Height: 8})
	m.Update(tea.KeyPressMsg{Code: tea.KeyLeft})
	_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	if cmd().(cells.AppliedMsg).Value != -5 {
		t.Fatal("apply changed after another key")
	}
	_, cmd = m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if cmd().(cells.CancelledMsg).Value != -5 || m.Value() != -5 {
		t.Fatal("cancel failed")
	}
	if _, err := New(cells.Config{}); err == nil {
		t.Fatal("invalid configuration accepted")
	}
}
