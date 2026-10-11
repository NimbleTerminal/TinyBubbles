//go:build tinygo

// Compile the choice picker with the cooperative runtime on a microcontroller.
package main

import (
	"image/color"
	bubbles "nimbleterminal.dev/tinybubbles"
	tea "nimbleterminal.dev/tinybubbles/tinytea"
	choicetea "nimbleterminal.dev/tinybubbles/tinytea/choice"
	"nimbleterminal.dev/tinybubbles/widgets/choice"
)

type display struct{}

func (display) Size() (int16, int16)              { return 128, 64 }
func (display) SetPixel(int16, int16, color.RGBA) {}
func (display) Display() error                    { return nil }

type model struct{ *choicetea.Model }

func (model) Init() tea.Cmd { return tea.Quit }
func main() {
	picker, err := choicetea.New(choice.Config{Label: "Sound", Options: []string{"Off", "On"}, Value: 1})
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
