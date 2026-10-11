// Package choice adapts the named choice picker to Bubble Tea and Bubbles help.
package choice

import (
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	cells "nimbleterminal.dev/tinybubbles/widgets/choice"
)

type AppliedMsg = cells.AppliedMsg
type CancelledMsg = cells.CancelledMsg
type KeyMap struct{ Previous, Next, Apply, Cancel key.Binding }

func DefaultKeys() KeyMap {
	return KeyMap{
		Previous: key.NewBinding(key.WithKeys("left"), key.WithHelp("left", "Previous")),
		Next:     key.NewBinding(key.WithKeys("right"), key.WithHelp("right", "Next")),
		Apply:    key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "Apply")),
		Cancel:   key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "Cancel")),
	}
}
func (k KeyMap) ShortHelp() []key.Binding  { return []key.Binding{k.Previous, k.Next, k.Apply} }
func (k KeyMap) FullHelp() [][]key.Binding { return [][]key.Binding{k.ShortHelp(), {k.Cancel}} }

type Model struct {
	*cells.Model
	KeyMap KeyMap
}

func New(config cells.Config) (*Model, error) {
	m, err := cells.New(config)
	if err != nil {
		return nil, err
	}
	return &Model{Model: m, KeyMap: DefaultKeys()}, nil
}
func (*Model) Init() tea.Cmd  { return nil }
func (*Model) View() tea.View { return tea.NewView("") }
func (m *Model) Keys() KeyMap {
	k := m.KeyMap
	k.Previous.SetEnabled(k.Previous.Enabled() && m.Len() > 1)
	k.Next.SetEnabled(k.Next.Enabled() && m.Len() > 1)
	return k
}
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.SetSize(msg.Width, msg.Height)
	case tea.KeyPressMsg:
		keys := m.Keys()
		switch {
		case key.Matches(msg, keys.Previous):
			m.Move(-1)
		case key.Matches(msg, keys.Next):
			m.Move(1)
		case key.Matches(msg, keys.Apply):
			event := AppliedMsg{Index: m.Apply(), Label: m.Label()}
			return m, func() tea.Msg { return event }
		case key.Matches(msg, keys.Cancel):
			event := CancelledMsg{Index: m.Cancel(), Label: m.Label()}
			return m, func() tea.Msg { return event }
		}
	}
	return m, nil
}
