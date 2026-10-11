// Package tinytea provides a cooperative subset of Bubble Tea v2's model API
// for memory-constrained displays. See RUNTIMES.md for behavioral differences.
package tinytea

import (
	"errors"

	bubbles "github.com/NimbleTerminal/TinyBubbles"
	"tinygo.org/x/drivers"
)

type Msg = any
type Cmd func() Msg
type View struct{ Content string }

func NewView(content string) View { return View{Content: content} }

type Model interface {
	Init() Cmd
	Update(Msg) (Model, Cmd)
	View() View
}
type CellViewer interface{ CellView() *bubbles.Buffer }
type WindowSizeMsg struct{ Width, Height int }
type QuitMsg struct{}

func Quit() Msg { return QuitMsg{} }

// KeyPressMsg supports basic button presses, without modifiers or release events.
type KeyPressMsg struct{ Code rune }

const (
	KeyEnter  rune = 13
	KeyEscape rune = 27
)
const (
	KeyUp rune = 0x10ffff + 2 + iota
	KeyDown
	KeyRight
	KeyLeft
)

// Program embeds the cooperative Step/Send/Stopped API. It has one owner:
// never call its methods concurrently or from interrupt handlers.
type Program struct {
	*bubbles.Program
	adapter *adapter
}

// NewProgram has the same display arguments as the Bubble Tea display adapter.
// Run returns the final model. Send returns synchronous rendering errors.
func NewProgram(model Model, display drivers.Displayer, font bubbles.Font, theme bubbles.Theme) (*Program, error) {
	if model == nil {
		return nil, errors.New("tinytea: nil model")
	}
	a := &adapter{model: model}
	p, err := bubbles.NewProgram(a, bubbles.Config{Display: display, Font: font, Theme: &theme})
	if err != nil {
		return nil, err
	}
	return &Program{Program: p, adapter: a}, nil
}
func (p *Program) Run() (Model, error) {
	err := p.Program.Run()
	return p.adapter.model, err
}

type adapter struct {
	model  Model
	buffer *bubbles.Buffer
}

func command(cmd Cmd) bubbles.Cmd {
	if cmd == nil {
		return nil
	}
	return func() bubbles.Msg {
		msg := cmd()
		if _, ok := msg.(QuitMsg); ok {
			return bubbles.Quit()
		}
		return msg
	}
}
func (a *adapter) Init() bubbles.Cmd { return command(a.model.Init()) }
func (a *adapter) Update(msg bubbles.Msg) (bubbles.Model, bubbles.Cmd) {
	var event Msg = msg
	switch msg := msg.(type) {
	case QuitMsg:
		return a, bubbles.Quit
	case bubbles.WindowSizeMsg:
		a.buffer = bubbles.NewBuffer(msg.Width, msg.Height)
		event = WindowSizeMsg{Width: msg.Width, Height: msg.Height}
	case bubbles.KeyMsg:
		switch msg.Key {
		case bubbles.KeyUp:
			event = KeyPressMsg{Code: KeyUp}
		case bubbles.KeyDown:
			event = KeyPressMsg{Code: KeyDown}
		case bubbles.KeyRight:
			event = KeyPressMsg{Code: KeyRight}
		case bubbles.KeyLeft:
			event = KeyPressMsg{Code: KeyLeft}
		case bubbles.KeySelect:
			event = KeyPressMsg{Code: KeyEnter}
		case bubbles.KeyBack:
			event = KeyPressMsg{Code: KeyEscape}
		case bubbles.KeyQuit:
			event = KeyPressMsg{Code: 'q'}
		}
	}
	next, cmd := a.model.Update(event)
	if next == nil {
		return nil, nil
	}
	a.model = next
	return a, command(cmd)
}
func (a *adapter) View() *bubbles.Buffer {
	if m, ok := a.model.(CellViewer); ok {
		return m.CellView()
	}
	a.buffer.Clear(bubbles.Normal)
	a.buffer.Text(0, 0, a.model.View().Content, bubbles.Normal)
	return a.buffer
}
