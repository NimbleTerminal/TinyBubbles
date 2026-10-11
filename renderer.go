package tinybubbles

import (
	"errors"
	"image/color"
	"tinygo.org/x/drivers"
)

// RectangleFiller is detected dynamically; plain Displayer implementations work too.
type RectangleFiller interface {
	FillRectangle(x, y, width, height int16, c color.RGBA) error
}

// Renderer stores one previous cell grid, never a pixel framebuffer.
type Renderer struct {
	display               drivers.Displayer
	font                  Font
	theme                 Theme
	width, height, cw, ch int16
	cols, rows            int
	previous              []Cell
	valid                 bool
	clip                  clippedDisplay
}

func NewRenderer(d drivers.Displayer, font Font, theme Theme) (*Renderer, error) {
	if d == nil {
		return nil, errors.New("tinybubbles: nil display")
	}
	if font == nil {
		font = DefaultFont()
	}
	w, h := d.Size()
	cw, ch := font.Size()
	if w <= 0 || h <= 0 || cw <= 0 || ch <= 0 || cw > w || ch > h {
		return nil, errors.New("tinybubbles: invalid display or font dimensions")
	}
	r := &Renderer{display: d, font: font, theme: theme, width: w, height: h, cw: cw, ch: ch, cols: int(w / cw), rows: int(h / ch)}
	r.previous = make([]Cell, r.cols*r.rows)
	r.clip.display = d
	return r, nil
}

func (r *Renderer) Size() (columns, rows int) { return r.cols, r.rows }
func (r *Renderer) Invalidate()               { r.valid = false }
func (r *Renderer) SetTheme(theme Theme)      { r.theme = theme; r.Invalidate() }

func (r *Renderer) fill(x, y, w, h int16, c color.RGBA) error {
	if f, ok := r.display.(RectangleFiller); ok {
		return f.FillRectangle(x, y, w, h, c)
	}
	for yy := y; yy < y+h; yy++ {
		for xx := x; xx < x+w; xx++ {
			r.display.SetPixel(xx, yy, c)
		}
	}
	return nil
}

// Render clips a view to the screen, clears missing cells, and flushes once if
// anything changed. On a driver error the whole frame is retried next time.
// Configure orientation before construction; recreate after a physical resize.
func (r *Renderer) Render(view *Buffer) error {
	full := !r.valid
	if full {
		if err := r.fill(0, 0, r.width, r.height, r.theme.Styles[Normal].Background); err != nil {
			return err
		}
	}
	changed := full
	for y := 0; y < r.rows; y++ {
		for x := 0; x < r.cols; x++ {
			cell := view.At(x, y)
			i := y*r.cols + x
			if !full && cell == r.previous[i] {
				continue
			}
			changed = true
			style := r.theme.style(cell.Style)
			px, py := int16(x)*r.cw, int16(y)*r.ch
			if !full || style.Background != r.theme.Styles[Normal].Background {
				if err := r.fill(px, py, r.cw, r.ch, style.Background); err != nil {
					r.valid = false
					return err
				}
			}
			if cell.Rune != 0 && cell.Rune != ' ' {
				r.clip.x, r.clip.y, r.clip.w, r.clip.h = px, py, r.cw, r.ch
				r.font.DrawGlyph(&r.clip, px, py, cell.Rune, style.Foreground)
			}
		}
	}
	if !changed {
		return nil
	}
	if err := r.display.Display(); err != nil {
		r.valid = false
		return err
	}
	for y := 0; y < r.rows; y++ {
		for x := 0; x < r.cols; x++ {
			r.previous[y*r.cols+x] = view.At(x, y)
		}
	}
	r.valid = true
	return nil
}

type clippedDisplay struct {
	display    drivers.Displayer
	x, y, w, h int16
}

func (d *clippedDisplay) Size() (int16, int16) { return d.display.Size() }
func (d *clippedDisplay) Display() error       { return nil }
func (d *clippedDisplay) SetPixel(x, y int16, c color.RGBA) {
	if x >= d.x && y >= d.y && x < d.x+d.w && y < d.y+d.h {
		d.display.SetPixel(x, y, c)
	}
}
