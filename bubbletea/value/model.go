// Package value adapts the bounded value picker to Bubble Tea and Bubbles help.
package value

import (
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	cells "nimbleterminal.dev/tinybubbles/widgets/value"
)

type AppliedMsg = cells.AppliedMsg
type CancelledMsg = cells.CancelledMsg
type KeyMap struct{ Decrease, Increase, Apply, Cancel key.Binding }

func DefaultKeys() KeyMap {
	return KeyMap{
		Decrease: key.NewBinding(key.WithKeys("left"), key.WithHelp("left", "Decrease")),
		Increase: key.NewBinding(key.WithKeys("right"), key.WithHelp("right", "Increase")),
		Apply:    key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "Apply")),
		Cancel:   key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "Cancel")),
	}
}
func (k KeyMap) ShortHelp() []key.Binding  { return []key.Binding{k.Decrease, k.Increase, k.Apply} }
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
	k.Decrease.SetEnabled(k.Decrease.Enabled() && m.CanDecrease())
	k.Increase.SetEnabled(k.Increase.Enabled() && m.CanIncrease())
	return k
}
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.SetSize(msg.Width, msg.Height)
	case tea.KeyPressMsg:
		keys := m.Keys()
		switch {
		case key.Matches(msg, keys.Decrease):
			m.Adjust(-1)
		case key.Matches(msg, keys.Increase):
			m.Adjust(1)
		case key.Matches(msg, keys.Apply):
			event := AppliedMsg{Value: m.Apply()}
			return m, func() tea.Msg { return event }
		case key.Matches(msg, keys.Cancel):
			event := CancelledMsg{Value: m.Cancel()}
			return m, func() tea.Msg { return event }
		}
	}
	return m, nil
}
