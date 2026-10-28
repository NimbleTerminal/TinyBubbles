// Package render copies a libghostty-vt screen into a TinyBubbles buffer.
//
// TinyBubbles cells hold one rune and one of eight theme styles, so each cell's
// resolved foreground and background is matched to the nearest theme style.
// Terminal default colors map to Normal. Underline, italic, strikethrough, and
// overline have no cell representation and are dropped. Wide characters keep
// their rune in the first cell and leave the spacer blank. Grapheme clusters
// keep their base codepoint.
//
// The renderer redraws only dirty rows while the buffer still holds its last
// frame, and allocates only its row scratch when the terminal widens.
package render

import (
	"image/color"

	bubbles "nimbleterminal.dev/tinybubbles"
	"nimbleterminal.dev/tinybubbles/tinyghostty/vt"
)

// Renderer keeps per-buffer drawing state. Use one renderer per buffer.
type Renderer struct {
	theme bubbles.Theme
	// Cursor draws the visible cursor with CursorStyle.
	Cursor      bool
	CursorStyle bubbles.StyleID
	// Bold, when not Normal, styles bold cells that use default colors.
	Bold bubbles.StyleID
	// ANSI maps the 16 palette foregrounds on the default background.
	// Other colors use the nearest theme style.
	ANSI [16]bubbles.StyleID

	row   []vt.Cell
	buf   *bubbles.Buffer
	w, h  int
	valid bool

	// The cell under the drawn cursor, restored before the next frame.
	cursorOn   bool
	cx, cy     int
	underCell  bubbles.Cell
	lastFG     uint32
	lastBG     uint32
	lastStyle  bubbles.StyleID
	lastCached bool
}

// New returns a renderer that draws the cursor as Selected and bold default
// text as Accent.
func New(theme bubbles.Theme) *Renderer {
	return &Renderer{theme: theme, Cursor: true, CursorStyle: bubbles.Selected, Bold: bubbles.Accent, ANSI: DefaultANSI}
}

// DefaultANSI maps black and bright black to Muted, cyan to Accent, magenta
// to Red, and white to Normal, for use with Theme.
var DefaultANSI = [16]bubbles.StyleID{
	bubbles.Muted, Red, Green, Yellow, Blue, Red, bubbles.Accent, bubbles.Normal,
	bubbles.Muted, Red, Green, Yellow, Blue, Red, bubbles.Accent, bubbles.Normal,
}

// SetTheme changes the style palette and forces a full redraw.
func (r *Renderer) SetTheme(theme bubbles.Theme) {
	r.theme = theme
	r.Invalidate()
}

// Invalidate forces the next Draw to repaint every row.
func (r *Renderer) Invalidate() {
	r.valid = false
	r.lastCached = false
}

// Draw snapshots term and updates buf. It reports whether buf changed. The
// buffer keeps its size; rows and columns outside the terminal stay blank.
func (r *Renderer) Draw(term *vt.Terminal, buf *bubbles.Buffer) (bool, error) {
	dirty, err := term.Update()
	if err != nil {
		return false, err
	}
	w, h := buf.Size()
	if buf != r.buf || w != r.w || h != r.h {
		r.buf, r.w, r.h = buf, w, h
		r.valid = false
	}
	full := !r.valid || dirty == vt.Full
	cursor := term.Cursor()
	cursorMoved := r.cursorOn != r.wantCursor(cursor) ||
		(r.cursorOn && (r.cx != int(cursor.X) || r.cy != int(cursor.Y)))
	if dirty == vt.Clean && r.valid && !cursorMoved {
		return false, nil
	}

	if full {
		buf.Clear(bubbles.Normal)
		r.cursorOn = false
	} else if r.cursorOn {
		buf.Set(r.cx, r.cy, r.underCell)
		r.cursorOn = false
	}

	cols, _ := term.Size()
	if cap(r.row) < cols {
		r.row = make([]vt.Cell, cols)
	}
	cells := r.row[:cols]
	for {
		y, n, ok := term.NextRow(cells, !full)
		if !ok {
			break
		}
		if y >= h {
			continue
		}
		x := 0
		for ; x < n && x < w; x++ {
			buf.Set(x, y, r.cell(cells[x]))
		}
		for ; x < w; x++ {
			buf.Set(x, y, bubbles.Cell{})
		}
	}

	if r.wantCursor(cursor) {
		x, y := int(cursor.X), int(cursor.Y)
		if x < w && y < h {
			r.underCell = buf.At(x, y)
			buf.Set(x, y, bubbles.Cell{Rune: r.underCell.Rune, Style: r.CursorStyle})
			r.cx, r.cy, r.cursorOn = x, y, true
		}
	}
	r.valid = true
	return true, term.MarkClean()
}

func (r *Renderer) wantCursor(c vt.Cursor) bool {
	return r.Cursor && c.Visible && c.InViewport
}

// cell converts one VT cell to a buffer cell.
func (r *Renderer) cell(c vt.Cell) bubbles.Cell {
	f := c.Flags
	if f&vt.Spacer != 0 {
		// The spacer shares the wide character's background.
		c.Rune = 0
	}
	out := bubbles.Cell{Rune: c.Rune}
	if f&vt.Invisible != 0 {
		out.Rune = 0
	}
	if f&(vt.FG|vt.BG|vt.Inverse|vt.Faint) == 0 {
		if f&vt.Bold != 0 {
			out.Style = r.Bold
		}
		return out
	}
	if f&(vt.BG|vt.Inverse|vt.Faint) == 0 && f&vt.FGPalette != 0 && c.FGIndex < 16 {
		out.Style = r.ANSI[c.FGIndex]
		return out
	}
	normal := r.theme.Styles[bubbles.Normal]
	fg, bg := rgb(normal.Foreground), rgb(normal.Background)
	if f&vt.FG != 0 {
		fg = c.FG
	}
	if f&vt.BG != 0 {
		bg = c.BG
	}
	if f&vt.Inverse != 0 {
		fg, bg = bg, fg
	}
	if f&vt.Faint != 0 {
		fg = mix(fg, bg)
	}
	out.Style = r.nearest(fg, bg)
	if out.Style == bubbles.Normal && f&vt.Bold != 0 {
		out.Style = r.Bold
	}
	return out
}

// nearest returns the theme style closest to fg and bg. Runs of equal colors
// are common, so the last match is cached.
func (r *Renderer) nearest(fg, bg uint32) bubbles.StyleID {
	if r.lastCached && fg == r.lastFG && bg == r.lastBG {
		return r.lastStyle
	}
	best, bestD := bubbles.Normal, uint32(1<<32-1)
	for i, s := range r.theme.Styles {
		d := dist(fg, rgb(s.Foreground)) + dist(bg, rgb(s.Background))
		if d < bestD {
			best, bestD = bubbles.StyleID(i), d
		}
	}
	r.lastFG, r.lastBG, r.lastStyle, r.lastCached = fg, bg, best, true
	return best
}

func rgb(c color.RGBA) uint32 { return uint32(c.R)<<16 | uint32(c.G)<<8 | uint32(c.B) }

// dist is a weighted squared RGB distance; green counts most.
func dist(a, b uint32) uint32 {
	dr := int32(a>>16&0xff) - int32(b>>16&0xff)
	dg := int32(a>>8&0xff) - int32(b>>8&0xff)
	db := int32(a&0xff) - int32(b&0xff)
	return uint32(2*dr*dr + 4*dg*dg + 3*db*db)
}

func mix(a, b uint32) uint32 {
	return (a>>1)&0x7f7f7f + (b>>1)&0x7f7f7f
}

// Theme returns a palette for terminal output. Monochrome uses
// bubbles.DefaultTheme(true). Color keeps the shared Normal, Accent, Selected,
// and Muted meanings and adds red, green, yellow, and blue.
func Theme(monochrome bool) bubbles.Theme {
	t := bubbles.DefaultTheme(monochrome)
	if monochrome {
		return t
	}
	black := color.RGBA{A: 255}
	for i, c := range []color.RGBA{
		{230, 80, 80, 255},  // Red
		{90, 210, 110, 255}, // Green
		{235, 200, 80, 255}, // Yellow
		{90, 140, 255, 255}, // Blue
	} {
		t.Styles[Red+bubbles.StyleID(i)] = bubbles.Style{Foreground: c, Background: black}
	}
	return t
}

// Style IDs that Theme adds after the shared four.
const (
	Red bubbles.StyleID = iota + 4
	Green
	Yellow
	Blue
)
