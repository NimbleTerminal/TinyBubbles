// Package status displays a compact, text-labelled device state.
package status

import (
	bubbles "nimbleterminal.dev/tinybubbles"
	"nimbleterminal.dev/tinybubbles/layout"
)

type State uint8

const (
	Offline State = iota
	Ready
	Busy
	Warning
	Error
)

func (s State) String() string {
	switch s {
	case Ready:
		return "Ready"
	case Busy:
		return "Busy"
	case Warning:
		return "Warning"
	case Error:
		return "Error"
	}
	return "Offline"
}

// Config supplies the device label and optional detail. Invalid states become Offline.
type Config struct {
	Label, Detail string
	State         State
}

// UpdateMsg replaces state and detail while retaining the device label.
type UpdateMsg struct {
	State  State
	Detail string
}

// Model owns one cell grid. Styles may be mapped to custom theme entries by the parent.
type Model struct {
	label, detail, heading string
	state                  State
	Styles                 [5]bubbles.StyleID
	buffer                 *bubbles.Buffer
}

func New(c Config) *Model {
	m := &Model{label: c.Label, Styles: [5]bubbles.StyleID{bubbles.Muted, bubbles.Accent, bubbles.Accent, bubbles.Selected, bubbles.Selected}, buffer: bubbles.NewBuffer(0, 0)}
	m.SetStatus(c.State, c.Detail)
	return m
}
func (m *Model) State() State   { return m.state }
func (m *Model) Detail() string { return m.detail }

// SetStatus updates text outside rendering. Status changes emit no commands or effects.
func (m *Model) SetStatus(state State, detail string) {
	if state > Error {
		state = Offline
	}
	m.state = state
	m.detail = detail
	symbols := [5]string{"-", "+", "~", "!", "x"}
	m.heading = symbols[state] + " " + state.String()
	if m.label != "" {
		m.heading += ": " + m.label
	}
}
func (m *Model) SetSize(w, h int) {
	m.buffer.Resize(w, h)
}

// CellView keeps the state first when space is tight. Detail occupies the second
// row when available. Text is clipped, never wrapped; remaining rows are cleared.
func (m *Model) CellView() *bubbles.Buffer {
	b := m.buffer
	b.Clear(bubbles.Normal)
	w, h := b.Size()
	if w == 0 || h == 0 {
		return b
	}
	layout.ClipText(b, layout.Rect{Y: 0, Width: w, Height: 1}, m.heading, m.Styles[m.state], layout.Start)
	if h > 1 {
		layout.ClipText(b, layout.Rect{Y: 1, Width: w, Height: 1}, m.detail, bubbles.Muted, layout.Start)
	}
	return b
}
