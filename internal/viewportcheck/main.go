//go:build tinygo

// Compile the text viewport with the cooperative runtime on a microcontroller.
package main

import (
	"image/color"
	bubbles "nimbleterminal.dev/tinybubbles"
	tea "nimbleterminal.dev/tinybubbles/tinytea"
	viewporttea "nimbleterminal.dev/tinybubbles/tinytea/viewport"
)

type display struct{}

func (display) Size() (int16, int16)              { return 128, 64 }
func (display) SetPixel(int16, int16, color.RGBA) {}
func (display) Display() error                    { return nil }

type model struct{ *viewporttea.Model }

func (model) Init() tea.Cmd { return tea.Quit }
func main() {
	view := viewporttea.New("Device information\nScroll to read more.")
	p, err := tea.NewProgram(model{view}, display{}, nil, bubbles.DefaultTheme(true))
	if err != nil {
		panic(err)
	}
	if _, err = p.Run(); err != nil {
		panic(err)
	}
}
