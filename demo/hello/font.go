package hello

import (
	bubbles "github.com/NimbleTerminal/TinyBubbles"
	"image/color"
	"tinygo.org/x/drivers"
)

// Private-use runes encode four tiles of a sprite or doubled text glyph. The
// renderer still sees ordinary cells and only repaints the ones that change.
const (
	bubbleRune rune = 0xe000
	ruleRune   rune = 0xe004
	largeRune  rune = 0xe100
)

type Font struct {
	base   bubbles.Font
	scaled scaledDisplay
}

func NewFont() *Font               { return &Font{base: bubbles.DefaultFont()} }
func (*Font) Size() (int16, int16) { return 6, 8 }
func (f *Font) DrawGlyph(d drivers.Displayer, x, y int16, r rune, c color.RGBA) {
	switch {
	case r >= bubbleRune && r < bubbleRune+4:
		tile := int(r - bubbleRune)
		for yy := 0; yy < 8; yy++ {
			for xx := 0; xx < 6; xx++ {
				px, py := xx+(tile%2)*6, yy+(tile/2)*8
				dx, dy := 2*px-11, 2*py-15
				dist := dx*dx + dy*dy
				if (dist >= 82 && dist <= 133) || (px == 3 && py >= 5 && py <= 6) {
					d.SetPixel(x+int16(xx), y+int16(yy), c)
				}
			}
		}
	case r == ruleRune:
		for xx := int16(0); xx < 6; xx++ {
			d.SetPixel(x+xx, y+4, c)
		}
	case r >= largeRune && r < largeRune+128*4:
		v := r - largeRune
		tile := v % 4
		f.scaled = scaledDisplay{display: d, x: x - int16(tile%2)*6, y: y - int16(tile/2)*8}
		f.base.DrawGlyph(&f.scaled, 0, 0, v/4, c)
		f.scaled.display = nil // do not retain the renderer's clipping adapter
	default:
		f.base.DrawGlyph(d, x, y, r, c)
	}
}

type scaledDisplay struct {
	display drivers.Displayer
	x, y    int16
}

func (*scaledDisplay) Size() (int16, int16) { return 6, 8 }
func (*scaledDisplay) Display() error       { return nil }
func (d *scaledDisplay) SetPixel(x, y int16, c color.RGBA) {
	for yy := int16(0); yy < 2; yy++ {
		for xx := int16(0); xx < 2; xx++ {
			d.display.SetPixel(d.x+2*x+xx, d.y+2*y+yy, c)
		}
	}
}
