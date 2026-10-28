// Package tinyghostty embeds a live Ghostty VT terminal in a TinyBubbles UI.
//
// Model is a Bubble Tea model. Program output enters through Write or
// OutputMsg; keys and paste are encoded for the program and written to Input
// together with the terminal's query replies. CellView draws the screen into
// a reused TinyBubbles buffer, redrawing only dirty rows.
//
// Packages vt and render hold the libghostty-vt binding and cell renderer.
package tinyghostty

import (
	"io"
	"strings"

	tea "charm.land/bubbletea/v2"
	bubbles "nimbleterminal.dev/tinybubbles"
	"nimbleterminal.dev/tinybubbles/tinyghostty/render"
	"nimbleterminal.dev/tinybubbles/tinyghostty/vt"
)

// OutputMsg carries program output into the terminal through Program.Send.
type OutputMsg []byte

// Sender returns a writer that copies each write into an OutputMsg and passes
// it to send, typically Program.Send. Use it to pump a PTY or serial port from
// another goroutine while the program loop owns the terminal.
func Sender(send func(tea.Msg)) io.Writer { return sender(send) }

type sender func(tea.Msg)

func (s sender) Write(p []byte) (int, error) {
	s(OutputMsg(append([]byte(nil), p...)))
	return len(p), nil
}

// Model owns a terminal, its renderer, and the cell buffer it draws into.
type Model struct {
	// Input receives encoded keys, paste text, and terminal replies for the
	// program running in the terminal. Nil discards them.
	Input io.Writer

	term   *vt.Terminal
	render *render.Renderer
	buf    *bubbles.Buffer
	err    error
	out    []byte
}

// New creates a cols×rows terminal drawn with render.Theme(false). Check Err
// for libghostty-vt failures; a failed model draws an empty buffer.
func New(cols, rows int) *Model {
	m := &Model{
		render: render.New(render.Theme(false)),
		buf:    bubbles.NewBuffer(cols, rows),
		out:    make([]byte, 0, 64),
	}
	m.term, m.err = vt.New(cols, rows)
	return m
}

// Err reports the first terminal or drawing error.
func (m *Model) Err() error { return m.err }

// Close frees the terminal.
func (m *Model) Close() { m.term.Close() }

// Terminal exposes the binding for mode queries and direct access.
func (m *Model) Terminal() *vt.Terminal { return m.term }

// Renderer exposes cursor, bold, and palette options. Call
// Renderer().SetTheme with the theme passed to the display program.
func (m *Model) Renderer() *render.Renderer { return m.render }

func (m *Model) fail(err error) {
	if m.err == nil {
		m.err = err
	}
}

// Write feeds program output to the terminal and forwards any query replies
// to Input. Call it only from the loop that owns the model.
func (m *Model) Write(p []byte) (int, error) {
	if m.term == nil {
		return 0, m.err
	}
	n, err := m.term.Write(p)
	m.flushReplies()
	return n, err
}

func (m *Model) flushReplies() {
	for {
		n := m.term.ReadReply(m.out[:cap(m.out)])
		if n == 0 {
			return
		}
		m.send(m.out[:n])
	}
}

func (m *Model) send(p []byte) {
	if m.Input != nil && len(p) > 0 {
		if _, err := m.Input.Write(p); err != nil {
			m.fail(err)
		}
	}
}

// SetSize resizes the terminal and its buffer. Parents call it when laying
// out the widget; a standalone model also follows tea.WindowSizeMsg.
func (m *Model) SetSize(cols, rows int) {
	if m.term == nil || cols < 1 || rows < 1 {
		return
	}
	if c, r := m.term.Size(); c == cols && r == rows {
		return
	}
	if err := m.term.Resize(cols, rows); err != nil {
		m.fail(err)
		return
	}
	m.buf.Resize(cols, rows)
}

// SendKey encodes a key press for the program.
func (m *Model) SendKey(k tea.Key) {
	if m.term == nil {
		return
	}
	m.out = AppendKey(m.out[:0], k, m.term.DECMode(vt.ModeCursorKeys))
	m.send(m.out)
}

// Paste sends text as one paste, bracketed when the program enabled it.
// Newlines become carriage returns, as typed Enter keys would.
func (m *Model) Paste(text string) {
	if m.term == nil || m.Input == nil {
		return
	}
	bracketed := m.term.DECMode(vt.ModeBracketedPaste)
	// A pasted end marker would let text escape the bracket.
	text = strings.ReplaceAll(text, "\x1b[201~", "")
	text = strings.ReplaceAll(strings.ReplaceAll(text, "\r\n", "\r"), "\n", "\r")
	if bracketed {
		m.send([]byte("\x1b[200~"))
	}
	m.send([]byte(text))
	if bracketed {
		m.send([]byte("\x1b[201~"))
	}
}

func (m *Model) Init() tea.Cmd { return nil }

// Update routes output, size, key, and paste messages. Parents should forward
// keys and paste only while the terminal has focus.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case OutputMsg:
		m.Write(msg)
	case tea.WindowSizeMsg:
		m.SetSize(msg.Width, msg.Height)
	case tea.KeyPressMsg:
		m.SendKey(tea.Key(msg))
	case tea.PasteMsg:
		m.Paste(msg.Content)
	}
	return m, nil
}

// CellView draws the current screen into the reused buffer.
func (m *Model) CellView() *bubbles.Buffer {
	if m.term != nil {
		if _, err := m.render.Draw(m.term, m.buf); err != nil {
			m.fail(err)
		}
	}
	return m.buf
}

// View is empty so display programs, which call it every frame, allocate
// nothing; they draw CellView. Use Text for a plain-text dump.
func (m *Model) View() tea.View { return tea.NewView("") }

// Text draws the screen and returns it as plain text, trimming each line.
func (m *Model) Text() string {
	buf := m.CellView()
	w, h := buf.Size()
	var b strings.Builder
	line := make([]rune, w)
	for y := 0; y < h; y++ {
		if y > 0 {
			b.WriteByte('\n')
		}
		for x := range line {
			if line[x] = buf.At(x, y).Rune; line[x] == 0 {
				line[x] = ' '
			}
		}
		b.WriteString(strings.TrimRight(string(line), " "))
	}
	return b.String()
}
