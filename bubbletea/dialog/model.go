// Package dialog adapts the confirmation dialog to Bubble Tea and Bubbles help.
package dialog

import (
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	cells "nimbleterminal.dev/tinybubbles/widgets/dialog"
)

type ResultMsg = cells.ResultMsg
type KeyMap struct{ Previous, Next, Select, Cancel key.Binding }

func DefaultKeys() KeyMap {
	return KeyMap{
		Previous: key.NewBinding(key.WithKeys("left"), key.WithHelp("left", "Previous")),
		Next:     key.NewBinding(key.WithKeys("right"), key.WithHelp("right", "Next")),
		Select:   key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "Select")),
		Cancel:   key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "Cancel")),
	}
}
func (k KeyMap) ShortHelp() []key.Binding  { return []key.Binding{k.Next, k.Select, k.Cancel} }
func (k KeyMap) FullHelp() [][]key.Binding { return [][]key.Binding{k.ShortHelp(), {k.Previous}} }

type Model struct {
	*cells.Model
	KeyMap KeyMap
}

func New(config cells.Config) *Model { return &Model{Model: cells.New(config), KeyMap: DefaultKeys()} }
func (*Model) Init() tea.Cmd         { return nil }
func (*Model) View() tea.View        { return tea.NewView("") }
func (m *Model) Keys() KeyMap {
	k := m.KeyMap
	for _, binding := range []*key.Binding{&k.Previous, &k.Next, &k.Select, &k.Cancel} {
		binding.SetEnabled(binding.Enabled() && !m.Done())
	}
	return k
}
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var result cells.Result
	var resolved bool
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
		case key.Matches(msg, keys.Select):
			result, resolved = m.Resolve()
		case key.Matches(msg, keys.Cancel):
			result, resolved = m.Cancel()
		}
	}
	if resolved {
		event := ResultMsg{Confirmed: result.Confirmed}
		return m, func() tea.Msg { return event }
	}
	return m, nil
}
