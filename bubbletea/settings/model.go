// Package settings demonstrates focus and composition on a small display.
package settings

import (
	"fmt"
	"time"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	bubbles "nimbleterminal.dev/tinybubbles"
	choice "nimbleterminal.dev/tinybubbles/bubbletea/choice"
	dialog "nimbleterminal.dev/tinybubbles/bubbletea/dialog"
	list "nimbleterminal.dev/tinybubbles/bubbletea/list"
	value "nimbleterminal.dev/tinybubbles/bubbletea/value"
	"nimbleterminal.dev/tinybubbles/layout"
	cc "nimbleterminal.dev/tinybubbles/widgets/choice"
	dc "nimbleterminal.dev/tinybubbles/widgets/dialog"
	lc "nimbleterminal.dev/tinybubbles/widgets/list"
	vc "nimbleterminal.dev/tinybubbles/widgets/value"
)

// FeatherButtons returns the three-button D0/D1/D2 configuration for the
// Reverse TFT Feather: D0 moves up or decreases (with hold-to-repeat), D1
// moves down or increases (with hold-to-repeat), and D2 selects/applies on a
// short tap or cancels on a 500ms long press.
func FeatherButtons() [3]bubbles.Button {
	return [3]bubbles.Button{
		{
			Key:            bubbles.KeyUp,
			Debounce:       25 * time.Millisecond,
			RepeatDelay:    400 * time.Millisecond,
			RepeatInterval: 100 * time.Millisecond,
		},
		{
			Key:            bubbles.KeyDown,
			Debounce:       25 * time.Millisecond,
			RepeatDelay:    400 * time.Millisecond,
			RepeatInterval: 100 * time.Millisecond,
		},
		{
			Key:       bubbles.KeySelect,
			LongKey:   bubbles.KeyBack,
			Debounce:  25 * time.Millisecond,
			LongPress: 500 * time.Millisecond,
		},
	}
}

// Values contains the settings applied during this session. Playback is an option index.
type Values struct {
	Volume   int32
	Sound    bool
	Playback int
}

// Model owns focus and routes child results back to the settings menu.
type Model struct {
	values          Values
	menu            *list.Model
	volume          *value.Model
	sound, playback *choice.Model
	reset           *dialog.Model
	focus           string
	pending         bool
	buffer          *bubbles.Buffer
}
type childResult struct{ message tea.Msg }

// New starts an in-memory settings session with volume 50, sound on, and single playback.
func New() *Model {
	volume, _ := value.New(vc.Config{Label: "Volume", Unit: "%", Min: 0, Max: 100, Step: 5, Value: 50})
	sound, _ := choice.New(cc.Config{Label: "Sound", Options: []string{"Off", "On"}, Value: 1})
	playback, _ := choice.New(cc.Config{Label: "Playback", Options: []string{"Single", "Repeat", "Shuffle"}})
	m := &Model{values: Values{50, true, 0}, menu: list.New(nil), volume: volume, sound: sound, playback: playback, reset: dialog.New(dc.Config{Title: "Reset settings?", Body: "Restore default volume, sound and playback?", Confirm: "Reset", Cancel: "Keep settings"}), buffer: bubbles.NewBuffer(0, 0)}
	m.refresh()
	return m
}
func (m *Model) Values() Values { return m.values }
func (*Model) Init() tea.Cmd    { return nil }
func (*Model) View() tea.View   { return tea.NewView("") }
func (m *Model) refresh() {
	sound := "Off"
	if m.values.Sound {
		sound = "On"
	}
	m.menu.SetItems([]lc.Item{{ID: "volume", Label: fmt.Sprintf("Volume: %d%%", m.values.Volume)}, {ID: "sound", Label: "Sound: " + sound}, {ID: "playback", Label: "Play: " + m.playback.Label()}, {ID: "reset", Label: "Reset defaults"}})
}
func (m *Model) active() tea.Model {
	switch m.focus {
	case "volume":
		return m.volume
	case "sound":
		return m.sound
	case "playback":
		return m.playback
	case "reset":
		return m.reset
	}
	return m.menu
}
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		w, h := m.buffer.Size()
		if w != msg.Width || h != msg.Height {
			m.buffer = bubbles.NewBuffer(msg.Width, msg.Height)
		}
		_, body, _ := menuRegions(layout.Bounds(m.buffer))
		m.menu.SetSize(body.Width, body.Height)
		m.volume.SetSize(msg.Width, msg.Height)
		m.sound.SetSize(msg.Width, msg.Height)
		m.playback.SetSize(msg.Width, msg.Height)
		m.reset.SetSize(msg.Width, msg.Height)
		return m, nil
	case childResult:
		if !m.pending {
			return m, nil
		}
		m.pending = false
		switch result := msg.message.(type) {
		case list.SelectedMsg:
			m.focus = result.Item.ID
			if m.focus == "reset" {
				m.reset.Reset()
			}
			return m, nil
		case value.AppliedMsg:
			m.values.Volume = result.Value
		case choice.AppliedMsg:
			if m.focus == "sound" {
				m.values.Sound = result.Index == 1
			} else {
				m.values.Playback = result.Index
			}
		case dialog.ResultMsg:
			if result.Confirmed {
				m.values = Values{50, true, 0}
				m.volume.SetValue(50)
				m.sound.SetValue(1)
				m.playback.SetValue(0)
			}
		}
		m.focus = ""
		m.refresh()
		return m, nil
	case tea.KeyPressMsg:
		if msg.Code == 'q' {
			return m, tea.Quit
		}
		// Allow a single pair of directional buttons (Up/Down or Left/Right)
		// to navigate both the vertical menu and horizontal child pickers.
		if m.focus == "" {
			switch msg.Code {
			case tea.KeyLeft:
				msg = tea.KeyPressMsg{Code: tea.KeyUp}
			case tea.KeyRight:
				msg = tea.KeyPressMsg{Code: tea.KeyDown}
			}
		} else {
			switch msg.Code {
			case tea.KeyUp:
				msg = tea.KeyPressMsg{Code: tea.KeyLeft}
			case tea.KeyDown:
				msg = tea.KeyPressMsg{Code: tea.KeyRight}
			}
		}
		if m.pending {
			return m, nil
		}
		_, cmd := m.active().Update(msg)
		if cmd != nil {
			m.pending = true
			return m, func() tea.Msg { return childResult{cmd()} }
		}
		return m, nil
	}
	if m.pending {
		return m, nil
	}
	_, cmd := m.active().Update(msg)
	if cmd != nil {
		m.pending = true
		return m, func() tea.Msg { return childResult{cmd()} }
	}
	return m, nil
}
func (m *Model) CellView() *bubbles.Buffer {
	if m.focus != "" {
		return m.active().(interface{ CellView() *bubbles.Buffer }).CellView()
	}
	b := m.buffer
	b.Clear(bubbles.Normal)
	header, body, footer := menuRegions(layout.Bounds(b))
	layout.Text(b, header, "SETTINGS", bubbles.Accent)
	layout.Draw(b, body, m.menu.CellView())
	layout.Text(b, footer, "Choose a setting", bubbles.Muted)
	return b
}

// Reserve header and footer first; the menu receives the remaining rows.
func menuRegions(bounds layout.Rect) (header, body, footer layout.Rect) {
	header, rest := bounds.Rows(1, 0)
	body, footer = rest.Rows(max(0, rest.Height-1), 0)
	return
}

// HelpMap is the Bubbles help contract without importing its terminal renderer.
type HelpMap interface {
	ShortHelp() []key.Binding
	FullHelp() [][]key.Binding
}

type bindings struct {
	short []key.Binding
	full  [][]key.Binding
}

func (b bindings) ShortHelp() []key.Binding  { return b.short }
func (b bindings) FullHelp() [][]key.Binding { return b.full }
func (m *Model) Keys() HelpMap {
	var k HelpMap
	switch m.focus {
	case "volume":
		k = m.volume.Keys()
	case "sound":
		k = m.sound.Keys()
	case "playback":
		k = m.playback.Keys()
	case "reset":
		k = m.reset.Keys()
	default:
		k = m.menu.Keys()
	}
	b := bindings{k.ShortHelp(), k.FullHelp()}
	if m.pending {
		for i := range b.short {
			b.short[i].SetEnabled(false)
		}
		for i := range b.full {
			for j := range b.full[i] {
				b.full[i][j].SetEnabled(false)
			}
		}
	}
	b.full = append(b.full, []key.Binding{key.NewBinding(key.WithKeys("q"), key.WithHelp("q", "Quit"))})
	return b
}
