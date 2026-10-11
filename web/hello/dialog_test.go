package web

import (
	tea "charm.land/bubbletea/v2"
	"github.com/NimbleMarkets/go-booba/actions"
	uv "github.com/charmbracelet/ultraviolet"
	"testing"
)

func TestDialogDemoControls(t *testing.T) {
	m, pixels, err := NewDialogDemo(&display{}, true)
	if err != nil {
		t.Fatal(err)
	}
	defer pixels.Close()
	m.Update(tea.WindowSizeMsg{Width: 200, Height: 100})
	m.View()
	_, cmd := m.Update(uv.UnknownOscEvent(actions.QueryProgramStatus))
	first := snapshot(t, cmd)
	if len(first.Short) != 3 || len(first.Actions) != 5 {
		t.Fatal("missing controls")
	}
	inner := pixels.Model().(*dialogDemo)
	_, cmd = inner.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	result := cmd()
	_, cmd = m.Update(result)
	done := snapshot(t, cmd)
	if inner.result != "Settings kept" || len(done.Short) != 1 || done.Actions[0].Label != "Ask again" {
		t.Fatal("result did not replace dialog controls")
	}
	inner.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if inner.result != "" || inner.dialog.ConfirmSelected() {
		t.Fatal("reopening lost safe default")
	}
	inner.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	_, cmd = inner.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	inner.Update(cmd())
	if inner.result != "Defaults restored" {
		t.Fatal("confirmation was not handled")
	}
}
