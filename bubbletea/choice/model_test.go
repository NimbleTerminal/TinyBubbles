package choice

import (
	tea "charm.land/bubbletea/v2"
	cells "nimbleterminal.dev/tinybubbles/widgets/choice"
	"testing"
)

func TestCaptureAndHelp(t *testing.T) {
	m, _ := New(cells.Config{Options: []string{"Off", "On"}})
	m.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m.Move(1)
	if got := cmd().(cells.AppliedMsg); got.Index != 1 || got.Label != "On" {
		t.Fatal(got)
	}
	_, cmd = m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if got := cmd().(cells.CancelledMsg); got.Index != 1 || m.Value() != 1 {
		t.Fatal(got)
	}
	m.KeyMap.Next.SetEnabled(false)
	m.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	if m.Value() != 1 || m.Keys().Next.Enabled() {
		t.Fatal("disabled key accepted")
	}
	one, _ := New(cells.Config{Options: []string{"only"}})
	if one.Keys().Next.Enabled() || one.Keys().Previous.Enabled() {
		t.Fatal("single choice help")
	}
}
