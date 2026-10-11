package hello

import "charm.land/bubbles/v2/key"

// KeyMap implements help.KeyMap without requiring the help renderer on devices.
// The same bindings drive Update, browser actions, and optional help views.
type KeyMap struct{ Pop, Color, Pause, Quit key.Binding }

func DefaultKeys() KeyMap {
	return KeyMap{
		Pop:   key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "Pop")),
		Color: key.NewBinding(key.WithKeys("right"), key.WithHelp("right", "Color")),
		Pause: key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "Pause")),
		Quit:  key.NewBinding(key.WithKeys("q"), key.WithHelp("q", "Quit")),
	}
}
func (k KeyMap) ShortHelp() []key.Binding  { return []key.Binding{k.Pop, k.Color, k.Pause} }
func (k KeyMap) FullHelp() [][]key.Binding { return [][]key.Binding{k.ShortHelp(), {k.Quit}} }

// SetColorEnabled disables palette switching when the display is monochrome.
func (m *Model) SetColorEnabled(enabled bool) { m.keys.Color.SetEnabled(enabled) }
