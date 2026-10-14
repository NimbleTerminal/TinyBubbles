package tinycharts

import (
	"math"

	bubbles "nimbleterminal.dev/tinybubbles"
	"nimbleterminal.dev/tinybubbles/layout"
)

// StreamMode selects the line rendering style for a Streamline chart.
type StreamMode uint8

const (
	// StreamModeBraille renders a 2x4 sub-cell Unicode Braille curve (U+2800..U+28FF).
	StreamModeBraille StreamMode = iota
	// StreamModeArc renders connected Unicode arc/box runes (╭╮╰╯─│) like ntcharts streamlinechart.
	StreamModeArc
	// StreamModeBox renders connected sharp Unicode box runes (┌┐└┘─│).
	StreamModeBox
)

// StreamlineOption configures a Streamline chart during construction.
type StreamlineOption func(*Streamline)

// WithStreamRange sets a fixed [minY, maxY] vertical range and disables AutoRange.
func WithStreamRange(minY, maxY float64) StreamlineOption {
	return func(s *Streamline) {
		if maxY > minY {
			s.minY = minY
			s.maxY = maxY
			s.AutoRange = false
		}
	}
}

// WithStreamAutoRange controls whether Push automatically expands [minY, maxY].
func WithStreamAutoRange(auto bool) StreamlineOption {
	return func(s *Streamline) {
		s.AutoRange = auto
	}
}

// WithStreamMode selects Braille, Arc, or Box line rendering.
func WithStreamMode(mode StreamMode) StreamlineOption {
	return func(s *Streamline) {
		s.mode = mode
	}
}

// WithStreamStyle sets the TinyBubbles StyleID used to draw the line.
func WithStreamStyle(style bubbles.StyleID) StreamlineOption {
	return func(s *Streamline) {
		s.Style = style
	}
}

// Streamline displays a continuous scrolling line moving from right to left,
// modeled after ntcharts/linechart/streamlinechart and wavelinechart.
type Streamline struct {
	AutoRange bool
	Style     bubbles.StyleID

	width   int
	height  int
	minY    float64
	maxY    float64
	mode    StreamMode
	data    []float64
	head    int
	count   int
	braille *BrailleGrid
	view    *bubbles.Buffer
}

// NewStreamline returns a Streamline chart initialized with the given cell dimensions and options.
func NewStreamline(width, height int, opts ...StreamlineOption) *Streamline {
	s := &Streamline{
		AutoRange: true,
		Style:     bubbles.Selected,
		minY:      0,
		maxY:      1,
		mode:      StreamModeBraille,
		braille:   NewBrailleGrid(width, height),
	}
	s.Resize(width, height)
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Width returns the cell width of the chart.
func (s *Streamline) Width() int { return s.width }

// Height returns the cell height of the chart.
func (s *Streamline) Height() int { return s.height }

// Range returns the current (minY, maxY) vertical range.
func (s *Streamline) Range() (minY, maxY float64) { return s.minY, s.maxY }

// SetRange updates the vertical (minY, maxY) range.
func (s *Streamline) SetRange(minY, maxY float64) {
	if maxY > minY {
		s.minY = minY
		s.maxY = maxY
	}
}

// Mode returns the active line rendering mode.
func (s *Streamline) Mode() StreamMode { return s.mode }

// SetMode changes the active line rendering mode (Braille, Arc, or Box).
func (s *Streamline) SetMode(mode StreamMode) { s.mode = mode }

// Resize adjusts the cell dimensions of the streamline chart, preserving recent samples.
func (s *Streamline) Resize(width, height int) {
	width = max(0, width)
	height = max(0, height)
	capLen := width * 2 // 2 sub-cell samples per column for Braille mode
	if capLen != len(s.data) && capLen > 0 {
		next := make([]float64, capLen)
		keep := min(s.count, capLen)
		for i := 0; i < keep; i++ {
			idx := (s.head - keep + i + len(s.data)) % len(s.data)
			next[i] = s.data[idx]
		}
		s.data = next
		s.head = keep % capLen
		s.count = keep
	} else if capLen == 0 {
		s.data = s.data[:0]
		s.head = 0
		s.count = 0
	}
	s.width = width
	s.height = height
	s.braille.Resize(width, height)
	if s.view == nil {
		s.view = bubbles.NewBuffer(width, height)
	} else {
		s.view.Resize(width, height)
	}
}

// Clear resets all stored samples in the streamline chart.
func (s *Streamline) Clear() {
	clear(s.data)
	s.head = 0
	s.count = 0
	s.braille.Clear()
	if s.view != nil {
		s.view.Clear(bubbles.Normal)
	}
}

// Push appends a sample to the right edge of the streamline chart.
func (s *Streamline) Push(v float64) {
	if math.IsNaN(v) || math.IsInf(v, 0) || len(s.data) == 0 {
		return
	}
	if s.AutoRange {
		if s.count == 0 {
			s.minY = v
			if v+1 > s.maxY {
				s.maxY = v + 1
			}
		} else {
			if v < s.minY {
				s.minY = v
			}
			if v > s.maxY {
				s.maxY = v
			}
		}
	}
	s.data[s.head] = v
	s.head = (s.head + 1) % len(s.data)
	if s.count < len(s.data) {
		s.count++
	}
}

// PushAll appends a slice of samples to the streamline chart.
func (s *Streamline) PushAll(vs []float64) {
	for _, v := range vs {
		s.Push(v)
	}
}

func (s *Streamline) scaleToRow(v float64, rows int) int {
	if rows <= 1 || s.maxY <= s.minY {
		return 0
	}
	norm := (v - s.minY) / (s.maxY - s.minY)
	if norm < 0 {
		norm = 0
	} else if norm > 1 {
		norm = 1
	}
	return rows - 1 - int(math.Round(norm*float64(rows-1)))
}

// Draw renders the streamline chart into dst within rect.
func (s *Streamline) Draw(dst *bubbles.Buffer, rect layout.Rect) {
	if dst == nil || rect.Width <= 0 || rect.Height <= 0 || s.count == 0 {
		return
	}
	if s.mode == StreamModeBraille {
		s.drawBraille(dst, rect)
		return
	}
	s.drawConnectedRunes(dst, rect)
}

func (s *Streamline) drawBraille(dst *bubbles.Buffer, rect layout.Rect) {
	if s.braille.cWidth != rect.Width || s.braille.cHeight != rect.Height {
		s.braille.Resize(rect.Width, rect.Height)
	} else {
		s.braille.Clear()
	}
	dotW, dotH := s.braille.DotSize()
	n := min(dotW, s.count)
	startX := dotW - n
	var prevX, prevY int
	for i := 0; i < n; i++ {
		idx := (s.head - n + i + len(s.data)) % len(s.data)
		dx := startX + i
		dy := s.scaleToRow(s.data[idx], dotH)
		if i == 0 {
			s.braille.Set(dx, dy)
		} else {
			s.braille.DrawLine(prevX, prevY, dx, dy)
		}
		prevX, prevY = dx, dy
	}
	s.braille.Draw(dst, rect, s.Style)
}

func (s *Streamline) drawConnectedRunes(dst *bubbles.Buffer, rect layout.Rect) {
	upRight, upLeft, downRight, downLeft := RuneArcUpRight, RuneArcUpLeft, RuneArcDownRight, RuneArcDownLeft
	if s.mode == StreamModeBox {
		upRight, upLeft, downRight, downLeft = RuneLineUpRight, RuneLineUpLeft, RuneLineDownRight, RuneLineDownLeft
	}

	cols := min(rect.Width, s.count)
	startX := rect.X + rect.Width - cols
	var prevRow int
	for i := 0; i < cols; i++ {
		idx := (s.head - cols + i + len(s.data)) % len(s.data)
		curRow := s.scaleToRow(s.data[idx], rect.Height)
		cx := startX + i
		if i == 0 || curRow == prevRow {
			dst.Set(cx, rect.Y+curRow, bubbles.Cell{Rune: RuneLineHorizontal, Style: s.Style})
		} else if curRow < prevRow {
			// Rising step from prevRow up to curRow.
			dst.Set(cx, rect.Y+prevRow, bubbles.Cell{Rune: upLeft, Style: s.Style})
			for ry := curRow + 1; ry < prevRow; ry++ {
				dst.Set(cx, rect.Y+ry, bubbles.Cell{Rune: RuneLineVertical, Style: s.Style})
			}
			dst.Set(cx, rect.Y+curRow, bubbles.Cell{Rune: downRight, Style: s.Style})
		} else {
			// Falling step from prevRow down to curRow.
			dst.Set(cx, rect.Y+prevRow, bubbles.Cell{Rune: downLeft, Style: s.Style})
			for ry := prevRow + 1; ry < curRow; ry++ {
				dst.Set(cx, rect.Y+ry, bubbles.Cell{Rune: RuneLineVertical, Style: s.Style})
			}
			dst.Set(cx, rect.Y+curRow, bubbles.Cell{Rune: upRight, Style: s.Style})
		}
		prevRow = curRow
	}
}

// View renders the streamline chart into its internal cell buffer and returns it.
func (s *Streamline) View() *bubbles.Buffer {
	if s.view == nil {
		s.view = bubbles.NewBuffer(s.width, s.height)
	}
	s.view.Clear(bubbles.Normal)
	s.Draw(s.view, layout.Rect{Width: s.width, Height: s.height})
	return s.view
}
