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
