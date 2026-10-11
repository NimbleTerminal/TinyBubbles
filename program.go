package tinybubbles

import (
	"errors"
	"time"
	"tinygo.org/x/drivers"
)

type Msg interface{}

// Cmd performs short work and returns a message. Commands execute cooperatively,
// one per Step, never in goroutines. Do not block waiting for input or timers.
type Cmd func() Msg

// Model is the original cell-view model API. New applications should use
// Bubble Tea or the tinytea package's Bubble Tea-shaped model API.
type Model interface {
	Init() Cmd
	Update(Msg) (Model, Cmd)
	View() *Buffer
}

type WindowSizeMsg struct {
	Width, Height           int // cells
	PixelWidth, PixelHeight int16
}
type QuitMsg struct{}

func Quit() Msg { return QuitMsg{} }

// Config accepts already-initialized hardware. Input returns nil when idle.
// Theme nil selects the default for Monochrome. PollInterval defaults to 10ms.
type Config struct {
	Display      drivers.Displayer
	Font         Font
	Theme        *Theme
	Monochrome   bool
	Input        func() Msg
	PollInterval time.Duration
}

var (
	ErrCommandQueueFull = errors.New("tinybubbles: command queue full")
	ErrNilModel         = errors.New("tinybubbles: nil model")
)

// Program is single-owner: call all methods from the device's main loop, not
// from interrupts or concurrent goroutines. Copy interrupt data into a safe
// device-owned queue and consume it in Input.
type Program struct {
	model            Model
	renderer         *Renderer
	input            func() Msg
	interval         time.Duration
	commands         [16]Cmd
	head, count      int
	started, stopped bool
}

func NewProgram(model Model, config Config) (*Program, error) {
	if model == nil {
		return nil, ErrNilModel
	}
	theme := DefaultTheme(config.Monochrome)
	if config.Theme != nil {
		theme = *config.Theme
	}
	r, err := NewRenderer(config.Display, config.Font, theme)
	if err != nil {
		return nil, err
	}
	interval := config.PollInterval
	if interval <= 0 {
		interval = 10 * time.Millisecond
	}
	return &Program{model: model, renderer: r, input: config.Input, interval: interval}, nil
}

func (p *Program) enqueue(cmd Cmd) error {
	if cmd == nil {
		return nil
	}
	if p.count == len(p.commands) {
		return ErrCommandQueueFull
	}
	p.commands[(p.head+p.count)%len(p.commands)] = cmd
	p.count++
	return nil
}

func (p *Program) update(msg Msg) error {
	if msg == nil || p.stopped {
		return nil
	}
	if _, ok := msg.(QuitMsg); ok {
		p.stopped = true
		p.commands = [16]Cmd{}
		p.count = 0
		return nil
	}
	model, cmd := p.model.Update(msg)
	if model == nil {
		return ErrNilModel
	}
	p.model = model
	return p.enqueue(cmd)
}

// Start calls Init once, then sends the initial window size before the first View.
// Init's command runs on the first Step, after the model knows its dimensions.
func (p *Program) Start() error {
	if p.started {
		return nil
	}
	p.started = true
	if err := p.enqueue(p.model.Init()); err != nil {
		return err
	}
	err := p.update(WindowSizeMsg{p.renderer.cols, p.renderer.rows, p.renderer.width, p.renderer.height})
	if err != nil {
		return err
	}
	return p.render()
}

func (p *Program) render() error {
	if p.stopped {
		return nil
	}
	return p.renderer.Render(p.model.View())
}

// Send synchronously delivers any application message and renders. It does not
// execute the returned command; call Step regularly to process commands.
func (p *Program) Send(msg Msg) error {
	if err := p.Start(); err != nil {
		return err
	}
	if err := p.update(msg); err != nil {
		return err
	}
	return p.render()
}

// Step polls input once, executes at most one queued command, then renders.
// Even idle steps retry a failed display flush and support a reused View buffer.
func (p *Program) Step() error {
	if err := p.Start(); err != nil {
		return err
	}
	if p.stopped {
		return nil
	}
	if p.input != nil {
		if err := p.update(p.input()); err != nil {
			return err
		}
	}
	if p.count > 0 && !p.stopped {
		cmd := p.commands[p.head]
		p.commands[p.head] = nil
		p.head = (p.head + 1) % len(p.commands)
		p.count--
		if err := p.update(cmd()); err != nil {
			return err
		}
	}
	return p.render()
}

func (p *Program) Stopped() bool { return p.stopped }

// SetTheme forces a repaint on the next Step or Send.
func (p *Program) SetTheme(theme Theme) { p.renderer.SetTheme(theme) }

// Run owns a polling loop until QuitMsg or an error. Use Step for custom scheduling.
func (p *Program) Run() error {
	for !p.stopped {
		if err := p.Step(); err != nil {
			return err
		}
		if !p.stopped {
			time.Sleep(p.interval)
		}
	}
	return nil
}
