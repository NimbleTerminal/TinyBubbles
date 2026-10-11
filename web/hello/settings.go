package web

import (
	"charm.land/bubbles/v2/help"
	tea "charm.land/bubbletea/v2"
	bubbles "nimbleterminal.dev/tinybubbles"
	displaytea "nimbleterminal.dev/tinybubbles/bubbletea"
	"nimbleterminal.dev/tinybubbles/bubbletea/settings"
	browser "nimbleterminal.dev/tinybubbles/web"
	"tinygo.org/x/drivers"
)

func NewSettingsDemo(display drivers.Displayer, monochrome bool) (tea.Model, *displaytea.DisplayModel, error) {
	pixels, err := displaytea.NewModel(settings.New(), display, bubbles.DefaultFont(), bubbles.DefaultTheme(monochrome))
	if err != nil {
		return nil, nil, err
	}
	wrapped, err := browser.WrapActions(pixels, "settings", func(model tea.Model) help.KeyMap {
		return model.(*displaytea.DisplayModel).Model().(*settings.Model).Keys()
	}, nil)
	if err != nil {
		pixels.Close()
		return nil, nil, err
	}
	return wrapped, pixels, nil
}
