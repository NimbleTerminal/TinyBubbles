//go:build tinygo

// Compile the progress bar with the cooperative runtime on a microcontroller.
package main

import (
	"image/color"
	bubbles "nimbleterminal.dev/tinybubbles"
	tea "nimbleterminal.dev/tinybubbles/tinytea"
	progresstea "nimbleterminal.dev/tinybubbles/tinytea/progress"
	"nimbleterminal.dev/tinybubbles/widgets/progress"
)

type display struct{}

func (display) Size() (int16, int16)              { return 128, 64 }
func (display) SetPixel(int16, int16, color.RGBA) {}
func (display) Display() error                    { return nil }

type model struct{ *progresstea.Model }

func (model) Init() tea.Cmd { return tea.Quit }
func main() {
	bar := progresstea.New(progress.Config{Label: "Loading", Done: 30, Total: 100})
	p, err := tea.NewProgram(model{bar}, display{}, nil, bubbles.DefaultTheme(true))
	if err != nil {
		panic(err)
	}
	if _, err = p.Run(); err != nil {
		panic(err)
	}
}
