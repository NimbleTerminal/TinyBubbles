// Package choice edits a named option, including two-option on/off settings.
package choice

import (
	"errors"
	"nimbleterminal.dev/tinybubbles/layout"
	"strconv"

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
	m.buffer.Resize(w, h)
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
		layout.ClipText(b, layout.Rect{Y: 0, Width: w, Height: 1}, m.label, bubbles.Accent, layout.Center)
		row = max(1, h/2-1)
	}
	layout.ClipText(b, layout.Rect{Y: row, Width: w, Height: 1}, m.Label(), bubbles.Selected, layout.Center)
	if h >= 5 {
		layout.ClipText(b, layout.Rect{Y: row + 1, Width: w, Height: 1}, m.position, bubbles.Muted, layout.Center)
	}
	if h >= 4 {
		status := "Applied"
		if m.Dirty() {
			status = "Unsaved"
		}
		layout.ClipText(b, layout.Rect{Y: h - 1, Width: w, Height: 1}, status, bubbles.Muted, layout.Center)
	}
	return b
}

// AppliedMsg captures the result emitted by runtime adapters.
type AppliedMsg struct {
	Index int
	Label string
}

// CancelledMsg captures the result emitted by runtime adapters.
type CancelledMsg struct {
	Index int
	Label string
}
