package list

import (
	tea "charm.land/bubbletea/v2"
	cells "nimbleterminal.dev/tinybubbles/widgets/list"
	"testing"
)

func TestActivationAndHelp(t *testing.T) {
	m := New([]cells.Item{{ID: "a", Label: "A"}, {ID: "b", Label: "B"}})
	m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m.Move(1)
	selected := cmd().(cells.SelectedMsg)
	if selected.Item.ID != "b" || selected.Index != 1 {
		t.Fatal("activation did not capture selection")
	}
	m.SetItems(nil)
	if m.Keys().Select.Enabled() {
		t.Fatal("empty list exposes select")
	}
	_, cmd = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd != nil {
		t.Fatal("empty list activated")
	}
	m.SetItems([]cells.Item{{ID: "a"}})
	if !m.Keys().Select.Enabled() {
		t.Fatal("select did not become available")
	}
	m.KeyMap.Next.SetEnabled(false)
	if m.Keys().Next.Enabled() {
		t.Fatal("custom disabled binding re-enabled")
	}
}
