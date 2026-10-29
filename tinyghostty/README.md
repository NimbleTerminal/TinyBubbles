# tinyghostty

[Back to the README](../README.md)

tinyghostty embeds a live [Ghostty](https://ghostty.org) VT terminal in a
TinyBubbles UI. Ghostty's freestanding `libghostty-vt` parses the program's
output; tinyghostty copies the screen into a TinyBubbles cell buffer and
encodes keys and paste for the program. It builds with Go on the host and with
TinyGo for WASI, the browser, Cortex-M, ESP32-C3 (RISC-V), and ESP32-S3
(Xtensa). The microcontroller builds pass link checks; none has run on
hardware. For how this differs from the general Go binding, see
[How it differs from go-libghostty](#how-it-differs-from-go-libghostty).

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

Microcontroller archives build one at a time. Each `make <chip>` links a
smoke binary as a check; it does not flash.

```sh
make lib-cortexm GHOSTTY_SRC=... ZIG=...  && make clue     # Cortex-M4F
make lib-esp32c3 GHOSTTY_SRC=... ZIG=...  && make esp32c3  # ESP32-C3, RISC-V
make lib-esp32s3 GHOSTTY_SRC=...          && make esp32s3  # ESP32-S3, Xtensa
```

Upstream Zig cannot target the ESP32-S3. `make lib-esp32s3` first runs
`make zig-xtensa`, which downloads a pinned Xtensa Zig build
([kassane/zig-espressif-bootstrap](https://github.com/kassane/zig-espressif-bootstrap)
`0.17.0-xtensa`) into `.tools/` and checks its sha256. That fork is
maintained outside Zig and Espressif. Set `ZIG_XTENSA=/path/to/zig` to use
your own.

```go
term := tinyghostty.New(40, 16)
if err := term.Err(); err != nil { ... }
defer term.Close()
term.Input = pty                              // keys, paste, and replies go here
term.Renderer().SetTheme(theme)               // match the display program's theme
go io.Copy(tinyghostty.Sender(p.Send), pty)   // program output arrives as OutputMsg
```

## Shell demo in your terminal

```sh
make shell                          # $SHELL on a 240×135 Feather screen
make shell ARGS="-display oled"     # 128×64 OLED, needs a 128×33 terminal
make shell ARGS="-mono -- htop"     # monochrome theme, any command
```

[examples/shell](examples/shell/main.go) runs a command in a PTY and draws the
emulated display in your terminal with half-block pixels, one terminal cell per
two pixels. Keys go through Bubble Tea and the widget's key encoder to the
PTY. Press ctrl+] to quit. The default Feather preset needs a terminal at least
240×69; shrink the font if it is smaller. Unix only. The 6×8 font is ASCII, so
other glyphs render as placeholders.

## Browser demo

```sh
make web    # http://127.0.0.1:8080
```

[web](web) builds a TinyGo-only page with the shared browser harness. Go's
`js/wasm` has no cgo, so Go cannot build it. A browser has no PTY, so a small
Go shell runs inside the WASM binary: `help`, `colors`, `styles`, `box`, `ls`,
`cat`, with Tab completion, history, Ctrl+C, and Ctrl+L. The demo sets the
harness's `Demo.TextInput`, so the focused display accepts free typing and
paste. Buttons below it run sample commands. The page offers the Feather TFT,
a 320×240 TFT, and a monochrome 128×64 OLED. `app.wasm` is about 6 MB.

The same demo is a tab in the main gallery: after `make lib`, run
`make web-serve-tinygo` from the repository root. CI's `web-tinygo` job
builds the Wasm archive from Ghostty v1.4.0 with Zig 0.17.0, so the published
gallery includes it.
`make -C web build` appends the licenses in `lib/licenses` to
`THIRD_PARTY_NOTICES.txt`: Ghostty and uucode, plus Wuffs when a host build
fetched it. The Wasm archive does not compile Wuffs.

## Using the widget

A parent model sizes the terminal with `SetSize`, forwards keys and paste only
while it has focus, and composes `CellView()` with `layout.Draw`. See
[examples/preview](examples/preview/main.go).

## Packages

| Package | Purpose |
| --- | --- |
| `tinyghostty` | Bubble Tea `Model`: `New`, `Write`, `SetSize`, `SendKey`, `Paste`, `CellView`, `Text`, `OutputMsg`, `Sender`, `AppendKey` |
| `tinyghostty/vt` | cgo binding over a C shim: `Terminal` lifetime, `Write`, `Resize`, `Update`/`NextRow`/`MarkClean`, `Cursor`, `Colors`, `DECMode`, `ReadReply`, `SetScrollback`, `HeapUsage` |
| `tinyghostty/render` | Copies a snapshot into a `*tinybubbles.Buffer`, redrawing only dirty rows; `Theme` adds red, green, yellow, and blue styles |

## How it differs from go-libghostty

[go-libghostty](https://pkg.go.dev/go.mitchellh.com/libghostty)
(`go.mitchellh.com/libghostty`) is the general Go binding for libghostty-vt.
Use it for desktop or server Go programs that need Ghostty's full API.
tinyghostty wraps a much smaller part of the same C library so it builds with
TinyGo and fits small displays.

| | go-libghostty | tinyghostty |
| --- | --- | --- |
| Compilers | Go | Go and TinyGo (`wasip1`, browser `wasm`, Cortex-M link) |
| Linking | `#cgo pkg-config`, static by default, `-tags dynamic` for shared | Static archives in `lib/` from `scripts/build-libghostty-vt.sh`; per-target `cgo_*.go` flag files, no pkg-config |
| Callbacks from C | `//export` trampolines with `cgo.Handle` for PTY writes, bell, title, clipboard, notifications, and other effects | None. Terminal replies collect in a 512-byte shim buffer; `ReadReply` drains it |
| Reading cells | Per-cell getters that copy Ghostty's style structs | One shim call per row fills flat 16-byte `vt.Cell` values: rune, RGB colors, palette indices, flags |
| Allocation | Ghostty's default allocator | A `malloc`-backed shim allocator, so the VT shares TinyGo's heap; `HeapUsage` reports it |
| Scrollback | Configurable with `WithMaxScrollbackBytes`, `WithMaxScrollbackLines`, and setters | Off by default to bound memory; `SetScrollback` sets a byte limit |
| API scope | Most of libghostty-vt: formatter, selection, search, snapshots, Kitty graphics, key and mouse encoders, OSC and SGR parsers | Terminal lifetime, VT input, resize, render state, cursor, colors, and mode queries |
| Keys | Ghostty's key encoder, including the Kitty keyboard protocol | A Go xterm encoder (`AppendKey`) that follows DECCKM; no Kitty protocol |
| On top | A Go API over the VT | A renderer into TinyBubbles cell buffers and a Bubble Tea model |

TinyGo 0.43 rejects go-libghostty's `#cgo pkg-config` line ("invalid #cgo
line: pkg-config"), checked against commit `8812ad0e0f79`. TinyGo also does
not expand `${SRCDIR}` or accept build constraints on `#cgo` lines, which is why
tinyghostty keeps its flags in tagged files. Its shim flattens Ghostty's union
colors and sized structs into plain integers, so TinyGo's cgo never handles
them.

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
| TinyGo `wasip1` | `lib/wasm` (`wasm32-freestanding`) | Tests and smoke pass under wasmtime; smoke binary 1.4 MB |
| TinyGo `wasm` (browser) | `lib/wasm` | Browser demo runs in Chrome |
| TinyGo `clue` | `lib/cortexm` (`make lib-cortexm`, Cortex-M4F) | `vt` smoke links: 779 KB flash, 6.4 KB static RAM. Not run on hardware |
| TinyGo `esp32c3-generic` | `lib/esp32c3` (`make lib-esp32c3`, RV32IMC) | `vt` smoke links: 905 KB flash, 11.6 KB static RAM. Not run on hardware |
| TinyGo `esp32s3-generic`, Feather `target.json` | `lib/esp32s3` (`make lib-esp32s3`, Xtensa fork) | `vt` smoke links: 595 KB flash, 19.6 KB static RAM. Not run on hardware |

Other Go hosts link `lib/<GOOS>_<GOARCH>` after `scripts/build-libghostty-vt.sh host`.
TinyGo's native darwin target lacks libc symbols the macOS archive needs, so
TinyGo builds use the freestanding archives only.

A 40×16 terminal holds about 64 KiB of Ghostty heap, on top of TinyBubbles'
model and renderer grids. New terminals keep no scrollback, so output does not
grow it; `SetScrollback` raises the limit. `vt.HeapUsage` reports current and
peak bytes.

Microcontroller notes:

- TinyGo picks the archive by build tag: `cortexm` links `lib/cortexm`,
  `esp32c3` links `lib/esp32c3`, and `esp32s3` links `lib/esp32s3`. Rebuild
  `lib/cortexm` for other Cortex-M CPUs, for example
  `scripts/build-libghostty-vt.sh thumb-freestanding-eabi:cortexm:cortex_m0plus`
  for the RP2040. Other chips, such as the original ESP32, have no archive yet.
- TinyGo's stack-size analysis cannot parse Zig's symbols, so link with
  `-scheduler=none`. Bubble Tea firmware also needs the vendored portability
  patches from `bubbletea/`.
- The ESP32-S3 archive builds through [scripts/xtensa](scripts/xtensa/build.zig),
  which uses Ghostty as a Zig dependency and changes two settings to work
  around bugs in the fork. With stripped symbols, the fork's literal pools name
  functions the object never defines. Without per-function sections, `L32R`
  literals fall out of range. ReleaseFast and ReleaseSafe crash the fork's
  register allocator, so the archive stays ReleaseSmall.
- On ESP32 targets TinyGo links with `--wrap=malloc` for its radio driver, so
  the shim calls the runtime's allocator directly (`TGVT_REAL_MALLOC`).
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
