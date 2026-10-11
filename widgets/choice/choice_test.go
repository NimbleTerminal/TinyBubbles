package choice

import "testing"

func TestChoices(t *testing.T) {
	for _, c := range []Config{{}, {Options: []string{"a"}, Value: -1}, {Options: []string{"a"}, Value: 1}} {
		if _, err := New(c); err == nil {
			t.Fatal("accepted invalid config")
		}
	}
	options := []string{"Single", "Repeat", "Shuffle"}
	m, _ := New(Config{Options: options})
	options[0] = "changed"
	if m.Label() != "Single" {
		t.Fatal("options not copied")
	}
	m.Move(-1)
	if m.Value() != 2 || !m.Dirty() {
		t.Fatal("previous did not wrap")
	}
	m.Cancel()
	if m.Value() != 0 || m.Dirty() {
		t.Fatal("cancel")
	}
	m.Move(1)
	m.Apply()
	m.Move(1)
	if m.Cancel() != 1 {
		t.Fatal("apply")
	}
	if m.SetValue(9) || m.Value() != 1 {
		t.Fatal("invalid set changed value")
	}
	m.SetValue(2)
	m.Move(1)
	if m.Value() != 0 {
		t.Fatal("next did not wrap")
	}
	toggle := NewToggle("Sound", true)
	if toggle.Label() != "On" {
		t.Fatal("toggle")
	}
	toggle.Move(1)
	if toggle.Label() != "Off" {
		t.Fatal("toggle wrap")
	}
	one, _ := New(Config{Options: []string{"only"}})
	if one.Move(1) || one.Move(0) {
		t.Fatal("single choice moved")
	}
}
func TestViewReuse(t *testing.T) {
	m := NewToggle("Sound", true)
	for _, size := range [][2]int{{0, 0}, {1, 1}, {3, 2}, {21, 8}, {40, 16}} {
		m.SetSize(size[0], size[1])
		b := m.CellView()
		m.SetSize(size[0], size[1])
		if b != m.CellView() {
			t.Fatal("buffer replaced")
		}
		if n := testing.AllocsPerRun(10, func() { m.CellView() }); n != 0 {
			t.Fatalf("view allocations: %v", n)
		}
	}
}
