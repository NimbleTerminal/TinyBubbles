// Package tinybubbles provides a cooperative Model/Update/View loop for embedded displays.
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
// Each rune occupies one cell; combining and wide characters are not composed.
type Buffer struct {
	width, height int
	cells         []Cell
}

// NewBuffer allocates a grid. Nonpositive dimensions produce an empty buffer.
func NewBuffer(width, height int) *Buffer {
	if width <= 0 || height <= 0 {
		return &Buffer{}
	}
	return &Buffer{width: width, height: height, cells: make([]Cell, width*height)}
}

func (b *Buffer) Size() (int, int) { return b.width, b.height }

// Set clips writes outside the grid.
func (b *Buffer) Set(x, y int, cell Cell) {
	if x >= 0 && y >= 0 && x < b.width && y < b.height {
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
	for i := range b.cells {
		b.cells[i] = Cell{Style: style}
	}
}

// Text writes UTF-8 text, clipping at the right edge. Newlines return to x.
// Carriage returns return to x; other control characters are ignored.
func (b *Buffer) Text(x, y int, text string, style StyleID) {
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
