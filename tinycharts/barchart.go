package tinycharts

import (
	"math"

	bubbles "nimbleterminal.dev/tinybubbles"
	"nimbleterminal.dev/tinybubbles/layout"
)

// BarData represents a single labeled bar in a BarChart.
type BarData struct {
	Label string
	Value float64
	Style bubbles.StyleID
}

// BarChartOption configures a BarChart during construction.
type BarChartOption func(*BarChart)

// WithBarMax sets the initial maximum value for scaling bars.
func WithBarMax(maxVal float64) BarChartOption {
	return func(b *BarChart) {
		if maxVal > 0 {
			b.max = maxVal
		}
	}
}

// WithBarHorizontal sets whether bars grow horizontally (left-to-right) or vertically (bottom-to-top).
func WithBarHorizontal(horizontal bool) BarChartOption {
	return func(b *BarChart) {
		b.Horizontal = horizontal
	}
}

// BarChart renders horizontal or vertical bars using fractional Unicode block elements
// (▏▎▍▌▋▊▉█ or ▁▂▃▄▅▆▇█), modeled after ntcharts/barchart.
type BarChart struct {
	AutoMaxValue bool
	Horizontal   bool

	width  int
	height int
	max    float64
	bars   []BarData
	view   *bubbles.Buffer
}

// NewBarChart returns a BarChart initialized with the given cell dimensions and options.
func NewBarChart(width, height int, opts ...BarChartOption) *BarChart {
	b := &BarChart{
		AutoMaxValue: true,
		Horizontal:   true,
		max:          1,
	}
	b.Resize(width, height)
	for _, opt := range opts {
		opt(b)
	}
	return b
}

// Resize updates the cell width and height of the bar chart.
func (b *BarChart) Resize(width, height int) {
	b.width = max(0, width)
	b.height = max(0, height)
	if b.view == nil {
		b.view = bubbles.NewBuffer(b.width, b.height)
	} else {
		b.view.Resize(b.width, b.height)
	}
}

// SetMax sets the maximum value used to scale bars.
func (b *BarChart) SetMax(maxVal float64) {
	if maxVal > 0 {
		b.max = maxVal
	}
}

// MaxValue returns the current maximum scaling value.
func (b *BarChart) MaxValue() float64 { return b.max }

// SetBars replaces the bar data slice.
func (b *BarChart) SetBars(bars []BarData) {
	b.bars = append(b.bars[:0], bars...)
	if b.AutoMaxValue {
		for _, item := range b.bars {
			if item.Value > b.max {
				b.max = item.Value
			}
		}
	}
}

// DrawGauge renders a single horizontal fractional-block gauge bar into dst within rect.
func DrawGauge(dst *bubbles.Buffer, rect layout.Rect, value, maxVal float64, fillStyle, trackStyle bubbles.StyleID) {
	if dst == nil || rect.Width <= 0 || rect.Height <= 0 || maxVal <= 0 {
		return
	}
	norm := value / maxVal
	if norm < 0 {
		norm = 0
	} else if norm > 1 {
		norm = 1
	}
	totalEighths := int(math.Round(norm * float64(rect.Width*8)))
	fullCols := totalEighths / 8
	rem := totalEighths % 8

	for y := 0; y < rect.Height; y++ {
		for x := 0; x < rect.Width; x++ {
			cx, cy := rect.X+x, rect.Y+y
			switch {
			case x < fullCols:
				dst.Set(cx, cy, bubbles.Cell{Rune: RuneFullBlock, Style: fillStyle})
			case x == fullCols && rem > 0:
				dst.Set(cx, cy, bubbles.Cell{Rune: LeftBlock(rem), Style: fillStyle})
			default:
				dst.Set(cx, cy, bubbles.Cell{Rune: RuneLightShade, Style: trackStyle})
			}
		}
	}
}

// Draw renders the bar chart into dst within rect.
func (b *BarChart) Draw(dst *bubbles.Buffer, rect layout.Rect) {
	if dst == nil || rect.Width <= 0 || rect.Height <= 0 || len(b.bars) == 0 || b.max <= 0 {
		return
	}
	if b.Horizontal {
		b.drawHorizontal(dst, rect)
		return
	}
	b.drawVertical(dst, rect)
}

func (b *BarChart) drawHorizontal(dst *bubbles.Buffer, rect layout.Rect) {
	labelW := 0
	for _, item := range b.bars {
		if len(item.Label) > labelW {
			labelW = len(item.Label)
		}
	}
	if labelW > rect.Width/3 {
		labelW = rect.Width / 3
	}
	barX := rect.X
	barW := rect.Width
	if labelW > 0 {
		barX = rect.X + labelW + 1
		barW = max(1, rect.Width-labelW-1)
	}

	rows := min(rect.Height, len(b.bars))
	for i := 0; i < rows; i++ {
		item := b.bars[i]
		ry := rect.Y + i
		if labelW > 0 {
			layout.ClipText(dst, layout.Rect{X: rect.X, Y: ry, Width: labelW, Height: 1}, item.Label, bubbles.Muted, layout.Start)
		}
		DrawGauge(dst, layout.Rect{X: barX, Y: ry, Width: barW, Height: 1}, item.Value, b.max, item.Style, bubbles.Muted)
	}
}

func (b *BarChart) drawVertical(dst *bubbles.Buffer, rect layout.Rect) {
	cols := min(rect.Width, len(b.bars))
	colW := max(1, rect.Width/cols)
	totalEighths := float64(rect.Height * 8)

	for i := 0; i < cols; i++ {
		item := b.bars[i]
		val := max(0.0, item.Value)
		scaled := int(math.Round((val / b.max) * totalEighths))
		if scaled > rect.Height*8 {
			scaled = rect.Height * 8
		}
		fullRows := scaled / 8
		rem := scaled % 8
		x0 := rect.X + i*colW
		barW := max(1, colW-1)
		if colW == 1 {
			barW = 1
		}
		for dx := 0; dx < barW && x0+dx < rect.X+rect.Width; dx++ {
			cx := x0 + dx
			for row := 0; row < fullRows; row++ {
				cy := rect.Y + rect.Height - 1 - row
				dst.Set(cx, cy, bubbles.Cell{Rune: RuneFullBlock, Style: item.Style})
			}
			if rem > 0 && fullRows < rect.Height {
				cy := rect.Y + rect.Height - 1 - fullRows
				dst.Set(cx, cy, bubbles.Cell{Rune: LowerBlock(rem), Style: item.Style})
			}
		}
	}
}

// View renders the bar chart into its internal cell buffer and returns it.
func (b *BarChart) View() *bubbles.Buffer {
	if b.view == nil {
		b.view = bubbles.NewBuffer(b.width, b.height)
	}
	b.view.Clear(bubbles.Normal)
	b.Draw(b.view, layout.Rect{Width: b.width, Height: b.height})
	return b.view
}
