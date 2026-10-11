package web

import (
	tea "charm.land/bubbletea/v2"
	"github.com/NimbleMarkets/go-booba/actions"
	uv "github.com/charmbracelet/ultraviolet"
	"testing"
)

func TestListDemoActions(t *testing.T) {
	m, pixels, err := NewListDemo(&display{}, true)
	if err != nil {
		t.Fatal(err)
	}
	defer pixels.Close()
	m.Update(tea.WindowSizeMsg{Width: 200, Height: 100})
	m.View()
	_, cmd := m.Update(uv.UnknownOscEvent(actions.QueryProgramStatus))
	s := snapshot(t, cmd)
	if len(s.Short) != 3 || len(s.Actions) != 4 {
		t.Fatalf("unexpected help: %+v", s)
	}
	inner := pixels.Model().(*listDemo)
	if w, h := inner.buffer.Size(); w != 21 || h != 8 {
		t.Fatalf("display size lost: %dx%d", w, h)
	}
	inner.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	_, cmd = inner.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	inner.Update(cmd())
	if inner.status != "Selected: Restart" {
		t.Fatalf("activation not handled: %s", inner.status)
	}
}
