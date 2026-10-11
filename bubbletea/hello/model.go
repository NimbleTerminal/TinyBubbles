// Package hello preserves TinyBubbles' greeting while using Bubble Tea commands.
package hello

import (
	"time"

	tea "charm.land/bubbletea/v2"
	bubbles "nimbleterminal.dev/tinybubbles"
	greeting "nimbleterminal.dev/tinybubbles/demo/hello"
)

type Model struct{ greeting *greeting.Model }

func New() *Model { return &Model{greeting: greeting.New()} }
func tick() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(time.Time) tea.Msg { return greeting.TickMsg{} })
}
func (*Model) Init() tea.Cmd { return tick() }
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var event bubbles.Msg
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		event = bubbles.WindowSizeMsg{Width: msg.Width, Height: msg.Height}
	case greeting.TickMsg:
		event = msg
		cmd = tick()
	case tea.KeyPressMsg:
		switch msg.Code {
		case tea.KeyEnter:
			event = bubbles.KeyMsg{Key: bubbles.KeySelect}
		case tea.KeyRight:
			event = bubbles.KeyMsg{Key: bubbles.KeyRight}
		case tea.KeyEscape:
			event = bubbles.KeyMsg{Key: bubbles.KeyBack}
		case 'q':
			return m, tea.Quit
		}
	}
	if event != nil {
		m.greeting.Update(event)
	}
	return m, cmd
}
func (*Model) View() tea.View { return tea.NewView("TinyBubbles + Bubble Tea v2.1") }
func (m *Model) CellView() *bubbles.Buffer {
	view := m.greeting.View()
	if view != nil {
		// Identify this runtime on the physical screen while retaining the
		// greeting's palette, large counter, and button hints.
		view.Text(1, 0, "BUBBLE TEA 2.1", view.At(1, 0).Style)
	}
	return view
}
