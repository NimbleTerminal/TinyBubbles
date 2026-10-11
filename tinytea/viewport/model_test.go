package viewport

import (
	tea "nimbleterminal.dev/tinybubbles/tinytea"
	"testing"
)

func TestNavigation(t *testing.T) {
	m := New("a\nb\nc\nd\ne\nf")
	m.Update(tea.WindowSizeMsg{Width: 4, Height: 3})
	m.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	if m.Offset() != 2 {
		t.Fatal("page")
	}
	m.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	if m.Offset() != 1 {
		t.Fatal("line")
	}
	m.Update(tea.KeyPressMsg{Code: tea.KeyLeft})
	if !m.AtTop() {
		t.Fatal("page up")
	}
}
