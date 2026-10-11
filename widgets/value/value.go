// Package value provides a bounded integer setting with apply and cancel.
package value

import (
	"errors"
	"nimbleterminal.dev/tinybubbles/layout"
	"strconv"

	bubbles "nimbleterminal.dev/tinybubbles"
)

// Config describes a setting. Values and bounds are inclusive; Step must be positive.
type Config struct {
	Label, Unit           string
	Min, Max, Step, Value int32
}

// Model keeps the draft separate from the last applied value.
// It owns no timers or device effects. The parent applies changes to hardware.
type Model struct {
	config         Config
	value, applied int32
	text, bounds   string
	buffer         *bubbles.Buffer
}

func New(config Config) (*Model, error) {
	if config.Min > config.Max {
		return nil, errors.New("value: minimum exceeds maximum")
	}
	if config.Step <= 0 {
		return nil, errors.New("value: step must be positive")
	}
	m := &Model{config: config, buffer: bubbles.NewBuffer(0, 0)}
	m.bounds = strconv.FormatInt(int64(config.Min), 10) + " .. " + strconv.FormatInt(int64(config.Max), 10)
	m.SetValue(config.Value)
	return m, nil
}
func (m *Model) Value() int32      { return m.value }
func (m *Model) Applied() int32    { return m.applied }
func (m *Model) Dirty() bool       { return m.value != m.applied }
func (m *Model) CanDecrease() bool { return m.value > m.config.Min }
func (m *Model) CanIncrease() bool { return m.value < m.config.Max }

// SetValue clamps an external value and replaces both the draft and applied value.
func (m *Model) SetValue(value int32) {
	m.value = min(m.config.Max, max(m.config.Min, value))
	m.applied = m.value
	m.refresh()
}
func (m *Model) refresh() { m.text = strconv.FormatInt(int64(m.value), 10) + m.config.Unit }

// Adjust moves one step in the requested direction. Endpoints clamp, never wrap.
func (m *Model) Adjust(direction int) bool {
	next := int64(m.value)
	if direction < 0 {
		next -= int64(m.config.Step)
	} else if direction > 0 {
		next += int64(m.config.Step)
	}
	next = min(int64(m.config.Max), max(int64(m.config.Min), next))
	if int32(next) == m.value {
		return false
	}
	m.value = int32(next)
	m.refresh()
	return true
}
func (m *Model) Apply() int32  { m.applied = m.value; return m.applied }
func (m *Model) Cancel() int32 { m.value = m.applied; m.refresh(); return m.value }
func (m *Model) SetSize(width, height int) {
	m.buffer.Resize(width, height)
}

// CellView reuses its grid. Small layouts keep the value and omit extra details.
func (m *Model) CellView() *bubbles.Buffer {
	b := m.buffer
	b.Clear(bubbles.Normal)
	w, h := b.Size()
	if w == 0 || h == 0 {
		return b
	}
	row := 0
	if h >= 3 {
		layout.ClipText(b, layout.Rect{Y: 0, Width: w, Height: 1}, m.config.Label, bubbles.Accent, layout.Center)
		row = h/2 - 1
	}
	if h >= 3 && row == 0 {
		row = 1
	}
	layout.ClipText(b, layout.Rect{Y: row, Width: w, Height: 1}, m.text, bubbles.Selected, layout.Center)
	if h >= 4 && w >= 3 {
		filled := 0
		if m.config.Max != m.config.Min {
			filled = int((int64(m.value) - int64(m.config.Min)) * int64(w-2) / (int64(m.config.Max) - int64(m.config.Min)))
		}
		b.Set(0, row+1, bubbles.Cell{Rune: '[', Style: bubbles.Muted})
		b.Set(w-1, row+1, bubbles.Cell{Rune: ']', Style: bubbles.Muted})
		for x := 0; x < w-2; x++ {
			r, style := '-', bubbles.Muted
			if x < filled {
				r, style = '=', bubbles.Accent
			}
			b.Set(x+1, row+1, bubbles.Cell{Rune: r, Style: style})
		}
	}
	if h >= 6 {
		layout.ClipText(b, layout.Rect{Y: h - 2, Width: w, Height: 1}, m.bounds, bubbles.Muted, layout.Center)
	}
	if h >= 4 {
		state := "Applied"
		if m.Dirty() {
			state = "Unsaved"
		}
		layout.ClipText(b, layout.Rect{Y: h - 1, Width: w, Height: 1}, state, bubbles.Muted, layout.Center)
	}
	return b
}

// AppliedMsg captures the result emitted by runtime adapters.
type AppliedMsg struct{ Value int32 }

// CancelledMsg captures the result emitted by runtime adapters.
type CancelledMsg struct{ Value int32 }
