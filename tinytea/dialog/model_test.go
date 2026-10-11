package dialog

import (
	tea "nimbleterminal.dev/tinybubbles/tinytea"
	cells "nimbleterminal.dev/tinybubbles/widgets/dialog"
	"testing"
)

func TestDecisions(t *testing.T) {
	m := New(cells.Config{})
	m.Update(tea.WindowSizeMsg{Width: 21, Height: 8})
	m.Update(tea.KeyPressMsg{Code: tea.KeyLeft})
	_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if _, again := m.Update(tea.KeyPressMsg{Code: tea.KeyEscape}); again != nil {
		t.Fatal("duplicate result")
	}
	m.Reset()
	if !cmd().(ResultMsg).Confirmed {
		t.Fatal("result not captured")
	}
	_, cmd = m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if cmd().(ResultMsg).Confirmed {
		t.Fatal("escape confirmed")
	}
}
