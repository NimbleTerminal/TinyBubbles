package dialog

import (
	tea "charm.land/bubbletea/v2"
	cells "nimbleterminal.dev/tinybubbles/widgets/dialog"
	"testing"
)

func TestDecisionCommandsAndHelp(t *testing.T) {
	m := New(cells.Config{})
	m.Update(tea.WindowSizeMsg{Width: 21, Height: 8})
	m.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.Keys().Select.Enabled() || m.Keys().Cancel.Enabled() {
		t.Fatal("resolved dialog exposes actions")
	}
	if _, repeat := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter}); repeat != nil {
		t.Fatal("duplicate result")
	}
	m.Reset()
	if !cmd().(ResultMsg).Confirmed {
		t.Fatal("result changed after reset")
	}
	if !m.Keys().Select.Enabled() {
		t.Fatal("reset did not enable controls")
	}
	m.KeyMap.Next.SetEnabled(false)
	m.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	if m.ConfirmSelected() {
		t.Fatal("disabled binding moved selection")
	}
	_, cmd = m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if cmd().(ResultMsg).Confirmed {
		t.Fatal("escape confirmed")
	}
}
