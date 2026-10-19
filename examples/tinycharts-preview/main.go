// Render each tinycharts chart type to a small PNG on the host for tinycharts/README.md.
// The scenes use the default 6×8 font on a 128×64 display, enlarged twice.
package main

import (
	"flag"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"

	bubbles "nimbleterminal.dev/tinybubbles"
	"nimbleterminal.dev/tinybubbles/layout"
	"nimbleterminal.dev/tinybubbles/tinycharts"
)

const width, height, scale = 128, 64, 2

const (
	styleGreen bubbles.StyleID = 4
	styleAmber bubbles.StyleID = 5
)

type display struct{ *image.RGBA }

func (d display) Size() (int16, int16)              { return width, height }
func (d display) SetPixel(x, y int16, c color.RGBA) { d.SetRGBA(int(x), int(y), c) }
func (d display) Display() error                    { return nil }

func must(err error) {
	if err != nil {
		panic(err)
	}
}

// scene draws a chart into a cell grid of the given size.
type scene struct {
	name string
	draw func(dst *bubbles.Buffer)
}

func main() {
	out := flag.String("out", filepath.Join("docs", "assets", "tinycharts"), "output directory")
	flag.Parse()
	must(os.MkdirAll(*out, 0755))
	theme := bubbles.DefaultTheme(false)
	theme.Styles[styleGreen] = bubbles.Style{Foreground: color.RGBA{80, 220, 120, 255}, Background: color.RGBA{A: 255}}
	theme.Styles[styleAmber] = bubbles.Style{Foreground: color.RGBA{255, 190, 60, 255}, Background: color.RGBA{A: 255}}
	for _, s := range scenes() {
		d := display{image.NewRGBA(image.Rect(0, 0, width, height))}
		r, err := bubbles.NewRenderer(d, nil, theme)
		must(err)
		buf := bubbles.NewBuffer(r.Size())
		s.draw(buf)
		must(r.Render(buf))
		large := image.NewRGBA(image.Rect(0, 0, width*scale, height*scale))
		for y := 0; y < height*scale; y++ {
			for x := 0; x < width*scale; x++ {
				large.SetRGBA(x, y, d.RGBAAt(x/scale, y/scale))
			}
		}
		f, err := os.Create(filepath.Join(*out, s.name+".png"))
		must(err)
		must(png.Encode(f, large))
		must(f.Close())
	}
}

// wave returns n samples of a damped sine in [0, max].
func wave(n int, max float64) []float64 {
	vs := make([]float64, n)
	for i := range vs {
		t := float64(i) / float64(n-1)
		vs[i] = max * (0.5 + 0.45*math.Sin(t*4*math.Pi)*(1-0.4*t))
	}
	return vs
}

func scenes() []scene {
	return []scene{
		{"braille", func(dst *bubbles.Buffer) {
			w, h := dst.Size()
			grid := tinycharts.NewBrailleGrid(w, h)
			dw, dh := grid.DotSize()
			grid.DrawLine(0, dh-1, dw-1, dh-1)
			grid.DrawLine(0, 0, 0, dh-1)
			px, py := 0, dh/2
			for x := 1; x < dw; x++ {
				y := int(float64(dh-1) * (0.5 - 0.42*math.Sin(float64(x)/float64(dw)*4*math.Pi)))
				grid.DrawLine(px, py, x, y)
				px, py = x, y
			}
			grid.Draw(dst, layout.Bounds(dst), bubbles.Accent)
		}},
		{"sparkline", func(dst *bubbles.Buffer) {
			w, h := dst.Size()
			top, bottom := layout.Bounds(dst).Rows(h/2, 0)
			spark := tinycharts.NewSparkline(w, top.Height, tinycharts.WithSparklineMax(100), tinycharts.WithSparklineStyle(styleGreen))
			spark.PushAll(wave(w, 100))
			spark.Draw(dst, top)
			spark2 := tinycharts.NewSparkline(w, bottom.Height, tinycharts.WithSparklineMax(100), tinycharts.WithSparklineStyle(styleAmber))
			for i := 0; i < w; i++ {
				spark2.Push(float64((i * 37) % 100))
			}
			spark2.Draw(dst, bottom)
		}},
		{"streamline", func(dst *bubbles.Buffer) {
			w, h := dst.Size()
			top, bottom := layout.Bounds(dst).Rows(h/2, 0)
			braille := tinycharts.NewStreamline(w, top.Height, tinycharts.WithStreamRange(0, 4095),
				tinycharts.WithStreamMode(tinycharts.StreamModeBraille), tinycharts.WithStreamStyle(bubbles.Accent))
			braille.PushAll(wave(w*2, 4095))
			braille.Draw(dst, top)
			arc := tinycharts.NewStreamline(w, bottom.Height, tinycharts.WithStreamRange(0, 100),
				tinycharts.WithStreamMode(tinycharts.StreamModeArc), tinycharts.WithStreamStyle(styleAmber))
			arc.PushAll(wave(w, 100))
			arc.Draw(dst, bottom)
		}},
		{"barchart", func(dst *bubbles.Buffer) {
			w, _ := dst.Size()
			left, right := layout.Bounds(dst).Columns(w/2, 1)
			bars := tinycharts.NewBarChart(left.Width, left.Height, tinycharts.WithBarMax(100), tinycharts.WithBarHorizontal(true))
			bars.SetBars([]tinycharts.BarData{
				{Label: "CPU", Value: 72, Style: bubbles.Accent},
				{Label: "PWM", Value: 45, Style: styleGreen},
				{Label: "BUS", Value: 88, Style: styleAmber},
				{Label: "RAM", Value: 31, Style: bubbles.Normal},
			})
			bars.Draw(dst, left)
			gauge, rest := right.Rows(1, 0)
			tinycharts.DrawGauge(dst, gauge, 64, 100, styleGreen, bubbles.Muted)
			vertical := tinycharts.NewBarChart(rest.Width, rest.Height, tinycharts.WithBarMax(100), tinycharts.WithBarHorizontal(false))
			vertical.SetBars([]tinycharts.BarData{
				{Label: "A", Value: 90, Style: bubbles.Accent},
				{Label: "B", Value: 55, Style: styleGreen},
				{Label: "C", Value: 20, Style: styleAmber},
				{Label: "D", Value: 70, Style: bubbles.Normal},
			})
			vertical.Draw(dst, rest)
		}},
		{"logicchart", func(dst *bubbles.Buffer) {
			w, h := dst.Size()
			logic := tinycharts.NewLogicChart(w, h, []tinycharts.LogicChannel{
				{Label: "CLK", Kind: tinycharts.LogicBit, Style: styleGreen},
				{Label: "PWM", Kind: tinycharts.LogicPWM, Style: bubbles.Accent},
				{Label: "SPI", Kind: tinycharts.LogicBus, Style: styleAmber},
			})
			for i := 0; i < w*2; i++ {
				clk := uint16(i / 2 % 2)
				pwm := uint16(i * 12 % 256)
				bus := uint16(0)
				if i%12 < 5 {
					bus = 1
				}
				logic.PushStep(clk, pwm, bus)
			}
			logic.Draw(dst, layout.Bounds(dst))
		}},
	}
}
