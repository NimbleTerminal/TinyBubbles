# `tinycharts`: Embedded & Cell-Buffer Charts for TinyBubbles

[Back to the README](../README.md) · [Layout](../layout/README.md) · [Widgets](../widgets/README.md) · [TinyHAL](../tinyhal/README.md)

`nimbleterminal.dev/tinybubbles/tinycharts` provides allocation-conscious chart
widgets for TinyBubbles [`*bubbles.Buffer`](../cell.go) grids and embedded
displays, modeled after [`NimbleMarkets/ntcharts`](https://github.com/NimbleMarkets/ntcharts).

## Procedural Unicode Chart Runes in `DefaultFont()`

TinyBubbles' [`TinyFont`](../font.go) procedurally rasterizes three standard
Unicode ranges into any `(Width × Height)` cell without requiring extra bitmap
font tables in Flash or RAM:

| Unicode Range | Runes | Purpose |
| --- | --- | --- |
| **Box & Arc Drawing** (`U+2500..U+257F`) | `─ ━ │ ┃ ┌ ┐ └ ┘ ├ ┤ ┬ ┴ ┼ ╭ ╮ ╯ ╰` | Connected step/wave lines, logic edges, and axes |
| **Block Elements** (`U+2580..U+259F`) | `▀ ▁ ▂ ▃ ▄ ▅ ▆ ▇ █ ▉ ▊ ▋ ▌ ▍ ▎ ▏ ▐ ░ ▒ ▓ ▔ ▕ ▖..▟` | Fractional sparklines, horizontal/vertical bars, and 2×2 quadrant blocks |
| **Braille Patterns** (`U+2800..U+28FF`) | `⠀ .. ⣿` | Sub-cell `2×4` dot matrix plotting inside a single character cell |

Because these runes are standard Unicode codepoints, the exact same chart
buffer renders on an MCU `ST7789` / `SSD1306` display, in the browser WASM
canvas, and in a terminal.

## Chart Types

| Type | File | Description |
| --- | --- | --- |
| [`BrailleGrid`](./runes.go) | `runes.go` | `2*W × 4*H` sub-cell dot matrix with Bresenham `DrawLine` and `Draw(dst, rect, style)` |
| [`Sparkline`](./sparkline.go) | `sparkline.go` | Scrolling right-to-left column chart using `▁▂▃▄▅▆▇█` (`Push`, `PushAll`, `AutoMaxValue`) |
| [`Streamline`](./streamline.go) | `streamline.go` | Scrolling time-series line chart supporting `StreamModeBraille` (`⠀..⣿`), `StreamModeArc` (`╭╮╰╯─│`), and `StreamModeBox` (`┌┐└┘─│`) |
| [`BarChart`](./barchart.go) | `barchart.go` | Horizontal (`▏▎▍▌▋▊▉█`) and vertical (`▁▂▃▄▅▆▇█`) bar charts plus single-row `DrawGauge` |
| [`LogicChart`](./logicchart.go) | `logicchart.go` | Multi-channel digital (`LogicBit`), PWM (`LogicPWM`), and bus (`LogicBus`) logic analyzer chart |

## Quickstart

```go
package main

import (
	bubbles "nimbleterminal.dev/tinybubbles"
	"nimbleterminal.dev/tinybubbles/layout"
	"nimbleterminal.dev/tinybubbles/tinycharts"
)

func drawCharts(dst *bubbles.Buffer) {
	_, h := dst.Size()
	top, bottom := layout.Bounds(dst).Rows(h/2, 1)

	// 1. High-resolution Braille streamline chart.
	stream := tinycharts.NewStreamline(
		top.Width, top.Height,
		tinycharts.WithStreamRange(0, 4095),
		tinycharts.WithStreamMode(tinycharts.StreamModeBraille),
		tinycharts.WithStreamStyle(bubbles.Selected),
	)
	stream.PushAll([]float64{512, 1024, 2048, 3500, 4095, 2800, 1200})
	stream.Draw(dst, top)

	// 2. Multi-channel digital & PWM logic analyzer chart.
	logic := tinycharts.NewLogicChart(bottom.Width, bottom.Height, []tinycharts.LogicChannel{
		{Label: "G13", Kind: tinycharts.LogicBit, Style: bubbles.Normal},
		{Label: "PWM", Kind: tinycharts.LogicPWM, Style: bubbles.Selected},
		{Label: "SPI", Kind: tinycharts.LogicBus, Style: bubbles.Accent},
	})
	logic.PushStep(1, 128, 0)
	logic.PushStep(0, 220, 1)
	logic.Draw(dst, bottom)
}
```

For standalone pixel targets (such as [`tinyhal/workbench`](../tinyhal/workbench)),
[`tinycharts.PaintBuffer(display, px, py, chart.View(), font, theme)`](./runes.go)
renders any chart's `*bubbles.Buffer` directly into a `drivers.Displayer`
framebuffer.
