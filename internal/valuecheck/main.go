//go:build tinygo

// Compile the value picker with the cooperative runtime on a microcontroller.
package main

import (
	"image/color"
	bubbles "nimbleterminal.dev/tinybubbles"
	tea "nimbleterminal.dev/tinybubbles/tinytea"
	valuetea "nimbleterminal.dev/tinybubbles/tinytea/value"
	"nimbleterminal.dev/tinybubbles/widgets/value"
)

type display struct{}

func (display) Size() (int16, int16)              { return 128, 64 }
func (display) SetPixel(int16, int16, color.RGBA) {}
func (display) Display() error                    { return nil }

type model struct{ *valuetea.Model }

func (model) Init() tea.Cmd { return tea.Quit }
func main() {
	picker, err := valuetea.New(value.Config{Label: "Volume", Unit: "%", Min: 0, Max: 100, Step: 5, Value: 50})
	if err != nil {
		panic(err)
	}
	p, err := tea.NewProgram(model{picker}, display{}, nil, bubbles.DefaultTheme(true))
	if err != nil {
		panic(err)
	}
	if _, err = p.Run(); err != nil {
		panic(err)
	}
}
