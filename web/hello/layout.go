package web

import (
	"fmt"
	"image/color"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	bubbles "nimbleterminal.dev/tinybubbles"
	displaytea "nimbleterminal.dev/tinybubbles/bubbletea"
	"nimbleterminal.dev/tinybubbles/layout"
	browser "nimbleterminal.dev/tinybubbles/web"
	"tinygo.org/x/drivers"
)

// layoutDemo lets the parent choose a layout; the helpers only divide cell regions.
type layoutDemo struct {
	buffer             *bubbles.Buffer
	tiles              [2]*bubbles.Buffer
	panels             [2]layout.Rect
	header, footer     layout.Rect
	mode, padding, gap int
	alignment          layout.Alignment
	title, caption     string
}

func newLayoutDemo() *layoutDemo {
	m := &layoutDemo{buffer: bubbles.NewBuffer(0, 0), padding: 1, gap: 1, alignment: layout.Center}
	for i, label := range []string{" A ", " B "} {
		m.tiles[i] = bubbles.NewBuffer(3, 1)
		m.tiles[i].Clear(bubbles.Selected)
		layout.Text(m.tiles[i], layout.Bounds(m.tiles[i]), label, bubbles.Selected)
	}
	m.reflow()
	return m
}
func (*layoutDemo) Init() tea.Cmd  { return nil }
func (*layoutDemo) View() tea.View { return tea.NewView("") }
func (m *layoutDemo) reflow() {
	var rest, body layout.Rect
	m.header, rest = layout.Bounds(m.buffer).Rows(1, 0)
	body, m.footer = rest.Rows(max(0, rest.Height-1), 0)
	body = body.Inset(m.padding, m.padding, m.padding, m.padding)
	columns := m.mode == 2 || m.mode == 0 && body.Width >= 32
	direction := "ROWS"
	if columns {
		direction = "COLUMNS"
		m.panels[0], m.panels[1] = body.Columns(max(0, body.Width-m.gap)/2, m.gap)
	} else {
		m.panels[0], m.panels[1] = body.Rows(max(0, body.Height-m.gap)/2, m.gap)
	}
	mode := [3]string{"AUTO", "ROWS", "COLUMNS"}[m.mode]
	m.title = mode
	if m.mode == 0 {
		m.title += " / " + direction
	}
	m.caption = fmt.Sprintf("P%d G%d %s", m.padding, m.gap, [3]string{"START", "CENTER", "END"}[m.alignment])
}
func (m *layoutDemo) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		w, h := m.buffer.Size()
		if w != msg.Width || h != msg.Height {
			m.buffer = bubbles.NewBuffer(msg.Width, msg.Height)
		}
	case tea.KeyPressMsg:
		k := layoutKeys()
		switch {
		case key.Matches(msg, k.mode):
			m.mode = (m.mode + 1) % 3
		case key.Matches(msg, k.padding):
			m.padding = 1 - m.padding
		case key.Matches(msg, k.align):
			m.alignment = (m.alignment + 1) % 3
		case key.Matches(msg, k.gap):
			m.gap = 1 - m.gap
		case key.Matches(msg, k.reset):
			m.mode = 0
			m.padding = 1
			m.gap = 1
			m.alignment = layout.Center
		case key.Matches(msg, k.quit):
			return m, tea.Quit
		default:
			return m, nil
		}
	default:
		return m, nil
	}
	m.reflow()
	return m, nil
}
func (m *layoutDemo) CellView() *bubbles.Buffer {
	b := m.buffer
	b.Clear(bubbles.Normal)
	layout.Text(b, m.header, m.title, bubbles.Accent)
	for i, r := range m.panels {
		// Visible patterns show region bounds on monochrome screens too.
		for y := 0; y < r.Height; y++ {
			for x := 0; x < r.Width; x++ {
				b.Set(r.X+x, r.Y+y, bubbles.Cell{Rune: [2]rune{'.', ':'}[i], Style: bubbles.StyleID(4 + i)})
			}
		}
		layout.Draw(b, r.Align(3, 1, m.alignment, m.alignment), m.tiles[i])
	}
	layout.Text(b, m.footer, m.caption, bubbles.Muted)
	return b
}

type layoutDemoKeys struct{ mode, padding, align, gap, reset, quit key.Binding }

func (k layoutDemoKeys) ShortHelp() []key.Binding { return []key.Binding{k.mode, k.padding, k.align} }
func (k layoutDemoKeys) FullHelp() [][]key.Binding {
	return [][]key.Binding{k.ShortHelp(), {k.gap, k.reset, k.quit}}
}
func layoutKeys() layoutDemoKeys {
	return layoutDemoKeys{
		mode:    key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "Arrangement")),
		padding: key.NewBinding(key.WithKeys("p"), key.WithHelp("p", "Padding")),
		align:   key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "Alignment")),
		gap:     key.NewBinding(key.WithKeys("g"), key.WithHelp("g", "Gap")),
		reset:   key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "Reset")),
		quit:    key.NewBinding(key.WithKeys("q"), key.WithHelp("q", "Quit")),
	}
}
func NewLayoutDemo(display drivers.Displayer, mono bool) (tea.Model, *displaytea.DisplayModel, error) {
	theme := bubbles.DefaultTheme(mono)
	if !mono {
		theme.Styles[4] = bubbles.Style{Foreground: color.RGBA{60, 110, 140, 255}, Background: color.RGBA{12, 28, 40, 255}}
		theme.Styles[5] = bubbles.Style{Foreground: color.RGBA{100, 130, 85, 255}, Background: color.RGBA{20, 34, 16, 255}}
	}
	pixels, err := displaytea.NewModel(newLayoutDemo(), display, bubbles.DefaultFont(), theme)
	if err != nil {
		return nil, nil, err
	}
	wrapped, err := browser.WrapActions(pixels, "layout", func(tea.Model) help.KeyMap { return layoutKeys() }, nil)
	if err != nil {
		pixels.Close()
		return nil, nil, err
	}
	return wrapped, pixels, nil
}
