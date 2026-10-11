package web

import (
	tea "charm.land/bubbletea/v2"
	"github.com/NimbleMarkets/go-booba/actions"
	uv "github.com/charmbracelet/ultraviolet"
	"nimbleterminal.dev/tinybubbles/layout"
	"testing"
)

func TestLayoutPlayground(t *testing.T) {
	m := newLayoutDemo()
	m.Update(tea.WindowSizeMsg{Width: 21, Height: 8})
	if m.panels[1].Y <= m.panels[0].Y || m.panels[0].X != 1 {
		t.Fatal("OLED should stack with padding")
	}
	m.Update(tea.WindowSizeMsg{Width: 40, Height: 16})
	if m.panels[1].X <= m.panels[0].X {
		t.Fatal("TFT should use columns")
	}
	m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.panels[1].Y <= m.panels[0].Y {
		t.Fatal("forced rows")
	}
	m.Update(tea.KeyPressMsg{Code: 'p'})
	if m.panels[0].X != 0 {
		t.Fatal("padding toggle")
	}
	m.Update(tea.KeyPressMsg{Code: 'a'})
	if m.alignment != layout.End {
		t.Fatal("alignment")
	}
	m.Update(tea.KeyPressMsg{Code: 'g'})
	if m.gap != 0 {
		t.Fatal("gap")
	}
	m.Update(tea.KeyPressMsg{Code: 'r'})
	if m.mode != 0 || m.padding != 1 || m.gap != 1 || m.alignment != layout.Center {
		t.Fatal("reset")
	}
	for _, sz := range [][2]int{{0, 0}, {1, 1}, {2, 2}, {21, 8}, {40, 16}, {53, 30}} {
		m.Update(tea.WindowSizeMsg{Width: sz[0], Height: sz[1]})
		b := m.CellView()
		title := m.title
		if n := testing.AllocsPerRun(10, func() { m.CellView() }); n != 0 {
			t.Fatal(n)
		}
		if b != m.CellView() || title != m.title {
			t.Fatal("view mutated state")
		}
		for _, r := range m.panels {
			if r.Width < 0 || r.Height < 0 || r.X+r.Width > sz[0] || r.Y+r.Height > sz[1] {
				t.Fatal("region outside display", sz, r)
			}
		}
	}
}
func TestLayoutDemoActions(t *testing.T) {
	m, pixels, err := NewLayoutDemo(&display{}, true)
	if err != nil {
		t.Fatal(err)
	}
	defer pixels.Close()
	m.Update(tea.WindowSizeMsg{Width: 21, Height: 8})
	m.View()
	_, cmd := m.Update(uv.UnknownOscEvent(actions.QueryProgramStatus))
	s := snapshot(t, cmd)
	if len(s.Short) != 3 || len(s.Actions) != 6 {
		t.Fatal("missing controls", s)
	}
	m.Update(tea.KeyPressMsg{Code: 'p'})
	if pixels.Model().(*layoutDemo).padding != 0 {
		t.Fatal("action not routed")
	}
}
