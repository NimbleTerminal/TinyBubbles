package value

import (
	"math"
	"testing"
)

func mustNew(t *testing.T, c Config) *Model {
	t.Helper()
	m, err := New(c)
	if err != nil {
		t.Fatal(err)
	}
	return m
}
func TestDraftApplyCancel(t *testing.T) {
	m := mustNew(t, Config{Min: 0, Max: 10, Step: 3, Value: 5})
	m.Adjust(1)
	if m.Value() != 8 || m.Applied() != 5 || !m.Dirty() {
		t.Fatal("draft changed applied value")
	}
	if m.Cancel() != 5 || m.Dirty() {
		t.Fatal("cancel did not restore applied value")
	}
	m.Adjust(-1)
	if m.Apply() != 2 || m.Dirty() {
		t.Fatal("apply did not save draft")
	}
	m.Adjust(-1)
	if m.Value() != 0 || m.CanDecrease() || m.Adjust(-1) {
		t.Fatal("lower bound")
	}
	if m.Cancel() != 2 {
		t.Fatal("cancel forgot last apply")
	}
	m.SetValue(99)
	if m.Value() != 10 || m.Applied() != 10 || m.Dirty() || m.CanIncrease() {
		t.Fatal("external value was not clamped/reset")
	}
	m.SetValue(-99)
	if m.Value() != 0 {
		t.Fatal("external lower bound")
	}
}
func TestRangeAndOverflow(t *testing.T) {
	for _, c := range []Config{{Min: 1, Max: 0, Step: 1}, {Step: 0}, {Step: -1}} {
		if _, err := New(c); err == nil {
			t.Fatal("invalid configuration accepted")
		}
	}
	m := mustNew(t, Config{Min: math.MinInt32, Max: math.MaxInt32, Step: math.MaxInt32, Value: math.MaxInt32 - 1})
	m.Adjust(1)
	if m.Value() != math.MaxInt32 {
		t.Fatal("positive overflow")
	}
	m.SetValue(math.MinInt32 + 1)
	m.Adjust(-1)
	if m.Value() != math.MinInt32 {
		t.Fatal("negative overflow")
	}
	if m.Adjust(0) {
		t.Fatal("zero direction changed value")
	}
	fixed := mustNew(t, Config{Min: 7, Max: 7, Step: 3})
	if fixed.Value() != 7 || fixed.Adjust(1) || fixed.Adjust(-1) {
		t.Fatal("fixed range")
	}
	fixed.SetSize(21, 8)
	fixed.CellView()
}
func TestLayoutAndReuse(t *testing.T) {
	m := mustNew(t, Config{Label: "Level\n", Unit: "%\r", Min: 0, Max: 100, Step: 5, Value: 50})
	m.SetSize(10, 6)
	b := m.CellView()
	if b.At(3, 2).Rune != '5' || b.At(4, 2).Rune != '0' {
		t.Fatal("value not centered")
	}
	if b.At(1, 3).Rune != '=' || b.At(5, 3).Rune != '-' {
		t.Fatal("incorrect meter")
	}
	if allocs := testing.AllocsPerRun(100, func() { m.CellView() }); allocs != 0 {
		t.Fatalf("view allocates: %v", allocs)
	}
	m.SetSize(10, 6)
	if m.CellView() != b {
		t.Fatal("same size replaced buffer")
	}
	for _, size := range [][2]int{{0, 0}, {-1, -1}, {1, 1}, {2, 2}, {3, 3}, {4, 4}, {5, 5}} {
		m.SetSize(size[0], size[1])
		m.CellView()
	}
	m.SetSize(3, 1)
	if m.CellView().At(0, 0).Rune != '5' {
		t.Fatal("compact layout lost value")
	}
}
