// Package dialog provides a two-choice confirmation prompt for small displays.
package dialog

import (
	bubbles "nimbleterminal.dev/tinybubbles"
	"strings"
	"unicode/utf8"
)

type Config struct{ Title, Body, Confirm, Cancel string }
type Result struct{ Confirmed bool }

// Model starts on Cancel and resolves at most once until Reset is called.
type Model struct {
	config        Config
	confirm, done bool
	result        Result
	buffer        *bubbles.Buffer
	lines         []string
}

func New(config Config) *Model {
	if config.Confirm == "" {
		config.Confirm = "Confirm"
	}
	if config.Cancel == "" {
		config.Cancel = "Cancel"
	}
	return &Model{config: config, buffer: bubbles.NewBuffer(0, 0)}
}
func (m *Model) ConfirmSelected() bool { return m.confirm }
func (m *Model) Done() bool            { return m.done }
func (m *Model) Reset()                { m.confirm = false; m.done = false; m.result = Result{} }
func (m *Model) Move(direction int) bool {
	if m.done || direction == 0 {
		return false
	}
	m.confirm = !m.confirm
	return true
}
func (m *Model) Resolve() (Result, bool) { return m.finish(m.confirm) }
func (m *Model) Cancel() (Result, bool)  { return m.finish(false) }
func (m *Model) finish(confirm bool) (Result, bool) {
	if m.done {
		return m.result, false
	}
	m.done = true
	m.confirm = confirm
	m.result = Result{Confirmed: confirm}
	return m.result, true
}
func (m *Model) SetSize(width, height int) {
	width, height = max(0, width), max(0, height)
	w, h := m.buffer.Size()
	if w == width && h == height {
		return
	}
	m.buffer = bubbles.NewBuffer(width, height)
	m.lines = wrap(m.config.Body, width, max(0, height-3))
}

// CellView reserves two rows for the choices so both stay visible on an OLED.
// Very short layouts show the selected choice; the parent should provide 4+ rows.
func (m *Model) CellView() *bubbles.Buffer {
	b := m.buffer
	b.Clear(bubbles.Normal)
	w, h := b.Size()
	if w == 0 || h == 0 {
		return b
	}
	if h == 1 {
		label := m.config.Cancel
		if m.confirm {
			label = m.config.Confirm
		}
		b.Set(0, 0, bubbles.Cell{Rune: '>', Style: bubbles.Selected})
		rowAt(b, 2, 0, label, bubbles.Selected)
		return b
	}
	if h >= 3 {
		row(b, 0, m.config.Title, bubbles.Accent)
	}
	for i, line := range m.lines {
		row(b, i+1, line, bubbles.Normal)
	}
	labels := [2]string{m.config.Cancel, m.config.Confirm}
	for i, label := range labels {
		selected := (i == 1) == m.confirm
		style, marker := bubbles.Normal, ' '
		if selected {
			style, marker = bubbles.Selected, '>'
		}
		if m.done {
			style = bubbles.Muted
		}
		for x := 0; x < w; x++ {
			b.Set(x, h-2+i, bubbles.Cell{Style: style})
		}
		b.Set(0, h-2+i, bubbles.Cell{Rune: marker, Style: style})
		rowAt(b, 2, h-2+i, label, style)
	}
	return b
}
func row(b *bubbles.Buffer, y int, text string, style bubbles.StyleID) {
	rowAt(b, 0, y, text, style)
}
func rowAt(b *bubbles.Buffer, x, y int, text string, style bubbles.StyleID) {
	w, _ := b.Size()
	if x >= w {
		return
	}
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

// Wrap once when sizing, keeping redraws free of text layout allocations.
func wrap(text string, width, height int) []string {
	if width <= 0 || height <= 0 {
		return nil
	}
	var lines []string
	line := ""
	for _, word := range strings.Fields(text) {
		if line != "" && utf8.RuneCountInString(line)+1+utf8.RuneCountInString(word) > width {
			lines = append(lines, line)
			line = ""
		}
		for utf8.RuneCountInString(word) > width {
			end := 0
			for i := 0; i < width; i++ {
				_, size := utf8.DecodeRuneInString(word[end:])
				end += size
			}
			lines = append(lines, word[:end])
			word = word[end:]
		}
		if word != "" {
			if line != "" {
				line += " "
			}
			line += word
		}
	}
	if line != "" {
		lines = append(lines, line)
	}
	if len(lines) > height {
		lines = lines[:height]
		last := []rune(lines[height-1])
		if len(last) >= width {
			last = last[:width-1]
		}
		lines[height-1] = string(last) + "."
	}
	return lines
}
