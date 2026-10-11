package web

import (
	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	bubbles "nimbleterminal.dev/tinybubbles"
	displaytea "nimbleterminal.dev/tinybubbles/bubbletea"
	dialogtea "nimbleterminal.dev/tinybubbles/bubbletea/dialog"
	browser "nimbleterminal.dev/tinybubbles/web"
	"nimbleterminal.dev/tinybubbles/widgets/dialog"
	"tinygo.org/x/drivers"
)

type dialogDemo struct {
	dialog *dialogtea.Model
	buffer *bubbles.Buffer
	result string
}

func newDialogDemo() *dialogDemo {
	return &dialogDemo{dialog: dialogtea.New(dialog.Config{
		Title: "Reset settings?", Body: "Restore the default volume and brightness?",
		Confirm: "Reset", Cancel: "Keep settings",
	}), buffer: bubbles.NewBuffer(0, 0)}
}
func (*dialogDemo) Init() tea.Cmd  { return nil }
func (*dialogDemo) View() tea.View { return tea.NewView("") }
func (m *dialogDemo) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.buffer = bubbles.NewBuffer(msg.Width, msg.Height)
		m.dialog.SetSize(msg.Width, msg.Height)
		return m, nil
	case dialogtea.ResultMsg:
		m.result = "Settings kept"
		if msg.Confirmed {
			m.result = "Defaults restored"
		}
		return m, nil
	case tea.KeyPressMsg:
		if msg.Code == 'q' {
			return m, tea.Quit
		}
		if m.result != "" && msg.Code == tea.KeyEnter {
			m.dialog.Reset()
			m.result = ""
			return m, nil
		}
	}
	if m.result != "" {
		return m, nil
	}
	_, cmd := m.dialog.Update(msg)
	return m, cmd
}
func (m *dialogDemo) CellView() *bubbles.Buffer {
	if m.result == "" {
		return m.dialog.CellView()
	}
	m.buffer.Clear(bubbles.Normal)
	m.buffer.Text(0, 0, m.result, bubbles.Accent)
	m.buffer.Text(0, 2, "Demo only", bubbles.Muted)
	m.buffer.Text(0, 4, "Enter: ask again", bubbles.Normal)
	return m.buffer
}

type dialogDemoKeys struct {
	short []key.Binding
	full  [][]key.Binding
}

func (k dialogDemoKeys) ShortHelp() []key.Binding  { return k.short }
func (k dialogDemoKeys) FullHelp() [][]key.Binding { return k.full }
func (m *dialogDemo) Keys() help.KeyMap {
	quit := key.NewBinding(key.WithKeys("q"), key.WithHelp("q", "Quit"))
	if m.result != "" {
		again := key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "Ask again"))
		return dialogDemoKeys{[]key.Binding{again}, [][]key.Binding{{again}, {quit}}}
	}
	keys := m.dialog.Keys()
	return dialogDemoKeys{keys.ShortHelp(), [][]key.Binding{keys.ShortHelp(), {keys.Previous, quit}}}
}
func NewDialogDemo(display drivers.Displayer, monochrome bool) (tea.Model, *displaytea.DisplayModel, error) {
	pixels, err := displaytea.NewModel(newDialogDemo(), display, bubbles.DefaultFont(), bubbles.DefaultTheme(monochrome))
	if err != nil {
		return nil, nil, err
	}
	wrapped, err := browser.WrapActions(pixels, "dialog", func(model tea.Model) help.KeyMap {
		return model.(*displaytea.DisplayModel).Model().(*dialogDemo).Keys()
	}, nil)
	if err != nil {
		pixels.Close()
		return nil, nil, err
	}
	return wrapped, pixels, nil
}
