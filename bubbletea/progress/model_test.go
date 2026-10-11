package progress

import (
	tea "charm.land/bubbletea/v2"
	cells "nimbleterminal.dev/tinybubbles/widgets/progress"
	"testing"
)

func TestMessages(t *testing.T) {
	m := New(cells.Config{Total: 100})
	m.Update(tea.WindowSizeMsg{Width: 21, Height: 8})
	if _, cmd := m.Update(UpdateMsg{Done: 120, Total: 100}); cmd != nil || !m.Complete() {
		t.Fatal("update")
	}
	m.Update(UpdateMsg{})
	m.Update(PulseMsg{})
	if !m.Indeterminate() {
		t.Fatal("unknown total")
	}
	w, h := m.CellView().Size()
	if w != 21 || h != 8 {
		t.Fatal("size")
	}
}
