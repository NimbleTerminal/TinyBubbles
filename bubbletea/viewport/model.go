// Package viewport adapts a text viewport to Bubble Tea and Bubbles help.
package viewport

import (
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	cells "nimbleterminal.dev/tinybubbles/widgets/viewport"
)

type KeyMap struct{ Up, Down, PageUp, PageDown, Top, Bottom key.Binding }

func DefaultKeys() KeyMap {
	return KeyMap{
		Up:       key.NewBinding(key.WithKeys("up"), key.WithHelp("up", "Scroll up")),
		Down:     key.NewBinding(key.WithKeys("down"), key.WithHelp("down", "Scroll down")),
		PageUp:   key.NewBinding(key.WithKeys("left"), key.WithHelp("left", "Page up")),
		PageDown: key.NewBinding(key.WithKeys("right"), key.WithHelp("right", "Page down")),
		Top:      key.NewBinding(key.WithKeys("home"), key.WithHelp("home", "Top")),
		Bottom:   key.NewBinding(key.WithKeys("end"), key.WithHelp("end", "Bottom")),
	}
}
func (k KeyMap) ShortHelp() []key.Binding { return []key.Binding{k.Up, k.Down, k.PageDown} }
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{k.ShortHelp(), {k.PageUp, k.Top, k.Bottom}}
}

type Model struct {
	*cells.Model
	KeyMap KeyMap
}

func New(content string) *Model { return &Model{cells.New(content), DefaultKeys()} }
func (*Model) Init() tea.Cmd    { return nil }
func (*Model) View() tea.View   { return tea.NewView("") }
func (m *Model) Keys() KeyMap {
	k := m.KeyMap
	for _, b := range []*key.Binding{&k.Up, &k.PageUp, &k.Top} {
		b.SetEnabled(b.Enabled() && !m.AtTop())
	}
	for _, b := range []*key.Binding{&k.Down, &k.PageDown, &k.Bottom} {
		b.SetEnabled(b.Enabled() && !m.AtBottom())
	}
	return k
}
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.SetSize(msg.Width, msg.Height)
	case tea.KeyPressMsg:
		k := m.Keys()
		switch {
		case key.Matches(msg, k.Up):
			m.Scroll(-1)
		case key.Matches(msg, k.Down):
			m.Scroll(1)
		case key.Matches(msg, k.PageUp):
			m.Page(-1)
		case key.Matches(msg, k.PageDown):
			m.Page(1)
		case key.Matches(msg, k.Top):
			m.GotoTop()
		case key.Matches(msg, k.Bottom):
			m.GotoBottom()
		}
	}
	return m, nil
}
