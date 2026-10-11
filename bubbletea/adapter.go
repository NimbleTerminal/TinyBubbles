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
	model *DisplayModel
}

// DisplayModel adds Displayer rendering to a Bubble Tea model.
type DisplayModel struct {
	ctx      context.Context
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
	a, err := NewModel(model, display, font, theme)
	if err != nil {
		return nil, err
	}
	cols, rows := a.Size()
	p := tea.NewProgram(a, tea.WithContext(a.Context()), tea.WithInput(nil), tea.WithOutput(io.Discard),
		tea.WithoutRenderer(), tea.WithoutSignalHandler(), tea.WithoutCatchPanics(),
		tea.WithEnvironment([]string{}), tea.WithWindowSize(cols, rows))
	return &Program{Program: p, model: a}, nil
}

// NewModel wraps a model with pixel rendering while allowing a host such as
// Booba to own the program and its input/output transport. Use WithoutRenderer
// and WithContext(wrapper.Context()) on that program, and Close after Run.
func NewModel(model tea.Model, display drivers.Displayer, font bubbles.Font, theme bubbles.Theme) (*DisplayModel, error) {
	if model == nil {
		return nil, errors.New("displaytea: nil model")
	}
	renderer, err := bubbles.NewRenderer(display, font, theme)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(context.Background())
	return &DisplayModel{model: model, renderer: renderer, ctx: ctx, cancel: cancel}, nil
}

// Context is canceled on a display error or Close.
func (a *DisplayModel) Context() context.Context { return a.ctx }
func (a *DisplayModel) Close()                   { a.cancel() }
func (a *DisplayModel) Err() error               { return a.err }

// Model returns the current inner model; access it only from the model loop.
func (a *DisplayModel) Model() tea.Model { return a.model }
func (a *DisplayModel) Size() (int, int) { return a.renderer.Size() }

// Run returns the application's final model and the first display/runtime error.
func (p *Program) Run() (tea.Model, error) {
	defer p.model.cancel()
	_, err := p.Program.Run()
	if p.model.err != nil {
		err = p.model.err
	}
	return p.model.model, err
}
func (a *DisplayModel) Init() tea.Cmd { return a.model.Init() }
func (a *DisplayModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if a.err != nil {
		return a, tea.Quit
	}
	// A host terminal resize must not change the physical display's cell grid.
	if _, ok := msg.(tea.WindowSizeMsg); ok {
		cols, rows := a.Size()
		msg = tea.WindowSizeMsg{Width: cols, Height: rows}
	}
	next, cmd := a.model.Update(msg)
	a.model = next
	return a, cmd
}
func (a *DisplayModel) View() tea.View {
	view := a.model.View()
	if a.err != nil {
		return view
	}
	var cells *bubbles.Buffer
	if m, ok := a.model.(CellViewer); ok {
		cells = m.CellView()
	} else {
		if a.buffer == nil {
			cols, rows := a.Size()
			a.buffer = bubbles.NewBuffer(cols, rows)
		}
		a.buffer.Clear(bubbles.Normal)
		a.buffer.Text(0, 0, view.Content, bubbles.Normal)
		cells = a.buffer
	}
	if err := a.renderer.Render(cells); err != nil {
		a.err = &bubbles.DisplayError{Err: err}
		a.cancel()
	}
	return view
}
