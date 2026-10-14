package tinycharts

import (
	"math"

	bubbles "nimbleterminal.dev/tinybubbles"
	"nimbleterminal.dev/tinybubbles/layout"
)

// SparklineOption configures a Sparkline during construction.
type SparklineOption func(*Sparkline)

// WithSparklineMax sets the initial maximum value for scaling sparkline columns.
func WithSparklineMax(maxVal float64) SparklineOption {
	return func(s *Sparkline) {
		if maxVal > 0 {
			s.max = maxVal
		}
	}
}

// WithSparklineAutoMax controls whether Push automatically expands MaxValue when exceeded.
func WithSparklineAutoMax(auto bool) SparklineOption {
	return func(s *Sparkline) {
		s.AutoMaxValue = auto
	}
}

// WithSparklineStyle sets the TinyBubbles StyleID used for sparkline columns.
func WithSparklineStyle(style bubbles.StyleID) SparklineOption {
	return func(s *Sparkline) {
		s.Style = style
	}
}

// Sparkline displays time-series values as vertical fractional-block columns
// scrolling from right to left, modeled after ntcharts/sparkline.
type Sparkline struct {
	AutoMaxValue bool
	Style        bubbles.StyleID

	width  int
	height int
	max    float64
	data   []float64
	head   int
	count  int
	view   *bubbles.Buffer
}

// NewSparkline returns a Sparkline initialized with the given cell dimensions and options.
func NewSparkline(width, height int, opts ...SparklineOption) *Sparkline {
	s := &Sparkline{
		AutoMaxValue: true,
		Style:        bubbles.Accent,
		max:          1,
	}
	s.Resize(width, height)
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Width returns the cell width of the sparkline.
func (s *Sparkline) Width() int { return s.width }

// Height returns the cell height of the sparkline.
func (s *Sparkline) Height() int { return s.height }

// MaxValue returns the current maximum scaling value.
func (s *Sparkline) MaxValue() float64 { return s.max }

// SetMax updates the maximum value used to scale columns.
func (s *Sparkline) SetMax(maxVal float64) {
	if maxVal > 0 {
		s.max = maxVal
	}
}

// Resize changes the sparkline's cell width and height, preserving recent samples.
func (s *Sparkline) Resize(width, height int) {
	width = max(0, width)
	height = max(0, height)
	if width != s.width && width > 0 {
		next := make([]float64, width)
		keep := min(s.count, width)
		for i := 0; i < keep; i++ {
			idx := (s.head - keep + i + len(s.data)) % len(s.data)
			next[i] = s.data[idx]
		}
		s.data = next
		s.head = keep % width
		s.count = keep
	} else if width == 0 {
		s.data = s.data[:0]
		s.head = 0
		s.count = 0
	}
	s.width = width
	s.height = height
	if s.view == nil {
		s.view = bubbles.NewBuffer(width, height)
	} else {
		s.view.Resize(width, height)
	}
}

// Clear resets all stored values in the sparkline.
func (s *Sparkline) Clear() {
	clear(s.data)
	s.head = 0
	s.count = 0
	if s.view != nil {
		s.view.Clear(bubbles.Normal)
	}
}

// Push appends a sample to the right edge of the sparkline. Negative values clamp to 0.
func (s *Sparkline) Push(v float64) {
	if math.IsNaN(v) || math.IsInf(v, 0) || len(s.data) == 0 {
		return
	}
	if v < 0 {
		v = 0
	}
	if s.AutoMaxValue && v > s.max {
		s.max = v
	}
	s.data[s.head] = v
	s.head = (s.head + 1) % len(s.data)
	if s.count < len(s.data) {
		s.count++
	}
}

// PushAll appends a slice of samples to the sparkline.
func (s *Sparkline) PushAll(vs []float64) {
	for _, v := range vs {
		s.Push(v)
	}
}

// Draw renders the sparkline columns into dst within rect, right-aligned to the newest sample.
func (s *Sparkline) Draw(dst *bubbles.Buffer, rect layout.Rect) {
	if dst == nil || rect.Width <= 0 || rect.Height <= 0 || s.count == 0 || s.max <= 0 {
		return
	}
	cols := min(rect.Width, s.count)
	startX := rect.X + rect.Width - cols
	totalEighths := float64(rect.Height * 8)

	for i := 0; i < cols; i++ {
		idx := (s.head - cols + i + len(s.data)) % len(s.data)
		val := s.data[idx]
		if val <= 0 {
			continue
		}
		scaled := int(math.Round((val / s.max) * totalEighths))
		if scaled <= 0 && val > 0 {
			scaled = 1
		}
		if scaled > rect.Height*8 {
			scaled = rect.Height * 8
		}
		fullRows := scaled / 8
		rem := scaled % 8
		cx := startX + i
		for row := 0; row < fullRows; row++ {
			cy := rect.Y + rect.Height - 1 - row
			dst.Set(cx, cy, bubbles.Cell{Rune: RuneFullBlock, Style: s.Style})
		}
		if rem > 0 && fullRows < rect.Height {
			cy := rect.Y + rect.Height - 1 - fullRows
			dst.Set(cx, cy, bubbles.Cell{Rune: LowerBlock(rem), Style: s.Style})
		}
	}
}

// View renders the sparkline into its internal cell buffer and returns it.
func (s *Sparkline) View() *bubbles.Buffer {
	if s.view == nil {
		s.view = bubbles.NewBuffer(s.width, s.height)
	}
	s.view.Clear(bubbles.Normal)
	s.Draw(s.view, layout.Rect{Width: s.width, Height: s.height})
	return s.view
}
