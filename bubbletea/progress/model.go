// Package progress adapts a progress bar to bubbletea. Parents own timers and effects.
package progress

import (
	tea "charm.land/bubbletea/v2"
	cells "nimbleterminal.dev/tinybubbles/widgets/progress"
)

type UpdateMsg = cells.UpdateMsg
type PulseMsg = cells.PulseMsg
type Model struct{ *cells.Model }

func New(config cells.Config) *Model { return &Model{cells.New(config)} }
func (*Model) Init() tea.Cmd         { return nil }
func (*Model) View() tea.View        { return tea.NewView("") }
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.SetSize(msg.Width, msg.Height)
	case UpdateMsg:
		m.SetProgress(msg.Done, msg.Total)
	case PulseMsg:
		m.Advance()
	}
	return m, nil
}
