package web

import (
	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	bubbles "nimbleterminal.dev/tinybubbles"
	displaytea "nimbleterminal.dev/tinybubbles/bubbletea"
	progresstea "nimbleterminal.dev/tinybubbles/bubbletea/progress"
	browser "nimbleterminal.dev/tinybubbles/web"
	"nimbleterminal.dev/tinybubbles/widgets/progress"
	"time"
	"tinygo.org/x/drivers"
)

type progressTick struct{ generation uint64 }
type progressDemo struct {
	*progresstea.Model
	paused     bool
	generation uint64
}

func newProgressDemo() *progressDemo {
	return &progressDemo{Model: progresstea.New(progress.Config{Label: "Loading", Total: 100})}
}
func (m *progressDemo) tick() tea.Cmd {
	g := m.generation
	return tea.Tick(200*time.Millisecond, func(time.Time) tea.Msg { return progressTick{g} })
}
func (m *progressDemo) Init() tea.Cmd { return m.tick() }
func (m *progressDemo) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case progressTick:
		if msg.generation != m.generation || m.paused || m.Complete() {
			return m, nil
		}
		if m.Indeterminate() {
			m.Advance()
		} else {
			m.SetProgress(m.Done()+5, m.Total())
		}
		if !m.Complete() {
			return m, m.tick()
		}
		return m, nil
	case tea.KeyPressMsg:
		keys := m.keys()
		switch {
		case key.Matches(msg, keys.quit):
			return m, tea.Quit
		case key.Matches(msg, keys.pause):
			m.paused = !m.paused
		case key.Matches(msg, keys.restart):
			m.SetProgress(0, m.Total())
			m.paused = false
		case key.Matches(msg, keys.mode):
			if m.Indeterminate() {
				m.SetProgress(0, 100)
			} else {
				m.SetProgress(0, 0)
			}
			m.paused = false
		default:
			return m, nil
		}
		m.generation++
		if !m.paused && !m.Complete() {
			return m, m.tick()
		}
		return m, nil
	}
	m.Model.Update(msg)
	return m, nil
}

type progressKeys struct{ pause, restart, mode, quit key.Binding }

func (k progressKeys) ShortHelp() []key.Binding  { return []key.Binding{k.pause, k.restart, k.mode} }
func (k progressKeys) FullHelp() [][]key.Binding { return [][]key.Binding{k.ShortHelp(), {k.quit}} }
func (m *progressDemo) keys() progressKeys {
	label := "Pause"
	if m.paused {
		label = "Resume"
	}
	mode := "Unknown total"
	if m.Indeterminate() {
		mode = "Known total"
	}
	k := progressKeys{
		pause:   key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", label)),
		restart: key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "Start over")),
		mode:    key.NewBinding(key.WithKeys("m"), key.WithHelp("m", mode)),
		quit:    key.NewBinding(key.WithKeys("q"), key.WithHelp("q", "Quit")),
	}
	k.pause.SetEnabled(!m.Complete())
	return k
}
func NewProgressDemo(display drivers.Displayer, monochrome bool) (tea.Model, *displaytea.DisplayModel, error) {
	pixels, err := displaytea.NewModel(newProgressDemo(), display, bubbles.DefaultFont(), bubbles.DefaultTheme(monochrome))
	if err != nil {
		return nil, nil, err
	}
	wrapped, err := browser.WrapActions(pixels, "progress", func(model tea.Model) help.KeyMap {
		return model.(*displaytea.DisplayModel).Model().(*progressDemo).keys()
	}, nil)
	if err != nil {
		pixels.Close()
		return nil, nil, err
	}
	return wrapped, pixels, nil
}
