package hello

import (
	"image"
	"image/color"
	bubbles "nimbleterminal.dev/tinybubbles"
	"reflect"
	"testing"
)

func TestAnimationAndControls(t *testing.T) {
	m := New()
	m.Update(bubbles.WindowSizeMsg{Width: 40, Height: 16})
	initial := m.bubbles
	for i := 0; i < 10; i++ {
		m.Update(TickMsg{})
	}
	if m.bubbles == initial {
		t.Fatal("animation did not move")
	}
	m.Update(bubbles.KeyMsg{Key: bubbles.KeyBack})
	paused := m.bubbles
	for i := 0; i < 10; i++ {
		m.Update(TickMsg{})
	}
	if m.bubbles != paused {
		t.Fatal("pause moved bubbles")
	}
	m.Update(bubbles.KeyMsg{Key: bubbles.KeySelect})
	if m.popped != 1 || m.bubbles[0].burst != 5 {
		t.Fatal("pop while paused")
	}
	m.Update(bubbles.KeyMsg{Key: bubbles.KeyBack})
	for i := 0; i < 5; i++ {
		m.Update(TickMsg{})
	}
	if m.bubbles[0].burst != 0 || m.bubbles[0].y != 11 {
		t.Fatal("burst did not respawn")
	}
	for i := 0; i < 4; i++ {
		m.Update(bubbles.KeyMsg{Key: bubbles.KeyRight})
	}
	if m.palette != 0 {
		t.Fatal("palette failed to wrap")
	}
	grid := m.View()
	state := *m
	m.View()
	if !reflect.DeepEqual(*m, state) || grid != m.View() {
		t.Fatal("View mutated animation or allocated a new grid")
	}
}

func TestLongRunAndSmallScreens(t *testing.T) {
	m := New()
	m.Update(TickMsg{})
	m.Update(bubbles.KeyMsg{Key: bubbles.KeySelect})
	if m.View() != nil {
		t.Fatal("uninitialized view")
	}
	for _, size := range [][2]int{{40, 16}, {21, 8}, {1, 1}, {0, 0}} {
		m.Update(bubbles.WindowSizeMsg{Width: size[0], Height: size[1]})
		for i := 0; i < 1000; i++ {
			m.Update(TickMsg{})
			m.View()
			if size[0] == 40 {
				for _, b := range m.bubbles {
					if b.x < 0 || b.x > 38 || b.y < 2 || b.y > 11 {
						t.Fatal(b)
					}
				}
			}
		}
	}
}

type testDisplay struct{ *image.RGBA }

func (d testDisplay) Size() (int16, int16) { return int16(d.Rect.Dx()), int16(d.Rect.Dy()) }
func (d testDisplay) SetPixel(x, y int16, c color.RGBA) {
	if !image.Pt(int(x), int(y)).In(d.Rect) {
		panic("pixel outside screen")
	}
	d.SetRGBA(int(x), int(y), c)
}
func (d testDisplay) Display() error { return nil }

func TestLargeGlyphTilesMatchDoubledFont(t *testing.T) {
	theme := Theme()
	d := testDisplay{image.NewRGBA(image.Rect(0, 0, 12, 16))}
	r, err := bubbles.NewRenderer(d, NewFont(), theme)
	if err != nil {
		t.Fatal(err)
	}
	g := bubbles.NewBuffer(2, 2)
	largeText(g, 0, 0, "H", bubbles.Normal)
	if err = r.Render(g); err != nil {
		t.Fatal(err)
	}
	base := testDisplay{image.NewRGBA(image.Rect(0, 0, 6, 8))}
	for y := 0; y < 8; y++ {
		for x := 0; x < 6; x++ {
			base.SetRGBA(x, y, theme.Styles[0].Background)
		}
	}
	bubbles.DefaultFont().DrawGlyph(base, 0, 0, 'H', theme.Styles[0].Foreground)
	for y := 0; y < 16; y++ {
		for x := 0; x < 12; x++ {
			if d.RGBAAt(x, y) != base.RGBAAt(x/2, y/2) {
				t.Fatalf("tile seam at %d,%d", x, y)
			}
		}
	}
}

func TestActualHelloRendering(t *testing.T) {
	d := testDisplay{image.NewRGBA(image.Rect(0, 0, 240, 135))}
	theme := Theme()
	p, err := bubbles.NewProgram(New(), bubbles.Config{Display: d, Font: NewFont(), Theme: &theme})
	if err != nil {
		t.Fatal(err)
	}
	if err = p.Start(); err != nil {
		t.Fatal(err)
	}
	before := append([]byte(nil), d.Pix...)
	for i := 0; i < 10; i++ {
		if err = p.Send(TickMsg{}); err != nil {
			t.Fatal(err)
		}
	}
	if reflect.DeepEqual(before, d.Pix) {
		t.Fatal("animation did not render")
	}
}
