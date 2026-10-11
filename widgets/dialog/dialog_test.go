package dialog

import "testing"

func TestDefaultAndSingleResult(t *testing.T) {
	m := New(Config{})
	if m.ConfirmSelected() {
		t.Fatal("confirmation is selected by default")
	}
	result, ok := m.Resolve()
	if !ok || result.Confirmed || !m.Done() {
		t.Fatal("default selection did not cancel")
	}
	if _, ok = m.Resolve(); ok || m.Move(1) {
		t.Fatal("resolved dialog accepted another decision")
	}
	m.Reset()
	m.Move(1)
	result, ok = m.Resolve()
	if !ok || !result.Confirmed {
		t.Fatal("explicit confirmation failed")
	}
	m.Reset()
	if m.ConfirmSelected() || m.Done() {
		t.Fatal("reset did not restore default")
	}
	m.Move(-1)
	result, ok = m.Cancel()
	if !ok || result.Confirmed || m.ConfirmSelected() {
		t.Fatal("cancel did not override highlighted confirmation")
	}
	if m.Move(0) {
		t.Fatal("zero direction moved")
	}
}
func TestWrappingAndClipping(t *testing.T) {
	got := wrap("Restore default volume and brightness?", 12, 3)
	want := []string{"Restore", "default", "volume and."}
	if len(got) != len(want) {
		t.Fatalf("lines: %q", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("wrap: %q", got)
		}
	}
	got = wrap("abcdefgh éééé", 3, 5)
	want = []string{"abc", "def", "gh", "ééé", "é"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("long-word wrap: %q", got)
		}
	}
	m := New(Config{Title: "Reset?", Body: "Restore the default volume and brightness?"})
	m.SetSize(21, 8)
	b := m.CellView()
	if b.At(0, 6).Rune != '>' || b.At(2, 6).Rune != 'C' || b.At(2, 7).Rune != 'C' {
		t.Fatal("choices missing from OLED layout")
	}
	m.Move(1)
	if m.CellView().At(0, 7).Rune != '>' {
		t.Fatal("highlight did not move")
	}
	for _, size := range [][2]int{{0, 0}, {1, 1}, {2, 2}, {3, 3}, {4, 4}} {
		m.SetSize(size[0], size[1])
		m.CellView()
	}
}
func TestViewReuseAndNoStateChanges(t *testing.T) {
	m := New(Config{Confirm: "An unusually long confirmation label for clipping", Body: "hello world"})
	m.SetSize(21, 8)
	m.Move(1)
	b := m.CellView()
	if allocs := testing.AllocsPerRun(100, func() { m.CellView() }); allocs != 0 {
		t.Fatalf("view allocates %v", allocs)
	}
	m.SetSize(21, 8)
	if b != m.CellView() || !m.ConfirmSelected() || m.Done() {
		t.Fatal("view or same size changed state")
	}
}
