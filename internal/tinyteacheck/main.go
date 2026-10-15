//go:build tinygo

package main

import (
	"image/color"
	bubbles "nimbleterminal.dev/tinybubbles"
	tea "nimbleterminal.dev/tinybubbles/tinytea"
	listtea "nimbleterminal.dev/tinybubbles/tinytea/list"
	"nimbleterminal.dev/tinybubbles/widgets/list"
)

type display struct{}

func (display) Size() (int16, int16)              { return 128, 64 }
func (display) SetPixel(int16, int16, color.RGBA) {}
func (display) Display() error                    { return nil }

type model struct {
	*listtea.Model
	timer tea.Timer
}

func (m *model) Init() tea.Cmd { return m.timer.After(0, tea.QuitMsg{}) }

func (*model) View() tea.View { return tea.NewView("TinyTea") }
func main() {
	p, err := tea.NewProgram(&model{Model: listtea.New([]list.Item{{ID: "menu", Label: "Menu"}})}, display{}, nil, bubbles.DefaultTheme(true))
	if err != nil {
		panic(err)
	}
	if _, err = p.Run(); err != nil {
		panic(err)
	}
}
