//go:build tinygo

package main

import (
	bubbles "github.com/NimbleTerminal/TinyBubbles"
	tea "github.com/NimbleTerminal/TinyBubbles/tinytea"
	"image/color"
)

type display struct{}

func (display) Size() (int16, int16)              { return 128, 64 }
func (display) SetPixel(int16, int16, color.RGBA) {}
func (display) Display() error                    { return nil }

type model struct{}

func (model) Init() tea.Cmd                         { return tea.Quit }
func (m model) Update(tea.Msg) (tea.Model, tea.Cmd) { return m, nil }
func (model) View() tea.View                        { return tea.NewView("TinyTea") }
func main() {
	p, err := tea.NewProgram(model{}, display{}, nil, bubbles.DefaultTheme(true))
	if err != nil {
		panic(err)
	}
	if _, err = p.Run(); err != nil {
		panic(err)
	}
}
