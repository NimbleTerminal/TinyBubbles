package list

import (
	tea "nimbleterminal.dev/tinybubbles/tinytea"
	cells "nimbleterminal.dev/tinybubbles/widgets/list"
	"testing"
)

func TestActivation(t *testing.T) {
	m := New([]cells.Item{{ID: "a"}, {ID: "b"}})
	m.Update(tea.WindowSizeMsg{Width: 10, Height: 2})
	m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m.Move(-1)
	if cmd().(SelectedMsg).Item.ID != "b" {
		t.Fatal("activation did not capture selected item")
	}
	m.SetItems(nil)
	_, cmd = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd != nil {
		t.Fatal("empty list activated")
	}
}
