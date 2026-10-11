package web

import (
	tea "charm.land/bubbletea/v2"
	"github.com/NimbleMarkets/go-booba/actions"
	uv "github.com/charmbracelet/ultraviolet"
	"testing"
)

func TestViewportActions(t *testing.T) {
	m, pixels, err := NewViewportDemo(&display{}, true)
	if err != nil {
		t.Fatal(err)
	}
	defer pixels.Close()
	m.Update(tea.WindowSizeMsg{Width: 21, Height: 8})
	m.View()
	_, cmd := m.Update(uv.UnknownOscEvent(actions.QueryProgramStatus))
	top := snapshot(t, cmd)
	for _, a := range top.Actions {
		if a.Label == "Scroll up" && a.Enabled {
			t.Fatal("top has up")
		}
	}
	m.Update(tea.KeyPressMsg{Code: tea.KeyEnd})
	_, cmd = m.Update(uv.UnknownOscEvent(actions.QueryProgramStatus))
	bottom := snapshot(t, cmd)
	found := false
	for _, a := range bottom.Actions {
		if a.Label == "Scroll down" && a.Enabled {
			t.Fatal("bottom has down")
		}
		if a.Label == "Scroll up" && a.Enabled {
			found = true
		}
	}
	if !found || !pixels.Model().(*viewportDemo).AtBottom() {
		t.Fatal("bottom actions")
	}
}
