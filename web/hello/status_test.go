package web

import (
	tea "charm.land/bubbletea/v2"
	"github.com/NimbleMarkets/go-booba/actions"
	uv "github.com/charmbracelet/ultraviolet"
	"nimbleterminal.dev/tinybubbles/widgets/status"
	"testing"
)

func TestStatusDemo(t *testing.T) {
	for _, mono := range []bool{false, true} {
		m, pixels, err := NewStatusDemo(&display{}, mono)
		if err != nil {
			t.Fatal(err)
		}
		m.Update(tea.WindowSizeMsg{Width: 21, Height: 8})
		m.View()
		_, cmd := m.Update(uv.UnknownOscEvent(actions.QueryProgramStatus))
		if len(snapshot(t, cmd).Short) != 3 {
			t.Fatal("controls")
		}
		inner := pixels.Model().(*statusDemo)
		for i := 0; i < 5; i++ {
			m.Update(tea.KeyPressMsg{Code: tea.KeyRight})
		}
		if inner.State() != status.Ready {
			t.Fatal("wrap")
		}
		m.Update(tea.KeyPressMsg{Code: tea.KeyLeft})
		if inner.State() != status.Offline {
			t.Fatal("previous")
		}
		m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
		if inner.State() != status.Ready || inner.Detail() != "Connected" {
			t.Fatal("reset")
		}
		pixels.Close()
	}
}
