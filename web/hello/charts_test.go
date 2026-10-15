package web

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/NimbleMarkets/go-booba/actions"
	uv "github.com/charmbracelet/ultraviolet"
	"nimbleterminal.dev/tinybubbles/tinycharts"
)

func TestChartsDemoLifecycleAndZeroAllocView(t *testing.T) {
	m := newChartsDemo()
	m.Update(tea.WindowSizeMsg{Width: 40, Height: 16})
	if m.panels[0].Width <= 0 || m.panels[1].Width <= 0 || m.panels[2].Width <= 0 || m.panels[3].Width <= 0 {
		t.Fatalf("expected 4 panels in 40x16 dashboard view, got %+v", m.panels)
	}

	step0 := m.step
	m.Update(chartsTickMsg{generation: m.generation})
	if m.step <= step0 {
		t.Fatalf("expected tick to advance chart samples")
	}

	// Cycle through Stream, Spark, Bars, Logic, and back to All.
	for want := 1; want < chartViewCount; want++ {
		m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
		if m.viewMode != want {
			t.Fatalf("expected viewMode %d, got %d", want, m.viewMode)
		}
		v := m.CellView()
		lit := 0
		w, h := v.Size()
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				if r := v.At(x, y).Rune; r != 0 && r != ' ' {
					lit++
				}
			}
		}
		if lit == 0 {
			t.Fatalf("expected non-blank cells in viewMode %d", want)
		}
	}
	m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.viewMode != chartViewAll {
		t.Fatalf("expected viewMode to wrap to chartViewAll, got %d", m.viewMode)
	}

	// Cycle line/bar style (Braille -> Arc -> Box) and horizontal/vertical bars.
	m.Update(tea.KeyPressMsg{Code: 'm'})
	if m.streamMode != tinycharts.StreamModeArc || m.horizBars {
		t.Fatalf("expected StreamModeArc and vertical bars after 'm'")
	}
	m.Update(tea.KeyPressMsg{Code: 'm'})
	if m.streamMode != tinycharts.StreamModeBox || !m.horizBars {
		t.Fatalf("expected StreamModeBox and horizontal bars after second 'm'")
	}

	// Inject burst spike.
	m.Update(tea.KeyPressMsg{Code: 'b'})
	if m.burstTTL == 0 {
		t.Fatalf("expected burstTTL > 0 after 'b'")
	}

	// Pause and verify stale ticks are ignored.
	m.Update(tea.KeyPressMsg{Code: 'p'})
	if !m.paused {
		t.Fatalf("expected paused after 'p'")
	}
	heldStep := m.step
	m.Update(chartsTickMsg{generation: m.generation})
	if m.step != heldStep {
		t.Fatalf("expected paused demo to ignore ticks")
	}

	// Reset restores defaults.
	m.Update(tea.KeyPressMsg{Code: 'r'})
	if m.paused || m.viewMode != chartViewAll || m.streamMode != tinycharts.StreamModeBraille || !m.horizBars {
		t.Fatalf("unexpected state after reset: paused=%v view=%d mode=%d horiz=%v", m.paused, m.viewMode, m.streamMode, m.horizBars)
	}

	// Verify zero-allocation CellView across all presets (OLED 21x8, Feather 40x16, Large 53x30).
	for _, sz := range [][2]int{{0, 0}, {21, 8}, {40, 16}, {53, 30}} {
		m.Update(tea.WindowSizeMsg{Width: sz[0], Height: sz[1]})
		b := m.CellView()
		title := m.title
		if n := testing.AllocsPerRun(10, func() { m.CellView() }); n != 0 {
			t.Fatalf("expected 0 allocations in CellView for %v, got %v", sz, n)
		}
		if b != m.CellView() || title != m.title {
			t.Fatalf("CellView mutated state for %v", sz)
		}
	}
}

func TestChartsDemoActions(t *testing.T) {
	m, pixels, err := NewChartsDemo(&display{}, false)
	if err != nil {
		t.Fatal(err)
	}
	defer pixels.Close()

	m.Update(tea.WindowSizeMsg{Width: 40, Height: 16})
	m.View()
	_, cmd := m.Update(uv.UnknownOscEvent(actions.QueryProgramStatus))
	s := snapshot(t, cmd)
	if len(s.Short) != 3 || len(s.Actions) != 6 {
		t.Fatalf("unexpected action metadata: short=%d total=%d", len(s.Short), len(s.Actions))
	}

	m.Update(tea.KeyPressMsg{Code: 'p'})
	_, cmd = m.Update(uv.UnknownOscEvent(actions.QueryProgramStatus))
	s = snapshot(t, cmd)
	foundResume := false
	for _, a := range s.Actions {
		if a.Label == "Resume" {
			foundResume = true
		}
	}
	if !foundResume {
		t.Fatalf("expected Resume label when paused")
	}
}
