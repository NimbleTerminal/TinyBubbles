package tinybubbles

import (
	"image/color"
	"tinygo.org/x/drivers"
	"tinygo.org/x/tinyfont"
)

// Font draws a glyph relative to a cell's top-left corner. The renderer clips
// glyph pixels to that cell. Implementations must not retain the display.
type Font interface {
	Size() (width, height int16)
	DrawGlyph(display drivers.Displayer, x, y int16, r rune, c color.RGBA)
}

// TinyFont adapts a TinyFont face to fixed cells; Baseline is relative to cell top.
type TinyFont struct {
	Face                    tinyfont.Fonter
	Width, Height, Baseline int16
}

func (f TinyFont) Size() (int16, int16) { return f.Width, f.Height }
func (f TinyFont) DrawGlyph(d drivers.Displayer, x, y int16, r rune, c color.RGBA) {
	if drawUnicodeChartGlyph(d, x, y, f.Width, f.Height, r, c) {
		return
	}
	glyph := f.Face.GetGlyph(r)
	if glyph == nil || glyph.Info().Rune != r {
		glyph = f.Face.GetGlyph('?')
	}
	if glyph != nil {
		glyph.Draw(d, x, y+f.Baseline, c)
	}
}

// DefaultFont fits Org01's five-pixel-wide glyphs into a 6x8 cell.
func DefaultFont() Font { return TinyFont{&tinyfont.Org01, 6, 8, 5} }

// drawUnicodeChartGlyph procedurally rasterizes Unicode Box-Drawing (U+2500..U+257F),
// Block Elements (U+2580..U+259F), and Braille Patterns (U+2800..U+28FF) into any
// (w x h) cell without requiring extra bitmap font tables in flash or RAM.
func drawUnicodeChartGlyph(d drivers.Displayer, x, y, w, h int16, r rune, c color.RGBA) bool {
	if w <= 0 || h <= 0 {
		return false
	}
	switch {
	case r >= 0x2800 && r <= 0x28FF:
		drawBrailleGlyph(d, x, y, w, h, byte(r-0x2800), c)
		return true
	case r >= 0x2580 && r <= 0x259F:
		return drawBlockGlyph(d, x, y, w, h, r, c)
	case r >= 0x2500 && r <= 0x257F:
		return drawBoxGlyph(d, x, y, w, h, r, c)
	default:
		return false
	}
}

func fillCellRect(d drivers.Displayer, x, y, w, h int16, c color.RGBA) {
	for yy := y; yy < y+h; yy++ {
		for xx := x; xx < x+w; xx++ {
			d.SetPixel(xx, yy, c)
		}
	}
}

func drawBrailleGlyph(d drivers.Displayer, x, y, w, h int16, bits byte, c color.RGBA) {
	if bits == 0 {
		return
	}
	// Unicode Braille 2x4 dot bit positions:
	// (0,0)=0x01 (1,0)=0x08
	// (0,1)=0x02 (1,1)=0x10
	// (0,2)=0x04 (1,2)=0x20
	// (0,3)=0x40 (1,3)=0x80
	masks := [4][2]byte{
		{0x01, 0x08},
		{0x02, 0x10},
		{0x04, 0x20},
		{0x40, 0x80},
	}
	colX := [2]int16{x + w/6, x + w/2 + w/6}
	dotW := max(int16(1), w/3)
	dotH := max(int16(1), h/6)
	for row := int16(0); row < 4; row++ {
		py := y + (row*h)/4
		for col := 0; col < 2; col++ {
			if bits&masks[row][col] != 0 {
				fillCellRect(d, colX[col], py, dotW, dotH, c)
			}
		}
	}
}

func drawBlockGlyph(d drivers.Displayer, x, y, w, h int16, r rune, c color.RGBA) bool {
	hw := max(int16(1), w/2)
	hh := max(int16(1), h/2)
	switch {
	case r == 0x2580: // ▀ Upper Half Block
		fillCellRect(d, x, y, w, hh, c)
		return true
	case r >= 0x2581 && r <= 0x2588: // ▁▂▃▄▅▆▇█ Lower 1/8 .. 8/8 Block
		n := int16(r - 0x2580)
		fh := max(int16(1), (n*h+4)/8)
		fillCellRect(d, x, y+h-fh, w, fh, c)
		return true
	case r >= 0x2589 && r <= 0x258F: // ▉▊▋▌▍▎▏ Left 7/8 .. 1/8 Block
		n := int16(0x2590 - r)
		fw := max(int16(1), (n*w+4)/8)
		fillCellRect(d, x, y, fw, h, c)
		return true
	case r == 0x2590: // ▐ Right Half Block
		fillCellRect(d, x+hw, y, w-hw, h, c)
		return true
	case r >= 0x2591 && r <= 0x2593: // ░▒▓ Shade blocks
		step := int16(4 - (r - 0x2590)) // 3, 2, 1
		for yy := int16(0); yy < h; yy++ {
			for xx := int16(0); xx < w; xx++ {
				if (xx+yy)%step == 0 {
					d.SetPixel(x+xx, y+yy, c)
				}
			}
		}
		return true
	case r == 0x2594: // ▔ Upper 1/8 Block
		fillCellRect(d, x, y, w, max(int16(1), h/8), c)
		return true
	case r == 0x2595: // ▕ Right 1/8 Block
		fw := max(int16(1), w/8)
		fillCellRect(d, x+w-fw, y, fw, h, c)
		return true
	case r >= 0x2596 && r <= 0x259F: // Quadrant blocks ▖..▟
		// Bitmask: UL=8, UR=4, LL=2, LR=1
		quadMask := [10]byte{
			0b0010, // U+2596 ▖ LL
			0b0001, // U+2597 ▗ LR
			0b1000, // U+2598 ▘ UL
			0b1011, // U+2599 ▙ UL+LL+LR
			0b1001, // U+259A ▚ UL+LR
			0b1110, // U+259B ▛ UL+UR+LL
			0b1101, // U+259C ▜ UL+UR+LR
			0b0100, // U+259D ▝ UR
			0b0110, // U+259E ▞ UR+LL
			0b0111, // U+259F ▟ UR+LL+LR
		}[r-0x2596]
		if quadMask&8 != 0 {
			fillCellRect(d, x, y, hw, hh, c)
		}
		if quadMask&4 != 0 {
			fillCellRect(d, x+hw, y, w-hw, hh, c)
		}
		if quadMask&2 != 0 {
			fillCellRect(d, x, y+hh, hw, h-hh, c)
		}
		if quadMask&1 != 0 {
			fillCellRect(d, x+hw, y+hh, w-hw, h-hh, c)
		}
		return true
	default:
		return false
	}
}

func drawBoxGlyph(d drivers.Displayer, x, y, w, h int16, r rune, c color.RGBA) bool {
	// Arm bitmask: left=1, right=2, up=4, down=8, heavy=16, arc=32
	var arms byte
	switch r {
	case '─', '╌', '┄', '┈':
		arms = 1 | 2
	case '━', '╍', '┅', '┉', '═':
		arms = 1 | 2 | 16
	case '│', '╎', '┆', '┊':
		arms = 4 | 8
	case '┃', '╏', '┇', '┋', '║':
		arms = 4 | 8 | 16
	case '┌', '┍', '┎', '┏', '╔':
		arms = 2 | 8
	case '┐', '┑', '┒', '┓', '╗':
		arms = 1 | 8
	case '└', '┕', '┖', '┗', '╚':
		arms = 2 | 4
	case '┘', '┙', '┚', '┛', '╝':
		arms = 1 | 4
	case '├', '┝', '┞', '┟', '┠', '┡', '┢', '┣', '╠':
		arms = 4 | 8 | 2
	case '┤', '┥', '┦', '┧', '┨', '┩', '┪', '┫', '╣':
		arms = 4 | 8 | 1
	case '┬', '┭', '┮', '┯', '┰', '┱', '┲', '┳', '╦':
		arms = 1 | 2 | 8
	case '┴', '┵', '┶', '┷', '┸', '┹', '┺', '┻', '╩':
		arms = 1 | 2 | 4
	case '┼', '┽', '┾', '┿', '╀', '╁', '╂', '╃', '╄', '╅', '╆', '╇', '╈', '╉', '╊', '╋', '╬':
		arms = 1 | 2 | 4 | 8
	case '╭':
		arms = 2 | 8 | 32
	case '╮':
		arms = 1 | 8 | 32
	case '╯':
		arms = 1 | 4 | 32
	case '╰':
		arms = 2 | 4 | 32
	case '╴', '╸':
		arms = 1
	case '╵', '╹':
		arms = 4
	case '╶', '╺':
		arms = 2
	case '╷', '╻':
		arms = 8
	case '╽', '╿':
		arms = 4 | 8
	default:
		return false
	}

	cx := x + w/2
	cy := y + h/2
	thick := int16(1)
	if arms&16 != 0 && w >= 4 && h >= 4 {
		thick = 2
	}

	if arms&1 != 0 { // left
		endX := cx + 1
		if arms&32 != 0 {
			endX = cx
		}
		fillCellRect(d, x, cy, max(int16(1), endX-x), thick, c)
	}
	if arms&2 != 0 { // right
		startX := cx
		if arms&32 != 0 && cx+1 < x+w {
			startX = cx + 1
		}
		fillCellRect(d, startX, cy, max(int16(1), x+w-startX), thick, c)
	}
	if arms&4 != 0 { // up
		endY := cy + 1
		if arms&32 != 0 {
			endY = cy
		}
		fillCellRect(d, cx, y, thick, max(int16(1), endY-y), c)
	}
	if arms&8 != 0 { // down
		startY := cy
		if arms&32 != 0 && cy+1 < y+h {
			startY = cy + 1
		}
		fillCellRect(d, cx, startY, thick, max(int16(1), y+h-startY), c)
	}
	if arms&32 == 0 {
		fillCellRect(d, cx, cy, thick, thick, c)
	}
	return true
}
