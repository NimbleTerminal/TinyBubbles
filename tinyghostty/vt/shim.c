#include <stdbool.h>
#include <stdlib.h>
#include <string.h>

#include <ghostty/vt.h>

#include "shim.h"

#define TGVT_REPLY_CAP 512

struct tgvt {
  GhosttyTerminal term;
  GhosttyRenderState state;
  GhosttyRenderStateRowIterator rows;
  GhosttyRenderStateRowCells cells;
  size_t reply_len, reply_dropped;
  uint8_t reply[TGVT_REPLY_CAP];
};

// Ghostty allocates through this vtable so the VT shares the host heap.
// Freestanding Wasm builds would otherwise grow linear memory behind TinyGo's
// allocator. Alignment is log2; the original pointer sits before each block.
static size_t tgvt_heap_cur, tgvt_heap_peak;

static void *tgvt_alloc(void *ctx, size_t len, uint8_t alignment,
                        uintptr_t ret) {
  (void)ctx;
  (void)ret;
  size_t align = (size_t)1 << alignment;
  if (align < sizeof(void *)) align = sizeof(void *);
  uint8_t *raw = malloc(len + align + sizeof(void *));
  if (raw == NULL) return NULL;
  tgvt_heap_cur += len;
  if (tgvt_heap_cur > tgvt_heap_peak) tgvt_heap_peak = tgvt_heap_cur;
  uintptr_t p = ((uintptr_t)(raw + sizeof(void *)) + align - 1) & ~(uintptr_t)(align - 1);
  ((void **)p)[-1] = raw;
  return (void *)p;
}

static bool tgvt_resize_mem(void *ctx, void *mem, size_t len, uint8_t alignment,
                            size_t new_len, uintptr_t ret) {
  (void)ctx;
  (void)mem;
  (void)alignment;
  (void)ret;
  if (new_len > len) return false;
  tgvt_heap_cur -= len - new_len;
  return true;
}

static void *tgvt_remap(void *ctx, void *mem, size_t len, uint8_t alignment,
                        size_t new_len, uintptr_t ret) {
  (void)ctx;
  (void)alignment;
  (void)ret;
  if (new_len > len) return NULL;
  tgvt_heap_cur -= len - new_len;
  return mem;
}

static void tgvt_free_mem(void *ctx, void *mem, size_t len, uint8_t alignment,
                          uintptr_t ret) {
  (void)ctx;
  (void)alignment;
  (void)ret;
  if (mem == NULL) return;
  tgvt_heap_cur -= len;
  free(((void **)mem)[-1]);
}

static const GhosttyAllocatorVtable tgvt_vtable = {
    tgvt_alloc, tgvt_resize_mem, tgvt_remap, tgvt_free_mem};
static const GhosttyAllocator tgvt_allocator = {NULL, &tgvt_vtable};

static void tgvt_on_reply(GhosttyTerminal term, void *userdata,
                          const uint8_t *data, size_t len) {
  (void)term;
  tgvt *t = userdata;
  size_t room = TGVT_REPLY_CAP - t->reply_len;
  if (len > room) {
    t->reply_dropped += len - room;
    len = room;
  }
  memcpy(t->reply + t->reply_len, data, len);
  t->reply_len += len;
}

tgvt *tgvt_new(uint16_t cols, uint16_t rows, int *result) {
  tgvt *t = calloc(1, sizeof(*t));
  int r = GHOSTTY_OUT_OF_MEMORY;
  if (t == NULL) goto fail;
  if ((r = ghostty_terminal_new(&tgvt_allocator, &t->term, cols, rows))) goto fail;
  if ((r = ghostty_render_state_new(&tgvt_allocator, &t->state))) goto fail;
  if ((r = ghostty_render_state_row_iterator_new(&tgvt_allocator, &t->rows))) goto fail;
  if ((r = ghostty_render_state_row_cells_new(&tgvt_allocator, &t->cells))) goto fail;
  if ((r = ghostty_terminal_set(t->term, GHOSTTY_TERMINAL_OPT_USERDATA, t))) goto fail;
  GhosttyTerminalWritePtyFn fn = tgvt_on_reply;
  if ((r = ghostty_terminal_set(t->term, GHOSTTY_TERMINAL_OPT_WRITE_PTY, (const void *)fn))) goto fail;
  // Small targets have no scrollback UI; keep only the visible screen.
  if ((r = tgvt_set_scrollback(t, 0))) goto fail;
  *result = GHOSTTY_SUCCESS;
  return t;
fail:
  tgvt_free(t);
  *result = r;
  return NULL;
}

void tgvt_free(tgvt *t) {
  if (t == NULL) return;
  if (t->cells) ghostty_render_state_row_cells_free(t->cells);
  if (t->rows) ghostty_render_state_row_iterator_free(t->rows);
  if (t->state) ghostty_render_state_free(t->state);
  if (t->term) ghostty_terminal_free(t->term);
  free(t);
}

void tgvt_reset(tgvt *t) { ghostty_terminal_reset(t->term); }

void tgvt_write(tgvt *t, const uint8_t *data, size_t len) {
  ghostty_terminal_vt_write(t->term, data, len);
}

int tgvt_set_scrollback(tgvt *t, size_t bytes) {
  return ghostty_terminal_set(t->term, GHOSTTY_TERMINAL_OPT_SCROLLBACK_MAX_BYTES, &bytes);
}

int tgvt_resize(tgvt *t, uint16_t cols, uint16_t rows) {
  return ghostty_terminal_resize(t->term, cols, rows, 0, 0);
}

int tgvt_update(tgvt *t) {
  int r = ghostty_render_state_update(t->state, t->term);
  if (r) return r;
  GhosttyRenderStateDirty dirty = GHOSTTY_RENDER_STATE_DIRTY_FULL;
  if ((r = ghostty_render_state_get(t->state, GHOSTTY_RENDER_STATE_DATA_DIRTY, &dirty))) return r;
  if ((r = ghostty_render_state_get(t->state, GHOSTTY_RENDER_STATE_DATA_ROW_ITERATOR, &t->rows))) return r;
  return (int)dirty;
}

static uint32_t tgvt_rgb(GhosttyColorRgb c) {
  return (uint32_t)c.r << 16 | (uint32_t)c.g << 8 | c.b;
}

int tgvt_next_row(tgvt *t, int dirty_only, tgvt_cell *out, uint16_t max,
                  uint16_t *y) {
  uint16_t row = 0;
  if (dirty_only) {
    if (!ghostty_render_state_row_iterator_next_dirty(t->rows, &row)) return -1;
  } else {
    if (!ghostty_render_state_row_iterator_next(t->rows)) return -1;
    ghostty_render_state_row_get(t->rows, GHOSTTY_RENDER_STATE_ROW_DATA_VIEWPORT_Y, &row);
  }
  *y = row;
  if (ghostty_render_state_row_get(t->rows, GHOSTTY_RENDER_STATE_ROW_DATA_CELLS, &t->cells)) return 0;

  int n = 0;
  while (n < max && ghostty_render_state_row_cells_next(t->cells)) {
    tgvt_cell *c = &out[n++];
    memset(c, 0, sizeof(*c));

    GhosttyCell raw = 0;
    GhosttyCellWide wide = GHOSTTY_CELL_WIDE_NARROW;
    if (ghostty_render_state_row_cells_get(t->cells, GHOSTTY_RENDER_STATE_ROW_CELLS_DATA_RAW, &raw) == GHOSTTY_SUCCESS)
      ghostty_cell_get(raw, GHOSTTY_CELL_DATA_WIDE, &wide);
    if (wide == GHOSTTY_CELL_WIDE_WIDE) c->flags |= TGVT_WIDE;
    if (wide == GHOSTTY_CELL_WIDE_SPACER_TAIL || wide == GHOSTTY_CELL_WIDE_SPACER_HEAD) c->flags |= TGVT_SPACER;

    uint32_t glen = 0;
    ghostty_render_state_row_cells_get(t->cells, GHOSTTY_RENDER_STATE_ROW_CELLS_DATA_GRAPHEMES_LEN, &glen);
    if (glen == 1) {
      ghostty_render_state_row_cells_get(t->cells, GHOSTTY_RENDER_STATE_ROW_CELLS_DATA_GRAPHEMES_BUF, &c->rune);
    } else if (glen > 1) {
      // Clusters keep their base codepoint; TinyBubbles cells hold one rune.
      uint32_t cps[8];
      if (glen <= 8) {
        ghostty_render_state_row_cells_get(t->cells, GHOSTTY_RENDER_STATE_ROW_CELLS_DATA_GRAPHEMES_BUF, cps);
        c->rune = cps[0];
      } else {
        ghostty_cell_get(raw, GHOSTTY_CELL_DATA_CODEPOINT, &c->rune);
      }
      c->flags |= TGVT_GRAPHEME;
    }

    GhosttyColorRgb rgb;
    if (ghostty_render_state_row_cells_get(t->cells, GHOSTTY_RENDER_STATE_ROW_CELLS_DATA_FG_COLOR, &rgb) == GHOSTTY_SUCCESS) {
      c->fg = tgvt_rgb(rgb);
      c->flags |= TGVT_FG;
    }
    if (ghostty_render_state_row_cells_get(t->cells, GHOSTTY_RENDER_STATE_ROW_CELLS_DATA_BG_COLOR, &rgb) == GHOSTTY_SUCCESS) {
      c->bg = tgvt_rgb(rgb);
      c->flags |= TGVT_BG;
    }

    bool styled = false;
    ghostty_render_state_row_cells_get(t->cells, GHOSTTY_RENDER_STATE_ROW_CELLS_DATA_HAS_STYLING, &styled);
    if (!styled) continue;
    GhosttyStyle st = GHOSTTY_INIT_SIZED(GhosttyStyle);
    if (ghostty_render_state_row_cells_get(t->cells, GHOSTTY_RENDER_STATE_ROW_CELLS_DATA_STYLE, &st)) continue;
    if (st.bold) c->flags |= TGVT_BOLD;
    if (st.italic) c->flags |= TGVT_ITALIC;
    if (st.faint) c->flags |= TGVT_FAINT;
    if (st.blink) c->flags |= TGVT_BLINK;
    if (st.inverse) c->flags |= TGVT_INVERSE;
    if (st.invisible) c->flags |= TGVT_INVISIBLE;
    if (st.strikethrough) c->flags |= TGVT_STRIKE;
    if (st.overline) c->flags |= TGVT_OVERLINE;
    if (st.underline) c->flags |= TGVT_UNDERLINE;
    if (st.fg_color.tag == GHOSTTY_STYLE_COLOR_PALETTE) {
      c->fg_index = st.fg_color.value.palette;
      c->flags |= TGVT_FG_INDEX;
    }
    if (st.bg_color.tag == GHOSTTY_STYLE_COLOR_PALETTE) {
      c->bg_index = st.bg_color.value.palette;
      c->flags |= TGVT_BG_INDEX;
    }
  }
  return n;
}

int tgvt_clean(tgvt *t) { return ghostty_render_state_clean(t->state); }

int tgvt_cursor_get(tgvt *t, tgvt_cursor *out) {
  GhosttyRenderStateCursor c = GHOSTTY_INIT_SIZED(GhosttyRenderStateCursor);
  int r = ghostty_render_state_get(t->state, GHOSTTY_RENDER_STATE_DATA_CURSOR, &c);
  memset(out, 0, sizeof(*out));
  if (r) return r;
  out->in_viewport = c.viewport_has_value;
  if (c.viewport_has_value) {
    out->x = c.viewport_x;
    out->y = c.viewport_y;
  }
  out->visible = c.visible;
  out->blinking = c.blinking;
  out->style = (uint8_t)c.visual_style;
  return 0;
}

int tgvt_colors_get(tgvt *t, tgvt_colors *out) {
  GhosttyRenderStateColors c = GHOSTTY_INIT_SIZED(GhosttyRenderStateColors);
  int r = ghostty_render_state_get(t->state, GHOSTTY_RENDER_STATE_DATA_COLORS, &c);
  memset(out, 0, sizeof(*out));
  if (r) return r;
  out->fg = tgvt_rgb(c.foreground);
  out->bg = tgvt_rgb(c.background);
  out->cursor = tgvt_rgb(c.cursor);
  out->cursor_set = c.cursor_has_value;
  return 0;
}

int tgvt_mode(tgvt *t, uint16_t mode, int ansi) {
  GhosttyTerminalModeConfig m = {ghostty_mode_new(mode, ansi != 0), false};
  int r = ghostty_terminal_get(t->term, GHOSTTY_TERMINAL_DATA_MODE, &m);
  return r ? r : (int)m.value;
}

size_t tgvt_read_reply(tgvt *t, uint8_t *buf, size_t cap) {
  size_t n = t->reply_len < cap ? t->reply_len : cap;
  memcpy(buf, t->reply, n);
  memmove(t->reply, t->reply + n, t->reply_len - n);
  t->reply_len -= n;
  return n;
}

size_t tgvt_reply_dropped(tgvt *t) { return t->reply_dropped; }

void tgvt_heap(size_t *cur, size_t *peak) {
  *cur = tgvt_heap_cur;
  *peak = tgvt_heap_peak;
}
