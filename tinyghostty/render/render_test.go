package render

import (
	"strings"
	"testing"

	bubbles "nimbleterminal.dev/tinybubbles"
	"nimbleterminal.dev/tinybubbles/tinyghostty/vt"
)

func setup(t *testing.T, cols, rows int) (*vt.Terminal, *Renderer, *bubbles.Buffer) {
	t.Helper()
	term, err := vt.New(cols, rows)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(term.Close)
	return term, New(Theme(false)), bubbles.NewBuffer(cols, rows)
}

func draw(t *testing.T, r *Renderer, term *vt.Terminal, buf *bubbles.Buffer) bool {
	t.Helper()
	changed, err := r.Draw(term, buf)
	if err != nil {
		t.Fatal(err)
	}
	return changed
}

func line(buf *bubbles.Buffer, y int) string {
	w, _ := buf.Size()
	var b strings.Builder
	for x := 0; x < w; x++ {
		if c := buf.At(x, y); c.Rune == 0 {
			b.WriteByte(' ')
		} else {
			b.WriteRune(c.Rune)
		}
	}
	return strings.TrimRight(b.String(), " ")
}

func TestStyles(t *testing.T) {
	term, r, buf := setup(t, 12, 2)
	r.Cursor = false
	term.WriteString("a\x1b[31mb\x1b[32mc\x1b[0m\x1b[7md\x1b[0m\x1b[1me\x1b[0m\x1b[2mf\x1b[0m\x1b[8mg\x1b[0m\x1b[38;2;80;200;255mh")
	draw(t, r, term, buf)
	want := []bubbles.StyleID{bubbles.Normal, Red, Green, bubbles.Selected, bubbles.Accent, bubbles.Muted, bubbles.Normal, bubbles.Accent}
	for x, s := range want {
		if got := buf.At(x, 0).Style; got != s {
			t.Errorf("cell %d style %d, want %d", x, got, s)
		}
	}
	if buf.At(6, 0).Rune != 0 {
		t.Error("invisible text drawn")
	}
	theme := Theme(true)
	mono := New(theme)
	mono.Cursor = false
	term.WriteString("\x1b[H\x1b[2J\x1b[31mr\x1b[0m\x1b[7mi")
	draw(t, mono, term, buf)
	// Compare drawn colors: unused monochrome styles render as Normal.
	if theme.Styles[buf.At(0, 0).Style] != theme.Styles[bubbles.Normal] ||
		theme.Styles[buf.At(1, 0).Style] != theme.Styles[bubbles.Selected] {
		t.Error("monochrome mapping")
	}
}

func TestDirtyRowsAndCursor(t *testing.T) {
	term, r, buf := setup(t, 8, 3)
	term.WriteString("one\r\ntwo")
	if !draw(t, r, term, buf) || line(buf, 0) != "one" || line(buf, 1) != "two" {
		t.Fatalf("first frame %q %q", line(buf, 0), line(buf, 1))
	}
	if c := buf.At(3, 1); c.Style != bubbles.Selected {
		t.Fatalf("cursor cell %+v", c)
	}
	if draw(t, r, term, buf) {
		t.Fatal("clean frame reported change")
	}
	// A partial redraw must not touch clean rows; mark one to detect it.
	buf.Set(7, 0, bubbles.Cell{Rune: '#'})
	term.WriteString("\x1b[3;1Hthree")
	draw(t, r, term, buf)
	if line(buf, 2) != "three" || buf.At(7, 0).Rune != '#' {
		t.Fatalf("partial frame %q %q", line(buf, 0), line(buf, 2))
	}
	if buf.At(3, 1).Style != bubbles.Normal || buf.At(5, 2).Style != bubbles.Selected {
		t.Fatal("cursor not moved")
	}
	term.WriteString("\x1b[?25l")
	draw(t, r, term, buf)
	if buf.At(5, 2).Style != bubbles.Normal {
		t.Fatal("hidden cursor drawn")
	}
}

func TestResizeAndWide(t *testing.T) {
	term, r, buf := setup(t, 6, 2)
	r.Cursor = false
	term.WriteString("\x1b[41m界\x1b[0mx")
	draw(t, r, term, buf)
	if buf.At(0, 0).Rune != '界' || buf.At(1, 0).Rune != 0 || buf.At(1, 0).Style != buf.At(0, 0).Style || buf.At(2, 0).Rune != 'x' {
		t.Fatalf("wide cells %+v %+v %+v", buf.At(0, 0), buf.At(1, 0), buf.At(2, 0))
	}
	// A larger buffer than the terminal stays blank outside it.
	buf.Resize(9, 4)
	buf.Set(8, 3, bubbles.Cell{Rune: '#'})
	draw(t, r, term, buf)
	if buf.At(8, 3).Rune != 0 || line(buf, 0) != "界 x" {
		t.Fatalf("resized buffer %q", line(buf, 0))
	}
	if err := term.Resize(4, 1); err != nil {
		t.Fatal(err)
	}
	buf.Resize(4, 1)
	draw(t, r, term, buf)
	if line(buf, 0) != "界 x" {
		t.Fatalf("resized terminal %q", line(buf, 0))
	}
}

func TestDrawAllocs(t *testing.T) {
	term, r, buf := setup(t, 40, 16)
	draw(t, r, term, buf)
	tick := []byte("\x1b[H\x1b[32mtick\x1b[0m")
	allocs := testing.AllocsPerRun(20, func() {
		term.Write(tick)
		r.Draw(term, buf)
	})
	if allocs != 0 {
		t.Fatalf("Draw allocates %.1f times per frame", allocs)
	}
}
