// Package displaytea adapts Bubble Tea v2 models to TinyBubbles displays.
package displaytea

import (
	"context"
	"errors"
	"io"

	tea "charm.land/bubbletea/v2"
	bubbles "nimbleterminal.dev/tinybubbles"
	"tinygo.org/x/drivers"
)

// CellViewer optionally supplies a pixel-oriented view with custom font tiles.
// Models still implement tea.Model, including View() tea.View.
type CellViewer interface{ CellView() *bubbles.Buffer }

// Program owns a real Bubble Tea program and a display renderer.
// Use Send to inject keys or application messages from peripheral goroutines.
type Program struct {
	*tea.Program
	model *adapter
}

type adapter struct {
	model    tea.Model
	renderer *bubbles.Renderer
	buffer   *bubbles.Buffer
	err      error
	cancel   context.CancelFunc
}

// NewProgram disables terminal input, output, signals, and panic recovery.
// Plain View content is clipped to the character grid; ANSI styling is not
// interpreted. Implement CellViewer for styled cells and custom font tiles.
func NewProgram(model tea.Model, display drivers.Displayer, font bubbles.Font, theme bubbles.Theme) (*Program, error) {
	if model == nil {
		return nil, errors.New("displaytea: nil model")
	}
	renderer, err := bubbles.NewRenderer(display, font, theme)
	if err != nil {
		return nil, err
	}
	cols, rows := renderer.Size()
	ctx, cancel := context.WithCancel(context.Background())
	a := &adapter{cancel: cancel, model: model, renderer: renderer, buffer: bubbles.NewBuffer(cols, rows)}
	p := tea.NewProgram(a, tea.WithContext(ctx), tea.WithInput(nil), tea.WithOutput(io.Discard),
		tea.WithoutRenderer(), tea.WithoutSignalHandler(), tea.WithoutCatchPanics(),
		tea.WithEnvironment([]string{}), tea.WithWindowSize(cols, rows))
	return &Program{Program: p, model: a}, nil
}

// Run returns the application's final model and the first display/runtime error.
func (p *Program) Run() (tea.Model, error) {
	defer p.model.cancel()
	_, err := p.Program.Run()
	if p.model.err != nil {
		err = p.model.err
	}
	return p.model.model, err
}
func (a *adapter) Init() tea.Cmd { return a.model.Init() }
func (a *adapter) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if a.err != nil {
		return a, tea.Quit
	}
	next, cmd := a.model.Update(msg)
	a.model = next
	return a, cmd
}
func (a *adapter) View() tea.View {
	view := a.model.View()
	if a.err != nil {
		return view
	}
	var cells *bubbles.Buffer
	if m, ok := a.model.(CellViewer); ok {
		cells = m.CellView()
	} else {
		a.buffer.Clear(bubbles.Normal)
		a.buffer.Text(0, 0, view.Content, bubbles.Normal)
		cells = a.buffer
	}
	a.err = a.renderer.Render(cells)
	if a.err != nil {
		a.cancel()
	}
	return view
}
