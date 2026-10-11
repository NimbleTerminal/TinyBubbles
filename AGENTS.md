# Working on TinyBubbles

TinyBubbles provides display rendering and device-focused UI for TinyGo, with
**Bubble Tea v2.1 as the primary runtime**. TinyTea is an optional cooperative
replacement with a documented subset of Bubble Tea's model API. Read
`RUNTIMES.md` before introducing dependencies or changing runtime semantics.

## Where this fits

- `bubbletea/` is a separate module importing real Bubble Tea v2.1.0. It is the
  primary backend, using TinyGo 0.43 and Go 1.26 plus reproducible bare-metal
  portability patches. Run `make prepare` there after changes to local dependencies.
- `tinytea/` exposes the supported Bubble Tea-shaped model API without importing
  Bubble Tea. Keep it small; unsupported timers and terminal APIs must stay explicit.
- The root package owns shared cells, fonts, themes, buttons, and rendering. Its
  original cooperative Program/Model API remains for compatibility with old demos.
- Widgets should prioritize small displays, three-button focus/navigation,
  monochrome/color adaptation, and mixed-size fonts. Reuse cell layouts and widget
  state; keep runtime-specific command scheduling in the owning model.
- tinypod supplies the Config plus devices precedent. tinyterm is a terminal
  renderer, and tinygui follows an immediate-mode paradigm. ntcharts and ntcharts3d
  are sibling documentation references, not dependencies for the rendering core.

Read `README.md` for the overview, `API.md` for framework semantics and memory
budgets, and `DEVICES.md` for wiring and adding targets. The Feather guide lives
at `devices/feather-reverse-tft/README.md`.

## Repository map

| Location | Responsibility |
| --- | --- |
| `bubbletea/` | Primary Bubble Tea backend, portability preparation, Feather demo |
| `web/` | Isolated WASM module, canvas Displayer, Booba help-to-HTML actions, static site |
| `tinytea/` | Optional cooperative subset with Bubble Tea-shaped models |
| `program.go` | Model, Msg, Cmd, Config, lifecycle, bounded command queue |
| `cell.go` | Reusable cell buffer, text placement, clipping |
| `renderer.go` | Changed-cell rendering, optional rectangle fill, flush/retry |
| `font.go`, `theme.go` | TinyFont adapter, pluggable fonts, compact palette |
| `input.go` | Standard key events and hardware-independent button debounce |
| `demo/model.go` | Counter shared by Pico and CLUE |
| `demo/hello/` | Animated greeting, bubble sprites, enlarged pop counter |
| `devices/` | Hardware-specific mains and any local target definitions |
| `examples/` | Host PNG/GIF previews using real models and renderers |
| `internal/compilecheck/` | Hardware-free ESP32-S3 firmware compile check |
| `docs/assets/hello.gif` | README demo asset generated from the hello model |
| `Makefile`, `.github/workflows/ci.yml` | Local checks and firmware CI |

The Go module/import path is `nimbleterminal.dev/tinybubbles`. The repository remains
`https://github.com/NimbleTerminal/TinyBubbles`; use NimbleTerminal for repository
badges, issue links, and remotes.

## Framework conventions

The cooperative lifecycle rules below describe TinyTea and the original root
runtime. Bubble Tea retains upstream scheduling and message semantics; consult
`RUNTIMES.md` instead of applying these rules to its program.

- Keep `machine`, bus initialization, display power/rotation, and physical pin
  assignments in `devices/<name>/main.go`. Use TinyGo/board build tags so ordinary
  `go test ./...` never imports `machine`.
- Accept the existing `drivers.Displayer` directly: `Size`, `SetPixel`, `Display`.
  Detect optional `FillRectangle(..., color.RGBA) error` with a type assertion;
  retain the SetPixel fallback. Do not require a richer driver interface.
- Keep the core free of terminal/OS APIs and heavy dependencies. TinyGo v0.43.0
  is the minimum supported compiler. The module language version is Go 1.23.
- The original root `Model` implements `Init() Cmd`, `Update(Msg) (Model, Cmd)`,
  and `View() *Buffer`. TinyTea instead uses `View() tinytea.View` with optional
  `CellView() *Buffer`, matching the supported Bubble Tea-shaped API.
  Any Go value can be a custom message. Reuse buffers rather than allocating one
  on every frame; keep animation state changes in `Update`, not `View`.
- `Start` queues Init's command, sends the initial `WindowSizeMsg`, and renders.
  `Step` polls once, executes at most one queued command, then renders. `Run`
  repeats this with a configurable sleep. `Send` updates/renders synchronously
  but leaves returned commands for a later Step.
- Commands are cooperative, not concurrent. Never wait for input, sleep, or do
  long network work inside a command. Device loops own timers; the hello demo
  feeds `TickMsg` from its poll function. The queue holds 16 commands and reports
  overflow with `ErrCommandQueueFull`.
- All program operations belong to one main loop. Interrupts and other tasks
  should hand events through a device-owned safe mechanism, not call `Send`.
- `KeyQuit` reaches the model; only `QuitMsg` stops the program. This permits
  application-specific confirmation. Preserve that distinction.
- The renderer owns one previous-cell grid, not a framebuffer. It clears stale
  cells, clips glyphs, flushes only changed frames, and invalidates on driver
  errors so a later render can retry. Preserve both buffered OLED and direct-write
  TFT behavior.
- A custom font receives a clipping display that it must not retain. The hello
  font encodes larger text and sprites as cell tiles; do not bypass the renderer
  to draw that demo directly onto hardware.
- `Displayer` cannot report color depth. Set Monochrome in Config and adapt layout
  from WindowSizeMsg. Configure physical orientation before program construction.
- Budget memory explicitly. With 6x8 cells, model plus renderer grids cost about
  16 bytes per cell (10 KiB for 40x16), in addition to drivers, fonts, heap, and
  stacks. Compiler static RAM figures exclude runtime allocations.

## Build and verification

Run commands from the repository root:

```sh
make test           # race-enabled Go tests and vet
make check          # formatting, modules, docs, tests, vet, and host builds
make ci             # all CI checks, previews, and TinyTea/Bubble Tea firmware
make firmware       # Pico, Pico 2, CLUE, ESP32-S3 check, Reverse TFT Feather
make hello          # primary build/bubbletea-feather.bin
make hello-tinytea  # original cooperative build/feather-reverse-tft.bin
make hello-preview  # build/hello.png and build/hello.gif
make snapshot       # snapshot.png for the shared counter
make web-serve      # build and serve WASM hello with generated HTML controls
make -C web test    # Go bridge tests, module checks, vet, Node key encoder tests
```

Pushes, pull requests, and manual dispatch run `.github/workflows/ci.yml`.
CI runs `make check`, generates previews, and compiles the firmware matrix with
TinyGo v0.43.0. Only explicitly listed previews and firmware files are uploaded
as artifacts; never broaden uploads to the entire `build/` directory.
The documentation check requires Python 3 and checks tracked and unignored new Markdown files.

Web CI builds standard Go WASM using Go 1.26.8 and Node.js 22, plus a separate
TinyGo v0.43 WASM build with a 128 KiB stack. Its only artifact directory is
`build/web/`. Keep Booba and its Bubble Tea WASM fork
replacement isolated in `web/go.mod`; hardware uses upstream v2.1. Browser
controls derive from Bubbles key bindings used by the model's Update method,
not a separate JavaScript action list. Read `web/README.md` for the
WithoutRenderer/raw-output contract. Test metadata changes and browser lifecycle
alongside the display behavior. Browser work does not authorize flashing hardware.

`TINYGO=/path/to/tinygo make firmware` selects a compiler binary. Use a Go version
supported by that compiler. Run `gofmt` on changed Go files. Add focused behavioral
tests for framework changes, particularly clipping, failed flushes, queue order,
debounce, and buffer reuse. For a simple visual adjustment, generate and inspect
the preview; do not add tests that merely repeat layout constants. Documentation
changes need link/content checks, not firmware rebuilds or flashes.

For core changes, run the host checks and firmware matrix. For board-specific
changes, build that target and run relevant host/model tests. Cross-compilation
does not validate wiring or physical rendering. State separately what compiled,
what ran on the host, and what a person confirmed on hardware.

After changing the shared `demo/hello` visual model, regenerate the README animation when applicable:

```sh
make hello-preview
cp build/hello.gif docs/assets/hello.gif
```

## Reverse TFT Feather workflow

The Adafruit ESP32-S3 Reverse TFT Feather (5691) is the hardware-tested example.
The user confirmed the hello demo working; the larger counter was subsequently
flashed successfully. Pico/Pico 2/CLUE currently have compile validation only.

Use `devices/feather-reverse-tft/target.json`, not another ESP32 board's target.
It inherits `esp32s3-generic` and adds the `feather_reverse_tft` build tag.
Board details are recorded in its main and README, including power enable,
landscape offsets, BGR order, and D0's active-low versus D1/D2's active-high inputs.
D0 pops, D1 changes palette, and D2 pauses. Preserve these controls unless asked
to change them.

Flash only as part of authorized hardware work; a documentation or unrelated
code task is not a request to replace connected firmware. Identify the connected
device and current serial port first, especially after switching boards.
The port can change when entering ROM download mode. Never assume a remembered
`/dev/cu.usbmodem...` name still belongs to the intended board.

```sh
tinygo flash -target=./devices/feather-reverse-tft/target.json \
  -port=<current-serial-port> ./devices/feather-reverse-tft
```

TinyGo flashing replaces existing firmware/bootloader contents. Preserve existing
CircuitPython application files before the first replacement. If automatic reset
cannot synchronize, ask the user to hold D0/BOOT, tap RESET, then release D0; rescan
the port before retrying. Report flash verification separately from seeing the UI
work. Do not claim to have observed a physical screen from compiler/flasher output.

`build/` is ignored and may contain private device backups, including credentials
in CircuitPython settings. Never commit, publish, or copy those backups into demo
assets, and do not print their contents. The README GIF should come only from
`examples/hello-preview`, not a copied device filesystem.

## Documentation and licensing

This project is MIT licensed, copyright (c) 2026 Neomantra Corp; see `LICENSE`.
Preserve third-party licenses and check font-face licenses when adding fonts or
assets. Do not assume the project's MIT license relicenses a dependency.

Follow the sibling projects' README style: title and badges, a visible demo,
quickstart, component/example tables, collaboration links, acknowledgements,
license, and the NimbleTerminal footer. Keep detailed API and device material in
their guides. Do not add release/version badges for nonexistent releases or
claim compatibility with untested boards.
