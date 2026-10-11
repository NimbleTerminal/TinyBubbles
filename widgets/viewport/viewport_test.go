package viewport

import (
	"strings"
	"testing"
)

func row(m *Model, y int) string {
	b := m.CellView()
	w, _ := b.Size()
	var s strings.Builder
	for x := 0; x < w; x++ {
		r := b.At(x, y).Rune
		if r == 0 {
			r = ' '
		}
		s.WriteRune(r)
	}
	return s.String()
}
func TestLayout(t *testing.T) {
	m := New("abé界f\r\n\n  x\tY\x00")
	m.SetSize(5, 10)
	if m.TotalLines() != 5 {
		t.Fatal(m.TotalLines())
	}
	if row(m, 0) != "abé界 " || row(m, 1) != "f    " || row(m, 2) != "     " || row(m, 3) != "  x  " || row(m, 4) != "   Y " {
		for i := 0; i < m.TotalLines(); i++ {
			t.Log(row(m, i))
		}
		t.Fatal("wrapping/whitespace")
	}
	m.SetContent("abcd\n")
	if m.TotalLines() != 2 {
		t.Fatal("exact width created extra row")
	}
	m.SetContent("")
	if m.TotalLines() != 0 || !m.AtTop() || !m.AtBottom() || row(m, 0) != "     " {
		t.Fatal("empty content")
	}
}
func TestScrollReflow(t *testing.T) {
	m := New("0123456789abcdefghijklmnopqrstuv")
	m.SetSize(5, 2)
	m.Scroll(2)
	if m.Offset() != 2 || row(m, 0)[:4] != "89ab" {
		t.Fatal("scroll")
	}
	m.SetSize(9, 2)
	if m.Offset() != 1 || row(m, 0)[:8] != "89abcdef" {
		t.Fatal("anchor", m.Offset(), row(m, 0))
	}
	m.Page(1)
	if !m.AtBottom() {
		t.Fatal("page")
	}
	m.Scroll(int(^uint(0) >> 1))
	if !m.AtBottom() {
		t.Fatal("positive overflow")
	}
	m.Scroll(-int(^uint(0)>>1) - 1)
	if !m.AtTop() {
		t.Fatal("negative overflow")
	}
	m.GotoBottom()
	m.SetSize(9, 20)
	if !m.AtTop() || !m.AtBottom() {
		t.Fatal("resize clamp")
	}
	m.SetContent("new")
	if m.Offset() != 0 {
		t.Fatal("replace reset")
	}
}
func TestScrollbarAndReuse(t *testing.T) {
	m := New("a\nb\nc\nd\ne\nf")
	m.SetSize(4, 3)
	if m.CellView().At(3, 0).Rune != '#' {
		t.Fatal("top thumb")
	}
	m.GotoBottom()
	if m.CellView().At(3, 2).Rune != '#' {
		t.Fatal("bottom thumb")
	}
	b := m.CellView()
	m.SetSize(4, 3)
	if m.CellView() != b {
		t.Fatal("buffer replaced")
	}
	if n := testing.AllocsPerRun(20, func() { m.Scroll(-1); m.Scroll(1); m.CellView() }); n != 0 {
		t.Fatal(n)
	}
	for _, sz := range [][2]int{{0, 0}, {0, 3}, {3, 0}, {1, 1}, {-1, -1}} {
		m.SetSize(sz[0], sz[1])
		m.GotoBottom()
		m.CellView()
		m.Scroll(-1)
	}
}

func TestResizePreservesPositionThroughCollapse(t *testing.T) {
	for _, size := range [][2]int{{0, 0}, {10, 0}, {0, 3}} {
		m := New("one\ntwo\nthree\nfour\nfive\nsix\nseven\neight")
		m.SetSize(10, 3)
		m.Scroll(2)
		m.SetSize(size[0], size[1])
		m.SetSize(0, 0)
		m.SetSize(10, 3)
		if m.Offset() != 2 {
			t.Fatalf("collapse %v lost offset: %d", size, m.Offset())
		}
		m.GotoBottom()
		m.SetSize(0, 0)
		m.SetSize(5, 2)
		if !m.AtBottom() {
			t.Fatal("collapsed tail lost bottom")
		}
	}
}

func TestResizeFollowsBottomButStartupStartsAtTop(t *testing.T) {
	m := New("abcdefghij\nklmnopqrst\nuvwxyz\nlast")
	m.SetSize(0, 0)
	m.SetSize(12, 2)
	if !m.AtTop() {
		t.Fatal("startup jumped to end")
	}
	m.GotoBottom()
	m.SetSize(5, 2)
	if !m.AtBottom() {
		t.Fatal("width reflow lost bottom")
	}
	m.SetSize(0, 0)
	m.SetContent("new\ndocument\nthird\nfourth")
	m.SetSize(5, 2)
	if !m.AtTop() {
		t.Fatal("new content inherited old tail")
	}
}
