// Package list provides a scrolling selection list for small cell displays.
package list

import bubbles "nimbleterminal.dev/tinybubbles"

// Item identifies an application action. Disabled items remain visible.
type Item struct {
	ID, Label string
	Disabled  bool
}

// Model owns selection and scrolling, without depending on a runtime.
// Items are copied on SetItems. CellView reuses its buffer.
type Model struct {
	items         []Item
	index, offset int
	buffer        *bubbles.Buffer
}

func New(items []Item) *Model {
	m := &Model{index: -1, buffer: bubbles.NewBuffer(0, 0)}
	m.SetItems(items)
	return m
}

// SetItems preserves the selected ID when it is still enabled.
func (m *Model) SetItems(items []Item) {
	selected, hadSelection := m.Selected()
	m.items = append([]Item(nil), items...)
	m.index = -1
	for i, item := range m.items {
		if item.Disabled {
			continue
		}
		if m.index < 0 {
			m.index = i
		}
		if hadSelection && item.ID == selected.ID {
			m.index = i
			break
		}
	}
	m.reveal()
}

func (m *Model) SetSize(width, height int) {
	width, height = max(0, width), max(0, height)
	w, h := m.buffer.Size()
	if w != width || h != height {
		m.buffer = bubbles.NewBuffer(width, height)
	}
	m.reveal()
}

func (m *Model) SelectedIndex() int { return m.index }
func (m *Model) Len() int           { return len(m.items) }
func (m *Model) Selected() (Item, bool) {
	if m.index < 0 || m.index >= len(m.items) {
		return Item{}, false
	}
	return m.items[m.index], true
}

// Move advances one enabled item in the given direction and wraps at the ends.
// A three-button device can use only Move(1), Select, and Back.
func (m *Model) Move(direction int) bool {
	if direction == 0 || m.index < 0 {
		return false
	}
	step := 1
	if direction < 0 {
		step = -1
	}
	for n, i := 0, m.index; n < len(m.items); n++ {
		i = (i + step + len(m.items)) % len(m.items)
		if !m.items[i].Disabled {
			changed := i != m.index
			m.index = i
			m.reveal()
			return changed
		}
	}
	return false
}

func (m *Model) reveal() {
	_, height := m.buffer.Size()
	m.offset = min(max(0, m.offset), max(0, len(m.items)-height))
	if m.index < 0 || height == 0 {
		return
	}
	if m.index < m.offset {
		m.offset = m.index
	}
	if m.index >= m.offset+height {
		m.offset = m.index - height + 1
	}
}

// CellView draws one row per item. The rightmost column marks the scrollbar.
// Labels occupy one line and truncate with a dot; control characters are skipped.
func (m *Model) CellView() *bubbles.Buffer {
	b := m.buffer
	b.Clear(bubbles.Normal)
	width, height := b.Size()
	if width == 0 || height == 0 {
		return b
	}
	if len(m.items) == 0 {
		line(b, 0, 0, "(empty)", width, bubbles.Muted)
		return b
	}
	textWidth := width
	scroll := len(m.items) > height && width > 1
	if scroll {
		textWidth--
	}
	for y := 0; y < height && m.offset+y < len(m.items); y++ {
		i := m.offset + y
		style := bubbles.Normal
		if m.items[i].Disabled {
			style = bubbles.Muted
		}
		if i == m.index {
			style = bubbles.Selected
		}
		for x := 0; x < textWidth; x++ {
			b.Set(x, y, bubbles.Cell{Style: style})
		}
		marker := ' '
		if i == m.index {
			marker = '>'
		}
		if m.items[i].Disabled {
			marker = '-'
		}
		b.Set(0, y, bubbles.Cell{Rune: marker, Style: style})
		line(b, 1, y, m.items[i].Label, textWidth, style)
	}
	if scroll {
		for y := 0; y < height; y++ {
			b.Set(width-1, y, bubbles.Cell{Rune: '|', Style: bubbles.Muted})
		}
		thumb := m.offset * (height - 1) / (len(m.items) - height)
		b.Set(width-1, thumb, bubbles.Cell{Rune: '#', Style: bubbles.Accent})
	}
	return b
}

func line(b *bubbles.Buffer, x, y int, text string, width int, style bubbles.StyleID) {
	if x >= width {
		return
	}
	for _, r := range text {
		if r < 32 || r == 127 {
			continue
		}
		if x >= width {
			b.Set(width-1, y, bubbles.Cell{Rune: '.', Style: style})
			break
		}
		b.Set(x, y, bubbles.Cell{Rune: r, Style: style})
		x++
	}
}
