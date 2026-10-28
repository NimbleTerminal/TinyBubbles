# tinyghostty

[Back to the README](../README.md)

tinyghostty embeds a live [Ghostty](https://ghostty.org) VT terminal in a
TinyBubbles UI. Ghostty's freestanding `libghostty-vt` parses the program's
output; tinyghostty copies the screen into a TinyBubbles cell buffer and
encodes keys and paste for the program. It builds with Go on the host and with
TinyGo for WASI. A Cortex-M4 link check passes; it has not run on hardware.

This is a nested module because it links a C library that you build first.
The root `make check` and `make ci` do not build it.

## Quickstart

You need Zig (the version pinned by Ghostty's `build.zig.zon`; 0.17.0 for
Ghostty 1.4.0) and a Ghostty checkout.

```sh
cd tinyghostty
make lib GHOSTTY_SRC=/path/to/ghostty ZIG=/path/to/zig  # host + wasm archives
make check        # Go tests and vet
make test-tinygo  # TinyGo wasip1 tests and smoke run (needs wasmtime)
make preview      # ../build/tinyghostty/tinyghostty.png
```

```go
term := tinyghostty.New(40, 16)
if err := term.Err(); err != nil { ... }
defer term.Close()
term.Input = pty                              // keys, paste, and replies go here
term.Renderer().SetTheme(theme)               // match the display program's theme
go io.Copy(tinyghostty.Sender(p.Send), pty)   // program output arrives as OutputMsg
```

A parent model sizes the terminal with `SetSize`, forwards keys and paste only
while it has focus, and composes `CellView()` with `layout.Draw`. See
[examples/preview](examples/preview/main.go).

## Packages

| Package | Purpose |
| --- | --- |
| `tinyghostty` | Bubble Tea `Model`: `New`, `Write`, `SetSize`, `SendKey`, `Paste`, `CellView`, `Text`, `OutputMsg`, `Sender`, `AppendKey` |
| `tinyghostty/vt` | cgo binding over a C shim: `Terminal` lifetime, `Write`, `Resize`, `Update`/`NextRow`/`MarkClean`, `Cursor`, `Colors`, `DECMode`, `ReadReply`, `SetScrollback`, `HeapUsage` |
| `tinyghostty/render` | Copies a snapshot into a `*tinybubbles.Buffer`, redrawing only dirty rows; `Theme` adds red, green, yellow, and blue styles |

## Ghostty API used

The shim in `vt/shim.c` wraps this subset and exposes only pointers, integers,
and fixed-layout structs. It flattens union colors and sized structs, copies a
whole row per call, and collects replies in a 512-byte buffer instead of
calling back into Go.

| Area | Ghostty functions |
| --- | --- |
| Terminal | `ghostty_terminal_new`, `_free`, `_reset`, `_resize`, `_vt_write`, `_get` (`DATA_MODE`), `_set` (`OPT_USERDATA`, `OPT_WRITE_PTY`, `OPT_SCROLLBACK_MAX_BYTES`) |
| Render state | `ghostty_render_state_new`, `_free`, `_update`, `_get` (`DIRTY`, `ROW_ITERATOR`, `CURSOR`, `COLORS`), `_clean` |
| Rows and cells | `ghostty_render_state_row_iterator_new`/`_next`/`_next_dirty`/`_free`, `_row_get` (`CELLS`, `VIEWPORT_Y`), `_row_cells_new`/`_next`/`_get`/`_free`, `ghostty_cell_get` (`WIDE`, `CODEPOINT`) |

Ghostty allocates through a shim allocator backed by `malloc`, so the VT shares
TinyGo's heap. On freestanding Wasm, Ghostty's own allocator would otherwise
grow linear memory behind TinyGo's allocator.

## Rendering

TinyBubbles cells hold one rune and one of eight theme styles.

- ANSI colors 0–15 on the default background use the renderer's `ANSI` table.
  Other colors pick the nearest theme style by foreground and background.
- Default colors map to `Normal`. Reverse video usually lands on `Selected`.
  Faint text blends toward the background and usually lands on `Muted`.
  Bold text in default colors uses `Bold` (Accent by default).
- The visible cursor is drawn as `CursorStyle` (Selected) at its cell.
- Wide characters keep their rune in the first cell. Grapheme clusters keep
  their base codepoint.
- Underline, italic, strikethrough, overline, and blink have no cell
  representation, so they are not drawn.

Frames after the first redraw only Ghostty's dirty rows, plus the cell under a
moved cursor. A steady-state 40×16 frame allocates nothing in `Draw`.

## Targets and memory

| Target | Archive | Status |
| --- | --- | --- |
| Go, darwin/arm64 | `lib/darwin_arm64` (native, no SIMD) | Tests pass |
| TinyGo `wasip1` | `lib/wasip1_wasm` (`wasm32-freestanding`) | Tests and smoke pass under wasmtime; smoke binary 1.4 MB |
| TinyGo `clue` | `lib/baremetal` (`make lib-clue`, Cortex-M4F) | `vt` smoke links: 779 KB flash, 6.7 KB static RAM. Not run on hardware |

Other Go hosts link `lib/<GOOS>_<GOARCH>` after `scripts/build-libghostty-vt.sh host`.
TinyGo's native darwin target lacks libc symbols the macOS archive needs, so
TinyGo builds use the freestanding archives only.

A 40×16 terminal holds about 64 KiB of Ghostty heap, on top of TinyBubbles'
model and renderer grids. New terminals keep no scrollback, so output does not
grow it; `SetScrollback` raises the limit. `vt.HeapUsage` reports current and
peak bytes.

Microcontroller notes:

- Build the archive for your CPU, for example
  `scripts/build-libghostty-vt.sh thumb-freestanding-eabi:baremetal:cortex_m0plus`
  for the RP2040. Every `baremetal` target links `lib/baremetal`.
- TinyGo's stack-size analysis cannot parse Zig's symbols, so link with
  `-scheduler=none`. Bubble Tea firmware also needs the vendored portability
  patches from `bubbletea/`.
- The ESP32-S3 Feather is not supported. Zig 0.17 has no `esp32s3` CPU, and
  a generic `xtensa-freestanding` build fails in Zig's compiler-rt.
- Ghostty's heap figure assumes a few hundred KiB of RAM. It will not fit
  boards with less.

## Not included

Kitty graphics, scrollback UI, mouse reporting, selection, clipboard, the Kitty
keyboard protocol, and dynamic linking. Keys use xterm encodings and respect
application cursor mode (DECCKM); paste respects bracketed paste mode.

## License

MIT, copyright (c) 2026 Neomantra Corp. Ghostty and libghostty-vt are MIT
licensed by Mitchell Hashimoto and Ghostty contributors; the built archives and
headers in `lib/` are not committed.
