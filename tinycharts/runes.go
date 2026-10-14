// Package tinycharts provides allocation-conscious chart widgets for TinyBubbles
// cell buffers and embedded displays, inspired by NimbleMarkets/ntcharts.
//
// Charts render standard Unicode Box-Drawing (U+2500..U+257F), Block Elements
// (U+2580..U+259F), and Braille Patterns (U+2800..U+28FF) into *tinybubbles.Buffer
// regions. TinyBubbles' DefaultFont procedurally rasterizes all three Unicode
// ranges without extra font tables in flash or RAM.
package tinycharts

import (
	bubbles "nimbleterminal.dev/tinybubbles"
	"nimbleterminal.dev/tinybubbles/layout"
	"tinygo.org/x/drivers"
)

// Box-drawing, arc, block, and Braille runes matching ntcharts/canvas/runes.
const (
	RuneLineHorizontal  = '\u2500' // ─
	RuneLineVertical    = '\u2502' // │
	RuneLineDownRight   = '\u250C' // ┌
	RuneLineDownLeft    = '\u2510' // ┐
	RuneLineUpRight     = '\u2514' // └
	RuneLineUpLeft      = '\u2518' // ┘
	RuneLineVertRight   = '\u251C' // ├
	RuneLineVertLeft    = '\u2524' // ┤
	RuneLineHorizUp     = '\u2534' // ┴
	RuneLineHorizDown   = '\u252C' // ┬
	RuneLineCross       = '\u253C' // ┼
	RuneArcDownRight    = '\u256D' // ╭
	RuneArcDownLeft     = '\u256E' // ╮
	RuneArcUpLeft       = '\u256F' // ╯
	RuneArcUpRight      = '\u2570' // ╰
	RuneUpperHalfBlock  = '\u2580' // ▀
	RuneLowerBlockOne   = '\u2581' // ▁
	RuneLowerBlockTwo   = '\u2582' // ▂
	RuneLowerBlockThree = '\u2583' // ▃
	RuneLowerBlockFour  = '\u2584' // ▄
	RuneLowerBlockFive  = '\u2585' // ▅
	RuneLowerBlockSix   = '\u2586' // ▆
	RuneLowerBlockSeven = '\u2587' // ▇
	RuneFullBlock       = '\u2588' // █
	RuneLeftBlockSeven  = '\u2589' // ▉
	RuneLeftBlockSix    = '\u258A' // ▊
	RuneLeftBlockFive   = '\u258B' // ▋
	RuneLeftBlockFour   = '\u258C' // ▌
	RuneLeftBlockThree  = '\u258D' // ▍
	RuneLeftBlockTwo    = '\u258E' // ▎
	RuneLeftBlockOne    = '\u258F' // ▏
	RuneRightHalfBlock  = '\u2590' // ▐
	RuneLightShade      = '\u2591' // ░
	RuneMediumShade     = '\u2592' // ▒
	RuneDarkShade       = '\u2593' // ▓
	RuneUpperOneEighth  = '\u2594' // ▔
	RuneBrailleBlank    = '\u2800' // ⠀
)

var lowerBlocks = [9]rune{
	' ',
	RuneLowerBlockOne,
	RuneLowerBlockTwo,
	RuneLowerBlockThree,
	RuneLowerBlockFour,
	RuneLowerBlockFive,
	RuneLowerBlockSix,
	RuneLowerBlockSeven,
	RuneFullBlock,
}

var leftBlocks = [9]rune{
	' ',
	RuneLeftBlockOne,
	RuneLeftBlockTwo,
	RuneLeftBlockThree,
	RuneLeftBlockFour,
	RuneLeftBlockFive,
	RuneLeftBlockSix,
	RuneLeftBlockSeven,
	RuneFullBlock,
}

var quadrantBlocks = [16]rune{
	' ', '\u2597', '\u2596', RuneLowerBlockFour,
	'\u259D', RuneRightHalfBlock, '\u259E', '\u259F',
	'\u2598', '\u259A', RuneLeftBlockFour, '\u2599',
	RuneUpperHalfBlock, '\u259C', '\u259B', RuneFullBlock,
}

// LowerBlock returns the lower fractional block rune for eighths in 0..8.
func LowerBlock(eighths int) rune {
	if eighths <= 0 {
		return ' '
	}
	if eighths >= 8 {
		return RuneFullBlock
	}
	return lowerBlocks[eighths]
}

// LeftBlock returns the left fractional block rune for eighths in 0..8.
func LeftBlock(eighths int) rune {
	if eighths <= 0 {
		return ' '
	}
	if eighths >= 8 {
		return RuneFullBlock
	}
	return leftBlocks[eighths]
}

// QuadrantBlock returns the Unicode 2x2 quadrant block rune for the given corners.
func QuadrantBlock(upperLeft, upperRight, lowerLeft, lowerRight bool) rune {
	mask := 0
	if upperLeft {
		mask |= 8
	}
	if upperRight {
		mask |= 4
	}
	if lowerLeft {
		mask |= 2
	}
	if lowerRight {
		mask |= 1
	}
	return quadrantBlocks[mask]
}

// BrailleGrid implements a 2D sub-cell dot matrix using Unicode Braille patterns (U+2800..U+28FF).
// Each cell represents a 2-wide by 4-high dot grid, providing 2*Width by 4*Height plotting resolution.
type BrailleGrid struct {
	cWidth  int
	cHeight int
	cells   []byte
}

var brailleDotBits = [4][2]byte{
	{0x01, 0x08},
	{0x02, 0x10},
	{0x04, 0x20},
	{0x40, 0x80},
}

// NewBrailleGrid allocates a BrailleGrid sized for cWidth by cHeight character cells.
func NewBrailleGrid(cWidth, cHeight int) *BrailleGrid {
	g := &BrailleGrid{}
	g.Resize(cWidth, cHeight)
	return g
}

// Resize adjusts the cell dimensions of the BrailleGrid, reusing backing storage when possible.
func (g *BrailleGrid) Resize(cWidth, cHeight int) {
	cWidth = max(0, cWidth)
	cHeight = max(0, cHeight)
	n := cWidth * cHeight
	if cap(g.cells) < n {
		g.cells = make([]byte, n)
	} else {
		g.cells = g.cells[:n]
	}
	g.cWidth = cWidth
	g.cHeight = cHeight
	g.Clear()
}

// CellSize returns the width and height in character cells.
func (g *BrailleGrid) CellSize() (width, height int) {
	return g.cWidth, g.cHeight
}

// DotSize returns the plotting resolution in sub-cell dots (2*cWidth, 4*cHeight).
func (g *BrailleGrid) DotSize() (width, height int) {
	return g.cWidth * 2, g.cHeight * 4
}

// Clear resets all Braille dots in the grid.
func (g *BrailleGrid) Clear() {
	clear(g.cells)
}

// Set enables the sub-cell dot at dot coordinates (dx, dy) where (0,0) is top-left.
func (g *BrailleGrid) Set(dx, dy int) {
	if dx < 0 || dy < 0 || dx >= g.cWidth*2 || dy >= g.cHeight*4 {
		return
	}
	cx, cy := dx/2, dy/4
	bit := brailleDotBits[dy%4][dx%2]
	g.cells[cy*g.cWidth+cx] |= bit
}

// DrawLine plots a connected Bresenham line between dot coordinates (x0, y0) and (x1, y1).
func (g *BrailleGrid) DrawLine(x0, y0, x1, y1 int) {
	dx := x1 - x0
	if dx < 0 {
		dx = -dx
	}
	sx := -1
	if x0 < x1 {
		sx = 1
	}
	dy := y1 - y0
	if dy > 0 {
		dy = -dy
	}
	sy := -1
	if y0 < y1 {
		sy = 1
	}
	err := dx + dy
	for {
		g.Set(x0, y0)
		if x0 == x1 && y0 == y1 {
			break
		}
		e2 := 2 * err
		if e2 >= dy {
			err += dy
			x0 += sx
		}
		if e2 <= dx {
			err += dx
			y0 += sy
		}
	}
}

// RuneAt returns the Unicode Braille rune at cell (cx, cy), or ' ' if empty.
func (g *BrailleGrid) RuneAt(cx, cy int) rune {
	if cx < 0 || cy < 0 || cx >= g.cWidth || cy >= g.cHeight {
		return ' '
	}
	bits := g.cells[cy*g.cWidth+cx]
	if bits == 0 {
		return ' '
	}
	return RuneBrailleBlank + rune(bits)
}

// Draw writes the non-empty Braille cells into dst clipped to rect using style.
func (g *BrailleGrid) Draw(dst *bubbles.Buffer, rect layout.Rect, style bubbles.StyleID) {
	if dst == nil || rect.Width <= 0 || rect.Height <= 0 {
		return
	}
	w := min(rect.Width, g.cWidth)
	h := min(rect.Height, g.cHeight)
	for cy := 0; cy < h; cy++ {
		for cx := 0; cx < w; cx++ {
			if r := g.RuneAt(cx, cy); r != ' ' {
				dst.Set(rect.X+cx, rect.Y+cy, bubbles.Cell{Rune: r, Style: style})
			}
		}
	}
}

// PaintBuffer renders a TinyBubbles cell Buffer onto a pixel Displayer at pixel offset (px, py)
// using the given Font and Theme. This allows standalone pixel targets (such as tinyhal/workbench)
// to embed tinycharts cell regions directly inside a pixel framebuffer.
func PaintBuffer(d drivers.Displayer, px, py int16, buf *bubbles.Buffer, font bubbles.Font, theme bubbles.Theme) {
	if d == nil || buf == nil {
		return
	}
	if font == nil {
		font = bubbles.DefaultFont()
	}
	cw, ch := font.Size()
	cols, rows := buf.Size()
	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			cell := buf.At(x, y)
			st := theme.Styles[bubbles.Normal]
			if int(cell.Style) < len(theme.Styles) {
				st = theme.Styles[cell.Style]
			}
			cx := px + int16(x)*cw
			cy := py + int16(y)*ch
			for yy := cy; yy < cy+ch; yy++ {
				for xx := cx; xx < cx+cw; xx++ {
					d.SetPixel(xx, yy, st.Background)
				}
			}
			if cell.Rune != 0 && cell.Rune != ' ' {
				font.DrawGlyph(d, cx, cy, cell.Rune, st.Foreground)
			}
		}
	}
}
