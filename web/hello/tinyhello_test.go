package web

import (
	"strings"
	"testing"

	"github.com/NimbleMarkets/go-booba/actions"
)

func TestTinyRuntimeActionsAndQuit(t *testing.T) {
	for _, mono := range []bool{false, true} {
		var output []string
		d := &display{}
		s, err := NewTinyDemo(d, mono, func(wire string) { output = append(output, wire) })
		if err != nil {
			t.Fatal(err)
		}
		if err := s.Program.Start(); err != nil {
			t.Fatal(err)
		}
		if d.frames == 0 {
			t.Fatal("TinyTea did not render")
		}
		if err := s.Input(actions.QueryProgramStatus); err != nil {
			t.Fatal(err)
		}
		if len(output) == 0 || !strings.Contains(output[0], "\x1b]7501;") {
			t.Fatal("missing help transport")
		}
		if s.model.inner.Keys().Color.Enabled() == mono {
			t.Fatal("wrong color availability")
		}
		for _, label := range []string{"Resume", "Pause"} {
			before := len(output)
			if err := s.Input("\x1b"); err != nil {
				t.Fatal(err)
			}
			if s.model.inner.Keys().Pause.Help().Desc != label || len(output) <= before {
				t.Fatal("pause did not update actions")
			}
		}
		if err := s.Tick(); err != nil {
			t.Fatal(err)
		}
		if err := s.Input("q"); err != nil {
			t.Fatal(err)
		}
		if !s.Program.Stopped() {
			t.Fatal("TinyTea did not execute its queued Quit command")
		}
		frames := d.frames
		if err := s.Tick(); err != nil {
			t.Fatal(err)
		}
		if err := s.Input("\r"); err != nil {
			t.Fatal(err)
		}
		if d.frames != frames {
			t.Fatal("stopped program still draws")
		}
	}
}
