// Package hello preserves TinyBubbles' greeting while using Bubble Tea commands.
package hello

import (
	"time"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	bubbles "nimbleterminal.dev/tinybubbles"
	greeting "nimbleterminal.dev/tinybubbles/demo/hello"
)

type Model struct {
	greeting *greeting.Model
	keys     KeyMap
	paused   bool
	hideHelp bool
}

func New() *Model { return &Model{greeting: greeting.New(), keys: DefaultKeys()} }
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
		switch {
		case key.Matches(msg, m.keys.Pop):
			event = bubbles.KeyMsg{Key: bubbles.KeySelect}
		case key.Matches(msg, m.keys.Color):
			event = bubbles.KeyMsg{Key: bubbles.KeyRight}
		case key.Matches(msg, m.keys.Pause):
			m.paused = !m.paused
			label := "Pause"
			if m.paused {
				label = "Resume"
			}
			m.keys.Pause.SetHelp("esc", label)
			event = bubbles.KeyMsg{Key: bubbles.KeyBack}
		case key.Matches(msg, m.keys.Quit):
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
		w, h := view.Size()
		// Identify this runtime on the physical screen while retaining the
		// greeting's palette, large counter, and button hints.
		title := "BUBBLE TEA 2.1"
		if w < 36 {
			title = "BUBBLE TEA"
		}
		view.Text(1, 0, title, view.At(1, 0).Style)
		if m.hideHelp {
			row := h - 1
			if w < 36 || h < 16 {
				row = h - 2
			}
			for x := 0; x < w; x++ {
				view.Set(x, row, bubbles.Cell{})
			}
		}
	}
	return view
}

// HideHelp hides the on-display button hints; the help map remains available.
func (m *Model) HideHelp(hide bool) { m.hideHelp = hide }
func (m *Model) Keys() KeyMap       { return m.keys }
