package vt

import (
	"strings"
	"testing"
)

func open(t *testing.T, cols, rows int) *Terminal {
	t.Helper()
	term, err := New(cols, rows)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(term.Close)
	return term
}

func screen(t *testing.T, term *Terminal) []string {
	t.Helper()
	if _, err := term.Update(); err != nil {
		t.Fatal(err)
	}
	cols, rows := term.Size()
	lines := make([]string, rows)
	cells := make([]Cell, cols)
	for {
		y, n, ok := term.NextRow(cells, false)
		if !ok {
			break
		}
		var b strings.Builder
		for _, c := range cells[:n] {
			if c.Flags&Spacer != 0 {
				continue
			}
			if c.Rune == 0 {
				b.WriteByte(' ')
			} else {
				b.WriteRune(c.Rune)
			}
		}
		lines[y] = strings.TrimRight(b.String(), " ")
	}
	return lines
}

func TestWriteAndStyle(t *testing.T) {
	term := open(t, 10, 3)
	term.WriteString("hi \x1b[1;31mred\x1b[0m\r\n\x1b[7mrev")
	lines := screen(t, term)
	if lines[0] != "hi red" || lines[1] != "rev" {
		t.Fatalf("screen %q", lines)
	}
	term.Update()
	cells := make([]Cell, 10)
	term.NextRow(cells, false)
	r := cells[3]
	if r.Rune != 'r' || r.Flags&(Bold|FG) != Bold|FG || r.FG == 0 {
		t.Fatalf("red cell %+v", r)
	}
	if cells[0].Flags != 0 {
		t.Fatalf("plain cell %+v", cells[0])
	}
	term.NextRow(cells, false)
	if cells[0].Flags&Inverse == 0 {
		t.Fatalf("inverse cell %+v", cells[0])
	}
}

func TestCursorAndModes(t *testing.T) {
	term := open(t, 10, 4)
	term.WriteString("\x1b[3;5H\x1b[?1h\x1b[?2004h")
	term.Update()
	c := term.Cursor()
	if !c.InViewport || c.X != 4 || c.Y != 2 || !c.Visible {
		t.Fatalf("cursor %+v", c)
	}
	if !term.DECMode(ModeCursorKeys) || !term.DECMode(ModeBracketedPaste) {
		t.Fatal("modes")
	}
	term.WriteString("\x1b[?25l")
	term.Update()
	if term.Cursor().Visible {
		t.Fatal("hidden cursor")
	}
}

func TestDirtyRows(t *testing.T) {
	term := open(t, 8, 4)
	if d, _ := term.Update(); d == Clean {
		t.Fatal("first frame clean")
	}
	term.MarkClean()
	if d, _ := term.Update(); d != Clean {
		t.Fatalf("unchanged frame %v", d)
	}
	term.MarkClean()
	term.WriteString("\x1b[3;1Hx")
	term.Update()
	cells := make([]Cell, 8)
	var rows []int
	for {
		y, _, ok := term.NextRow(cells, true)
		if !ok {
			break
		}
		rows = append(rows, y)
	}
	// The cursor leaves row 0, so that row is redrawn too.
	if len(rows) != 2 || rows[0] != 0 || rows[1] != 2 {
		t.Fatalf("dirty rows %v", rows)
	}
}

func TestWideAndResize(t *testing.T) {
	term := open(t, 6, 2)
	term.WriteString("a界b")
	term.Update()
	cells := make([]Cell, 6)
	term.NextRow(cells, false)
	if cells[1].Rune != '界' || cells[1].Flags&Wide == 0 || cells[2].Flags&Spacer == 0 || cells[3].Rune != 'b' {
		t.Fatalf("wide %+v", cells[:4])
	}
	if err := term.Resize(3, 2); err != nil {
		t.Fatal(err)
	}
	if c, r := term.Size(); c != 3 || r != 2 {
		t.Fatal("size")
	}
	// Short destination slices clip the row.
	term.Update()
	if _, n, _ := term.NextRow(cells[:2], false); n != 2 {
		t.Fatalf("clipped %d", n)
	}
}

func TestReply(t *testing.T) {
	term := open(t, 10, 4)
	term.WriteString("\x1b[2;3H\x1b[6n")
	buf := make([]byte, 32)
	n := term.ReadReply(buf)
	if string(buf[:n]) != "\x1b[2;3R" {
		t.Fatalf("reply %q", buf[:n])
	}
	if term.ReadReply(buf) != 0 {
		t.Fatal("drained")
	}
}

func TestClosed(t *testing.T) {
	term, _ := New(2, 2)
	term.Close()
	term.Close()
	if _, err := term.WriteString("x"); err != ErrClosed {
		t.Fatal(err)
	}
}

func TestHeapUsage(t *testing.T) {
	before, _ := HeapUsage()
	term, err := New(40, 16)
	if err != nil {
		t.Fatal(err)
	}
	term.Update()
	during, peak := HeapUsage()
	scroll := func() int {
		for i := 0; i < 2000; i++ {
			term.WriteString("\x1b[3" + string(rune('0'+i%8)) + "mscrolling output line\r\n")
		}
		term.Update()
		n, _ := HeapUsage()
		return n
	}
	// New styles may grow the page once; without scrollback, history does not.
	if a, b := scroll(), scroll(); a != b || a > 2*during {
		t.Fatalf("scrollback grew heap %d -> %d -> %d", during, a, b)
	}
	term.Close()
	after, _ := HeapUsage()
	t.Logf("40x16 terminal: %d bytes held, peak %d", during-before, peak)
	if during <= before || after != before {
		t.Fatalf("heap before=%d during=%d after=%d", before, during, after)
	}
}
