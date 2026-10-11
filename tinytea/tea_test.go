package tinytea

import (
	bubbles "github.com/NimbleTerminal/TinyBubbles"
	"image/color"
	"testing"

	"tinygo.org/x/drivers"
)

type display struct{ flushes int }

func (*display) Size() (int16, int16)              { return 120, 64 }
func (*display) SetPixel(int16, int16, color.RGBA) {}
func (d *display) Display() error                  { d.flushes++; return nil }

type countMsg struct{}
type model struct {
	width, count int
	ready        bool
}

func (m *model) Init() Cmd { return func() Msg { m.ready = m.width == 20; return countMsg{} } }
func (m *model) Update(msg Msg) (Model, Cmd) {
	switch msg := msg.(type) {
	case WindowSizeMsg:
		m.width = msg.Width
	case countMsg:
		m.count++
	case KeyPressMsg:
		if msg.Code == KeyEnter {
			m.count++
		}
		if msg.Code == 'q' {
			return m, Quit
		}
	}
	return m, nil
}
func (*model) View() View { return NewView("TinyTea") }
func TestCooperativeModelAndKeys(t *testing.T) {
	m := &model{}
	d := &display{}
	p, err := NewProgram(m, d, nil, bubbles.DefaultTheme(false))
	if err != nil {
		t.Fatal(err)
	}
	if err = p.Start(); err != nil {
		t.Fatal(err)
	}
	if m.count != 0 || m.width != 20 {
		t.Fatalf("unexpected Start: %+v", m)
	}
	if err = p.Step(); err != nil {
		t.Fatal(err)
	}
	if !m.ready || m.count != 1 {
		t.Fatalf("command ran before dimensions: %+v", m)
	}
	if err = p.Send(bubbles.KeyMsg{Key: bubbles.KeySelect}); err != nil {
		t.Fatal(err)
	}
	if m.count != 2 {
		t.Fatal("button not mapped to Enter")
	}
	if err = p.Send(KeyPressMsg{Code: 'q'}); err != nil {
		t.Fatal(err)
	}
	final, err := p.Run()
	if err != nil {
		t.Fatal(err)
	}
	if final != m || !p.Stopped() || d.flushes == 0 {
		t.Fatal("incorrect final state")
	}
}

type cellModel struct {
	model
	cells *bubbles.Buffer
}

func (m *cellModel) Init() Cmd                 { return nil }
func (m *cellModel) Update(Msg) (Model, Cmd)   { return m, nil }
func (m *cellModel) CellView() *bubbles.Buffer { return m.cells }

type font struct {
	glyph      rune
	foreground color.RGBA
}

func (*font) Size() (int16, int16) { return 6, 8 }
func (f *font) DrawGlyph(_ drivers.Displayer, _, _ int16, r rune, c color.RGBA) {
	f.glyph = r
	f.foreground = c
}
func TestCustomCellRendering(t *testing.T) {
	cells := bubbles.NewBuffer(20, 8)
	cells.Set(0, 0, bubbles.Cell{Rune: 'X', Style: bubbles.Accent})
	f := &font{}
	theme := bubbles.DefaultTheme(false)
	p, err := NewProgram(&cellModel{cells: cells}, &display{}, f, theme)
	if err != nil {
		t.Fatal(err)
	}
	if err = p.Step(); err != nil {
		t.Fatal(err)
	}
	if f.glyph != 'X' || f.foreground != theme.Styles[bubbles.Accent].Foreground {
		t.Fatalf("custom font/style lost: %+v", f)
	}
}
