// Package hello is a small, animated greeting playground with no hardware imports.
package hello

import (
	bubbles "github.com/NimbleTerminal/TinyBubbles"
	"image/color"
	"strconv"
)

// TickMsg advances one animation frame. The device owns the timer.
type TickMsg struct{}

const bubbleCount = 10

type bubble struct{ x, y, age, burst int }

type Model struct {
	grid                         *bubbles.Buffer
	bubbles                      [bubbleCount]bubble
	frame, popped, palette, next int
	paused                       bool
	seed                         uint32
}

func New() *Model                { return &Model{seed: 5691} }
func (*Model) Init() bubbles.Cmd { return nil }

func (m *Model) Update(msg bubbles.Msg) (bubbles.Model, bubbles.Cmd) {
	switch msg := msg.(type) {
	case bubbles.WindowSizeMsg:
		m.grid = bubbles.NewBuffer(msg.Width, msg.Height)
		for i := range m.bubbles {
			m.spawn(i, true)
		}
	case TickMsg:
		if m.paused || m.grid == nil {
			break
		}
		m.frame++
		for i := range m.bubbles {
			b := &m.bubbles[i]
			if b.burst > 0 {
				b.burst--
				if b.burst == 0 {
					m.spawn(i, false)
				}
				continue
			}
			b.age++
			if b.age%(3+i%3) == 0 {
				b.y--
			}
			if b.y < 2 {
				m.spawn(i, false)
			}
		}
	case bubbles.KeyMsg:
		switch msg.Key {
		case bubbles.KeySelect:
			if m.grid == nil {
				break
			}
			// Keep button feedback visible even when paused.
			for n := 0; n < bubbleCount; n++ {
				i := (m.next + n) % bubbleCount
				if m.bubbles[i].burst == 0 {
					m.bubbles[i].burst = 5
					m.popped++
					m.next = (i + 1) % bubbleCount
					break
				}
			}
		case bubbles.KeyRight:
			m.palette = (m.palette + 1) % 4
		case bubbles.KeyBack:
			m.paused = !m.paused
		case bubbles.KeyQuit:
			return m, bubbles.Quit
		}
	}
	return m, nil
}

func (m *Model) random(n int) int {
	m.seed = m.seed*1664525 + 1013904223
	if n <= 0 {
		return 0
	}
	return int(m.seed>>8) % n
}

func (m *Model) spawn(i int, initial bool) {
	w, h := m.grid.Size()
	y := h - 5
	if initial {
		y = 2 + m.random(h-6)
	}
	// Keep the greeting's central area clear: bubbles rise along both sides.
	x := m.random(6)
	if i%2 == 1 {
		x = w - 8 + m.random(6)
	}
	m.bubbles[i] = bubble{x: x, y: y, age: i}
}

var paletteNames = [4]string{"LAGOON", "BLOSSOM", "SUNSHINE", "MINT"}

func (m *Model) View() *bubbles.Buffer {
	if m.grid == nil {
		return nil
	}
	g := m.grid
	w, h := g.Size()
	g.Clear(bubbles.Normal)
	accent := bubbles.StyleID(4 + m.palette)
	g.Text(1, 0, "TINYBUBBLES", accent)
	state := "LIVE"
	if m.paused {
		state = "PAUSED"
	}
	g.Text(w-len(state)-1, 0, state, bubbles.Muted)
	if w < 36 || h < 16 {
		g.Text(0, 2, "Hello, tiny world!", accent)
		g.Text(0, 4, "Pops: "+strconv.Itoa(m.popped), bubbles.Normal)
		g.Text(0, h-2, "0:POP 1:COLOR 2:PAUSE", bubbles.Muted)
		return g
	}
	for x := 1; x < w-1; x++ {
		g.Set(x, 1, bubbles.Cell{Rune: ruleRune, Style: bubbles.Muted})
	}
	for i, b := range m.bubbles {
		style := accent
		if i%3 == 0 {
			style = bubbles.Accent
		}
		if b.burst > 0 {
			g.Text(b.x, b.y, "+*", accent)
			g.Text(b.x, b.y+1, "*+", accent)
		} else {
			for ty := 0; ty < 2; ty++ {
				for tx := 0; tx < 2; tx++ {
					g.Set(b.x+tx, b.y+ty, bubbles.Cell{Rune: bubbleRune + rune(ty*2+tx), Style: style})
				}
			}
		}
	}
	largeText(g, (w-12)/2, 4, "HELLO,", accent)
	largeText(g, (w-22)/2, 7, "TINY WORLD!", bubbles.Normal)
	tagline := "SMALL SCREEN. BIG HELLO."
	g.Text((w-len(tagline))/2, 10, tagline, bubbles.Muted)
	counter := "POPS " + strconv.Itoa(m.popped)
	palette := paletteNames[m.palette]
	x := (w - 2*len(counter) - 3 - len(palette)) / 2
	largeText(g, x, h-4, counter, accent)
	g.Text(x+2*len(counter)+3, h-3, palette, bubbles.Muted)
	for x := 1; x < w-1; x++ {
		g.Set(x, h-2, bubbles.Cell{Rune: ruleRune, Style: bubbles.Muted})
	}
	g.Text(1, h-1, "0 POP     1 COLOR     2 PAUSE", bubbles.Normal)
	return g
}

func largeText(g *bubbles.Buffer, x, y int, s string, style bubbles.StyleID) {
	for _, r := range s {
		for ty := 0; ty < 2; ty++ {
			for tx := 0; tx < 2; tx++ {
				g.Set(x+tx, y+ty, bubbles.Cell{Rune: largeRune + r*4 + rune(ty*2+tx), Style: style})
			}
		}
		x += 2
	}
}

// Theme keeps a consistent dark background; palette selection changes cell IDs.
func Theme() bubbles.Theme {
	bg := color.RGBA{8, 15, 29, 255}
	t := bubbles.DefaultTheme(false)
	for i := range t.Styles {
		t.Styles[i] = bubbles.Style{Foreground: color.RGBA{237, 243, 248, 255}, Background: bg}
	}
	t.Styles[bubbles.Accent].Foreground = color.RGBA{86, 133, 161, 255}
	t.Styles[bubbles.Muted].Foreground = color.RGBA{123, 153, 180, 255}
	colors := [4]color.RGBA{{77, 228, 239, 255}, {255, 125, 183, 255}, {255, 209, 102, 255}, {136, 239, 173, 255}}
	for i, c := range colors {
		t.Styles[4+i].Foreground = c
	}
	return t
}
