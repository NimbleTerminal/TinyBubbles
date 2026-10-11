package status

import (
	bubbles "nimbleterminal.dev/tinybubbles"
	"testing"
)

func TestStatesAndClearing(t *testing.T) {
	m := New(Config{Label: "Wi-Fi", State: Ready, Detail: "Connected"})
	m.SetSize(21, 2)
	for i, symbol := range []rune{'-', '+', '~', '!', 'x'} {
		m.SetStatus(State(i), "detail")
		b := m.CellView()
		if b.At(0, 0).Rune != symbol || m.State() != State(i) {
			t.Fatal("state", i)
		}
		if m.State().String() == "" {
			t.Fatal("empty state text")
		}
	}
	m.SetStatus(Error, "Long error detail")
	m.CellView()
	m.SetStatus(Ready, "")
	b := m.CellView()
	for x := 0; x < 21; x++ {
		if b.At(x, 1).Rune != 0 {
			t.Fatal("stale detail")
		}
	}
	m.SetStatus(State(99), "invalid")
	if m.State() != Offline || State(99).String() != "Offline" {
		t.Fatal("invalid state")
	}
	m.Styles[Offline] = bubbles.Selected
	if m.CellView().At(0, 0).Style != bubbles.Selected {
		t.Fatal("custom style")
	}
}
func TestSmallLayoutsAndReuse(t *testing.T) {
	m := New(Config{Label: "Wi\nFi", State: Warning, Detail: "a\nb"})
	for _, sz := range [][2]int{{0, 0}, {1, 1}, {3, 1}, {21, 2}, {40, 16}} {
		m.SetSize(sz[0], sz[1])
		b := m.CellView()
		m.SetSize(sz[0], sz[1])
		if b != m.CellView() {
			t.Fatal("new buffer")
		}
		if n := testing.AllocsPerRun(10, func() { m.CellView() }); n != 0 {
			t.Fatal(n)
		}
	}
	m.SetSize(1, 1)
	if m.CellView().At(0, 0).Rune != '!' {
		t.Fatal("lost symbol")
	}
	m.SetSize(3, 1)
	if m.CellView().At(2, 0).Rune != '.' {
		t.Fatal("no clipping marker")
	}
	m.SetSize(21, 3)
	b := m.CellView()
	if b.At(1, 1).Rune != 'b' || b.At(0, 2).Rune != 0 {
		t.Fatal("control characters escaped row")
	}
}
