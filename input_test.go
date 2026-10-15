package tinybubbles

import (
	"testing"
	"time"
)

func TestButtonDebouncesPressAndRelease(t *testing.T) {
	b := Button{Key: KeySelect, Debounce: 25 * time.Millisecond}
	start := time.Unix(0, 0)
	steps := []struct {
		ms          int
		down, event bool
	}{
		{0, false, false}, {1, true, false}, {5, false, false}, {10, true, false},
		{34, true, false}, {35, true, true}, {100, true, false},
		{101, false, false}, {110, true, false}, {120, false, false}, {145, false, false},
		{150, true, false}, {175, true, true},
	}
	for _, s := range steps {
		msg, ok := b.Update(s.down, start.Add(time.Duration(s.ms)*time.Millisecond))
		if ok != s.event || (ok && msg.Key != KeySelect) {
			t.Fatalf("at %dms: %v %v", s.ms, msg, ok)
		}
	}
}

func TestButtonHoldToRepeat(t *testing.T) {
	b := Button{
		Key:            KeyDown,
		Debounce:       10 * time.Millisecond,
		RepeatDelay:    200 * time.Millisecond,
		RepeatInterval: 50 * time.Millisecond,
	}
	t0 := time.Unix(0, 0)

	// Start press at t=0; debounced at t=10ms.
	if _, ok := b.Update(true, t0); ok {
		t.Fatal("debounce bypassed")
	}
	if msg, ok := b.Update(true, t0.Add(10*time.Millisecond)); !ok || msg != (KeyMsg{KeyDown}) {
		t.Fatalf("expected initial press at 10ms, got %v %v", msg, ok)
	}
	// Held before RepeatDelay (10ms + 199ms = 209ms): no repeat yet.
	if _, ok := b.Update(true, t0.Add(209*time.Millisecond)); ok {
		t.Fatal("repeated before RepeatDelay")
	}
	// First repeat at 210ms (10ms + 200ms).
	if msg, ok := b.Update(true, t0.Add(210*time.Millisecond)); !ok || msg != (KeyMsg{KeyDown}) {
		t.Fatalf("expected first repeat at 210ms, got %v %v", msg, ok)
	}
	// Subsequent repeats at 260ms and 310ms.
	if msg, ok := b.Update(true, t0.Add(260*time.Millisecond)); !ok || msg != (KeyMsg{KeyDown}) {
		t.Fatalf("expected second repeat at 260ms, got %v %v", msg, ok)
	}
	if msg, ok := b.Update(true, t0.Add(310*time.Millisecond)); !ok || msg != (KeyMsg{KeyDown}) {
		t.Fatalf("expected third repeat at 310ms, got %v %v", msg, ok)
	}
	// Release at 320ms, debounced at 330ms: stops repeating.
	b.Update(false, t0.Add(320*time.Millisecond))
	if _, ok := b.Update(false, t0.Add(330*time.Millisecond)); ok {
		t.Fatal("unexpected press on release")
	}
	if _, ok := b.Update(false, t0.Add(400*time.Millisecond)); ok {
		t.Fatal("repeated while released")
	}
}

func TestButtonShortAndLongPressWithRelease(t *testing.T) {
	b := Button{
		Key:         KeyUp,
		LongKey:     KeyBack,
		Debounce:    10 * time.Millisecond,
		LongPress:   400 * time.Millisecond,
		EmitRelease: true,
	}
	t0 := time.Unix(0, 0)

	// 1. Short tap: press at 0ms, debounced at 10ms (deferred), release at 100ms, debounced at 110ms.
	if msg := b.Poll(true, t0); msg != nil {
		t.Fatalf("unexpected at 0ms: %v", msg)
	}
	if msg := b.Poll(true, t0.Add(10*time.Millisecond)); msg != nil {
		t.Fatalf("expected short press to wait until release or LongPress, got %v", msg)
	}
	if msg := b.Poll(false, t0.Add(100*time.Millisecond)); msg != nil {
		t.Fatalf("unexpected before release debounce: %v", msg)
	}
	if msg := b.Poll(false, t0.Add(110*time.Millisecond)); msg != (KeyMsg{KeyUp}) {
		t.Fatalf("expected short KeyUp on release at 110ms, got %v", msg)
	}
	if msg := b.Poll(false, t0.Add(120*time.Millisecond)); msg != (KeyReleaseMsg{KeyUp}) {
		t.Fatalf("expected pending KeyReleaseMsg{KeyUp} at 120ms, got %v", msg)
	}

	// 2. Long hold: press at 200ms, debounced at 210ms, held past 610ms -> emits LongKey (KeyBack).
	b.Poll(true, t0.Add(200*time.Millisecond))
	if msg := b.Poll(true, t0.Add(210*time.Millisecond)); msg != nil {
		t.Fatalf("unexpected at 210ms: %v", msg)
	}
	if msg := b.Poll(true, t0.Add(609*time.Millisecond)); msg != nil {
		t.Fatalf("fired before LongPress at 609ms: %v", msg)
	}
	if msg := b.Poll(true, t0.Add(610*time.Millisecond)); msg != (KeyMsg{KeyBack}) {
		t.Fatalf("expected LongKey KeyBack at 610ms, got %v", msg)
	}
	// Further hold without RepeatDelay does not re-fire LongKey.
	if msg := b.Poll(true, t0.Add(700*time.Millisecond)); msg != nil {
		t.Fatalf("unexpected duplicate LongKey at 700ms: %v", msg)
	}
	// Release after long press emits only KeyReleaseMsg{KeyBack}, not short KeyUp.
	b.Poll(false, t0.Add(710*time.Millisecond))
	if msg := b.Poll(false, t0.Add(720*time.Millisecond)); msg != (KeyReleaseMsg{KeyBack}) {
		t.Fatalf("expected KeyReleaseMsg{KeyBack} at 720ms, got %v", msg)
	}
}

func TestBufferClippingAndUnicode(t *testing.T) {
	b := NewBuffer(3, 2)
	b.Text(-1, 0, "AéBC\nDEF", Accent)
	if b.At(0, 0) != (Cell{'é', Accent}) || b.At(2, 0).Rune != 'C' || b.At(0, 1).Rune != 'E' {
		t.Fatal("text placement")
	}
	b.Set(-1, -1, Cell{'X', Normal})
	b.Set(3, 2, Cell{'X', Normal})
	if b.At(3, 0) != (Cell{}) {
		t.Fatal("out of bounds")
	}
	b.Clear(Selected)
	if b.At(1, 1) != (Cell{Style: Selected}) {
		t.Fatal("clear")
	}
	if w, h := NewBuffer(-1, 2).Size(); w != 0 || h != 0 {
		t.Fatal("negative dimensions")
	}
}

func TestButtonPollerPreservesSimultaneousPresses(t *testing.T) {
	p := ButtonPoller{Buttons: []Button{{Key: KeyLeft, Debounce: time.Millisecond}, {Key: KeyRight, Debounce: time.Millisecond}}}
	now := time.Now()
	pressed := func(int) bool { return true }
	if msg := p.Poll(now, pressed); msg != nil {
		t.Fatal("debounce bypassed")
	}
	for _, key := range []Key{KeyLeft, KeyRight} {
		msg := p.Poll(now.Add(time.Millisecond), pressed)
		if msg != (KeyMsg{key}) {
			t.Fatalf("got %v want %v", msg, key)
		}
	}
	if msg := p.Poll(now.Add(2*time.Millisecond), pressed); msg != nil {
		t.Fatal("held key repeated")
	}
	empty := ButtonPoller{}
	if msg := empty.Poll(now, pressed); msg != nil {
		t.Fatal(msg)
	}
}
