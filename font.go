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
