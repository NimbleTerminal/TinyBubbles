package tinybubbles

import "testing"

func TestBufferResize(t *testing.T) {
	b := NewBuffer(3, 2)
	b.Set(1, 1, Cell{Rune: 'X', Style: Accent})
	original := &b.cells[0]
	if b.Resize(3, 2) || &b.cells[0] != original || b.At(1, 1).Rune != 'X' {
		t.Fatal("unchanged resize lost cells")
	}
	if n := testing.AllocsPerRun(100, func() { b.Resize(3, 2) }); n != 0 {
		t.Fatal("unchanged resize allocated", n)
	}
	if !b.Resize(2, 3) || b.At(1, 1) != (Cell{}) {
		t.Fatal("changed shape did not clear cells")
	}
	if !b.Resize(-1, 3) {
		t.Fatal("empty resize not reported")
	}
	if w, h := b.Size(); w != 0 || h != 0 {
		t.Fatal(w, h)
	}
	if b.Resize(0, 10) {
		t.Fatal("equivalent empty size changed")
	}
	if !b.Resize(1, 1) || b.At(0, 0) != (Cell{}) {
		t.Fatal("regrow failed")
	}
}

func TestNilBufferIsEmpty(t *testing.T) {
	var b *Buffer
	if w, h := b.Size(); w != 0 || h != 0 {
		t.Fatal(w, h)
	}
	if b.At(0, 0) != (Cell{}) || b.Resize(4, 4) {
		t.Fatal("nil buffer not empty")
	}
	b.Set(0, 0, Cell{Rune: 'x'})
	b.Clear(Accent)
	b.Text(0, 0, "hello\nworld", Normal)
}
