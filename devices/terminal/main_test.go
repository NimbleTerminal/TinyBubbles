package main

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/NimbleMarkets/go-booba/actions"
	"github.com/NimbleMarkets/go-booba/actions/bubblehelp"
	"github.com/NimbleMarkets/go-booba/actions/termhost"
	"nimbleterminal.dev/ntpicture"
	displaytea "nimbleterminal.dev/tinybubbles/bubbletea"
	teamodel "nimbleterminal.dev/tinybubbles/bubbletea/hello"
	"nimbleterminal.dev/tinybubbles/demo/hello"
)

func testApp(t *testing.T) *app {
	t.Helper()
	fb := &display{RGBA: image.NewRGBA(image.Rect(0, 0, width, height))}
	model := teamodel.New()
	inner, err := displaytea.NewModel(model, fb, hello.NewFont(), hello.Theme())
	if err != nil {
		t.Fatal(err)
	}
	pic := ntpicture.New(ntpicture.Options{})
	t.Cleanup(func() { inner.Close(); pic.Close() })
	return &app{inner: inner, model: model, keys: model.Keys(), fb: fb, pic: pic, wantKitty: true}
}

func TestManualModeOverridesAutoSelection(t *testing.T) {
	a := testApp(t)
	a.pic.SetKittyCapability(ntpicture.KittyCapabilitySupported)
	a.pic.SetMode(ntpicture.PictureKitty)
	a.Update(tea.KeyPressMsg{Code: 'k'})
	if a.wantKitty || a.pic.Mode() != ntpicture.PictureGlyph {
		t.Fatal("manual glyph selection was overridden by automatic Kitty selection")
	}
	a.Update(ntpicture.KittyCapabilityMsg{Capability: ntpicture.KittyCapabilitySupported})
	if a.pic.Mode() != ntpicture.PictureGlyph {
		t.Fatal("late capability reply changed manual selection")
	}
}

func TestPictureMessagesDoNotResubmitFramebuffer(t *testing.T) {
	a := testApp(t)
	a.wantKitty = false
	a.fb.frames++
	generation := a.pic.Status().Generation
	a.Update(ntpicture.ErrorMsg{Owner: a.pic, Generation: generation, Err: errors.New("encode failed")})
	if a.status != "encode failed" {
		t.Fatalf("status = %q", a.status)
	}
	if a.shown != 0 || a.pic.Status().Generation != generation {
		t.Fatal("picture message resubmitted framebuffer")
	}
	a.Update(ntpicture.ErrorMsg{Owner: a.pic, Generation: generation + 1, Err: errors.New("obsolete")})
	a.Update(ntpicture.ErrorMsg{Err: errors.New("other picture")})
	if a.status != "encode failed" {
		t.Fatalf("unrelated error replaced status: %q", a.status)
	}
}

func TestUnpacedHelloDoesNotScheduleTimer(t *testing.T) {
	m := &unpacedHello{teamodel.New()}
	if _, ok := m.Init()().(hello.TickMsg); !ok {
		t.Fatal("missing initial tick")
	}
	if _, cmd := m.Update(hello.TickMsg{}); cmd != nil {
		t.Fatal("uncapped model scheduled timer")
	}
	if _, cmd := m.Update(tea.KeyPressMsg{Code: 'q'}); cmd == nil {
		t.Fatal("quit command was discarded")
	}
}

func TestUncappedCompletionKeepsAdvancingAndPauses(t *testing.T) {
	a := testApp(t)
	a.wantKitty = false
	a.uncapped = true
	a.Update(tea.WindowSizeMsg{Width: 240, Height: 68})
	// Even unchanged pixels must not break the completion-driven frame loop.
	for range 100 {
		generation := a.pic.Status().Generation
		a.Update(ntpicture.PresentedMsg{Owner: a.pic, Generation: generation})
		if a.pic.Status().Generation <= generation {
			t.Fatal("animation stopped scheduling pictures")
		}
	}
	a.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	generation := a.pic.Status().Generation
	a.Update(ntpicture.PresentedMsg{Owner: a.pic, Generation: generation})
	if a.pic.Status().Generation != generation {
		t.Fatal("paused animation advanced")
	}
	a.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if a.pic.Status().Generation <= generation {
		t.Fatal("resume did not restart pictures")
	}
	generation = a.pic.Status().Generation
	a.Update(ntpicture.PresentedMsg{Owner: a.pic, Generation: generation - 1})
	if a.pic.Status().Generation != generation {
		t.Fatal("stale completion advanced animation")
	}
}

// TestHostedActionsFollowHelpMap runs the publisher and host the way the
// program loop does: commands execute, raw output goes to the host's writer,
// and messages come back to Update, until the hosted bar appears.
func TestHostedActionsFollowHelpMap(t *testing.T) {
	a := testApp(t)
	a.wantKitty = false
	published, err := bubblehelp.NewModel(a, func(tea.Model) bubblehelp.Description {
		return bubblehelp.Description{Context: "hello", Keys: a.helpKeys(), Status: actions.ProgramStatus{Summary: "test"}}
	}, bubblehelp.ModelOptions{Instance: "test", Adapter: bubblehelp.Options{IncludeDisabled: true}})
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	host, output := termhost.New(published, &out, termhost.Options{Timeout: time.Millisecond, Styles: &termhost.Styles{}})
	msgs := make(chan tea.Msg, 64)
	var run func(tea.Cmd)
	run = func(cmd tea.Cmd) {
		if cmd == nil {
			return
		}
		go func() {
			switch msg := cmd().(type) {
			case nil:
			case tea.BatchMsg:
				for _, c := range msg {
					run(c)
				}
			default:
				msgs <- msg
			}
		}()
	}
	host.Update(tea.WindowSizeMsg{Width: 240, Height: 70})
	run(host.Init())
	deadline := time.After(2 * time.Second)
	for host.Snapshot() == nil {
		select {
		case msg := <-msgs:
			if raw, ok := msg.(tea.RawMsg); ok {
				fmt.Fprint(output, raw.Msg)
				continue
			}
			_, cmd := host.Update(msg)
			run(cmd)
		case <-deadline:
			t.Fatalf("no hosted snapshot; mode %v output %q", host.Mode(), out.String())
		}
	}
	if host.Mode() != termhost.Hosting {
		t.Fatalf("mode %v", host.Mode())
	}
	labels := map[string]bool{}
	for _, action := range host.Snapshot().Actions {
		labels[action.Label] = action.Enabled
	}
	// Kitty is unknown in tests, so its toggle is published but disabled.
	if !labels["Pop"] || !labels["Color"] || !labels["Quit"] || labels["Kitty graphics"] {
		t.Fatalf("actions %+v", labels)
	}
	if v := host.View(); !strings.Contains(v.Content, "Pop") || !strings.Contains(v.Content, "More") || v.MouseMode != tea.MouseModeCellMotion {
		t.Fatalf("view %q mouse %v", v.Content, v.MouseMode)
	}
	view := host.View()
	lines := strings.Split(view.Content, "\n")
	row := len(lines) - 1
	pauseX := strings.Index(lines[row], "Pause")
	if pauseX < 0 {
		t.Fatal("missing pause button")
	}
	_, click := host.Update(tea.MouseClickMsg{X: pauseX, Y: row, Button: tea.MouseLeft})
	if click == nil {
		t.Fatal("pause click did not produce a key")
	}
	host.Update(click())
	if a.helpKeys().hello.Pause.Help().Desc != "Resume" {
		t.Fatal("mouse click did not reach hello model")
	}
	moreX := strings.Index(lines[row], "More")
	if moreX < 0 {
		t.Fatal("missing More button")
	}
	host.Update(tea.MouseClickMsg{X: moreX, Y: row, Button: tea.MouseLeft})
	if !strings.Contains(host.View().Content, "Quit") {
		t.Fatal("More did not reveal Quit")
	}
	if strings.Contains(out.String(), "\x1b]7501;state=") {
		t.Fatalf("report leaked to the terminal: %q", out.String())
	}
}

func TestActionLabelsFollowModelAndPicture(t *testing.T) {
	a := testApp(t)
	a.wantKitty = false
	a.pic.SetKittyCapability(ntpicture.KittyCapabilitySupported)
	if keys := a.helpKeys(); keys.hello.Pause.Help().Desc != "Pause" || !keys.kitty.Enabled() {
		t.Fatal("initial actions do not match the model and capability")
	}
	a.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if a.helpKeys().hello.Pause.Help().Desc != "Resume" {
		t.Fatal("pause action was not relabeled")
	}
	a.Update(tea.KeyPressMsg{Code: 'k'})
	if keys := a.helpKeys(); keys.kitty.Help().Desc != "Half-blocks" || !keys.kitty.Enabled() {
		t.Fatal("Kitty action does not offer the reverse transition")
	}
	a.Update(tea.KeyPressMsg{Code: 'k'})
	a.pic.SetKittyCapability(ntpicture.KittyCapabilityUnsupported)
	if a.helpKeys().kitty.Enabled() {
		t.Fatal("unavailable Kitty action is enabled")
	}
}
