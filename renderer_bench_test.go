package tinybubbles

import (
	"image/color"
	"testing"
)

type benchPixelDisplay struct {
	w, h            int16
	pixels, flushes int64
}

func (d *benchPixelDisplay) Size() (int16, int16)              { return d.w, d.h }
func (d *benchPixelDisplay) SetPixel(_, _ int16, _ color.RGBA) { d.pixels++ }
func (d *benchPixelDisplay) Display() error                    { d.flushes++; return nil }

type benchFillDisplay struct {
	benchPixelDisplay
	rects int64
}

func (d *benchFillDisplay) FillRectangle(_, _, _, _ int16, _ color.RGBA) error {
	d.rects++
	return nil
}

func populateBenchBuffer(buf *Buffer) {
	w, h := buf.Size()
	buf.Clear(Normal)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			r := rune('A' + (x+y)%26)
			st := StyleID((x + y) % 4)
			buf.Set(x, y, Cell{Rune: r, Style: st})
		}
	}
}

func BenchmarkRendererFirstFrame_OLED_PixelFallback(b *testing.B) {
	disp := &benchPixelDisplay{w: 128, h: 64}
	font := DefaultFont()
	theme := DefaultTheme(true)
	buf := NewBuffer(21, 8)
	populateBenchBuffer(buf)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r, err := NewRenderer(disp, font, theme)
		if err != nil {
			b.Fatal(err)
		}
		if err := r.Render(buf); err != nil {
			b.Fatal(err)
		}
	}
	b.ReportMetric(float64(disp.pixels)/float64(b.N), "pixels/op")
	b.ReportMetric(float64(disp.flushes)/float64(b.N), "flushes/op")
}

func BenchmarkRendererFirstFrame_FeatherTFT_FillRect(b *testing.B) {
	disp := &benchFillDisplay{benchPixelDisplay: benchPixelDisplay{w: 240, h: 135}}
	font := DefaultFont()
	theme := DefaultTheme(false)
	buf := NewBuffer(40, 16)
	populateBenchBuffer(buf)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r, err := NewRenderer(disp, font, theme)
		if err != nil {
			b.Fatal(err)
		}
		if err := r.Render(buf); err != nil {
			b.Fatal(err)
		}
	}
	b.ReportMetric(float64(disp.pixels)/float64(b.N), "pixels/op")
	b.ReportMetric(float64(disp.rects)/float64(b.N), "rects/op")
	b.ReportMetric(float64(disp.flushes)/float64(b.N), "flushes/op")
}

func BenchmarkRendererSteadyState_Unchanged(b *testing.B) {
	disp := &benchFillDisplay{benchPixelDisplay: benchPixelDisplay{w: 240, h: 135}}
	r, err := NewRenderer(disp, DefaultFont(), DefaultTheme(false))
	if err != nil {
		b.Fatal(err)
	}
	buf := NewBuffer(40, 16)
	populateBenchBuffer(buf)
	if err := r.Render(buf); err != nil {
		b.Fatal(err)
	}
	disp.pixels, disp.rects, disp.flushes = 0, 0, 0

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := r.Render(buf); err != nil {
			b.Fatal(err)
		}
	}
	b.ReportMetric(float64(disp.pixels)/float64(b.N), "pixels/op")
	b.ReportMetric(float64(disp.flushes)/float64(b.N), "flushes/op")
}

func BenchmarkRendererSteadyState_CounterUpdate(b *testing.B) {
	disp := &benchFillDisplay{benchPixelDisplay: benchPixelDisplay{w: 240, h: 135}}
	r, err := NewRenderer(disp, DefaultFont(), DefaultTheme(false))
	if err != nil {
		b.Fatal(err)
	}
	buf := NewBuffer(40, 16)
	populateBenchBuffer(buf)
	if err := r.Render(buf); err != nil {
		b.Fatal(err)
	}
	disp.pixels, disp.rects, disp.flushes = 0, 0, 0

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Mutate 4 counter digits per frame.
		buf.Set(18, 8, Cell{Rune: rune('0' + (i/1000)%10), Style: Selected})
		buf.Set(19, 8, Cell{Rune: rune('0' + (i/100)%10), Style: Selected})
		buf.Set(20, 8, Cell{Rune: rune('0' + (i/10)%10), Style: Selected})
		buf.Set(21, 8, Cell{Rune: rune('0' + i%10), Style: Selected})
		if err := r.Render(buf); err != nil {
			b.Fatal(err)
		}
	}
	b.ReportMetric(float64(disp.pixels)/float64(b.N), "pixels/op")
	b.ReportMetric(float64(disp.rects)/float64(b.N), "rects/op")
	b.ReportMetric(float64(disp.flushes)/float64(b.N), "flushes/op")
}

func BenchmarkRendererUnicodeChartGlyphs(b *testing.B) {
	disp := &benchFillDisplay{benchPixelDisplay: benchPixelDisplay{w: 240, h: 135}}
	r, err := NewRenderer(disp, DefaultFont(), DefaultTheme(false))
	if err != nil {
		b.Fatal(err)
	}
	buf := NewBuffer(40, 16)
	chartRunes := [...]rune{
		'\u28ff', '\u2847', '\u28b8', '\u2819',
		'\u2581', '\u2584', '\u2588', '\u258c',
		'\u2592', '\u259a', '\u2500', '\u2502',
		'\u256d', '\u256e', '\u256f', '\u2570',
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for y := 0; y < 16; y++ {
			for x := 0; x < 40; x++ {
				ch := chartRunes[(x+y+i)%len(chartRunes)]
				buf.Set(x, y, Cell{Rune: ch, Style: Accent})
			}
		}
		if err := r.Render(buf); err != nil {
			b.Fatal(err)
		}
	}
	b.ReportMetric(float64(disp.pixels)/float64(b.N), "pixels/op")
	b.ReportMetric(float64(disp.rects)/float64(b.N), "rects/op")
	b.ReportMetric(float64(disp.flushes)/float64(b.N), "flushes/op")
}
