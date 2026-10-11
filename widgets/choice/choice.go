// Package choice edits a named option, including two-option on/off settings.
package choice

import (
	"errors"
	"strconv"
	"unicode/utf8"

	bubbles "nimbleterminal.dev/tinybubbles"
)

type Config struct {
	Label   string
	Options []string
	Value   int
}
type Model struct {
	label          string
	options        []string
	value, applied int
	position       string
	buffer         *bubbles.Buffer
}

func New(config Config) (*Model, error) {
	if len(config.Options) == 0 {
		return nil, errors.New("choice: at least one option is required")
	}
	if config.Value < 0 || config.Value >= len(config.Options) {
		return nil, errors.New("choice: value is outside options")
	}
	m := &Model{label: config.Label, options: append([]string(nil), config.Options...), buffer: bubbles.NewBuffer(0, 0)}
	m.SetValue(config.Value)
	return m, nil
}
func NewToggle(label string, enabled bool) *Model {
	index := 0
	if enabled {
		index = 1
	}
	m, _ := New(Config{Label: label, Options: []string{"Off", "On"}, Value: index})
	return m
}
func (m *Model) Value() int    { return m.value }
func (m *Model) Applied() int  { return m.applied }
func (m *Model) Label() string { return m.options[m.value] }
func (m *Model) Len() int      { return len(m.options) }
func (m *Model) Dirty() bool   { return m.value != m.applied }
func (m *Model) SetValue(index int) bool {
	if index < 0 || index >= len(m.options) {
		return false
	}
	m.value = index
	m.applied = index
	m.refresh()
	return true
}
func (m *Model) refresh() {
	m.position = strconv.Itoa(m.value+1) + " / " + strconv.Itoa(len(m.options))
}
func (m *Model) Move(direction int) bool {
	if direction == 0 || len(m.options) < 2 {
		return false
	}
	step := 1
	if direction < 0 {
		step = -1
	}
	m.value = (m.value + step + len(m.options)) % len(m.options)
	m.refresh()
	return true
}
func (m *Model) Apply() int  { m.applied = m.value; return m.value }
func (m *Model) Cancel() int { m.value = m.applied; m.refresh(); return m.value }
func (m *Model) SetSize(w, h int) {
	w, h = max(0, w), max(0, h)
	oldW, oldH := m.buffer.Size()
	if w != oldW || h != oldH {
		m.buffer = bubbles.NewBuffer(w, h)
	}
}
func (m *Model) CellView() *bubbles.Buffer {
	b := m.buffer
	b.Clear(bubbles.Normal)
	w, h := b.Size()
	if w == 0 || h == 0 {
		return b
	}
	row := 0
	if h >= 3 {
		center(b, 0, m.label, bubbles.Accent)
		row = max(1, h/2-1)
	}
	center(b, row, m.Label(), bubbles.Selected)
	if h >= 5 {
		center(b, row+1, m.position, bubbles.Muted)
	}
	if h >= 4 {
		status := "Applied"
		if m.Dirty() {
			status = "Unsaved"
		}
		center(b, h-1, status, bubbles.Muted)
	}
	return b
}
func center(b *bubbles.Buffer, y int, text string, style bubbles.StyleID) {
	w, _ := b.Size()
	x := max(0, (w-utf8.RuneCountInString(text))/2)
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
