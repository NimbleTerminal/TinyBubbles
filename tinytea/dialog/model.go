// Package dialog adapts the confirmation dialog to cooperative TinyTea.
package dialog

import (
	tea "nimbleterminal.dev/tinybubbles/tinytea"
	cells "nimbleterminal.dev/tinybubbles/widgets/dialog"
)

type ResultMsg struct{ Confirmed bool }
type Model struct{ *cells.Model }

func New(config cells.Config) *Model { return &Model{cells.New(config)} }
func (*Model) Init() tea.Cmd         { return nil }
func (*Model) View() tea.View        { return tea.NewView("") }
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var result cells.Result
	var resolved bool
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.SetSize(msg.Width, msg.Height)
	case tea.KeyPressMsg:
		switch msg.Code {
		case tea.KeyLeft:
			m.Move(-1)
		case tea.KeyRight:
			m.Move(1)
		case tea.KeyEnter:
			result, resolved = m.Resolve()
		case tea.KeyEscape:
			result, resolved = m.Cancel()
		}
	}
	if resolved {
		event := ResultMsg{result.Confirmed}
		return m, func() tea.Msg { return event }
	}
	return m, nil
}
