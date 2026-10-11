package demo

import (
	bubbles "github.com/NimbleTerminal/TinyBubbles"
	"testing"
)

func TestDemoAdaptsAndHandlesKeys(t *testing.T) {
	for _, size := range [][2]int{{21, 8}, {40, 16}, {40, 30}, {53, 30}, {8, 3}} {
		m := New()
		m.Update(bubbles.WindowSizeMsg{Width: size[0], Height: size[1]})
		m.Update(bubbles.KeyMsg{Key: bubbles.KeySelect})
		m.Update(bubbles.KeyMsg{Key: bubbles.KeyRight})
		m.Update(bubbles.KeyMsg{Key: bubbles.KeyLeft})
		if m.count != 1 {
			t.Fatal(m.count)
		}
		if w, h := m.View().Size(); w != size[0] || h != size[1] {
			t.Fatal("grid size")
		}
		m.Update(bubbles.KeyMsg{Key: bubbles.KeyBack})
		if m.count != 0 {
			t.Fatal("reset")
		}
		_, cmd := m.Update(bubbles.KeyMsg{Key: bubbles.KeyQuit})
		if _, ok := cmd().(bubbles.QuitMsg); !ok {
			t.Fatal("quit")
		}
	}
}
