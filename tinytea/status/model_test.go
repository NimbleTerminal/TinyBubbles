package status

import (
	tea "nimbleterminal.dev/tinybubbles/tinytea"
	cells "nimbleterminal.dev/tinybubbles/widgets/status"
	"testing"
)

func TestUpdate(t *testing.T) {
	m := New(cells.Config{Label: "Network"})
	m.Update(tea.WindowSizeMsg{Width: 21, Height: 2})
	_, cmd := m.Update(UpdateMsg{State: cells.Error, Detail: "Failed"})
	if cmd != nil || m.State() != cells.Error || m.Detail() != "Failed" {
		t.Fatal("state update")
	}
	w, h := m.CellView().Size()
	if w != 21 || h != 2 {
		t.Fatal("size")
	}
	m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.State() != cells.Error {
		t.Fatal("widget consumed input")
	}
}
