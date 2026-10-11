package viewport

import (
	tea "charm.land/bubbletea/v2"
	"strings"
	"testing"
)

func TestKeys(t *testing.T) {
	m := New(strings.Repeat("line\n", 20))
	m.Update(tea.WindowSizeMsg{Width: 21, Height: 4})
	if m.Keys().Up.Enabled() || !m.Keys().Down.Enabled() {
		t.Fatal("top help")
	}
	m.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	if m.Offset() != 3 {
		t.Fatal("page")
	}
	m.KeyMap.Down.SetEnabled(false)
	m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	if m.Offset() != 3 {
		t.Fatal("disabled key")
	}
	m.Update(tea.KeyPressMsg{Code: tea.KeyEnd})
	if !m.AtBottom() || m.Keys().PageDown.Enabled() {
		t.Fatal("bottom")
	}
	m.Update(tea.KeyPressMsg{Code: tea.KeyHome})
	if !m.AtTop() {
		t.Fatal("home")
	}
	m.SetContent("short")
	if m.Keys().Down.Enabled() {
		t.Fatal("short help")
	}
}
