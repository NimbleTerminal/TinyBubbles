// Package list adapts the shared display list to the cooperative TinyTea runtime.
package list

import (
	tea "nimbleterminal.dev/tinybubbles/tinytea"
	cells "nimbleterminal.dev/tinybubbles/widgets/list"
)

type SelectedMsg = cells.SelectedMsg
type Model struct{ *cells.Model }

func New(items []cells.Item) *Model { return &Model{cells.New(items)} }
func (*Model) Init() tea.Cmd        { return nil }
func (*Model) View() tea.View       { return tea.NewView("") }
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.SetSize(msg.Width, msg.Height)
	case tea.KeyPressMsg:
		switch msg.Code {
		case tea.KeyUp:
			m.Move(-1)
		case tea.KeyDown:
			m.Move(1)
		case tea.KeyEnter:
			if item, ok := m.Selected(); ok {
				selected := SelectedMsg{Item: item, Index: m.SelectedIndex()}
				return m, func() tea.Msg { return selected }
			}
		}
	}
	return m, nil
}
