package web

import (
	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"image/color"
	bubbles "nimbleterminal.dev/tinybubbles"
	displaytea "nimbleterminal.dev/tinybubbles/bubbletea"
	statustea "nimbleterminal.dev/tinybubbles/bubbletea/status"
	browser "nimbleterminal.dev/tinybubbles/web"
	"nimbleterminal.dev/tinybubbles/widgets/status"
	"tinygo.org/x/drivers"
)

var statusDetails = [5]string{"No connection", "Connected", "Connecting...", "Weak signal", "Connection failed"}

type statusDemo struct{ *statustea.Model }

func newStatusDemo() *statusDemo {
	return &statusDemo{statustea.New(status.Config{Label: "Wi-Fi", State: status.Ready, Detail: statusDetails[status.Ready]})}
}
func (m *statusDemo) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if k, ok := msg.(tea.KeyPressMsg); ok {
		keys := statusKeys()
		s := int(m.State())
		switch {
		case key.Matches(k, keys.next):
			s = (s + 1) % 5
		case key.Matches(k, keys.previous):
			s = (s + 4) % 5
		case key.Matches(k, keys.reset):
			s = int(status.Ready)
		case key.Matches(k, keys.quit):
			return m, tea.Quit
		default:
			return m, nil
		}
		m.SetStatus(status.State(s), statusDetails[s])
		return m, nil
	}
	m.Model.Update(msg)
	return m, nil
}

type statusDemoKeys struct{ previous, next, reset, quit key.Binding }

func (k statusDemoKeys) ShortHelp() []key.Binding  { return []key.Binding{k.previous, k.next, k.reset} }
func (k statusDemoKeys) FullHelp() [][]key.Binding { return [][]key.Binding{k.ShortHelp(), {k.quit}} }
func statusKeys() statusDemoKeys {
	return statusDemoKeys{
		previous: key.NewBinding(key.WithKeys("left"), key.WithHelp("left", "Previous state")),
		next:     key.NewBinding(key.WithKeys("right"), key.WithHelp("right", "Next state")),
		reset:    key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "Reset")),
		quit:     key.NewBinding(key.WithKeys("q"), key.WithHelp("q", "Quit")),
	}
}
func NewStatusDemo(display drivers.Displayer, monochrome bool) (tea.Model, *displaytea.DisplayModel, error) {
	m := newStatusDemo()
	theme := bubbles.DefaultTheme(monochrome)
	if !monochrome {
		// Color supplements the state text and ASCII symbols, never replaces them.
		for i, c := range []color.RGBA{{90, 230, 140, 255}, {255, 205, 70, 255}, {255, 100, 100, 255}} {
			theme.Styles[4+i] = bubbles.Style{Foreground: c, Background: color.RGBA{A: 255}}
		}
		m.Styles[status.Ready] = 4
		m.Styles[status.Warning] = 5
		m.Styles[status.Error] = 6
	}
	pixels, err := displaytea.NewModel(m, display, bubbles.DefaultFont(), theme)
	if err != nil {
		return nil, nil, err
	}
	wrapped, err := browser.WrapActions(pixels, "status", func(tea.Model) help.KeyMap { return statusKeys() }, nil)
	if err != nil {
		pixels.Close()
		return nil, nil, err
	}
	return wrapped, pixels, nil
}
