package main

import (
	"errors"
	"image"
	"testing"

	tea "charm.land/bubbletea/v2"
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
	return &app{inner: inner, keys: model.Keys(), fb: fb, pic: pic, wantKitty: true}
}

func TestManualModeOverridesAutoSelection(t *testing.T) {
	ntpicture.ForceKittyCapability(ntpicture.KittyCapabilitySupported)
	t.Cleanup(func() { ntpicture.ForceKittyCapability(ntpicture.KittyCapabilityUnknown) })
	a := testApp(t)
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
