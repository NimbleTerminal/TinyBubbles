# Working on TinyBubbles

TinyGo compiles Go for microcontrollers. Bubble Tea is a Go framework for
interactive terminal apps. TinyBubbles brings Bubble Tea to small displays and
runs the same UI in a browser for development.

**Use Bubble Tea v2.1 by default.** TinyTea is an optional smaller runtime with
limited functionality. Read [RUNTIMES.md](RUNTIMES.md) before changing runtime
behavior or dependencies.

Go imports use `nimbleterminal.dev/tinybubbles`. Repository links and remotes use
`https://github.com/NimbleTerminal/TinyBubbles`.

## Code map

| Location | Purpose |
| --- | --- |
| `cell.go`, `renderer.go` | Shared cell buffers and Displayer rendering |
| `font.go`, `theme.go`, `input.go` | Fonts, palettes, key events, button debounce |
| `bubbletea/` | Primary runtime adapter, portability patches, Feather main |
| `bubbletea/hello/` | Bubble Tea hello model and shared Bubbles key bindings |
| `tinytea/` | Optional cooperative subset of the Bubble Tea model API |
| `program.go` | Original cooperative runtime used by TinyTea and older demos |
| `demo/hello/` | Shared greeting, custom font tiles, enlarged counter |
| `demo/model.go` | Counter used by Pico and CLUE |
| `devices/` | Board setup, pins, drivers, and local targets |
| `web/` | Separate WASM module, canvas driver, Booba transport, demo site |
| `web/site/demos.js` | Gallery entries and their WASM asset paths |
| `examples/` | Host PNG/GIF previews |
| `internal/` | Firmware compile checks |
| `Makefile`, `.github/workflows/ci.yml` | Local checks and CI |

Read [API.md](API.md) for rendering and memory budgets, [DEVICES.md](DEVICES.md)
for boards, and [web/README.md](web/README.md) for the browser bridge.

## Shared code and widgets

- Keep the core small and free of terminal, OS, and browser dependencies.
  TinyGo 0.43.0 is the minimum compiler; the root module uses Go 1.23 syntax.
- Accept `drivers.Displayer`: `Size`, `SetPixel`, and `Display`. Detect optional
  `FillRectangle(..., color.RGBA) error` with a type assertion. Keep the pixel fallback.
- Keep `machine`, buses, pins, power, and rotation in device mains with board
  build tags. Ordinary Go tests must not import `machine`.
- Design widgets for small screens, three-button navigation, monochrome/color,
  and mixed font sizes. Share widget state and cell layouts across runtimes;
  let the owning model schedule commands and timers.
- Reuse buffers. Change application state in `Update`, not `View`.
- The renderer keeps a previous-cell grid, clips glyphs, clears stale cells,
  and flushes changed frames. Preserve retry behavior after display failures
  and support both buffered OLED and direct-write TFT drivers.
- Fonts receive a clipping display and must not retain it. Large hello glyphs
  are cell tiles; keep them going through the renderer.
- `Displayer` does not report color depth. Choose the theme explicitly, pass
  monochrome settings where supported, and size layouts from `WindowSizeMsg`.
  Set physical orientation before creating the program.
- Budget model and renderer grids at about 16 bytes per cell: 10 KiB for 40×16.
  Drivers, fonts, heap, and stacks add to that. Static RAM figures exclude heap
  allocations. The browser driver has its own RGBA framebuffer.

## Runtime rules

Bubble Tea owns its event loop, commands, goroutines, and timers. Device code
can inject keys and custom messages through `Program.Send`. Use `CellView()`
for styled cells and custom fonts; plain view content is clipped text, without
ANSI interpretation. The firmware backend uses upstream v2.1 plus patches in
an ignored vendor directory. Run `make -C bubbletea prepare` after changing
local dependencies; never patch the module cache or another checkout.

These rules apply only to TinyTea and the original root runtime:

- Root models return `*Buffer` from `View`; TinyTea models return `tinytea.View`
  and may implement `CellView() *Buffer`.
- `Start` queues Init's command, sends the initial size, and renders. `Step`
  polls, executes at most one queued command, and renders. `Send` updates and
  renders immediately but leaves returned commands for a later step.
- Commands run cooperatively. Do not block, sleep, or wait for network input
  inside them. Device loops supply ticks. The command queue holds 16 entries
  and reports `ErrCommandQueueFull` on overflow.
- Keep program operations on one loop. Interrupts and other tasks must hand
  events to that loop rather than call `Send` directly.
- `KeyQuit` reaches the model; `QuitMsg` stops the program.

## Browser demos

Hello World has a Bubble Tea / TinyTea selector. Each selection starts the
chosen runtime loop from scratch and preserves the display preset. The browser
bundle includes both runtimes; do not use its size to compare firmware memory.
`web/tinyhello.go` adapts only hello: it reuses the visual model and bindings,
supplies host ticks, and uses TinyTea commands for quitting. It is not a general
Bubble Tea command adapter. Keep TinyTea program calls on one host loop; JS
callbacks enqueue input. Firmware uses `make hello` or `make hello-tinytea`.

- Keep Booba and its pinned Bubble Tea WASM fork in `web/go.mod`. The firmware
  module uses upstream v2.1. Root module replacements do not inherit web settings.
- Use `displaytea.NewModel` when Booba owns the program. Pass its context and
  cell size to the program, check `Err()` after Run, and call `Close()`.
- `WithoutRenderer()` still allows `tea.Raw` output. Keep Booba's input and
  output connected for capability replies and action metadata. Sending output
  to `io.Discard` breaks this path. `View.ProgramStatus` needs terminal rendering;
  the demo uses Booba's `bubblehelp` raw-output path instead.
- Define actions in Bubbles key bindings used by both Update and the help map.
  HTML buttons must follow that metadata. Keep widget behavior in Go.
- Register working demos in `web/site/demos.js` with a category, source link,
  and WASM path. Add the asset build step too. All entries in a build must use
  the same compiler/runtime shim. Demo links preserve the display preset and
  reload the page for a fresh program.
- Keep the page compact. Explain TinyGo, Bubble Tea, and TinyBubbles plainly.
  Put the display and controls first; collapse supporting details.

## Build and check

Run from the repository root:

```sh
make check          # root tests, vet, build, modules, formatting, docs
make ci             # all modules, previews, firmware, Go/TinyGo WASM
make firmware       # Pico, Pico 2, CLUE, ESP32-S3, cooperative Feather
make hello          # Bubble Tea Feather firmware
make hello-tinytea  # original cooperative Feather firmware
make hello-preview  # build/hello.png and build/hello.gif
make snapshot       # counter snapshot.png
make web-serve      # build Go WASM and serve on port 8080
make web-tinygo     # build TinyGo WASM with a 128 KiB stack
make -C web test    # Go bridge tests, vet, module checks, JS encoder tests
make -C bubbletea prepare test  # prepare dependencies and test the adapter
```

Root tests do not cover the nested modules. Firmware selects Go 1.26.0; web
selects Go 1.26.8. Web tests need Node.js 22+. Docs and asset builds need Python 3.
Use `TINYGO=/path/to/tinygo` to select a compiler. Rebuild when switching WASM
compilers so `wasm_exec.js` matches the binary.

Choose checks that match the change:

- Core: host checks and the firmware matrix.
- Board setup: that target's build and relevant model tests.
- Runtime/bridge: that module's tests and affected firmware or WASM build.
- Browser UI: inspect the page and check actions, navigation, and lifecycle.
- Visuals: inspect the rendered preview. Avoid tests that repeat layout constants.
- Docs: `make check-docs` and `git diff --check`; no firmware build or flash.

Use focused tests for clipping, failed flushes, command order, debounce, and
buffer reuse. Run `gofmt` on changed Go files. State separately what compiled,
what ran in a browser or on the host, and what was confirmed on hardware.

When the shared hello visuals change, refresh the README animation if needed:

```sh
make hello-preview
cp build/hello.gif docs/assets/hello.gif
```

CI runs on pushes, pull requests, and manual dispatch. It uploads named previews,
firmware files, and `build/web/`. It does not flash or deploy. Never upload the
whole `build/` directory.

## Hardware and private files

The Adafruit ESP32-S3 Reverse TFT Feather (5691) is hardware-tested. Pico,
Pico 2, and CLUE have compile checks only. Use the Feather's local
`devices/feather-reverse-tft/target.json`. D0 pops, D1 changes palette, and D2
pauses; preserve those controls unless asked to change them. Wiring, power,
rotation, and pin polarity are in its main and device guide.

Flash only for authorized hardware work. Identify the board and current serial
port first; ports can change after reset. Follow the selected runtime's flash
guide. Preserve existing CircuitPython files before replacing that firmware.
If automatic reset fails, ask the user to hold D0/BOOT, tap RESET, then release
D0; rescan the port before retrying. Flasher success does not prove the screen works.

`build/` may contain private device backups and credentials. Never read, print,
commit, publish, or copy those backups into demo assets. Generate the README GIF
from `examples/hello-preview`.

## Docs and licensing

Write plainly: say what a feature does, why it matters, and how to use it.
Keep detailed API, device, and browser instructions in their guides. Follow the
ntcharts/ntcharts3d README structure: badges, demo, quickstart, tables,
collaboration links, acknowledgements, license, and NimbleTerminal footer.
Do not claim untested board support or add badges for unreleased versions.

TinyBubbles is MIT licensed, copyright (c) 2026 Neomantra Corp. Preserve dependency
and font licenses. Web distributions include `THIRD_PARTY_NOTICES.txt`; update
its inputs when changing bundled dependencies.
