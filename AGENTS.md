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
| `layout/` | Allocation-free cell regions, padding, alignment, and buffer composition |
| `font.go`, `theme.go`, `input.go` | Fonts, procedural Unicode chart glyphs, palettes, key events, button debounce/repeat/long-press |
| `bubbletea/` | Primary runtime adapter, portability patches, Feather `hello` and `feather-settings` mains |
| `bubbletea/hello/` | Bubble Tea hello model and shared Bubbles key bindings |
| `tinytea/` | Optional cooperative subset of the Bubble Tea model API with cooperative timers (`Tick`, `Every`, `After`, `Timer`) |
| `tinyhal/` | In-WASM hardware abstraction layer for TinyGo `machine` pins, buses, peripheral models, and VCD trace recording (`tinyhal/workbench/` for the standalone MCU bench) |
| `program.go` | Original cooperative runtime used by TinyTea and older demos |
| `demo/hello/` | Shared greeting, custom font tiles, enlarged counter |
| `demo/model.go` | Counter used by Pico and CLUE |
| `widgets/` | Shared state and cell layouts for list, value, choice, dialog, progress, status, and viewport |
| `tinycharts/` | Cell-buffer sparklines, Braille/arc/box streamlines, bar charts, and logic analyzer waveforms |
| `bubbletea/<widget>/` | Bubble Tea adapters and Bubbles help bindings |
| `tinytea/<widget>/` | Cooperative widget adapters |
| `bubbletea/settings/` | Composed menu, focus routing, child command handling, and three-button Feather config |
| `devices/` | Board setup, pins, drivers, and local targets; shared Feather initialization in `feather-reverse-tft/board` |
| `tinyghostty/` | Nested module: Ghostty VT terminal widget over a static libghostty-vt (`vt` cgo shim, `render` to cell buffers, Bubble Tea model); needs `make -C tinyghostty lib`, not in `make ci` |
| `devices/terminal/` | Nested module: hello display drawn in a terminal through ntpicture, help bindings hosted as clickable Terminal Actions; native and wasip1 builds |
| `web/` | Reusable browser harness module, actions, canvas, LEDs, scaffold tool |
| `web/hello/` | Demo gallery consumer module (Hello, TinyHAL, TinyCharts, layout, Settings, widgets) and WASM main |
| `examples/` | Host PNG/GIF previews |
| `internal/` | Firmware compile checks and Go/TinyGo benchmark suite (`internal/bench`) |
| `Makefile`, `Taskfile.yml`, `.github/workflows/ci.yml` | Local checks, task runner aliases, and CI |

Read [API.md](API.md) for rendering and memory budgets, [DEVICES.md](DEVICES.md)
for boards, and [web/README.md](web/README.md) for the browser bridge.

## Shared code and widgets

- Keep the core small and free of terminal, OS, and browser dependencies.
  TinyGo 0.43.0 is the minimum compiler; the root module uses Go 1.23 syntax.
- Accept `drivers.Displayer`: `Size`, `SetPixel`, and `Display`. Detect optional
  `FillRectangle(..., color.RGBA) error` with a type assertion. Keep the pixel fallback.
- Keep `machine`, buses, pins, power, and rotation in device mains or board packages with board
  build tags. Ordinary Go tests must not import `machine`.
- Design widgets for small screens, three-button navigation, monochrome/color,
  and mixed font sizes. Share widget state and cell layouts across runtimes;
  let the owning model schedule commands and timers.
- Reuse buffers with `Buffer.Resize`. Change application state in `Update`, not `View`.
- Use `layout.ClipText` for one-line labels; it skips controls when aligning.
  Define result messages in shared widgets and alias them in runtime adapters.
- Use `layout` to split cell regions and compose buffers. Size children in `Update`,
  clear the parent before drawing, and keep focus routing in the owning model.
  Parents choose responsive breakpoints; layout does not infer them. See
  [layout/README.md](layout/README.md) and `web/hello/layout.go`.
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

Lists copy their items on replacement and reuse the cell buffer while drawing.
Forward input only to the focused widget. Selection commands capture the item
at activation; parents decide what selecting it does. Value and choice pickers
keep draft and applied values separate; only parents perform device or storage
effects. `bubbletea/settings` demonstrates focus routing and consumes child
commands before accepting more input. Dialogs default to Cancel and emit one
decision until reset; parents own focus and the confirmed action. Progress bars
receive counts and pulses from the parent; they own no timers, key bindings, or
completion effects. Status indicators receive state and detail from the parent;
keep symbols and text meaningful without color. Text viewports hard-wrap at cell
boundaries, cache line offsets, preserve plain-text spacing, and allocate only a
visible grid without ANSI interpretation. See [widgets/README.md](widgets/README.md).

## Runtime rules

Bubble Tea owns its event loop, commands, goroutines, and timers. Device code
injects keys and custom messages through `Program.Send`. Use `CellView()`
for styled cells and custom fonts; plain view content is clipped text without
ANSI interpretation. The firmware backend uses upstream v2.1 plus patches in
an ignored vendor directory. Run `make -C bubbletea prepare` after changing
local dependencies; never patch the module cache or another checkout.

These rules apply only to TinyTea and the original root runtime:

- Root models return `*Buffer` from `View`; TinyTea models return `tinytea.View`
  and may implement `CellView() *Buffer`.
- `Start` queues `Init`'s command, sends the initial size, and renders. `Step`
  polls input and cooperative timers, executes at most one queued command, and
  renders. `Send` updates and renders immediately but leaves returned commands
  for a later step.
- Commands run cooperatively. Do not block, sleep, or wait for network input
  inside them. Use `tinytea.Tick`, `Every`, `After`, or a model-owned `Timer`
  (`Tick`, `Every`, `After`, `Repeat`, `Cancel`), or supply ticks from the
  device loop. The command queue holds 16 entries and reports
  `ErrCommandQueueFull` on overflow.
- Keep program operations on one loop. Interrupts and other tasks must hand
  events to that loop rather than call `Send` directly.
- `Run` is fail-fast in both runtimes. Rendering failures use `*DisplayError`.
  Cooperative device mains retry only those errors in paced `Step` loops; model
  and queue errors remain fatal. Bubble Tea cancels on display failure.
- `KeyQuit` reaches the model; `QuitMsg` stops the program.

## Browser demos

The gallery includes Hello, TinyHAL workbench, TinyCharts, layout, Settings, list, value,
dialog, progress, status, and viewport demos. Settings demonstrates choice
pickers. All entries except Hello use Bubble Tea only. New widgets have compile
and browser checks; do not describe them as hardware-tested.

Hello World has a Bubble Tea / TinyTea selector. Each selection starts the
chosen runtime loop from scratch and preserves the display preset. The browser
bundle includes both runtimes; do not use its size to compare firmware memory.
`web/hello/tinyhello.go` adapts only hello: it reuses the visual model and bindings,
supplies host ticks, and uses TinyTea commands for quitting. It is not a general
Bubble Tea command adapter. Keep TinyTea program calls on one host loop; JS
callbacks enqueue input. Firmware uses `make hello`, `make settings`, or `make hello-tinytea`.

- Keep Booba and its pinned Bubble Tea WASM fork in `web/go.mod`. The terminal
  module also depends on go-booba, for Terminal Actions only. The firmware
  module uses upstream v2.1. Root module replacements do not inherit web settings.
- Use `displaytea.NewModel` when Booba owns the program. Pass its context and
  cell size to the program, check `Err()` after `Run`, and call `Close()`.
- `WithoutRenderer()` still allows `tea.Raw` output. Keep Booba's input and
  output connected for capability replies and action metadata. Sending output
  to `io.Discard` breaks this path. `View.ProgramStatus` needs terminal rendering;
  the demo uses Booba's `bubblehelp` raw-output path instead.
- Define actions in Bubbles key bindings used by both `Update` and the help map.
  HTML buttons follow that metadata. Keep widget behavior in Go. Multiple
  key aliases produce multiple buttons, so avoid redundant aliases in short help.
  Disabled actions remain in metadata with `Enabled: false`; test that flag.
- Supply display presets and gallery entries through Go `web.Config`. The
  shared page receives them at startup. Compile the app models into `app.wasm`
  and select one using the returned selection. Links preserve presets and reload.
- Keep app models and renderer dependencies in `web/hello`, not the reusable
  harness. Consumers generate pages with `go tool tinybubbles-web --assets`;
  only `index.html` and `style.css` may override generated page assets. Do not
  copy JavaScript, logos, notices, or licenses. Scaffold refreshes preserve `go.mod`.
  Each consumer web module must repeat the Bubble Tea WASM replacement.
- `WrapActions` uses Booba's eight-character hashed IDs and validates the initial OSC payload. Keep
  host tests for distinct binding IDs and the 2048-byte message limit. Keyboard input
  must be restricted to enabled bindings in the current help map.
- Hidden browser tabs throttle timers to about once per second and slow every
  `time.Sleep`. Keep the page visible during timing checks. Do not add Web Audio
  as part of harness work.
- Keep the page compact. Explain TinyGo, Bubble Tea, and TinyBubbles plainly.
  Put the display and controls first; collapse supporting details.

## Build and check

Run from the repository root:

```sh
make check          # root tests, vet, build, modules, formatting, docs
make ci             # all modules, previews, firmware, tinyhal-wasi, Go/TinyGo WASM
make firmware       # Pico, Pico 2, CLUE, ESP32-S3, cooperative Feather
make hello          # Bubble Tea Feather hello + settings firmware
make settings       # Bubble Tea Feather three-button settings firmware
make hello-tinytea  # original cooperative Feather firmware
make tinyhal-wasi   # TinyHAL machine/ST7789/APA102/VCD smoke demo under TinyGo wasip1
make bench          # Go testing.B benchmarks plus Go vs. TinyGo native/WASI comparison
make hello-preview  # build/hello.png and build/hello.gif
make snapshot       # counter snapshot.png
make widgets-preview # widget screenshots in docs/assets/widgets/
make tinycharts-preview # chart screenshots in docs/assets/tinycharts/
make web-serve      # build Go WASM and serve on port 8080
make web-tinygo     # build TinyGo WASM with a 128 KiB stack
make -C web test    # Go bridge tests, vet, module checks, JS key mapping tests
make -C bubbletea test          # refresh vendored dependencies and test the adapter/widgets
make -C devices/terminal check  # tidy, vet, test, and build the terminal target (native + wasip1)
```

Root tests do not cover the nested modules. Firmware selects Go 1.26.0; web and
the terminal target select Go 1.26.8. Web tests need Node.js 22+. Docs and asset builds need Python 3.
Use `TINYGO=/path/to/tinygo` to select a compiler. Rebuild when switching WASM
compilers so `wasm_exec.js` matches the binary.

Choose checks that match the change:

- Core, widgets, charts, HAL, and layout: host checks and the relevant firmware/WASI checks.
  Widget adapters also need their nested module tests.
- Board setup: that target's build and relevant model/smoke tests.
- Runtime/bridge: that module's tests and affected firmware or WASM build.
- Browser UI: inspect the page and check actions, navigation, and lifecycle.
- Visuals: inspect the rendered preview. Avoid tests that repeat layout constants.
- Docs: `make check-docs` and `git diff --check`; no firmware build or flash.

Use focused tests for clipping, failed flushes, command order, button debounce/repeat/long-press,
cooperative timers, buffer reuse, draft/apply/cancel behavior, viewport reflow, and focus routing.
Run `gofmt` on changed Go files. State separately what compiled, what ran in a browser or on the host,
and what was confirmed on hardware.

When the shared hello visuals change, refresh the README animation if needed:

```sh
make hello-preview
cp build/hello.gif docs/assets/hello.gif
```

When widget or chart rendering changes, run `make widgets-preview` or
`make tinycharts-preview` to refresh the screenshots in `docs/assets/` used by
`widgets/README.md` and `tinycharts/README.md`.

CI runs on pushes, pull requests, and manual dispatch. It uploads named previews,
firmware files, and `build/web/`. After all checks pass on `main`, it deploys
the TinyGo WASM site to GitHub Pages. Pull requests and tags do not deploy.
It does not flash hardware. Never upload the whole `build/` directory.

## Hardware and private files

The Adafruit ESP32-S3 Reverse TFT Feather (5691) hello demo is hardware-tested.
Pico, Pico 2, CLUE, and the Feather settings target have compile and `tinyhal` smoke checks.
Use the Feather's local `devices/feather-reverse-tft/target.json`. In `hello`, D0 pops,
D1 changes palette, and D2 pauses; preserve those controls unless asked to change them.
In `feather-settings`, D0 moves up/decreases (hold to repeat), D1 moves down/increases
(hold to repeat), and D2 selects/applies on a short tap or cancels on a long press.
Wiring, power, rotation, and pin polarity are in the board package and device guide.

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
