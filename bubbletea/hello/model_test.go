package hello

import (
	"reflect"
	"testing"

	tea "charm.land/bubbletea/v2"
	bubbles "nimbleterminal.dev/tinybubbles"
)

func cells(m *Model) []bubbles.Cell {
	b := m.CellView()
	w, h := b.Size()
	result := make([]bubbles.Cell, 0, w*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			result = append(result, b.At(x, y))
		}
	}
	return result
}

func TestDisabledColorBindingDoesNotChangeDisplay(t *testing.T) {
	m := New()
	m.Update(tea.WindowSizeMsg{Width: 40, Height: 16})
	before := cells(m)
	m.SetColorEnabled(false)
	m.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	if !reflect.DeepEqual(before, cells(m)) {
		t.Fatal("disabled key changed the palette")
	}
	m.SetColorEnabled(true)
	m.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	if reflect.DeepEqual(before, cells(m)) {
		t.Fatal("enabled key did not change the palette")
	}
}

func TestPauseHelpFollowsState(t *testing.T) {
	m := New()
	for _, want := range []string{"Resume", "Pause"} {
		m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
		if got := m.Keys().Pause.Help().Desc; got != want {
			t.Fatalf("got %q, want %q", got, want)
		}
	}
}
