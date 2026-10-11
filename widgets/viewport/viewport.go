// Package viewport displays vertically scrollable plain text on a cell grid.
package viewport

import (
	bubbles "nimbleterminal.dev/tinybubbles"
	"strings"
	"unicode/utf8"
)

type line struct{ start, end int }

// Model caches line boundaries when content or size changes. It owns no input or timers.
type Model struct {
	text          string
	lines         []line
	offset        int
	anchor        int
	bottom        bool
	width, height int
	buffer        *bubbles.Buffer
}

func New(content string) *Model {
	m := &Model{buffer: bubbles.NewBuffer(0, 0)}
	m.SetContent(content)
	return m
}

// SetContent replaces the document and returns to the top. Tabs become four spaces,
// CRLF becomes newline, and other control characters are removed. ANSI is not parsed.
func (m *Model) SetContent(content string) {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	var b strings.Builder
	for _, r := range content {
		switch {
		case r == '\t':
			b.WriteString("    ")
		case r == '\n' || r >= 32 && r != 127:
			b.WriteRune(r)
		}
	}
	m.text = b.String()
	m.offset, m.anchor, m.bottom = 0, 0, false
	m.layout(0)
}
func (m *Model) Offset() int     { return m.offset }
func (m *Model) TotalLines() int { return len(m.lines) }
func (m *Model) limit() int {
	if m.width <= 0 || m.height <= 0 {
		return 0
	}
	return max(0, len(m.lines)-m.height)
}
func (m *Model) AtTop() bool    { return m.offset == 0 }
func (m *Model) AtBottom() bool { return m.offset >= m.limit() }

// Scroll moves visual rows, clamps at both ends, and never wraps.
func (m *Model) Scroll(delta int) bool {
	if m.width == 0 || m.height == 0 {
		return false
	}
	old := m.offset
	if delta > 0 {
		m.offset += min(delta, m.limit()-m.offset)
	} else if delta < 0 {
		m.offset += max(delta, -m.offset)
	}
	return old != m.offset
}
func (m *Model) Page(direction int) bool {
	if direction < 0 {
		return m.Scroll(-max(1, m.height-1))
	}
	if direction > 0 {
		return m.Scroll(max(1, m.height-1))
	}
	return false
}
func (m *Model) GotoTop()    { m.offset, m.anchor, m.bottom = 0, 0, false }
func (m *Model) GotoBottom() { m.offset = m.limit(); m.bottom = true }

// SetSize uses cells. Reflow keeps the old top line's first character visible,
// then clamps to the final page. A column is reserved for the scrollbar at widths >=2.
func (m *Model) SetSize(w, h int) {
	oldW, oldH := m.buffer.Size()
	if !m.buffer.Resize(w, h) {
		return
	}
	if oldW > 0 && oldH > 0 {
		m.bottom = m.AtBottom()
		m.anchor = 0
		if len(m.lines) > 0 {
			m.anchor = m.lines[m.offset].start
		}
	}
	m.width, m.height = m.buffer.Size()
	m.layout(m.anchor)
	if m.width > 0 && m.height > 0 && m.bottom {
		m.GotoBottom()
	}
}

func (m *Model) layout(anchor int) {
	m.lines = m.lines[:0]
	m.offset = 0
	width := m.width
	if width >= 2 {
		width--
	}
	if width <= 0 || m.text == "" {
		return
	}
	start, col := 0, 0
	for i, r := range m.text {
		if r == '\n' {
			m.lines = append(m.lines, line{start, i})
			start = i + 1
			col = 0
			continue
		}
		if col == width {
			m.lines = append(m.lines, line{start, i})
			start = i
			col = 0
		}
		col++
	}
	m.lines = append(m.lines, line{start, len(m.text)})
	for i, l := range m.lines {
		if l.start > anchor {
			break
		}
		m.offset = i
	}
	m.offset = min(m.offset, m.limit())
}

// CellView reuses the buffer; scrolling and drawing allocate nothing.
func (m *Model) CellView() *bubbles.Buffer {
	b := m.buffer
	b.Clear(bubbles.Normal)
	w, h := b.Size()
	for y := 0; y < h && m.offset+y < len(m.lines); y++ {
		l := m.lines[m.offset+y]
		x := 0
		for pos := l.start; pos < l.end; {
			r, n := utf8.DecodeRuneInString(m.text[pos:l.end])
			b.Set(x, y, bubbles.Cell{Rune: r})
			pos += n
			x++
		}
	}
	if w >= 2 && m.limit() > 0 {
		thumb := max(1, h*h/len(m.lines))
		top := int(int64(m.offset) * int64(h-thumb) / int64(m.limit()))
		for y := 0; y < h; y++ {
			r := '|'
			style := bubbles.Muted
			if y >= top && y < top+thumb {
				r = '#'
				style = bubbles.Accent
			}
			b.Set(w-1, y, bubbles.Cell{Rune: r, Style: style})
		}
	}
	return b
}
