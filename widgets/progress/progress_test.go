package progress

import "testing"

func row(m *Model, y int) string {
	b := m.CellView()
	w, _ := b.Size()
	s := make([]rune, w)
	for x := range s {
		s[x] = b.At(x, y).Rune
		if s[x] == 0 {
			s[x] = ' '
		}
	}
	return string(s)
}
func TestCounts(t *testing.T) {
	m := New(Config{Total: 100})
	m.SetSize(12, 1)
	if got := row(m, 0); got != "[----------]" {
		t.Fatal(got)
	}
	m.SetProgress(50, 100)
	if got := row(m, 0); got != "[=====-----]" {
		t.Fatal(got)
	}
	m.SetProgress(200, 100)
	if !m.Complete() || m.Done() != 100 || row(m, 0) != "[==========]" {
		t.Fatal("clamp or full bar")
	}
	m.SetProgress(1<<31, ^uint32(0))
	if row(m, 0) != "[=====-----]" {
		t.Fatal("overflow")
	}
	m.SetProgress(99, 0)
	if m.Complete() || !m.Indeterminate() || m.Done() != 0 {
		t.Fatal("unknown total")
	}
	first := row(m, 0)
	m.Advance()
	if row(m, 0) == first {
		t.Fatal("no activity")
	}
	for i := 0; i < 9; i++ {
		m.Advance()
	}
	if row(m, 0) != first {
		t.Fatal("activity did not wrap")
	}
	m.SetProgress(25, 100)
	first = row(m, 0)
	m.Advance()
	if row(m, 0) != first {
		t.Fatal("pulse changed determinate bar")
	}
}
func TestLayouts(t *testing.T) {
	m := New(Config{Label: "Load\nfiles", Done: 1, Total: 2})
	for _, size := range [][2]int{{0, 0}, {1, 1}, {2, 2}, {3, 3}, {21, 8}, {40, 16}} {
		m.SetSize(size[0], size[1])
		b := m.CellView()
		m.SetSize(size[0], size[1])
		if m.CellView() != b {
			t.Fatal("buffer replaced")
		}
		if n := testing.AllocsPerRun(10, func() { m.CellView() }); n != 0 {
			t.Fatal(n)
		}
	}
	m.SetSize(12, 3)
	if row(m, 0) != "Loadfiles   " || row(m, 2) != "50%         " {
		t.Fatal("label or percent", row(m, 0), row(m, 2))
	}
	m.SetProgress(0, 0)
	m.SetSize(1, 1)
	for i := 0; i < 10; i++ {
		m.Advance()
		m.CellView()
	}
}

func TestNarrowIndeterminateKeepsMoving(t *testing.T) {
	for _, w := range []int{1, 2, 3, 4, 5} {
		m := New(Config{})
		m.SetSize(w, 1)
		before := make([]rune, w)
		b := m.CellView()
		for x := range before {
			before[x] = b.At(x, 0).Rune
		}
		m.Advance()
		b = m.CellView()
		changed := false
		for x := range before {
			if b.At(x, 0).Rune != before[x] {
				changed = true
			}
		}
		if !changed {
			t.Fatalf("width %d did not animate", w)
		}
		if m.Complete() {
			t.Fatal("indeterminate marked complete")
		}
	}
}
