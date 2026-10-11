package web

import (
	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"fmt"
	bubbles "nimbleterminal.dev/tinybubbles"
	displaytea "nimbleterminal.dev/tinybubbles/bubbletea"
	valuetea "nimbleterminal.dev/tinybubbles/bubbletea/value"
	browser "nimbleterminal.dev/tinybubbles/web"
	"nimbleterminal.dev/tinybubbles/widgets/value"
	"tinygo.org/x/drivers"
)

type valueDemo struct {
	picker *valuetea.Model
	buffer *bubbles.Buffer
	status string
}

func newValueDemo() (*valueDemo, error) {
	picker, err := valuetea.New(value.Config{Label: "Volume", Unit: "%", Min: 0, Max: 100, Step: 5, Value: 50})
	if err != nil {
		return nil, err
	}
	return &valueDemo{picker: picker, buffer: bubbles.NewBuffer(0, 0), status: "Adjust, then apply"}, nil
}
func (*valueDemo) Init() tea.Cmd  { return nil }
func (*valueDemo) View() tea.View { return tea.NewView("") }
func (m *valueDemo) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.buffer = bubbles.NewBuffer(msg.Width, msg.Height)
		m.picker.SetSize(msg.Width, max(0, msg.Height-1))
		return m, nil
	case valuetea.AppliedMsg:
		m.status = fmt.Sprintf("Applied: %d%%", msg.Value)
		return m, nil
	case valuetea.CancelledMsg:
		m.status = fmt.Sprintf("Restored: %d%%", msg.Value)
		return m, nil
	case tea.KeyPressMsg:
		if msg.Code == 'q' {
			return m, tea.Quit
		}
	}
	_, cmd := m.picker.Update(msg)
	return m, cmd
}
func (m *valueDemo) CellView() *bubbles.Buffer {
	m.buffer.Clear(bubbles.Normal)
	w, h := m.buffer.Size()
	view := m.picker.CellView()
	for y := 0; y < h-1; y++ {
		for x := 0; x < w; x++ {
			m.buffer.Set(x, y, view.At(x, y))
		}
	}
	if h > 0 {
		m.buffer.Text(0, h-1, m.status, bubbles.Muted)
	}
	return m.buffer
}

type valueDemoKeys struct{ valuetea.KeyMap }

func (k valueDemoKeys) FullHelp() [][]key.Binding {
	return [][]key.Binding{k.ShortHelp(), {k.Cancel, key.NewBinding(key.WithKeys("q"), key.WithHelp("q", "Quit"))}}
}
func NewValueDemo(display drivers.Displayer, monochrome bool) (tea.Model, *displaytea.DisplayModel, error) {
	m, err := newValueDemo()
	if err != nil {
		return nil, nil, err
	}
	pixels, err := displaytea.NewModel(m, display, bubbles.DefaultFont(), bubbles.DefaultTheme(monochrome))
	if err != nil {
		return nil, nil, err
	}
	wrapped, err := browser.WrapActions(pixels, "value", func(model tea.Model) help.KeyMap {
		return valueDemoKeys{model.(*displaytea.DisplayModel).Model().(*valueDemo).picker.Keys()}
	}, nil)
	if err != nil {
		pixels.Close()
		return nil, nil, err
	}
	return wrapped, pixels, nil
}
