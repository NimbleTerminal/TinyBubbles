package web

import (
	tea "charm.land/bubbletea/v2"
	"github.com/NimbleMarkets/go-booba/actions"
	uv "github.com/charmbracelet/ultraviolet"
	"nimbleterminal.dev/tinybubbles/bubbletea/settings"
	"testing"
)

func TestSettingsActionsFollowFocus(t *testing.T) {
	m, pixels, err := NewSettingsDemo(&display{}, true)
	if err != nil {
		t.Fatal(err)
	}
	defer pixels.Close()
	m.Update(tea.WindowSizeMsg{Width: 21, Height: 8})
	m.View()
	_, cmd := m.Update(uv.UnknownOscEvent(actions.QueryProgramStatus))
	initial := snapshot(t, cmd)
	if len(initial.Short) != 3 {
		t.Fatal("menu controls missing")
	}
	inner := pixels.Model().(*settings.Model)
	_, cmd = inner.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	_, cmd = m.Update(cmd())
	editor := snapshot(t, cmd)
	found := false
	for _, a := range editor.Actions {
		if a.Label == "Apply" {
			found = true
		}
	}
	if !found {
		t.Fatal("editor controls missing")
	}
	_, cmd = inner.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	_, cmd = m.Update(cmd())
	menu := snapshot(t, cmd)
	for _, a := range menu.Actions {
		if a.Label == "Apply" {
			t.Fatal("stale editor controls")
		}
	}
}
