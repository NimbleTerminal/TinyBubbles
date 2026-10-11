// Package tinybubbles provides cell rendering, fonts, and themes for embedded
// displays. Use the bubbletea module for the primary runtime or tinytea for the
// cooperative subset. The original root runtime remains for compatibility.
package tinybubbles

// StyleID indexes the renderer's eight-entry theme palette.
type StyleID uint8

const (
	Normal StyleID = iota
	Accent
	Selected
	Muted
)

// Cell occupies eight bytes on typical 32-bit targets. A zero rune is blank.
type Cell struct {
	Rune  rune
	Style StyleID
}

// Buffer is a row-major character grid. Reuse it between View calls.
// A nil buffer behaves as an empty grid: reads return zero values and writes,
// clears, and resizes do nothing. Resize cannot create a buffer through a nil pointer.
// Each rune occupies one cell; combining and wide characters are not composed.
type Buffer struct {
	width, height int
	cells         []Cell
}

// NewBuffer allocates a grid. Nonpositive dimensions produce an empty buffer.
func NewBuffer(width, height int) *Buffer {
	b := &Buffer{}
	b.Resize(width, height)
	return b
}

// Resize clears and reallocates the grid only when its dimensions change.
// Unchanged dimensions preserve cells and allocate nothing. Nonpositive
// dimensions normalize to an empty 0×0 grid. It reports whether the size changed.
func (b *Buffer) Resize(width, height int) bool {
	if b == nil {
		return false
	}
	if width <= 0 || height <= 0 {
		width, height = 0, 0
	}
	if b.width == width && b.height == height {
		return false
	}
	b.width, b.height = width, height
	b.cells = make([]Cell, width*height)
	return true
}

func (b *Buffer) Size() (int, int) {
	if b == nil {
		return 0, 0
	}
	return b.width, b.height
}

// Set clips writes outside the grid.
func (b *Buffer) Set(x, y int, cell Cell) {
	if b != nil && x >= 0 && y >= 0 && x < b.width && y < b.height {
		b.cells[y*b.width+x] = cell
	}
}

// At returns a blank cell outside the grid.
func (b *Buffer) At(x, y int) Cell {
	if b == nil || x < 0 || y < 0 || x >= b.width || y >= b.height {
		return Cell{}
	}
	return b.cells[y*b.width+x]
}

func (b *Buffer) Clear(style StyleID) {
	if b == nil {
		return
	}
	for i := range b.cells {
		b.cells[i] = Cell{Style: style}
	}
}

// Text writes UTF-8 text, clipping at the right edge. Newlines return to x.
// Carriage returns return to x; other control characters are ignored.
func (b *Buffer) Text(x, y int, text string, style StyleID) {
	if b == nil {
		return
	}
	start := x
	for _, r := range text {
		switch r {
		case '\n':
			x = start
			y++
		case '\r':
			x = start
		default:
			if r >= 32 && r != 127 {
				b.Set(x, y, Cell{r, style})
				x++
			}
		}
	}
}
