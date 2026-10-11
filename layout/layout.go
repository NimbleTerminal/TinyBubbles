// Package layout divides cell grids and composes existing buffers without allocation.
// Parents own widget sizing, input focus, and buffer lifetimes.
package layout

import bubbles "nimbleterminal.dev/tinybubbles"

// Rect describes a region in cells, not pixels. Nonpositive extents are empty.
type Rect struct{ X, Y, Width, Height int }

func Bounds(b *bubbles.Buffer) Rect {
	if b == nil {
		return Rect{}
	}
	w, h := b.Size()
	return Rect{Width: w, Height: h}
}
func (r Rect) normalized() Rect { r.Width = max(0, r.Width); r.Height = max(0, r.Height); return r }

// Inset removes padding. Negative padding is ignored; excessive padding leaves
// an empty region. Left and top take priority when opposite sides cannot fit.
func (r Rect) Inset(top, right, bottom, left int) Rect {
	r = r.normalized()
	left = min(max(0, left), r.Width)
	top = min(max(0, top), r.Height)
	r.X += left
	r.Y += top
	r.Width -= left
	r.Height -= top
	r.Width -= min(max(0, right), r.Width)
	r.Height -= min(max(0, bottom), r.Height)
	return r
}

// Rows reserves first rows at the top, skips gap rows, and returns the remainder.
// The first region takes priority on small screens. All sizes clamp to available space.
func (r Rect) Rows(first, gap int) (Rect, Rect) {
	r = r.normalized()
	n := min(max(0, first), r.Height)
	g := min(max(0, gap), r.Height-n)
	return Rect{r.X, r.Y, r.Width, n}, Rect{r.X, r.Y + n + g, r.Width, r.Height - n - g}
}

// Columns reserves first columns at the left, then a gap, then the remainder.
func (r Rect) Columns(first, gap int) (Rect, Rect) {
	r = r.normalized()
	n := min(max(0, first), r.Width)
	g := min(max(0, gap), r.Width-n)
	return Rect{r.X, r.Y, n, r.Height}, Rect{r.X + n + g, r.Y, r.Width - n - g, r.Height}
}

type Alignment uint8

const (
	Start Alignment = iota
	Center
	End
)

// Align places a child within the region. Oversized children are cropped to fit.
// Odd spare cells go after a centered child; invalid alignment acts like Start.
func (r Rect) Align(width, height int, horizontal, vertical Alignment) Rect {
	r = r.normalized()
	w, h := min(max(0, width), r.Width), min(max(0, height), r.Height)
	return Rect{r.X + offset(r.Width-w, horizontal), r.Y + offset(r.Height-h, vertical), w, h}
}
func offset(spare int, a Alignment) int {
	switch a {
	case Center:
		return spare / 2
	case End:
		return spare
	}
	return 0
}

// Draw copies source cells, including blank cells and styles, into the region's
// top left. It clips to both buffers and the region. Uncovered destination cells
// remain unchanged: clear the parent before composing each frame. Self-copy is supported.
func Draw(dst *bubbles.Buffer, region Rect, src *bubbles.Buffer) {
	if dst == nil || src == nil {
		return
	}
	region = region.normalized()
	sw, sh := src.Size()
	dw, dh := dst.Size()
	x0, y0 := max(0, region.X), max(0, region.Y)
	x1, y1 := min(dw, region.X+min(region.Width, sw)), min(dh, region.Y+min(region.Height, sh))
	if dst == src && (region.Y > 0 || region.Y == 0 && region.X > 0) {
		for y := y1 - 1; y >= y0; y-- {
			for x := x1 - 1; x >= x0; x-- {
				dst.Set(x, y, src.At(x-region.X, y-region.Y))
			}
		}
		return
	}
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			dst.Set(x, y, src.At(x-region.X, y-region.Y))
		}
	}
}

// Text draws plain text clipped to the region. Newlines start another row;
// carriage returns reset the column and other ASCII controls are ignored.
func Text(dst *bubbles.Buffer, region Rect, text string, style bubbles.StyleID) {
	if dst == nil || region.Width <= 0 || region.Height <= 0 {
		return
	}
	x, y := 0, 0
	for _, r := range text {
		switch r {
		case '\n':
			x = 0
			y++
		case '\r':
			x = 0
		default:
			if r >= 32 && r != 127 {
				if x < region.Width {
					dst.Set(region.X+x, region.Y+y, bubbles.Cell{Rune: r, Style: style})
				}
				x++
			}
		}
		if y >= region.Height {
			return
		}
	}
}

// ClipText draws one line, ignoring ASCII controls (including newline and DEL).
// Each remaining rune occupies one cell. Overflow replaces the last cell with
// a dot; width one preserves the first rune. Alignment counts only displayed runes. It allocates
// nothing and leaves cells outside the text unchanged.
func ClipText(dst *bubbles.Buffer, region Rect, text string, style bubbles.StyleID, align Alignment) {
	if dst == nil || region.Width <= 0 || region.Height <= 0 {
		return
	}
	count := 0
	for _, r := range text {
		if r >= 32 && r != 127 {
			count++
		}
	}
	width := min(count, region.Width)
	x := region.X + offset(region.Width-width, align)
	end := x + width
	for _, r := range text {
		if r < 32 || r == 127 {
			continue
		}
		if x >= end {
			break
		}
		if region.Width > 1 && count > region.Width && x == end-1 {
			r = '.'
		}
		dst.Set(x, region.Y, bubbles.Cell{Rune: r, Style: style})
		x++
	}
}
