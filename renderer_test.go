package tinybubbles

import (
	"errors"
	"image/color"
	"testing"
	"tinygo.org/x/drivers"
)

type testDisplay struct {
	w, h            int16
	pixels          []color.RGBA
	writes, flushes int
	err             error
}

func newDisplay(w, h int16) *testDisplay {
	return &testDisplay{w: w, h: h, pixels: make([]color.RGBA, int(w)*int(h))}
}
func (d *testDisplay) Size() (int16, int16) { return d.w, d.h }
func (d *testDisplay) SetPixel(x, y int16, c color.RGBA) {
	if x < 0 || y < 0 || x >= d.w || y >= d.h {
		panic("out-of-bounds pixel")
	}
	d.writes++
	d.pixels[int(y)*int(d.w)+int(x)] = c
}
func (d *testDisplay) Display() error { d.flushes++; return d.err }

type fillDisplay struct {
	*testDisplay
	fills   int
	fillErr error
}

func (d *fillDisplay) FillRectangle(x, y, w, h int16, c color.RGBA) error {
	d.fills++
	if d.fillErr != nil {
		return d.fillErr
	}
	for yy := y; yy < y+h; yy++ {
		for xx := x; xx < x+w; xx++ {
			d.SetPixel(xx, yy, c)
		}
	}
	return nil
}

type testFont struct{}

func (testFont) Size() (int16, int16) { return 2, 3 }
func (testFont) DrawGlyph(d drivers.Displayer, x, y int16, r rune, c color.RGBA) {
	// Deliberately overflow all four cell edges to test the clipping adapter.
	for yy := y - 1; yy <= y+3; yy++ {
		for xx := x - 1; xx <= x+2; xx++ {
			d.SetPixel(xx, yy, c)
		}
	}
}
func makeRenderer(t *testing.T, d drivers.Displayer) *Renderer {
	t.Helper()
	r, err := NewRenderer(d, testFont{}, DefaultTheme(false))
	if err != nil {
		t.Fatal(err)
	}
	return r
}
func TestRenderDiffClippingAndRemoval(t *testing.T) {
	d := newDisplay(5, 7)
	r := makeRenderer(t, d)
	b := NewBuffer(2, 2)
	b.Set(0, 0, Cell{'X', Accent})
	if err := r.Render(b); err != nil {
		t.Fatal(err)
	}
	fg := DefaultTheme(false).Styles[Accent].Foreground
	bg := DefaultTheme(false).Styles[Normal].Background
	for y := 0; y < 7; y++ {
		for x := 0; x < 5; x++ {
			want := bg
			if x < 2 && y < 3 {
				want = fg
			}
			if d.pixels[y*5+x] != want {
				t.Fatalf("pixel %d,%d", x, y)
			}
		}
	}
	writes := d.writes
	if err := r.Render(b); err != nil {
		t.Fatal(err)
	}
	if d.writes != writes || d.flushes != 1 {
		t.Fatal("unchanged frame touched display")
	}
	b.Clear(Normal)
	if err := r.Render(b); err != nil {
		t.Fatal(err)
	}
	if d.writes-writes != 6 || d.flushes != 2 {
		t.Fatal("expected exactly one cell repaint")
	}
	for _, pixel := range d.pixels {
		if pixel != bg {
			t.Fatal("stale glyph")
		}
	}
	b.Set(1, 1, Cell{'Y', Normal})
	_ = r.Render(b)
	_ = r.Render(nil)
	for _, pixel := range d.pixels {
		if pixel != bg {
			t.Fatal("nil view did not clear screen")
		}
	}
}
func TestFillAccelerationAndErrorRetry(t *testing.T) {
	d := &fillDisplay{testDisplay: newDisplay(4, 6)}
	r := makeRenderer(t, d)
	b := NewBuffer(2, 2)
	boom := errors.New("bus failed")
	d.fillErr = boom
	if err := r.Render(b); err != boom {
		t.Fatal(err)
	}
	if d.flushes != 0 {
		t.Fatal("flushed failed fill")
	}
	d.fillErr = nil
	d.err = boom
	if err := r.Render(b); err != boom {
		t.Fatal(err)
	}
	fills := d.fills
	d.err = nil
	if err := r.Render(b); err != nil {
		t.Fatal(err)
	}
	if d.fills-fills != 1 {
		t.Fatal("failed frame must be redrawn completely")
	}
	fills = d.fills
	_ = r.Render(b)
	if d.fills != fills {
		t.Fatal("unnecessary fill")
	}
	theme := DefaultTheme(true)
	theme.Styles[Normal].Background = color.RGBA{8, 8, 8, 255}
	r.SetTheme(theme)
	_ = r.Render(b)
	if d.pixels[0] != theme.Styles[Normal].Background {
		t.Fatal("theme not invalidated")
	}
}
func TestRendererValidationAndStyleFallback(t *testing.T) {
	if _, err := NewRenderer(nil, nil, Theme{}); err == nil {
		t.Fatal("nil display accepted")
	}
	if _, err := NewRenderer(newDisplay(1, 1), testFont{}, Theme{}); err == nil {
		t.Fatal("oversize font accepted")
	}
	d := newDisplay(4, 6)
	r := makeRenderer(t, d)
	b := NewBuffer(1, 1)
	b.Set(0, 0, Cell{'X', 255})
	if err := r.Render(b); err != nil {
		t.Fatal(err)
	}
	if d.pixels[0] != DefaultTheme(false).Styles[Normal].Foreground {
		t.Fatal("invalid style did not fall back")
	}
}
func TestDefaultTinyFont(t *testing.T) {
	d := newDisplay(128, 64)
	r, err := NewRenderer(d, nil, DefaultTheme(true))
	if err != nil {
		t.Fatal(err)
	}
	b := NewBuffer(21, 8)
	b.Text(0, 0, "Hello, TinyBubbles!\n\u2603", Normal)
	if err = r.Render(b); err != nil {
		t.Fatal(err)
	}
	lit := 0
	for _, p := range d.pixels {
		if p.R > 0 {
			lit++
		}
	}
	if lit == 0 {
		t.Fatal("font drew nothing")
	}
}

func TestFullRepaintSkipsMatchingBackgroundFills(t *testing.T) {
	d := &fillDisplay{testDisplay: newDisplay(4, 6)}
	theme := DefaultTheme(false)
	theme.Styles[Selected].Background = color.RGBA{R: 111, A: 255}
	r, err := NewRenderer(d, testFont{}, theme)
	if err != nil {
		t.Fatal(err)
	}
	b := NewBuffer(2, 2)
	b.Set(1, 1, Cell{Style: Selected})
	if err := r.Render(b); err != nil {
		t.Fatal(err)
	}
	if d.fills != 2 {
		t.Fatalf("want clear plus different background, got %d", d.fills)
	}
	if d.pixels[5*4+3] != theme.Styles[Selected].Background || d.pixels[0] != theme.Styles[Normal].Background {
		t.Fatal("background colors lost")
	}
	b.Set(1, 1, Cell{})
	before := d.fills
	if err := r.Render(b); err != nil {
		t.Fatal(err)
	}
	if d.fills != before+1 || d.pixels[5*4+3] != theme.Styles[Normal].Background {
		t.Fatal("incremental clear skipped")
	}
}
