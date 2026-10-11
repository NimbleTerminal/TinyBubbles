// Package status adapts device status to tinytea. Parents own input and effects.
package status

import (
	tea "nimbleterminal.dev/tinybubbles/tinytea"
	cells "nimbleterminal.dev/tinybubbles/widgets/status"
)

type UpdateMsg = cells.UpdateMsg
type Model struct{ *cells.Model }

func New(config cells.Config) *Model { return &Model{cells.New(config)} }
func (*Model) Init() tea.Cmd         { return nil }
func (*Model) View() tea.View        { return tea.NewView("") }
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.SetSize(msg.Width, msg.Height)
	case UpdateMsg:
		m.SetStatus(msg.State, msg.Detail)
	}
	return m, nil
}
