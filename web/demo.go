// Package web assembles the browser demo without adding browser dependencies to firmware.
package web

import (
	tea "charm.land/bubbletea/v2"
	"github.com/NimbleMarkets/go-booba/actions"
	"github.com/NimbleMarkets/go-booba/actions/bubblehelp"
	bubbles "nimbleterminal.dev/tinybubbles"
	displaytea "nimbleterminal.dev/tinybubbles/bubbletea"
	"nimbleterminal.dev/tinybubbles/bubbletea/hello"
	greeting "nimbleterminal.dev/tinybubbles/demo/hello"
	"tinygo.org/x/drivers"
)

// NewDemo uses the device model and renderer, with help metadata published by
// Booba over the program's ordinary output channel. No help UI is rendered.
func NewDemo(display drivers.Displayer, monochrome bool) (tea.Model, *displaytea.DisplayModel, error) {
	m := hello.New()
	m.HideHelp(true)
	m.SetColorEnabled(!monochrome)
	theme := greeting.Theme()
	if monochrome {
		theme = bubbles.DefaultTheme(true)
	}
	pixels, err := displaytea.NewModel(m, display, greeting.NewFont(), theme)
	if err != nil {
		return nil, nil, err
	}
	wrapped, err := bubblehelp.NewModel(pixels, func(model tea.Model) bubblehelp.Description {
		helloModel := model.(*displaytea.DisplayModel).Model().(*hello.Model)
		return bubblehelp.Description{Context: "hello", Keys: helloModel.Keys(), Status: actions.ProgramStatus{Summary: "Hello, tiny world"}}
	}, bubblehelp.ModelOptions{Instance: "tinybubbles-hello", Adapter: bubblehelp.Options{IncludeDisabled: true}})
	if err != nil {
		pixels.Close()
		return nil, nil, err
	}
	return wrapped, pixels, nil
}
