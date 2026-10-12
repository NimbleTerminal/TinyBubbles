package web

import (
	tea "charm.land/bubbletea/v2"
	"github.com/NimbleMarkets/go-booba/actions"
	uv "github.com/charmbracelet/ultraviolet"
	"testing"
)

func TestValueDemoActions(t *testing.T) {
	m, pixels, err := NewValueDemo(&display{}, true)
	if err != nil {
		t.Fatal(err)
	}
	defer pixels.Close()
	m.Update(tea.WindowSizeMsg{Width: 200, Height: 100})
	m.View()
	_, cmd := m.Update(uv.UnknownOscEvent(actions.QueryProgramStatus))
	first := snapshot(t, cmd)
	if len(first.Actions) != 5 {
		t.Fatal("missing value actions")
	}
	for i := 0; i < 10; i++ {
		_, cmd = m.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	}
	bounded := snapshot(t, cmd)
	disabled := false
	for _, a := range bounded.Actions {
		if len(a.Keys) == 1 && a.Keys[0].Key == "right" {
			disabled = !a.Enabled
		}
	}
	if !disabled || bounded.Revision <= first.Revision {
		t.Fatal("limit did not disable browser action")
	}
	inner := pixels.Model().(*valueDemo)
	_, cmd = inner.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	inner.Update(cmd())
	if inner.status != "Applied: 100%" {
		t.Fatal(inner.status)
	}
	inner.Update(tea.KeyPressMsg{Code: tea.KeyLeft})
	_, cmd = inner.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	inner.Update(cmd())
	if inner.status != "Restored: 100%" {
		t.Fatal(inner.status)
	}
}
