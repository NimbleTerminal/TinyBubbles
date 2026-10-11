package web

import (
	"encoding/base64"
	"encoding/json"
	"image/color"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/NimbleMarkets/go-booba/actions"
	uv "github.com/charmbracelet/ultraviolet"
)

type display struct{ frames int }

func (*display) Size() (int16, int16)              { return 128, 64 }
func (*display) SetPixel(int16, int16, color.RGBA) {}
func (d *display) Display() error                  { d.frames++; return nil }

// Decode the actual command payload, including the OSC/base64 envelope, rather
// than inspecting the model's keys independently of its transport.
func snapshot(t *testing.T, cmd tea.Cmd) actions.Snapshot {
	t.Helper()
	if cmd == nil {
		t.Fatal("missing action update")
	}
	msg := cmd()
	if batch, ok := msg.(tea.BatchMsg); ok {
		for _, c := range batch {
			if c != nil {
				return snapshot(t, c)
			}
		}
		t.Fatal("empty action batch")
	}
	raw, ok := msg.(tea.RawMsg)
	if !ok {
		t.Fatalf("unexpected command: %T", msg)
	}
	wire := raw.Msg.(string)
	_, encoded, ok := strings.Cut(wire, ":msg=")
	if !ok {
		t.Fatalf("missing OSC status: %q", wire)
	}
	data, err := base64.StdEncoding.DecodeString(strings.TrimSuffix(encoded, "\x1b\\"))
	if err != nil {
		t.Fatal(err)
	}
	_, payload, ok := strings.Cut(string(data), actions.MessageMarker)
	if !ok {
		t.Fatal("missing action payload")
	}
	var s actions.Snapshot
	if err := json.Unmarshal([]byte(payload), &s); err != nil {
		t.Fatal(err)
	}
	if err := s.Validate(); err != nil {
		t.Fatal(err)
	}
	return s
}

func TestHelpActionsFollowModelAndDisplay(t *testing.T) {
	for _, mono := range []bool{false, true} {
		d := &display{}
		m, pixels, err := NewDemo(d, mono)
		if err != nil {
			t.Fatal(err)
		}
		defer pixels.Close()
		m, _ = m.Update(tea.WindowSizeMsg{Width: 200, Height: 100})
		m.View()
		if d.frames == 0 {
			t.Fatal("no display output")
		}
		var cmd tea.Cmd
		m, cmd = m.Update(uv.UnknownOscEvent(actions.QueryProgramStatus))
		first := snapshot(t, cmd)
		if len(first.Actions) != 4 || len(first.Short) != 3 || len(first.Full) != 2 {
			t.Fatalf("wrong help groups: %+v", first)
		}
		for _, a := range first.Actions {
			if a.Label == "Color" && a.Enabled == mono {
				t.Fatal("color availability ignores display")
			}
		}
		m, cmd = m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
		paused := snapshot(t, cmd)
		if paused.Revision <= first.Revision {
			t.Fatal("revision did not advance")
		}
		found := false
		for _, a := range paused.Actions {
			found = found || a.Label == "Resume"
		}
		if !found {
			t.Fatal("pause did not update the HTML action label")
		}
		_, cmd = m.Update(tea.KeyPressMsg{Code: 'q'})
		if cmd == nil {
			t.Fatal("quit binding lost")
		}
	}
}
