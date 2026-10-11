package web

import (
	"encoding/json"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"github.com/NimbleMarkets/go-booba/actions"
	"github.com/NimbleMarkets/go-booba/actions/bubblehelp"
	bubbles "nimbleterminal.dev/tinybubbles"
	"nimbleterminal.dev/tinybubbles/bubbletea/hello"
	greeting "nimbleterminal.dev/tinybubbles/demo/hello"
	"nimbleterminal.dev/tinybubbles/tinytea"
	"tinygo.org/x/drivers"
)

// TinyHello reuses hello's state, bindings, and view under TinyTea's loop.
// It adapts only this demo: the host supplies ticks and TinyTea owns quitting.
// It is not a general adapter for arbitrary Bubble Tea commands.
type TinyHello struct{ inner *hello.Model }

func (*TinyHello) Init() tinytea.Cmd { return nil }
func (m *TinyHello) Update(msg tinytea.Msg) (tinytea.Model, tinytea.Cmd) {
	var event tea.Msg
	switch msg := msg.(type) {
	case tinytea.WindowSizeMsg:
		event = tea.WindowSizeMsg{Width: msg.Width, Height: msg.Height}
	case greeting.TickMsg:
		event = msg
	case tinytea.KeyPressMsg:
		code := msg.Code
		switch code {
		case tinytea.KeyEnter:
			code = tea.KeyEnter
		case tinytea.KeyEscape:
			code = tea.KeyEscape
		case tinytea.KeyRight:
			code = tea.KeyRight
		}
		press := tea.KeyPressMsg{Code: code}
		if key.Matches(press, m.inner.Keys().Quit) {
			return m, tinytea.Quit
		}
		event = press
	default:
		return m, nil
	}
	// The hello model's only scheduled command is its next animation tick.
	// TinyTea's host supplies that tick instead of running tea.Tick.
	m.inner.Update(event)
	return m, nil
}
func (m *TinyHello) View() tinytea.View { return tinytea.NewView("TinyBubbles + TinyTea") }
func (m *TinyHello) CellView() *bubbles.Buffer {
	b := m.inner.CellView()
	if b != nil {
		width, _ := b.Size()
		title := "TINYTEA       "
		if width < 36 {
			title = "TINYTEA   "
		}
		b.Text(1, 0, title, b.At(1, 0).Style)
	}
	return b
}

// TinySession owns the cooperative program and its browser action publisher.
// Call all its methods from one host loop, including Input and Tick.
type TinySession struct {
	Program   *tinytea.Program
	model     *TinyHello
	publisher *actions.Publisher
	supported bool
}

func NewTinyDemo(display drivers.Displayer, mono bool, output func(string)) (*TinySession, error) {
	m := &TinyHello{inner: hello.New()}
	m.inner.HideHelp(true)
	m.inner.SetColorEnabled(!mono)
	theme := greeting.Theme()
	if mono {
		theme = bubbles.DefaultTheme(true)
	}
	p, err := tinytea.NewProgram(m, display, greeting.NewFont(), theme)
	if err != nil {
		return nil, err
	}
	publisher, err := actions.NewPublisher("tinybubbles-hello-tinytea", func(data []byte) error {
		var snapshot actions.Snapshot
		if err := json.Unmarshal(data, &snapshot); err != nil {
			return err
		}
		wire, err := actions.EncodeProgramStatus(snapshot, actions.ProgramStatus{Summary: "Hello, tiny world"})
		if err == nil {
			output(wire)
		}
		return err
	})
	if err != nil {
		return nil, err
	}
	return &TinySession{Program: p, model: m, publisher: publisher}, nil
}
func (s *TinySession) publish() error {
	if !s.supported || s.Program.Stopped() {
		return nil
	}
	snapshot, _, err := bubblehelp.Snapshot(s.model.inner.Keys(), "tinybubbles-hello-tinytea", "hello", 0, bubblehelp.Options{IncludeDisabled: true})
	if err != nil {
		return err
	}
	_, err = s.publisher.Publish(snapshot)
	return err
}

// Input accepts complete gestures from this page's restricted key encoder,
// plus Booba's capability reply. It is not a terminal byte-stream parser.
func (s *TinySession) Input(input string) error {
	if s.Program.Stopped() {
		return nil
	}
	if input == actions.QueryProgramStatus || input == "\x1b]7501;?\a" {
		s.supported = true
		if err := s.publish(); err != nil {
			return err
		}
		return s.publisher.Refresh()
	}
	var code rune
	switch input {
	case "\r":
		code = tinytea.KeyEnter
	case "\x1b":
		code = tinytea.KeyEscape
	case "\x1b[C", "\x1bOC":
		code = tinytea.KeyRight
	case "q":
		code = 'q'
	default:
		return nil
	}
	if err := s.Program.Send(tinytea.KeyPressMsg{Code: code}); err != nil {
		return err
	}
	// Send queues commands; Step executes the queued Quit on this same loop.
	if err := s.Program.Step(); err != nil {
		return err
	}
	return s.publish()
}
func (s *TinySession) Tick() error {
	if s.Program.Stopped() {
		return nil
	}
	if err := s.Program.Send(greeting.TickMsg{}); err != nil {
		return err
	}
	return s.Program.Step()
}
