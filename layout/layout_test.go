package layout

import (
	bubbles "nimbleterminal.dev/tinybubbles"
	"testing"
)

func TestRegions(t *testing.T) {
	r := Rect{2, 3, 20, 8}
	a, b := r.Rows(2, 1)
	if a != (Rect{2, 3, 20, 2}) || b != (Rect{2, 6, 20, 5}) {
		t.Fatal(a, b)
	}
	a, b = r.Columns(7, 2)
	if a != (Rect{2, 3, 7, 8}) || b != (Rect{11, 3, 11, 8}) {
		t.Fatal(a, b)
	}
	if got := r.Inset(1, 2, 3, 4); got != (Rect{6, 4, 14, 4}) {
		t.Fatal(got)
	}
	if got := r.Align(5, 3, Center, End); got != (Rect{9, 8, 5, 3}) {
		t.Fatal(got)
	}
	a, b = r.Rows(99, 99)
	if a != r || b.Height != 0 {
		t.Fatal("oversize", a, b)
	}
	a, b = r.Columns(-2, -2)
	if a.Width != 0 || b != r {
		t.Fatal("negative")
	}
	if got := r.Inset(99, 99, 99, 99); got.Width != 0 || got.Height != 0 {
		t.Fatal(got)
	}
	if got := r.Align(99, 99, Alignment(99), Start); got != r {
		t.Fatal(got)
	}
	if Bounds(nil) != (Rect{}) {
		t.Fatal("nil bounds")
	}
}
func TestDrawClippingAndStyles(t *testing.T) {
	dst := bubbles.NewBuffer(5, 3)
	src := bubbles.NewBuffer(4, 2)
	src.Text(0, 0, "abcd\nefgh", bubbles.Selected)
	Draw(dst, Rect{-1, 1, 3, 2}, src)
	if dst.At(0, 1) != (bubbles.Cell{Rune: 'b', Style: bubbles.Selected}) || dst.At(1, 2).Rune != 'g' || dst.At(2, 1).Rune != 0 || dst.At(0, 0).Rune != 0 {
		t.Fatal("clipping")
	}
	src.Clear(bubbles.Accent)
	Draw(dst, Rect{0, 1, 1, 1}, src)
	if dst.At(0, 1) != (bubbles.Cell{Style: bubbles.Accent}) || dst.At(1, 1).Rune != 'c' {
		t.Fatal("blank copy or region leak")
	}
	Draw(nil, Rect{}, src)
	Draw(dst, Rect{}, nil)
}
func TestSelfCopy(t *testing.T) {
	for _, r := range []Rect{{1, 0, 4, 3}, {0, 1, 5, 2}, {-1, 0, 4, 3}, {0, -1, 5, 2}, {-1, 1, 4, 2}} {
		b := bubbles.NewBuffer(5, 3)
		b.Text(0, 0, "abcde\nfghij\nklmno", bubbles.Normal)
		original := bubbles.NewBuffer(5, 3)
		Draw(original, Bounds(original), b)
		expected := bubbles.NewBuffer(5, 3)
		Draw(expected, Bounds(expected), b)
		Draw(expected, r, original)
		Draw(b, r, b)
		for y := 0; y < 3; y++ {
			for x := 0; x < 5; x++ {
				if b.At(x, y) != expected.At(x, y) {
					t.Fatal("overlap", r, x, y)
				}
			}
		}
	}
}
func TestTextAndAllocations(t *testing.T) {
	b := bubbles.NewBuffer(6, 4)
	r := Rect{1, 1, 3, 2}
	Text(b, r, "abcd\né界\x00z\nno", bubbles.Accent)
	if b.At(3, 1).Rune != 'c' || b.At(4, 1).Rune != 0 || b.At(2, 2).Rune != '界' || b.At(1, 3).Rune != 0 {
		t.Fatal("text escaped region")
	}
	child := bubbles.NewBuffer(3, 2)
	if n := testing.AllocsPerRun(50, func() {
		a, _ := Bounds(b).Inset(1, 0, 0, 0).Rows(2, 0)
		a, _ = a.Columns(3, 0)
		Draw(b, a.Align(3, 2, Center, Center), child)
		Text(b, a, "ok", bubbles.Normal)
	}); n != 0 {
		t.Fatal(n)
	}
}
