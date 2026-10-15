// Command bench runs portable rendering, widget, charting, and TinyHAL
// benchmarks under both the standard Go toolchain (`go run ./internal/bench`)
// and TinyGo WASI (`tinygo run -target=wasip1 ./internal/bench`) to track
// compiler, allocator, and renderer improvements across versions.
package main

import (
	"fmt"
	"image/color"
	"runtime"
	"time"

	bubbles "nimbleterminal.dev/tinybubbles"
	"nimbleterminal.dev/tinybubbles/demo/hello"
	"nimbleterminal.dev/tinybubbles/layout"
	"nimbleterminal.dev/tinybubbles/tinycharts"
	"nimbleterminal.dev/tinybubbles/tinyhal/workbench"
)

type counterDisplay struct {
	w, h            int16
	pixels, rects   int64
	flushes         int64
	supportFillRect bool
}

func (d *counterDisplay) Size() (int16, int16)              { return d.w, d.h }
func (d *counterDisplay) SetPixel(_, _ int16, _ color.RGBA) { d.pixels++ }
func (d *counterDisplay) Display() error                    { d.flushes++; return nil }

type rectDisplay struct{ counterDisplay }

func (d *rectDisplay) FillRectangle(_, _, _, _ int16, _ color.RGBA) error {
	d.rects++
	return nil
}

type benchCase struct {
	name  string
	iters int
	fn    func(iters int) (pixels, rects int64)
}

func main() {
	fmt.Printf("TinyBubbles Compiler & Rendering Benchmark Suite (%s %s/%s)\n",
		runtime.Version(), runtime.GOOS, runtime.GOARCH)
	fmt.Println("--------------------------------------------------------------------------------")
	fmt.Printf("%-34s %8s %10s %10s %10s %10s\n",
		"Benchmark", "Iters", "ns/op", "B/op", "allocs/op", "px+rect/op")
	fmt.Println("--------------------------------------------------------------------------------")

	cases := []benchCase{
		{
			name:  "Renderer/FirstFrame/TFT_FillRect",
			iters: 500,
			fn:    benchFirstFrameTFT,
		},
		{
			name:  "Renderer/SteadyState/Unchanged",
			iters: 5000,
			fn:    benchSteadyUnchanged,
		},
		{
			name:  "Renderer/SteadyState/Counter4Cells",
			iters: 2000,
			fn:    benchSteadyCounter,
		},
		{
			name:  "Renderer/UnicodeChartGlyphs",
			iters: 500,
			fn:    benchUnicodeChartGlyphs,
		},
		{
			name:  "Hello/StepAndRender/TFT",
			iters: 1000,
			fn:    benchHelloStepAndRender,
		},
		{
			name:  "TinyCharts/Dashboard4Panels",
			iters: 1000,
			fn:    benchTinyChartsDashboard,
		},
		{
			name:  "TinyHAL/WorkbenchTick/TFT_Scope",
			iters: 200,
			fn:    benchWorkbenchTickScope,
		},
		{
			name:  "TinyHAL/WorkbenchTick/TFT_Logic",
			iters: 200,
			fn:    benchWorkbenchTickLogic,
		},
	}

	for _, tc := range cases {
		// Warm-up run.
		_, _ = tc.fn(min(10, tc.iters))

		runtime.GC()
		var m0, m1 runtime.MemStats
		runtime.ReadMemStats(&m0)
		start := time.Now()
		pixels, rects := tc.fn(tc.iters)
		elapsed := time.Since(start)
		runtime.ReadMemStats(&m1)

		nsPerOp := elapsed.Nanoseconds() / int64(tc.iters)
		bytesPerOp := int64(m1.TotalAlloc-m0.TotalAlloc) / int64(tc.iters)
		allocsPerOp := int64(m1.Mallocs-m0.Mallocs) / int64(tc.iters)
		opsPerFrame := (pixels + rects) / int64(tc.iters)

		fmt.Printf("%-34s %8d %10d %10d %10d %10d\n",
			tc.name, tc.iters, nsPerOp, bytesPerOp, allocsPerOp, opsPerFrame)
	}
	fmt.Println("--------------------------------------------------------------------------------")
}

func fillTextGrid(buf *bubbles.Buffer) {
	w, h := buf.Size()
	buf.Clear(bubbles.Normal)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			buf.Set(x, y, bubbles.Cell{
				Rune:  rune('A' + (x+y)%26),
				Style: bubbles.StyleID((x + y) % 4),
			})
		}
	}
}

func benchFirstFrameTFT(iters int) (int64, int64) {
	disp := &rectDisplay{counterDisplay: counterDisplay{w: 240, h: 135}}
	font := bubbles.DefaultFont()
	theme := bubbles.DefaultTheme(false)
	buf := bubbles.NewBuffer(40, 16)
	fillTextGrid(buf)
	for i := 0; i < iters; i++ {
		r, _ := bubbles.NewRenderer(disp, font, theme)
		_ = r.Render(buf)
	}
	return disp.pixels, disp.rects
}

func benchSteadyUnchanged(iters int) (int64, int64) {
	disp := &rectDisplay{counterDisplay: counterDisplay{w: 240, h: 135}}
	r, _ := bubbles.NewRenderer(disp, bubbles.DefaultFont(), bubbles.DefaultTheme(false))
	buf := bubbles.NewBuffer(40, 16)
	fillTextGrid(buf)
	_ = r.Render(buf)
	disp.pixels, disp.rects = 0, 0
	for i := 0; i < iters; i++ {
		_ = r.Render(buf)
	}
	return disp.pixels, disp.rects
}

func benchSteadyCounter(iters int) (int64, int64) {
	disp := &rectDisplay{counterDisplay: counterDisplay{w: 240, h: 135}}
	r, _ := bubbles.NewRenderer(disp, bubbles.DefaultFont(), bubbles.DefaultTheme(false))
	buf := bubbles.NewBuffer(40, 16)
	fillTextGrid(buf)
	_ = r.Render(buf)
	disp.pixels, disp.rects = 0, 0
	for i := 0; i < iters; i++ {
		buf.Set(18, 8, bubbles.Cell{Rune: rune('0' + (i/1000)%10), Style: bubbles.Selected})
		buf.Set(19, 8, bubbles.Cell{Rune: rune('0' + (i/100)%10), Style: bubbles.Selected})
		buf.Set(20, 8, bubbles.Cell{Rune: rune('0' + (i/10)%10), Style: bubbles.Selected})
		buf.Set(21, 8, bubbles.Cell{Rune: rune('0' + i%10), Style: bubbles.Selected})
		_ = r.Render(buf)
	}
	return disp.pixels, disp.rects
}

func benchUnicodeChartGlyphs(iters int) (int64, int64) {
	disp := &rectDisplay{counterDisplay: counterDisplay{w: 240, h: 135}}
	r, _ := bubbles.NewRenderer(disp, bubbles.DefaultFont(), bubbles.DefaultTheme(false))
	buf := bubbles.NewBuffer(40, 16)
	runes := [...]rune{
		'\u28ff', '\u2847', '\u28b8', '\u2819',
		'\u2581', '\u2584', '\u2588', '\u258c',
		'\u2592', '\u259a', '\u2500', '\u2502',
		'\u256d', '\u256e', '\u256f', '\u2570',
	}
	for i := 0; i < iters; i++ {
		for y := 0; y < 16; y++ {
			for x := 0; x < 40; x++ {
				buf.Set(x, y, bubbles.Cell{Rune: runes[(x+y+i)%len(runes)], Style: bubbles.Accent})
			}
		}
		_ = r.Render(buf)
	}
	return disp.pixels, disp.rects
}

func benchHelloStepAndRender(iters int) (int64, int64) {
	disp := &rectDisplay{counterDisplay: counterDisplay{w: 240, h: 135}}
	m := hello.New()
	r, _ := bubbles.NewRenderer(disp, hello.NewFont(), hello.Theme())
	m.Update(bubbles.WindowSizeMsg{Width: 40, Height: 16})
	_ = r.Render(m.View())
	disp.pixels, disp.rects = 0, 0
	for i := 0; i < iters; i++ {
		m.Update(hello.TickMsg{})
		_ = r.Render(m.View())
	}
	return disp.pixels, disp.rects
}

func benchTinyChartsDashboard(iters int) (int64, int64) {
	disp := &rectDisplay{counterDisplay: counterDisplay{w: 240, h: 135}}
	r, _ := bubbles.NewRenderer(disp, bubbles.DefaultFont(), bubbles.DefaultTheme(false))
	buf := bubbles.NewBuffer(40, 16)
	top, bottom := layout.Bounds(buf).Rows(7, 1)
	p0, p1 := top.Columns(19, 1)
	p2, p3 := bottom.Columns(19, 1)

	sl := tinycharts.NewStreamline(p0.Width, p0.Height, tinycharts.WithStreamRange(0, 100))
	sp := tinycharts.NewSparkline(p1.Width, p1.Height, tinycharts.WithSparklineMax(100), tinycharts.WithSparklineAutoMax(false))
	bc := tinycharts.NewBarChart(p2.Width, p2.Height, tinycharts.WithBarMax(100))
	bc.SetBars([]tinycharts.BarData{
		{Label: "ADC", Value: 65, Style: bubbles.Accent},
		{Label: "PWM", Value: 80, Style: bubbles.Selected},
	})
	lc := tinycharts.NewLogicChart(p3.Width, p3.Height, []tinycharts.LogicChannel{
		{Label: "CLK", Kind: tinycharts.LogicBit, Style: bubbles.Accent},
		{Label: "PWM", Kind: tinycharts.LogicPWM, Style: bubbles.Selected},
	})
	for i := 0; i < 40; i++ {
		sl.Push(float64((i * 7) % 100))
		sp.Push(float64((i * 11) % 100))
		lc.PushStep(uint16(i&1), uint16((i*19)&0xff))
	}

	for i := 0; i < iters; i++ {
		sl.Push(float64((i * 13) % 100))
		sp.Push(float64((i * 17) % 100))
		lc.PushStep(uint16(i&1), uint16((i*29)&0xff))
		buf.Clear(bubbles.Normal)
		sl.Draw(buf, p0)
		sp.Draw(buf, p1)
		bc.Draw(buf, p2)
		lc.Draw(buf, p3)
		_ = r.Render(buf)
	}
	return disp.pixels, disp.rects
}

func benchWorkbenchTickScope(iters int) (int64, int64) {
	disp := &counterDisplay{w: 240, h: 135}
	wb := workbench.New(workbench.Config{
		Width:  240,
		Height: 135,
		Target: disp,
	})
	wb.PressMode()
	disp.pixels = 0
	for i := 0; i < iters; i++ {
		wb.Tick()
	}
	return disp.pixels, 0
}

func benchWorkbenchTickLogic(iters int) (int64, int64) {
	disp := &counterDisplay{w: 240, h: 135}
	wb := workbench.New(workbench.Config{
		Width:  240,
		Height: 135,
		Target: disp,
	})
	wb.ToggleView()
	disp.pixels = 0
	for i := 0; i < iters; i++ {
		wb.Tick()
	}
	return disp.pixels, 0
}
