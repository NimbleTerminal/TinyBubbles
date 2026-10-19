# Changelog

This file lists what changed in each TinyBubbles release, newest first. It
follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and
[Semantic Versioning](https://semver.org/spec/v2.0.0.html). Before 1.0, a minor
release may change public APIs; the notes will say so.

## v0.2.0 - 2026-10-19

New packages and targets. Existing v0.1.1 APIs are unchanged; the additions
below are additive.

### Input and runtimes

- **Button interactions.** `Button` and `ButtonPoller` add debounce,
  hold-to-repeat, short-tap versus long-press keys, and release events, so a
  three-button board can drive menus and pickers.
- **TinyTea cooperative timers.** `Tick`, `Every`, `After`, and a reusable,
  cancellable `Timer` (`Tick`, `Every`, `After`, `Repeat`, `Cancel`) schedule
  work without goroutines. [RUNTIMES.md](RUNTIMES.md) describes the model.
- **Three-button Feather settings firmware.** `make settings` builds the composed
  settings menu for the Reverse TFT Feather: D0 moves up or decreases, D1 moves
  down or increases, and D2 selects on a tap or cancels on a long press.
  Compiled and smoke-tested with TinyHAL; not yet confirmed on hardware.

### TinyCharts

- **`tinycharts` package.** Allocation-conscious charts for cell buffers:
  `BrailleGrid`, `Sparkline`, `Streamline` (Braille, arc, and box modes),
  `BarChart` with `DrawGauge`, and a multi-channel `LogicChart` for digital,
  PWM, and bus traces. `PaintBuffer` draws any chart into a `drivers.Displayer`.
- **Procedural chart glyphs.** `DefaultFont()` rasterizes the Unicode box and
  arc drawing, block element, and Braille ranges into any cell size, with no
  extra bitmap tables. See the [TinyCharts guide](tinycharts/README.md).

### TinyHAL

- **`tinyhal` package.** An in-WASM hardware abstraction layer for TinyGo
  `machine` pins and buses with peripheral models (ST7789, SSD1306, APA102,
  sensors, UART) and an IEEE 1364 VCD trace recorder. Board smoke tests run
  the Pico, CLUE, and Feather setups on the host.
- **MCU workbench.** `tinyhal/workbench` is a standalone bench with simulated
  buttons, PWM LEDs, an ADC oscilloscope, and a logic analyzer view. It runs in
  the browser gallery and under TinyGo `wasip1` (`make tinyhal-wasi`).

### Terminal target

- **`devices/terminal` module.** Draws the hello display inside a terminal
  through [ntpicture](https://github.com/NimbleTerminal/ntpicture) as Kitty
  graphics or half-block cells, and hosts the help bindings as clickable
  Terminal Actions from go-booba. It builds natively and for `wasip1` with Go
  or TinyGo. `make -C devices/terminal check` covers tidy, vet, tests, and builds.

### Browser

- Added the TinyHAL workbench and TinyCharts demos to the gallery.
- TinyGo WASM and WASI builds use the Boehm garbage collector.

### Build and docs

- Added `Taskfile.yml` mirroring every Makefile target; `make check-docs`
  keeps the two in sync.
- Added `make bench` with Go `testing.B` benchmarks and Go versus TinyGo
  native and WASI comparisons (`internal/bench`).
- Added host screenshot generators (`make widgets-preview`,
  `make tinycharts-preview`); the widget and chart guides open with preview tables.
- Tightened [AGENTS.md](AGENTS.md) and the README; feature tables list the
  new packages.

Downstream modules can update `nimbleterminal.dev/tinybubbles`,
`nimbleterminal.dev/tinybubbles/bubbletea`, and
`nimbleterminal.dev/tinybubbles/web` to v0.2.0 together.

## v0.1.1 - 2026-10-11

### Browser

- Published the [live demo gallery](https://nimbleterminal.github.io/TinyBubbles/)
  on GitHub Pages. Successful CI runs on `main` deploy the TinyGo WASM build.
  New pushes wait for an active main-branch run instead of cancelling its deployment.
- Replaced the harness's custom action IDs with Booba's eight-character hashed
  IDs. Initial action metadata still fails clearly if it exceeds 2048 bytes,
  including the summary. Action IDs are opaque; identify bindings by key metadata.
- Replaced the local key encoder with Booba's encoder, which follows cursor,
  backarrow, and newline modes. Enabled help bindings still control which keys
  are accepted. Printable Unicode is supported without a fallback.
- Forced asset refreshes preserve consumer-owned `demos.js` files unless a
  previous manifest lists them as generated assets.
- Added a GitHub logo to the gallery's Source link and linked it to the README.

Downstream browser consumers can update `nimbleterminal.dev/tinybubbles/web`
to v0.1.1 and rebuild to regenerate the harness assets. No firmware or widget
behavior changed.

## v0.1.0 - 2026-10-11

First release. Entries are grouped by area.

### Runtimes

- **Bubble Tea on small displays.** A real Bubble Tea v2.1 program renders to
  any TinyGo display driver (`drivers.Displayer`). Models keep their normal
  commands, timers, and messages. Device code sends button presses and custom
  messages with `Program.Send`.
- **TinyTea, a smaller optional runtime.** It accepts the same basic model
  shape but runs cooperatively: one short command per step, no goroutines, and
  ticks supplied by the device loop. Use it when Bubble Tea does not fit.
  [RUNTIMES.md](RUNTIMES.md) lists the differences and how to switch.
- **Firmware patches for Bubble Tea.** Upstream Bubble Tea does not build for
  bare-metal TinyGo. The `bubbletea` module vendors it and applies patches with
  `make -C bubbletea prepare`. See the
  [backend guide](bubbletea/README.md#portability-changes).
- **Display errors.** Both runtimes return display failures from `Run`. Cooperative
  device examples retry transient failures through paced `Step` loops.

### Rendering

- **Cell buffers.** Text lives in a grid of character cells, each holding one
  rune and a style ID. Buffers are reused between frames. Writes are clipped,
  so oversized content never corrupts memory. Nil buffers behave as empty grids.
- **Changed-cell renderer.** Only cells that changed are redrawn, and the
  display is flushed only when something changed. A failed flush is retried
  on the next frame. Drivers that offer `FillRectangle` get faster background
  fills; others fall back to pixel writes. Full repaints avoid redundant background
  fills. Custom-cell models do not allocate a fallback text grid.
- **Fonts and themes.** The default font is TinyFont's Org01 face in 6×8
  pixel cells. Custom fonts can draw large glyphs and image tiles through the
  same renderer. Themes have eight styles. `DefaultTheme(true)` is black and
  white; `DefaultTheme(false)` adds color.

### Widgets

Seven widgets share one state and drawing implementation, with adapters for
both runtimes. They are built for small screens and three-button navigation,
and each stays readable in monochrome. See the
[widget guide](widgets/README.md).

- **List.** Scrolls, wraps at the ends, skips disabled items, and reports the
  selected item to the parent.
- **Value picker.** Edits a bounded number. It keeps a draft separate from the
  applied value, so Cancel restores the last applied setting.
- **Choice picker.** The same draft, apply, and cancel pattern for a fixed set
  of named options, including on/off toggles.
- **Confirmation dialog.** Defaults to Cancel and gives exactly one answer per
  prompt. The parent performs the confirmed action.
- **Progress bar.** Shows a known total as a percentage, or an activity
  indicator when the total is unknown, including on one-cell bars. The parent
  owns the timer.
- **Status indicator.** Offline, Ready, Busy, Warning, and Error with distinct
  ASCII symbols, so the state is clear without color.
- **Text viewport.** Scrolls and pages through plain text. It wraps at cell
  boundaries and allocates only the visible grid. It keeps the reading position
  through collapsed layouts and keeps bottom-pinned views at the end after reflow.
- **Settings example.** Combines the list, value, choice, and dialog widgets
  and shows how a parent routes focus and consumes child results.

### Layout

- **Layout helpers.** Split a grid into rows and columns, add padding, align a
  child, and copy child buffers into a parent. They allocate nothing and work
  with either runtime. One-cell labels keep their first character, and centered
  labels ignore control characters. See the [layout guide](layout/README.md).

### Browser

- **Demo gallery.** `make web-serve` runs Hello World, every widget, the
  Settings screen, and a layout playground in a browser with OLED and TFT
  presets. Hello World can switch between Bubble Tea and TinyTea. Links keep
  the chosen demo and preset.
- **Reusable browser harness.** The `web` module gives any TinyGo display
  project a canvas display, simulated LEDs, keyboard input, and HTML buttons
  generated from the model's Bubbles help map. Disabled actions show as
  disabled buttons. `go tool tinybubbles-web` scaffolds a new project. See the
  [browser guide](web/README.md). The harness uses go-booba v0.9.1.
- **Scaffold updates.** Builds generate harness JavaScript and support assets;
  consumers can override HTML and CSS. Refreshes preserve existing `go.mod`
  files, even with `--force`, and remove obsolete generated assets.

### Devices

- **Adafruit ESP32-S3 Reverse TFT Feather.** The Hello demo runs on real
  hardware with the Bubble Tea runtime. D0 pops bubbles, D1 changes colors,
  and D2 pauses the animation.
- **Raspberry Pi Pico, Pico 2, and Adafruit CLUE.** Counter examples compile
  with TinyGo 0.43.0 but have not been run on hardware. They use the original
  cooperative runtime, not Bubble Tea. See [DEVICES.md](DEVICES.md).

### Project

- **Tooling.** `make check` runs host tests, vet, formatting, module, and docs
  checks. `make ci` adds previews, firmware for every board, and Go and TinyGo
  WASM builds. GitHub Actions runs the same steps.
- **Toolchains.** TinyGo 0.43.0 is the minimum compiler. The root module uses
  Go 1.23 language features. The firmware module selects Go 1.26.0 and the web
  modules select Go 1.26.8.
- **Guides.** [API.md](API.md), [DEVICES.md](DEVICES.md),
  [RUNTIMES.md](RUNTIMES.md), and the widget, layout, browser, and backend
  guides. MIT license, with dependency and font licenses preserved.

### Known limitations

- Plain text views are clipped, not interpreted. ANSI colors, terminal
  styling, and mouse regions are ignored.
- One rune fills one cell. There is no wide-character or combining-character
  layout.
- Only the Feather has been tested on hardware. Other boards have compile
  checks only.
- TinyTea has no `Tick` or `Every` helpers. The device loop supplies ticks.
- The Bubble Tea runtime costs about 260 KiB of flash and 91 KiB of static RAM
  on the Feather, before heap and stacks. Small RP2040 and nRF52 boards have
  not been measured.
