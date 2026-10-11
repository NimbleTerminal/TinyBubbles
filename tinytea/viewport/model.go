// Package viewport adapts a text viewport to cooperative TinyTea.
package viewport

import (
	tea "nimbleterminal.dev/tinybubbles/tinytea"
	cells "nimbleterminal.dev/tinybubbles/widgets/viewport"
)

type Model struct{ *cells.Model }

func New(content string) *Model { return &Model{cells.New(content)} }
func (*Model) Init() tea.Cmd    { return nil }
func (*Model) View() tea.View   { return tea.NewView("") }
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.SetSize(msg.Width, msg.Height)
	case tea.KeyPressMsg:
		switch msg.Code {
		case tea.KeyUp:
			m.Scroll(-1)
		case tea.KeyDown:
			m.Scroll(1)
		case tea.KeyLeft:
			m.Page(-1)
		case tea.KeyRight:
			m.Page(1)
		}
	}
	return m, nil
}
