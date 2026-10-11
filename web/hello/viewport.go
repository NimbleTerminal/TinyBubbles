package web

import (
	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	bubbles "nimbleterminal.dev/tinybubbles"
	displaytea "nimbleterminal.dev/tinybubbles/bubbletea"
	viewporttea "nimbleterminal.dev/tinybubbles/bubbletea/viewport"
	browser "nimbleterminal.dev/tinybubbles/web"
	"tinygo.org/x/drivers"
)

const viewportText = `TINY DEVICE GUIDE

Read a long document on a small screen. Up and Down move one row. Left and Right move a page with one row of overlap.

DISPLAY
The same Go model draws on an OLED, a color TFT, or this browser canvas. Select another display preset to try a different size.

BUTTONS
A three-button device can map Up, Down, and Back. The parent owns Back and decides which screen to open next.

TEXT
Blank lines and indentation are preserved. Long lines wrap at cell boundaries. Each rune occupies one cell using the display's font.

DEVICE LOG
  00:01 power ready
  00:02 display ready
  00:03 input ready
  00:04 app started

MEMORY
Keep documents small on embedded devices. The viewport stores the text, line boundaries, and its visible cell grid. It does not allocate a grid for the entire document.

END OF GUIDE
You reached the bottom. Use Top under More to start again.`

type viewportDemo struct{ *viewporttea.Model }

func (m *viewportDemo) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if k, ok := msg.(tea.KeyPressMsg); ok && key.Matches(k, viewportQuit()) {
		return m, tea.Quit
	}
	_, cmd := m.Model.Update(msg)
	return m, cmd
}
func viewportQuit() key.Binding { return key.NewBinding(key.WithKeys("q"), key.WithHelp("q", "Quit")) }

type viewportDemoKeys struct{ viewporttea.KeyMap }

func (k viewportDemoKeys) FullHelp() [][]key.Binding {
	return append(k.KeyMap.FullHelp(), []key.Binding{viewportQuit()})
}
func NewViewportDemo(display drivers.Displayer, mono bool) (tea.Model, *displaytea.DisplayModel, error) {
	pixels, err := displaytea.NewModel(&viewportDemo{viewporttea.New(viewportText)}, display, bubbles.DefaultFont(), bubbles.DefaultTheme(mono))
	if err != nil {
		return nil, nil, err
	}
	wrapped, err := browser.WrapActions(pixels, "viewport", func(model tea.Model) help.KeyMap {
		return viewportDemoKeys{model.(*displaytea.DisplayModel).Model().(*viewportDemo).Keys()}
	}, nil)
	if err != nil {
		pixels.Close()
		return nil, nil, err
	}
	return wrapped, pixels, nil
}
