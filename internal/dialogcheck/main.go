//go:build tinygo

// Compile the confirmation dialog with TinyTea on a microcontroller.
package main

import (
	"image/color"
	bubbles "nimbleterminal.dev/tinybubbles"
	tea "nimbleterminal.dev/tinybubbles/tinytea"
	dialogtea "nimbleterminal.dev/tinybubbles/tinytea/dialog"
	"nimbleterminal.dev/tinybubbles/widgets/dialog"
)

type display struct{}

func (display) Size() (int16, int16)              { return 128, 64 }
func (display) SetPixel(int16, int16, color.RGBA) {}
func (display) Display() error                    { return nil }

type model struct{ *dialogtea.Model }

func (model) Init() tea.Cmd { return tea.Quit }
func main() {
	prompt := dialogtea.New(dialog.Config{Title: "Reset?", Body: "Restore default settings?"})
	p, err := tea.NewProgram(model{prompt}, display{}, nil, bubbles.DefaultTheme(true))
	if err != nil {
		panic(err)
	}
	if _, err = p.Run(); err != nil {
		panic(err)
	}
}
