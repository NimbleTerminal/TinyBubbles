// Package vt binds the small part of libghostty-vt that TinyBubbles needs:
// terminal lifetime, VT input, resize, render snapshots, cursor, colors, and
// mode queries. A C shim flattens Ghostty's structs so the package builds with
// both Go and TinyGo cgo against a static libghostty-vt.a.
//
// Build the library first with scripts/build-libghostty-vt.sh. It installs
// headers into lib/include and archives into lib/<GOOS>_<GOARCH>. Go links the
// host archive; TinyGo links lib/wasm, lib/cortexm, lib/esp32c3, or
// lib/esp32s3. TinyGo rejects build constraints on #cgo lines and does not
// expand ${SRCDIR}, so cgo_*.go files carry each toolchain's and target's
// flags; TinyGo resolves relative paths itself.
//
// A Terminal is not safe for concurrent use.
package vt

// #include "shim.h"
import "C"

import (
	"errors"
	"unsafe"
)

// Cell is one screen cell. Colors are 0xRRGGBB and valid only when FG or BG
// is set; unset colors use the terminal defaults from Colors. Palette colors
// also report their index with FGPalette or BGPalette.
type Cell struct {
	Rune             rune
	FG, BG           uint32
	Flags            Flags
	FGIndex, BGIndex uint8
}

type Flags uint16

const (
	FG Flags = 1 << iota
	BG
	Bold
	Italic
	Faint
	Blink
	Inverse
	Invisible
	Strike
	Overline
	Underline
	// Wide marks the first cell of a double-width character; the next cell
	// is a Spacer and should not be drawn.
	Wide
	Spacer
	// Grapheme marks a cluster reduced to its base codepoint.
	Grapheme
	// FGPalette and BGPalette mark colors chosen by palette index.
	FGPalette
	BGPalette
)

// The Go structs mirror the shim's; a size mismatch fails to compile.
var (
	_ = [1]struct{}{}[unsafe.Sizeof(Cell{})-unsafe.Sizeof(C.tgvt_cell{})]
	_ = [1]struct{}{}[unsafe.Sizeof(Cursor{})-unsafe.Sizeof(C.tgvt_cursor{})]
	_ = [1]struct{}{}[unsafe.Sizeof(Colors{})-unsafe.Sizeof(C.tgvt_colors{})]
)

type CursorStyle uint8

const (
	CursorBar CursorStyle = iota
	CursorBlock
	CursorUnderline
	CursorBlockHollow
)

// Cursor reports the cursor position in viewport cells. X and Y are valid only
// when InViewport is true. Visible follows DECTCEM.
type Cursor struct {
	X, Y       uint16
	InViewport bool
	Visible    bool
	Blinking   bool
	Style      CursorStyle
}

// Colors are the effective default foreground, background, and cursor colors.
type Colors struct {
	FG, BG, Cursor uint32
	CursorSet      bool
	_              [3]uint8
}

// Dirty reports how much of the screen changed since the last Clean.
type Dirty int

const (
	Clean Dirty = iota
	Partial
	Full
)

// Mode is a terminal mode number. ANSI modes use ANSIMode.
type Mode uint16

const (
	ModeCursorKeys     Mode = 1    // DECCKM: application cursor keys
	ModeCursorVisible  Mode = 25   // DECTCEM
	ModeAltScreen      Mode = 1049 // alternate screen with saved cursor
	ModeBracketedPaste Mode = 2004
)

// Error is a GhosttyResult code.
type Error int

func (e Error) Error() string {
	switch e {
	case -1:
		return "ghostty: out of memory"
	case -2:
		return "ghostty: invalid value"
	case -3:
		return "ghostty: out of space"
	case -4:
		return "ghostty: no value"
	}
	return "ghostty: error"
}

var ErrClosed = errors.New("ghostty: terminal closed")

// Terminal owns a Ghostty terminal and its render state.
type Terminal struct {
	p          *C.tgvt
	cols, rows int
	// cgo out-parameters live here; locals passed by pointer escape.
	y      C.uint16_t
	cursor Cursor
}

// New creates a cols×rows terminal. Dimensions are clamped to 1..65535.
func New(cols, rows int) (*Terminal, error) {
	cols, rows = clamp(cols), clamp(rows)
	var res C.int
	p := C.tgvt_new(C.uint16_t(cols), C.uint16_t(rows), &res)
	if p == nil {
		return nil, Error(res)
	}
	return &Terminal{p: p, cols: cols, rows: rows}, nil
}

func clamp(n int) int {
	if n < 1 {
		return 1
	}
	if n > 0xffff {
		return 0xffff
	}
	return n
}

// Close frees the terminal. Later calls report ErrClosed or do nothing.
func (t *Terminal) Close() {
	if t != nil && t.p != nil {
		C.tgvt_free(t.p)
		t.p = nil
	}
}

func (t *Terminal) Size() (cols, rows int) { return t.cols, t.rows }

// Write feeds VT bytes from the application. It always consumes all of p.
func (t *Terminal) Write(p []byte) (int, error) {
	if t.p == nil {
		return 0, ErrClosed
	}
	if len(p) > 0 {
		C.tgvt_write(t.p, (*C.uint8_t)(unsafe.Pointer(&p[0])), C.size_t(len(p)))
	}
	return len(p), nil
}

// WriteString avoids a copy for constant sequences on Go; TinyGo may copy.
func (t *Terminal) WriteString(s string) (int, error) {
	return t.Write([]byte(s))
}

// Reset performs a full terminal reset (RIS).
func (t *Terminal) Reset() {
	if t.p != nil {
		C.tgvt_reset(t.p)
	}
}

// Resize reflows the screen. Dimensions are clamped to 1..65535.
func (t *Terminal) Resize(cols, rows int) error {
	if t.p == nil {
		return ErrClosed
	}
	cols, rows = clamp(cols), clamp(rows)
	if r := C.tgvt_resize(t.p, C.uint16_t(cols), C.uint16_t(rows)); r != 0 {
		return Error(r)
	}
	t.cols, t.rows = cols, rows
	return nil
}

// SetScrollback limits retained history in bytes. New terminals keep none.
// Ghostty prunes whole pages, so the limit is approximate.
func (t *Terminal) SetScrollback(bytes int) error {
	if t.p == nil {
		return ErrClosed
	}
	if bytes < 0 {
		bytes = 0
	}
	if r := C.tgvt_set_scrollback(t.p, C.size_t(bytes)); r != 0 {
		return Error(r)
	}
	return nil
}

// Update snapshots the screen for drawing and rewinds the row iterator.
func (t *Terminal) Update() (Dirty, error) {
	if t.p == nil {
		return Clean, ErrClosed
	}
	r := C.tgvt_update(t.p)
	if r < 0 {
		return Clean, Error(r)
	}
	return Dirty(r), nil
}

// NextRow copies the next snapshot row into cells and returns its viewport y
// and cell count. With dirtyOnly, unchanged rows are skipped. It returns
// ok=false after the last row. Call Update first.
func (t *Terminal) NextRow(cells []Cell, dirtyOnly bool) (y, n int, ok bool) {
	if t.p == nil {
		return 0, 0, false
	}
	var out *C.tgvt_cell
	max := len(cells)
	if max > 0xffff {
		max = 0xffff
	}
	if max > 0 {
		out = (*C.tgvt_cell)(unsafe.Pointer(&cells[0]))
	}
	d := C.int(0)
	if dirtyOnly {
		d = 1
	}
	r := C.tgvt_next_row(t.p, d, out, C.uint16_t(max), &t.y)
	if r < 0 {
		return 0, 0, false
	}
	return int(t.y), int(r), true
}

// MarkClean clears global and per-row dirty state after a complete draw.
func (t *Terminal) MarkClean() error {
	if t.p == nil {
		return ErrClosed
	}
	if r := C.tgvt_clean(t.p); r != 0 {
		return Error(r)
	}
	return nil
}

// Cursor reads the snapshot cursor.
func (t *Terminal) Cursor() Cursor {
	t.cursor = Cursor{}
	if t.p != nil {
		C.tgvt_cursor_get(t.p, (*C.tgvt_cursor)(unsafe.Pointer(&t.cursor)))
	}
	return t.cursor
}

// Colors reads the snapshot default colors.
func (t *Terminal) Colors() Colors {
	var c Colors
	if t.p != nil {
		C.tgvt_colors_get(t.p, (*C.tgvt_colors)(unsafe.Pointer(&c)))
	}
	return c
}

// DECMode reports a DEC private mode such as ModeCursorKeys.
func (t *Terminal) DECMode(m Mode) bool { return t.mode(m, false) }

// ANSIMode reports an ANSI mode such as 4 (insert).
func (t *Terminal) ANSIMode(m Mode) bool { return t.mode(m, true) }

func (t *Terminal) mode(m Mode, ansi bool) bool {
	if t.p == nil {
		return false
	}
	a := C.int(0)
	if ansi {
		a = 1
	}
	return C.tgvt_mode(t.p, C.uint16_t(m), a) == 1
}

// ReadReply drains bytes the terminal wants to send back to the application,
// such as cursor position and device attribute reports. The shim holds 512
// bytes; overflow is counted by ReplyDropped.
func (t *Terminal) ReadReply(p []byte) int {
	if t.p == nil || len(p) == 0 {
		return 0
	}
	return int(C.tgvt_read_reply(t.p, (*C.uint8_t)(unsafe.Pointer(&p[0])), C.size_t(len(p))))
}

func (t *Terminal) ReplyDropped() int {
	if t.p == nil {
		return 0
	}
	return int(C.tgvt_reply_dropped(t.p))
}

// HeapUsage reports bytes libghostty-vt currently holds and the peak since
// start, across all terminals. It excludes malloc and alignment overhead.
func HeapUsage() (current, peak int) {
	var c, p C.size_t
	C.tgvt_heap(&c, &p)
	return int(c), int(p)
}
