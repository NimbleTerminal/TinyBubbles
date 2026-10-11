// Package choice adapts the named choice picker to cooperative TinyTea.
package choice

import (
	tea "nimbleterminal.dev/tinybubbles/tinytea"
	cells "nimbleterminal.dev/tinybubbles/widgets/choice"
)

type AppliedMsg struct {
	Index int
	Label string
}
type CancelledMsg struct {
	Index int
	Label string
}
type Model struct{ *cells.Model }

func New(config cells.Config) (*Model, error) {
	m, err := cells.New(config)
	if err != nil {
		return nil, err
	}
	return &Model{m}, nil
}
func (*Model) Init() tea.Cmd  { return nil }
func (*Model) View() tea.View { return tea.NewView("") }
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
			event := AppliedMsg{m.Apply(), m.Label()}
			return m, func() tea.Msg { return event }
		case tea.KeyEscape:
			event := CancelledMsg{m.Cancel(), m.Label()}
			return m, func() tea.Msg { return event }
		}
	}
	return m, nil
}
