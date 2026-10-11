//go:build tinygo

// Compile the status bar with the cooperative runtime on a microcontroller.
package main

import (
	"image/color"
	bubbles "nimbleterminal.dev/tinybubbles"
	tea "nimbleterminal.dev/tinybubbles/tinytea"
	statustea "nimbleterminal.dev/tinybubbles/tinytea/status"
	"nimbleterminal.dev/tinybubbles/widgets/status"
)

type display struct{}

func (display) Size() (int16, int16)              { return 128, 64 }
func (display) SetPixel(int16, int16, color.RGBA) {}
func (display) Display() error                    { return nil }

type model struct{ *statustea.Model }

func (model) Init() tea.Cmd { return tea.Quit }
func main() {
	bar := statustea.New(status.Config{Label: "Wi-Fi", State: status.Ready, Detail: "Connected"})
	p, err := tea.NewProgram(model{bar}, display{}, nil, bubbles.DefaultTheme(true))
	if err != nil {
		panic(err)
	}
	if _, err = p.Run(); err != nil {
		panic(err)
	}
}
