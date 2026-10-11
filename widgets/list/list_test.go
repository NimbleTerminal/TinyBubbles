package list

import (
	bubbles "nimbleterminal.dev/tinybubbles"
	"testing"
)

func items() []Item {
	return []Item{{ID: "a", Label: "Alpha"}, {ID: "b", Label: "Blocked", Disabled: true}, {ID: "c", Label: "Charlie"}, {ID: "d", Label: "Delta"}}
}
func TestNavigationAndScroll(t *testing.T) {
	m := New(items())
	m.SetSize(8, 2)
	m.Move(1)
	if m.SelectedIndex() != 2 || m.offset != 1 {
		t.Fatalf("skip/scroll: index=%d offset=%d", m.index, m.offset)
	}
	view := m.CellView()
	if view.At(0, 0).Rune != '-' || view.At(0, 1).Style != bubbles.Selected {
		t.Fatal("disabled and selected rows are not distinguished")
	}
	m.Move(1)
	m.Move(1)
	if m.SelectedIndex() != 0 || m.offset != 0 {
		t.Fatal("forward wrap did not reveal first row")
	}
	m.Move(-1)
	if m.SelectedIndex() != 3 || m.offset != 2 {
		t.Fatal("reverse wrap did not reveal last row")
	}
	m.SetSize(8, 1)
	if m.offset != 3 {
		t.Fatal("shrinking hid selection")
	}
	m.SetSize(8, 8)
	if m.offset != 0 {
		t.Fatal("growing left unnecessary scroll offset")
	}
}
func TestItemsAndEmptyStates(t *testing.T) {
	original := items()
	m := New(original)
	original[0].Label = "Changed"
	item, _ := m.Selected()
	if item.Label != "Alpha" {
		t.Fatal("items were not copied")
	}
	m.Move(1)
	m.SetItems([]Item{{ID: "c", Label: "Charlie"}, {ID: "a", Label: "Alpha"}})
	if m.SelectedIndex() != 0 {
		t.Fatal("selection ID was not preserved")
	}
	m.SetItems([]Item{{ID: "c", Disabled: true}, {ID: "a", Label: "Alpha"}})
	item, _ = m.Selected()
	if item.ID != "a" {
		t.Fatal("disabled selection was preserved")
	}
	m.SetItems([]Item{{ID: "a", Disabled: true}})
	if _, ok := m.Selected(); ok || m.Move(1) {
		t.Fatal("disabled-only list has a selection")
	}
	m.SetItems(nil)
	m.SetSize(10, 2)
	if m.SelectedIndex() != -1 || m.CellView().At(0, 0).Rune != '(' {
		t.Fatal("empty list")
	}
}
func TestClippingAndReuse(t *testing.T) {
	m := New([]Item{{Label: "AB\nCDEFG"}, {Label: "Second"}})
	m.SetSize(5, 2)
	view := m.CellView()
	if view.At(4, 0).Rune != '.' || view.At(1, 1).Rune != 'S' {
		t.Fatal("label leaked to another row or did not truncate")
	}
	if allocs := testing.AllocsPerRun(100, func() { m.CellView() }); allocs != 0 {
		t.Fatalf("view allocated %v times", allocs)
	}
	m.SetSize(5, 2)
	if m.CellView() != view {
		t.Fatal("same size allocated another grid")
	}
	for _, size := range [][2]int{{1, 1}, {0, 2}, {2, 0}, {-1, -1}} {
		m.SetSize(size[0], size[1])
		m.Move(1)
		m.CellView()
	}
}
