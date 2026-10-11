// Package list adapts the small-display list to Bubble Tea and Bubbles help.
package list

import (
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	bubbles "nimbleterminal.dev/tinybubbles"
	cells "nimbleterminal.dev/tinybubbles/widgets/list"
)

// SelectedMsg reports activation, separate from moving the highlight.
type SelectedMsg = cells.SelectedMsg

type KeyMap struct{ Previous, Next, Select key.Binding }

func DefaultKeys() KeyMap {
	return KeyMap{
		Previous: key.NewBinding(key.WithKeys("up"), key.WithHelp("up", "Previous")),
		Next:     key.NewBinding(key.WithKeys("down"), key.WithHelp("down", "Next")),
		Select:   key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "Select")),
	}
}
func (k KeyMap) ShortHelp() []key.Binding  { return []key.Binding{k.Previous, k.Next, k.Select} }
func (k KeyMap) FullHelp() [][]key.Binding { return [][]key.Binding{k.ShortHelp()} }

type Model struct {
	*cells.Model
	KeyMap KeyMap
}

func New(items []cells.Item) *Model { return &Model{Model: cells.New(items), KeyMap: DefaultKeys()} }
func (*Model) Init() tea.Cmd        { return nil }
func (m *Model) Keys() KeyMap {
	keys := m.KeyMap
	_, enabled := m.Selected()
	keys.Previous.SetEnabled(enabled && keys.Previous.Enabled())
	keys.Next.SetEnabled(enabled && keys.Next.Enabled())
	keys.Select.SetEnabled(enabled && keys.Select.Enabled())
	return keys
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
		case key.Matches(msg, keys.Select):
			item, ok := m.Selected()
			if ok {
				selected := SelectedMsg{Item: item, Index: m.SelectedIndex()}
				return m, func() tea.Msg { return selected }
			}
		}
	}
	return m, nil
}
func (*Model) View() tea.View              { return tea.NewView("") }
func (m *Model) CellView() *bubbles.Buffer { return m.Model.CellView() }
