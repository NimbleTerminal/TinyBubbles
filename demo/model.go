// Package demo contains the same counter model for every device.
package demo

import (
	bubbles "nimbleterminal.dev/tinybubbles"
	"strconv"
)

type Model struct {
	count int
	view  *bubbles.Buffer
}

func New() *Model                  { return &Model{} }
func (m *Model) Init() bubbles.Cmd { return nil }
func (m *Model) Update(msg bubbles.Msg) (bubbles.Model, bubbles.Cmd) {
	switch msg := msg.(type) {
	case bubbles.WindowSizeMsg:
		m.view = bubbles.NewBuffer(msg.Width, msg.Height)
	case bubbles.KeyMsg:
		switch msg.Key {
		case bubbles.KeyUp, bubbles.KeyRight, bubbles.KeySelect:
			m.count++
		case bubbles.KeyDown, bubbles.KeyLeft:
			m.count--
		case bubbles.KeyBack:
			m.count = 0
		case bubbles.KeyQuit:
			return m, bubbles.Quit
		}
	}
	return m, nil
}
func (m *Model) View() *bubbles.Buffer {
	if m.view == nil {
		return nil
	}
	w, h := m.view.Size()
	m.view.Clear(bubbles.Normal)
	if w < 16 || h < 6 {
		m.view.Text(0, 0, "Count", bubbles.Accent)
		m.view.Text(0, 1, strconv.Itoa(m.count), bubbles.Selected)
		return m.view
	}
	x := 0
	if w >= 32 {
		x = (w - 24) / 2
	}
	m.view.Text(x, 0, "TinyBubbles", bubbles.Accent)
	m.view.Text(x, 2, "Count: "+strconv.Itoa(m.count), bubbles.Selected)
	m.view.Text(x, h-3, "Left/A: -", bubbles.Muted)
	m.view.Text(x, h-2, "Right/B: +", bubbles.Muted)
	if h >= 12 {
		m.view.Text(x, 5, "One model. Any display.", bubbles.Normal)
	}
	return m.view
}
