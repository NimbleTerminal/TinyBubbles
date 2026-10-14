package web

import (
	"image/color"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/NimbleMarkets/go-booba/actions"
	uv "github.com/charmbracelet/ultraviolet"
)

type tinyhalTestLEDs struct {
	writes int
	colors []color.RGBA
}

func (l *tinyhalTestLEDs) WriteColors(colors []color.RGBA) error {
	l.writes++
	l.colors = append(l.colors[:0], colors...)
	return nil
}

func TestTinyHALDemoActionsAndHardwareFlow(t *testing.T) {
	for _, mono := range []bool{false, true} {
		d := &display{}
		leds := &tinyhalTestLEDs{}
		m, err := NewTinyHALDemo(d, leds, mono)
		if err != nil {
			t.Fatal(err)
		}
		if d.frames == 0 {
			t.Fatal("expected initial tinyhal display frame flush")
		}
		if leds.writes == 0 || len(leds.colors) != 3 {
			t.Fatalf("expected 3 LED colors initialized, got writes=%d len=%d", leds.writes, len(leds.colors))
		}

		// Verify initial action snapshot and enabled manual ADC controls.
		m, cmd := m.Update(uv.UnknownOscEvent(actions.QueryProgramStatus))
		first := snapshot(t, cmd)
		var adcPlusEnabled, ledOffFound bool
		for _, a := range first.Actions {
			if a.Label == "ADC +" && a.Enabled {
				adcPlusEnabled = true
			}
			if a.Label == "D1: LED off" {
				ledOffFound = true
			}
		}
		if !adcPlusEnabled || !ledOffFound {
			t.Fatalf("unexpected initial tinyhal actions: %+v", first.Actions)
		}

		// Press 'm' (D0 button) to switch to Sine wave mode; ADC + / - must become disabled.
		m, cmd = m.Update(tea.KeyPressMsg{Code: 'm'})
		sineSnap := snapshot(t, cmd)
		for _, a := range sineSnap.Actions {
			if (a.Label == "ADC +" || a.Label == "ADC -") && a.Enabled {
				t.Fatalf("expected %s to be disabled in Sine wave mode", a.Label)
			}
		}

		// Press Enter (D1 button) to toggle GPIO13 LED off; label must change to "D1: LED on".
		m, cmd = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
		ledSnap := snapshot(t, cmd)
		var ledOnFound, viewLogicFound, exportVCDFound bool
		for _, a := range ledSnap.Actions {
			if a.Label == "D1: LED on" {
				ledOnFound = true
			}
			if a.Label == "View: Logic" && a.Enabled {
				viewLogicFound = true
			}
			if a.Label == "Export .vcd" && a.Enabled {
				exportVCDFound = true
			}
		}
		if !ledOnFound || !viewLogicFound || !exportVCDFound {
			t.Fatalf("expected D1/View/Export actions, got %+v", ledSnap.Actions)
		}

		// Press 'v' to switch to the VCD Logic Analyzer view; label must change to "View: Scope".
		m, cmd = m.Update(tea.KeyPressMsg{Code: 'v'})
		viewSnap := snapshot(t, cmd)
		var viewScopeFound bool
		for _, a := range viewSnap.Actions {
			if a.Label == "View: Scope" {
				viewScopeFound = true
			}
		}
		if !viewScopeFound {
			t.Fatalf("expected 'v' press to update action label to 'View: Scope', got %+v", viewSnap.Actions)
		}

		// Press 'x' to exercise VCD export.
		m, _ = m.Update(tea.KeyPressMsg{Code: 'x'})
	}
}
