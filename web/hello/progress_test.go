package web

import (
	tea "charm.land/bubbletea/v2"
	"github.com/NimbleMarkets/go-booba/actions"
	uv "github.com/charmbracelet/ultraviolet"
	"testing"
)

func TestProgressLifecycle(t *testing.T) {
	m := newProgressDemo()
	m.Update(progressTick{})
	if m.Done() != 5 {
		t.Fatal("tick")
	}
	m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m.Update(progressTick{})
	if m.Done() != 5 || !m.paused {
		t.Fatal("pause")
	}
	_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd == nil || m.paused {
		t.Fatal("resume")
	}
	m.Update(progressTick{})
	if m.Done() != 5 {
		t.Fatal("stale tick accepted")
	}
	for i := 0; i < 19; i++ {
		_, cmd = m.Update(progressTick{m.generation})
	}
	if !m.Complete() || cmd != nil || m.keys().pause.Enabled() {
		t.Fatal("completion")
	}
	m.Update(tea.KeyPressMsg{Code: 'r'})
	if m.Done() != 0 || !m.keys().pause.Enabled() {
		t.Fatal("restart")
	}
	m.Update(tea.KeyPressMsg{Code: 'm'})
	if !m.Indeterminate() {
		t.Fatal("mode")
	}
	_, cmd = m.Update(progressTick{m.generation})
	if cmd == nil {
		t.Fatal("unknown activity stopped")
	}
}
func TestProgressActions(t *testing.T) {
	m, pixels, err := NewProgressDemo(&display{}, true)
	if err != nil {
		t.Fatal(err)
	}
	defer pixels.Close()
	m.Update(tea.WindowSizeMsg{Width: 21, Height: 8})
	m.View()
	_, cmd := m.Update(uv.UnknownOscEvent(actions.QueryProgramStatus))
	s := snapshot(t, cmd)
	if len(s.Short) != 3 {
		t.Fatal("missing controls")
	}
	inner := pixels.Model().(*progressDemo)
	inner.paused = true
	_, cmd = m.Update(uv.UnknownOscEvent(actions.QueryProgramStatus))
	s = snapshot(t, cmd)
	found := false
	for _, a := range s.Actions {
		if a.Label == "Resume" {
			found = true
		}
	}
	if !found {
		t.Fatal("pause label")
	}
}
