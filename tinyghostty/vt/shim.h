// Flat C ABI over libghostty-vt for TinyGo cgo.
//
// TinyGo's cgo handles pointers, integers, and plain structs well, but not
// unions, sized-struct initializers, inline functions, or C-to-Go callbacks.
// This shim hides those behind fixed-layout structs and one call per row.
#ifndef TINYGHOSTTY_SHIM_H
#define TINYGHOSTTY_SHIM_H

#include <stddef.h>
#include <stdint.h>

typedef struct tgvt tgvt;

// tgvt_cell matches vt.Cell: 16 bytes, no padding.
typedef struct {
  uint32_t rune;  // base codepoint; 0 is empty
  uint32_t fg;    // 0xRRGGBB when TGVT_FG is set
  uint32_t bg;    // 0xRRGGBB when TGVT_BG is set
  uint16_t flags; // TGVT_* bits
  uint8_t fg_index; // ANSI palette index when TGVT_FG_INDEX is set
  uint8_t bg_index; // ANSI palette index when TGVT_BG_INDEX is set
} tgvt_cell;

enum {
  TGVT_FG = 1 << 0,
  TGVT_BG = 1 << 1,
  TGVT_BOLD = 1 << 2,
  TGVT_ITALIC = 1 << 3,
  TGVT_FAINT = 1 << 4,
  TGVT_BLINK = 1 << 5,
  TGVT_INVERSE = 1 << 6,
  TGVT_INVISIBLE = 1 << 7,
  TGVT_STRIKE = 1 << 8,
  TGVT_OVERLINE = 1 << 9,
  TGVT_UNDERLINE = 1 << 10,
  TGVT_WIDE = 1 << 11,
  TGVT_SPACER = 1 << 12,
  TGVT_GRAPHEME = 1 << 13,
  TGVT_FG_INDEX = 1 << 14,
  TGVT_BG_INDEX = 1 << 15,
};

// tgvt_cursor matches vt.Cursor: 8 bytes.
typedef struct {
  uint16_t x, y;
  uint8_t in_viewport, visible, blinking, style;
} tgvt_cursor;

// tgvt_colors matches vt.Colors: 16 bytes.
typedef struct {
  uint32_t fg, bg, cursor;
  uint8_t cursor_set, pad[3];
} tgvt_colors;

// tgvt_new returns NULL and stores a GhosttyResult in *result on failure.
tgvt *tgvt_new(uint16_t cols, uint16_t rows, int *result);
void tgvt_free(tgvt *t);
void tgvt_reset(tgvt *t);
void tgvt_write(tgvt *t, const uint8_t *data, size_t len);
int tgvt_resize(tgvt *t, uint16_t cols, uint16_t rows);
// New terminals keep no scrollback. Ghostty prunes at page granularity.
int tgvt_set_scrollback(tgvt *t, size_t bytes);

// tgvt_update snapshots the terminal and rewinds the row iterator. It returns
// 0 (clean), 1 (partial), 2 (full), or a negative GhosttyResult.
int tgvt_update(tgvt *t);
// tgvt_next_row advances the row iterator, copies at most max cells, and
// returns the count, or -1 when no rows remain.
int tgvt_next_row(tgvt *t, int dirty_only, tgvt_cell *out, uint16_t max,
                  uint16_t *y);
int tgvt_clean(tgvt *t);
int tgvt_cursor_get(tgvt *t, tgvt_cursor *out);
int tgvt_colors_get(tgvt *t, tgvt_colors *out);

// tgvt_mode returns 0 or 1, or a negative GhosttyResult.
int tgvt_mode(tgvt *t, uint16_t mode, int ansi);

// Bytes the terminal writes back to the application (device reports and
// query replies) collect in a fixed buffer. tgvt_read_reply drains it.
size_t tgvt_read_reply(tgvt *t, uint8_t *buf, size_t cap);
size_t tgvt_reply_dropped(tgvt *t);

// tgvt_heap reports bytes Ghostty holds through the shim allocator, excluding
// malloc and alignment overhead, across all terminals.
void tgvt_heap(size_t *cur, size_t *peak);

#endif
