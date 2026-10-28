package demo

import (
	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"tinygo.org/x/drivers"

	bubbles "nimbleterminal.dev/tinybubbles"
	displaytea "nimbleterminal.dev/tinybubbles/bubbletea"
	"nimbleterminal.dev/tinybubbles/layout"
	"nimbleterminal.dev/tinybubbles/tinyghostty"
	"nimbleterminal.dev/tinybubbles/tinyghostty/render"
	browser "nimbleterminal.dev/tinybubbles/web"
)

// KeyMap holds the action buttons. Each types a command into the shell; all
// other keys go to the terminal.
type KeyMap struct{ Help, Colors, Styles, Clear key.Binding }

func DefaultKeys() KeyMap {
	return KeyMap{
		Help:   key.NewBinding(key.WithKeys("f1"), key.WithHelp("F1", "help")),
		Colors: key.NewBinding(key.WithKeys("f2"), key.WithHelp("F2", "colors")),
		Styles: key.NewBinding(key.WithKeys("f3"), key.WithHelp("F3", "styles")),
		Clear:  key.NewBinding(key.WithKeys("f4"), key.WithHelp("F4", "clear")),
	}
}

func (k KeyMap) ShortHelp() []key.Binding  { return []key.Binding{k.Help, k.Colors, k.Styles, k.Clear} }
func (k KeyMap) FullHelp() [][]key.Binding { return [][]key.Binding{k.ShortHelp()} }

// Model composes a title row and the terminal. It owns sizing and focus.
type Model struct {
	Keys    KeyMap
	term    *tinyghostty.Model
	shell   *Shell
	buf     *bubbles.Buffer
	body    layout.Rect
	started bool
}

// New creates the terminal and shell. Check Err for libghostty-vt failures.
func New(theme bubbles.Theme) *Model {
	term := tinyghostty.New(1, 1)
	term.Renderer().SetTheme(theme)
	m := &Model{Keys: DefaultKeys(), term: term, buf: bubbles.NewBuffer(0, 0)}
	m.shell = NewShell(term, term.Terminal().Size)
	term.Input = m.shell
	return m
}

func (m *Model) Err() error { return m.term.Err() }
func (m *Model) Close()     { m.term.Close() }

func (m *Model) Init() tea.Cmd { return nil }

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.buf.Resize(msg.Width, msg.Height)
		_, m.body = layout.Bounds(m.buf).Rows(1, 0)
		m.term.SetSize(m.body.Width, m.body.Height)
		// Print the banner once the terminal has its real width.
		if !m.started && m.body.Width > 0 {
			m.started = true
			m.shell.Start()
		}
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, m.Keys.Help):
			m.shell.Type("help\r")
		case key.Matches(msg, m.Keys.Colors):
			m.shell.Type("colors\r")
		case key.Matches(msg, m.Keys.Styles):
			m.shell.Type("styles\r")
		case key.Matches(msg, m.Keys.Clear):
			m.shell.Type("clear\r")
		default:
			m.term.Update(msg)
		}
	default:
		m.term.Update(msg)
	}
	m.shell.Process()
	return m, nil
}

func (m *Model) View() tea.View { return tea.NewView("") }

func (m *Model) CellView() *bubbles.Buffer {
	m.buf.Clear(bubbles.Normal)
	title, _ := layout.Bounds(m.buf).Rows(1, 0)
	layout.ClipText(m.buf, title, " tinyghostty", bubbles.Selected, layout.Start)
	layout.Draw(m.buf, m.body, m.term.CellView())
	return m.buf
}

// Text returns the terminal screen as plain text for tests.
func (m *Model) Text() string { return m.term.Text() }

// NewDemo wraps the model for a display and exposes its buttons as browser
// actions. Close the returned display model after the program ends.
func NewDemo(display drivers.Displayer, monochrome bool) (tea.Model, *displaytea.DisplayModel, error) {
	theme := render.Theme(monochrome)
	m := New(theme)
	if err := m.Err(); err != nil {
		return nil, nil, err
	}
	pixels, err := displaytea.NewModel(m, display, bubbles.DefaultFont(), theme)
	if err != nil {
		m.Close()
		return nil, nil, err
	}
	wrapped, err := browser.WrapActions(pixels, "tinyghostty", func(tea.Model) help.KeyMap { return m.Keys }, nil)
	if err != nil {
		pixels.Close()
		m.Close()
		return nil, nil, err
	}
	return wrapped, pixels, nil
}
