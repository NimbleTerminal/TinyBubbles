package choice

import (
	tea "nimbleterminal.dev/tinybubbles/tinytea"
	cells "nimbleterminal.dev/tinybubbles/widgets/choice"
	"testing"
)

func TestCaptureAndHelp(t *testing.T) {
	m, _ := New(cells.Config{Options: []string{"Off", "On"}})
	m.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m.Move(1)
	if got := cmd().(AppliedMsg); got.Index != 1 || got.Label != "On" {
		t.Fatal(got)
	}
	_, cmd = m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if got := cmd().(CancelledMsg); got.Index != 1 || m.Value() != 1 {
		t.Fatal(got)
	}
}
