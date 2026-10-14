package tinycharts

import (
	bubbles "nimbleterminal.dev/tinybubbles"
	"nimbleterminal.dev/tinybubbles/layout"
)

// LogicChannelKind describes how a channel in a LogicChart renders its samples.
type LogicChannelKind uint8

const (
	// LogicBit renders a 1-bit digital square wave with rising and falling edges.
	LogicBit LogicChannelKind = iota
	// LogicPWM renders an 8-bit PWM duty level (0..255) using fractional block elements or pulse modulation.
	LogicPWM
	// LogicBus renders a serial/parallel bus line (SPI/I2C/UART) that expands into packet blocks on activity.
	LogicBus
)

// LogicChannel defines a named trace lane in a LogicChart.
type LogicChannel struct {
	Label string
	Kind  LogicChannelKind
	Style bubbles.StyleID
}

// LogicChart renders multi-channel digital, PWM, and bus waveforms into a TinyBubbles
// cell Buffer using Unicode box-drawing (┌┐└┘─) and fractional block (▁▂▃▄▅▆▇█) runes.
type LogicChart struct {
	width    int
	height   int
	channels []LogicChannel
	samples  [][]uint16 // per-channel ring buffer of encoded samples
	head     int
	count    int
	capLen   int
	view     *bubbles.Buffer
}

// NewLogicChart creates a multi-channel logic analyzer chart for the given cell dimensions and channels.
func NewLogicChart(width, height int, channels []LogicChannel) *LogicChart {
	c := &LogicChart{
		channels: append([]LogicChannel(nil), channels...),
	}
	c.Resize(width, height)
	return c
}

// Resize adjusts the cell dimensions of the logic chart, preserving recent history.
func (c *LogicChart) Resize(width, height int) {
	width = max(0, width)
	height = max(0, height)
	if width != c.capLen && width > 0 {
		next := make([][]uint16, len(c.channels))
		keep := min(c.count, width)
		for ch := range c.channels {
			next[ch] = make([]uint16, width)
			if ch < len(c.samples) && len(c.samples[ch]) > 0 {
				for i := 0; i < keep; i++ {
					idx := (c.head - keep + i + len(c.samples[ch])) % len(c.samples[ch])
					next[ch][i] = c.samples[ch][idx]
				}
			}
		}
		c.samples = next
		c.head = keep % width
		c.count = keep
		c.capLen = width
	} else if width == 0 {
		c.samples = nil
		c.head = 0
		c.count = 0
		c.capLen = 0
	}
	c.width = width
	c.height = height
	if c.view == nil {
		c.view = bubbles.NewBuffer(width, height)
	} else {
		c.view.Resize(width, height)
	}
}

// Clear resets all recorded waveform samples.
func (c *LogicChart) Clear() {
	for ch := range c.samples {
		clear(c.samples[ch])
	}
	c.head = 0
	c.count = 0
	if c.view != nil {
		c.view.Clear(bubbles.Normal)
	}
}

// PushStep appends one sample column across all channels.
// For LogicBit channels, non-zero values mean logic high.
// For LogicPWM channels, values in 0..255 represent the PWM duty cycle.
// For LogicBus channels, non-zero values indicate an active bus packet transfer.
func (c *LogicChart) PushStep(values ...uint16) {
	if c.capLen <= 0 || len(c.samples) == 0 {
		return
	}
	for ch := range c.channels {
		var v uint16
		if ch < len(values) {
			v = values[ch]
		}
		c.samples[ch][c.head] = v
	}
	c.head = (c.head + 1) % c.capLen
	if c.count < c.capLen {
		c.count++
	}
}

// Draw renders the multi-channel logic analyzer into dst within rect.
func (c *LogicChart) Draw(dst *bubbles.Buffer, rect layout.Rect) {
	if dst == nil || rect.Width <= 0 || rect.Height <= 0 || len(c.channels) == 0 {
		return
	}
	labelW := 0
	for _, ch := range c.channels {
		if len(ch.Label) > labelW {
			labelW = len(ch.Label)
		}
	}
	if labelW > rect.Width/3 {
		labelW = rect.Width / 3
	}
	traceX := rect.X
	traceW := rect.Width
	if labelW > 0 {
		traceX = rect.X + labelW + 1
		traceW = max(1, rect.Width-labelW-1)
	}

	numCh := len(c.channels)
	twoRow := rect.Height >= numCh*2
	laneH := 1
	if twoRow {
		laneH = 2
	}

	cols := min(traceW, c.count)
	startX := traceX + traceW - cols

	for chIdx, ch := range c.channels {
		ry := rect.Y + chIdx*laneH
		if ry >= rect.Y+rect.Height {
			break
		}
		if labelW > 0 {
			labelY := ry + laneH - 1
			layout.ClipText(dst, layout.Rect{X: rect.X, Y: labelY, Width: labelW, Height: 1}, ch.Label, bubbles.Muted, layout.Start)
		}
		if cols == 0 {
			continue
		}

		var prevHigh bool
		for i := 0; i < cols; i++ {
			idx := (c.head - cols + i + c.capLen) % c.capLen
			val := c.samples[chIdx][idx]
			cx := startX + i

			switch ch.Kind {
			case LogicBit:
				high := val != 0
				if twoRow {
					if i == 0 || high == prevHigh {
						if high {
							dst.Set(cx, ry, bubbles.Cell{Rune: RuneLineHorizontal, Style: ch.Style})
						} else {
							dst.Set(cx, ry+1, bubbles.Cell{Rune: RuneLineHorizontal, Style: ch.Style})
						}
					} else if high {
						// Rising edge: ┌ on top row, ┘ on bottom row.
						dst.Set(cx, ry, bubbles.Cell{Rune: RuneLineDownRight, Style: ch.Style})
						dst.Set(cx, ry+1, bubbles.Cell{Rune: RuneLineUpLeft, Style: ch.Style})
					} else {
						// Falling edge: ┐ on top row, └ on bottom row.
						dst.Set(cx, ry, bubbles.Cell{Rune: RuneLineDownLeft, Style: ch.Style})
						dst.Set(cx, ry+1, bubbles.Cell{Rune: RuneLineUpRight, Style: ch.Style})
					}
				} else {
					r := RuneLowerBlockOne
					if high {
						r = RuneUpperHalfBlock
					}
					if i > 0 && high != prevHigh {
						if high {
							r = RuneLineUpRight
						} else {
							r = RuneLineDownLeft
						}
					}
					dst.Set(cx, ry, bubbles.Cell{Rune: r, Style: ch.Style})
				}
				prevHigh = high

			case LogicPWM:
				duty := min(int(val), 255)
				eighths := (duty*8 + 127) / 255
				if twoRow {
					total16 := (duty*16 + 127) / 255
					if total16 > 8 {
						dst.Set(cx, ry+1, bubbles.Cell{Rune: RuneFullBlock, Style: ch.Style})
						dst.Set(cx, ry, bubbles.Cell{Rune: LowerBlock(total16 - 8), Style: ch.Style})
					} else {
						dst.Set(cx, ry+1, bubbles.Cell{Rune: LowerBlock(max(1, total16)), Style: ch.Style})
					}
				} else {
					dst.Set(cx, ry, bubbles.Cell{Rune: LowerBlock(max(1, eighths)), Style: ch.Style})
				}

			case LogicBus:
				active := val != 0
				if twoRow {
					if active {
						dst.Set(cx, ry, bubbles.Cell{Rune: RuneUpperHalfBlock, Style: ch.Style})
						dst.Set(cx, ry+1, bubbles.Cell{Rune: RuneLowerBlockFour, Style: ch.Style})
					} else {
						dst.Set(cx, ry+1, bubbles.Cell{Rune: RuneLineHorizontal, Style: bubbles.Muted})
					}
				} else {
					if active {
						dst.Set(cx, ry, bubbles.Cell{Rune: RuneMediumShade, Style: ch.Style})
					} else {
						dst.Set(cx, ry, bubbles.Cell{Rune: RuneLineHorizontal, Style: bubbles.Muted})
					}
				}
			}
		}
	}
}

// View renders the logic chart into its internal cell buffer and returns it.
func (c *LogicChart) View() *bubbles.Buffer {
	if c.view == nil {
		c.view = bubbles.NewBuffer(c.width, c.height)
	}
	c.view.Clear(bubbles.Normal)
	c.Draw(c.view, layout.Rect{Width: c.width, Height: c.height})
	return c.view
}
