package web

import (
	"fmt"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	bubbles "nimbleterminal.dev/tinybubbles"
	displaytea "nimbleterminal.dev/tinybubbles/bubbletea"
	listtea "nimbleterminal.dev/tinybubbles/bubbletea/list"
	browser "nimbleterminal.dev/tinybubbles/web"
	"nimbleterminal.dev/tinybubbles/widgets/list"
	"tinygo.org/x/drivers"
)

type listDemo struct {
	list   *listtea.Model
	buffer *bubbles.Buffer
	status string
}

func newListDemo() *listDemo {
	return &listDemo{list: listtea.New([]list.Item{
		{ID: "music", Label: "Music"},
		{ID: "brightness", Label: "Brightness"},
		{ID: "volume", Label: "Volume"},
		{ID: "wireless", Label: "Wireless (unavailable)", Disabled: true},
		{ID: "theme", Label: "Theme"},
		{ID: "timer", Label: "Sleep timer"},
		{ID: "buttons", Label: "Button mapping"},
		{ID: "battery", Label: "Battery"},
		{ID: "storage", Label: "Storage"},
		{ID: "display", Label: "Display calibration and orientation"},
		{ID: "about", Label: "About"},
		{ID: "restart", Label: "Restart"},
	}), buffer: bubbles.NewBuffer(0, 0)}
}
func (*listDemo) Init() tea.Cmd { return nil }
func (m *listDemo) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.buffer = bubbles.NewBuffer(msg.Width, msg.Height)
		m.list.SetSize(msg.Width, max(0, msg.Height-2))
		return m, nil
	case listtea.SelectedMsg:
		m.status = "Selected: " + msg.Item.Label
		return m, nil
	case tea.KeyPressMsg:
		if msg.Code == 'q' {
			return m, tea.Quit
		}
	}
	_, cmd := m.list.Update(msg)
	return m, cmd
}
func (*listDemo) View() tea.View { return tea.NewView("") }
func (m *listDemo) CellView() *bubbles.Buffer {
	m.buffer.Clear(bubbles.Normal)
	w, h := m.buffer.Size()
	m.buffer.Text(0, 0, fmt.Sprintf("MENU %d/%d", m.list.SelectedIndex()+1, m.list.Len()), bubbles.Accent)
	view := m.list.CellView()
	for y := 0; y < h-2; y++ {
		for x := 0; x < w; x++ {
			m.buffer.Set(x, y+1, view.At(x, y))
		}
	}
	if h > 1 {
		status := m.status
		if status == "" {
			status = "Choose an item"
		}
		m.buffer.Text(0, h-1, status, bubbles.Muted)
	}
	return m.buffer
}

type listDemoKeys struct{ listtea.KeyMap }

func (k listDemoKeys) FullHelp() [][]key.Binding {
	return [][]key.Binding{k.ShortHelp(), {key.NewBinding(key.WithKeys("q"), key.WithHelp("q", "Quit"))}}
}
func NewListDemo(display drivers.Displayer, monochrome bool) (tea.Model, *displaytea.DisplayModel, error) {
	pixels, err := displaytea.NewModel(newListDemo(), display, bubbles.DefaultFont(), bubbles.DefaultTheme(monochrome))
	if err != nil {
		return nil, nil, err
	}
	wrapped, err := browser.WrapActions(pixels, "menu", func(model tea.Model) help.KeyMap {
		return listDemoKeys{model.(*displaytea.DisplayModel).Model().(*listDemo).list.Keys()}
	}, nil)
	if err != nil {
		pixels.Close()
		return nil, nil, err
	}
	return wrapped, pixels, nil
}
