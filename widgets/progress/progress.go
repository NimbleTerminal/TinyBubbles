// Package progress displays completed work or activity when the total is unknown.
package progress

import (
	bubbles "nimbleterminal.dev/tinybubbles"
	"strconv"
)

// Config supplies the label and initial work counts. Total zero means unknown.
type Config struct {
	Label       string
	Done, Total uint32
}

// UpdateMsg replaces the work counts. Runtime adapters accept this message.
type UpdateMsg struct{ Done, Total uint32 }

// PulseMsg advances the unknown-total indicator by one cell.
// The parent decides when to send it; the widget starts no timers.
type PulseMsg struct{}

type Model struct {
	label, text string
	done, total uint32
	phase       int
	buffer      *bubbles.Buffer
}

func New(config Config) *Model {
	m := &Model{label: config.Label, buffer: bubbles.NewBuffer(0, 0)}
	m.SetProgress(config.Done, config.Total)
	return m
}
func (m *Model) Done() uint32        { return m.done }
func (m *Model) Total() uint32       { return m.total }
func (m *Model) Indeterminate() bool { return m.total == 0 }
func (m *Model) Complete() bool      { return m.total > 0 && m.done == m.total }

// SetProgress clamps Done to Total. Unknown totals discard the completed count.
func (m *Model) SetProgress(done, total uint32) {
	if total == 0 {
		done = 0
	}
	m.done = min(done, total)
	m.total = total
	m.phase = 0
	m.text = "Working..."
	if total > 0 {
		m.text = strconv.FormatUint(uint64(m.done)*100/uint64(total), 10) + "%"
	}
}
func (m *Model) Advance() {
	if m.Indeterminate() {
		w, _ := m.buffer.Size()
		m.phase = (m.phase + 1) % max(1, innerWidth(w))
	}
}
func (m *Model) SetSize(w, h int) {
	w, h = max(0, w), max(0, h)
	ow, oh := m.buffer.Size()
	if w != ow || h != oh {
		m.buffer = bubbles.NewBuffer(w, h)
		m.phase = 0
	}
}
func innerWidth(w int) int {
	if w >= 3 {
		return w - 2
	}
	return w
}

// CellView draws without allocation. One-row layouts show only the bar.
func (m *Model) CellView() *bubbles.Buffer {
	b := m.buffer
	b.Clear(bubbles.Normal)
	w, h := b.Size()
	if w == 0 || h == 0 {
		return b
	}
	row := 0
	if h >= 3 {
		write(b, 0, m.label, bubbles.Accent)
		row = h / 2
	}
	n, start := innerWidth(w), 0
	if w >= 3 {
		start = 1
		b.Set(0, row, bubbles.Cell{Rune: '[', Style: bubbles.Muted})
		b.Set(w-1, row, bubbles.Cell{Rune: ']', Style: bubbles.Muted})
	}
	filled := 0
	if m.total > 0 {
		filled = int(uint64(m.done) * uint64(n) / uint64(m.total))
	}
	for x := 0; x < n; x++ {
		r, style := '-', bubbles.Muted
		active := x < filled
		if m.Indeterminate() {
			active = (x-m.phase+n)%n < max(1, n/4)
		}
		if active {
			r, style = '=', bubbles.Accent
		}
		b.Set(start+x, row, bubbles.Cell{Rune: r, Style: style})
	}
	if h >= 2 {
		write(b, row+1, m.text, bubbles.Normal)
	}
	return b
}
func write(b *bubbles.Buffer, y int, text string, style bubbles.StyleID) {
	w, _ := b.Size()
	x := 0
	for _, r := range text {
		if r < 32 || r == 127 {
			continue
		}
		if x >= w {
			b.Set(w-1, y, bubbles.Cell{Rune: '.', Style: style})
			break
		}
		b.Set(x, y, bubbles.Cell{Rune: r, Style: style})
		x++
	}
}
