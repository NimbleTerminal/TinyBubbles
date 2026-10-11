# TinyBubbles - TinyGo Display UIs

<p>
    <a href="https://github.com/NimbleTerminal/TinyBubbles/actions/workflows/ci.yml"><img src="https://github.com/NimbleTerminal/TinyBubbles/actions/workflows/ci.yml/badge.svg" alt="Build Status"></a>
    <a href="https://pkg.go.dev/nimbleterminal.dev/tinybubbles"><img src="https://pkg.go.dev/badge/nimbleterminal.dev/tinybubbles.svg" alt="Go Reference"></a>
    <a href="https://tinygo.org"><img src="https://img.shields.io/badge/TinyGo-0.43%2B-blue" alt="TinyGo 0.43 or newer"></a>
    <a href="./LICENSE"><img src="https://img.shields.io/badge/License-MIT-blue.svg" alt="MIT License"></a>
</p>

**TinyBubbles brings Bubble Tea to small embedded displays.**
[TinyGo](https://tinygo.org) compiles Go for microcontrollers.
[Bubble Tea](https://github.com/charmbracelet/bubbletea) is a Go framework for
interactive terminal apps. TinyBubbles lets you use it on a tiny screen and try
the same Go UI in your browser.

Bubble Tea v2.1 is the primary runtime. [TinyTea](RUNTIMES.md) is an optional,
smaller replacement with limited functionality. Both use the same cells,
fonts, and renderer.

[![Hello, Tiny World: floating bubbles, color changes, and an enlarged pop counter](docs/assets/hello.gif)](devices/feather-reverse-tft/README.md)

**Hello, Tiny World** · [Source](demo/hello/model.go) · [Device guide](devices/feather-reverse-tft/README.md)

The preview renders the actual demo model and font on the desktop, enlarged 3×.
The demo also runs on the **Adafruit ESP32-S3 Reverse TFT Feather**: D0 pops bubbles,
D1 changes colors, and D2 pauses or resumes animation.

Initialize any [`drivers.Displayer`](https://github.com/tinygo-org/drivers/blob/v0.36.0/displayer.go)
in a device entry point, then pass it to the Bubble Tea display backend. Models
use real Bubble Tea commands, timers, and messages. Views can provide plain text
or reusable cell buffers with custom fonts, including large glyphs and counters.

| Component | Description |
| :--- | :--- |
| [Bubble Tea backend](bubbletea/README.md) | Real Bubble Tea v2.1 Program with a Displayer renderer. |
| [TinyTea](RUNTIMES.md) | Optional cooperative subset with a similar model API for smaller devices. |
| [Widgets](widgets/README.md) | Lists, value and choice pickers, dialogs, progress, status, and scrollable text. |
| [Layout](layout/README.md) | Cell regions, padding, splits, alignment, and clipped composition. |
| [Cell buffer](cell.go) | Reusable character grids with UTF-8 text and clipped writes. |
| [Renderer](API.md#displayer-and-rendering) | Changed-cell drawing and optional FillRectangle acceleration. |
| [Fonts and themes](API.md#fonts-and-themes) | TinyFont, custom glyphs, palettes, and monochrome/color defaults. |
| [Browser playground](web/README.md) | WASM demos with OLED/TFT presets and HTML controls from Go help maps. |
| [Devices](DEVICES.md) | Hardware initialization, buttons, buses, and power controls. |

## Quickstart

Try the browser demo from a checkout:

```sh
make web-serve
```

Open <http://127.0.0.1:8080>. This needs Go and Python 3; the build selects
Go 1.26.8. Pick a display, pop bubbles, change colors, or pause the animation.
Explore the [demo gallery](#demo-apps), from individual widgets to a composed
settings screen and an interactive layout playground. Hello World also lets you
switch between Bubble Tea and TinyTea.

To build firmware, install **TinyGo 0.43.0 or newer**:

```sh
make hello          # Bubble Tea firmware: build/bubbletea-feather.bin
make hello-preview  # desktop PNG/GIF of the shared greeting
```

The firmware build selects Go 1.26.0 and applies the required portability patches.
This is an early port; read the [backend guide](bubbletea/README.md) for its
limitations and flash instructions. See the [device guide](DEVICES.md) for wiring
and adding a board.

## Usage

Use `nimbleterminal.dev/tinybubbles` for Go imports. The repository and issue
tracker remain at [NimbleTerminal/TinyBubbles](https://github.com/NimbleTerminal/TinyBubbles).

Use Bubble Tea's model interface and the separate `bubbletea` module:

```go
import (
    tea "charm.land/bubbletea/v2"
    bubbles "nimbleterminal.dev/tinybubbles"
    displaytea "nimbleterminal.dev/tinybubbles/bubbletea"
)

type model struct{}
func (model) Init() tea.Cmd { return nil }
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    if key, ok := msg.(tea.KeyPressMsg); ok && key.Code == 'q' {
        return m, tea.Quit
    }
    return m, nil
}
func (model) View() tea.View { return tea.NewView("Hello, tiny world!") }

// Inside your initialized device main:
p, err := displaytea.NewProgram(model{}, display, nil, bubbles.DefaultTheme(false))
if err != nil { panic(err) }
if _, err := p.Run(); err != nil { panic(err) }
```

Send keys and custom messages through `p.Send`. Add `CellView() *bubbles.Buffer`
to render styled cells or custom font tiles. Plain View content is clipped to
the grid; ANSI and terminal graphics are not interpreted. Read the
[backend guide](bubbletea/README.md) before using terminal-specific libraries.

For a smaller cooperative runtime, use [TinyTea](tinytea/tea.go). It supports the
basic model signatures and the same cell rendering, but has different command,
timer, and input semantics. The [runtime comparison and migration guide](RUNTIMES.md)
explains the supported subset and how to switch. Device-focused widgets can share
state and cell layouts across both runtimes.

## Demo Apps

| Demo | Source | What it shows |
| :--- | :--- | :--- |
| Hello, Tiny World | [bubbletea/hello](./bubbletea/hello) | Floating bubbles, large text and counter, four palettes, and pause/resume on the Reverse TFT Feather. |
| Layout playground | [web/hello/layout.go](web/hello/layout.go) | Responsive rows and columns, padding, gaps, and alignment. |
| Settings | [bubbletea/settings](bubbletea/settings) | A menu combining values, choices, toggles, and confirmation. |
| Scrolling menu | [bubbletea/list](bubbletea/list) | Selection, disabled items, and scrolling. |
| Value picker | [bubbletea/value](bubbletea/value) | Bounded numbers with apply and cancel. |
| Confirmation dialog | [bubbletea/dialog](bubbletea/dialog) | Safe default and one decision per prompt. |
| Progress bar | [bubbletea/progress](bubbletea/progress) | Known totals and indeterminate activity. |
| Status indicator | [bubbletea/status](bubbletea/status) | State symbols, labels, and optional detail. |
| Text viewport | [bubbletea/viewport](bubbletea/viewport) | Scrollable plain text with paging. |
| Counter | [demo](./demo/model.go) | One model adapting to a monochrome OLED or color TFT. |
| Hello preview | [examples/hello-preview](./examples/hello-preview) | The hello demo rendered to PNG and an animated GIF with ordinary Go. |
| Counter snapshot | [examples/snapshot](./examples/snapshot) | A desktop Displayer implementation that writes a PNG. |

The first nine entries run in the browser gallery. Choice pickers appear in
Settings. Only Hello World has a runtime selector; the other browser demos use
Bubble Tea. All seven widgets have shared state and layouts plus adapters for
both runtimes. See the [widget guide](widgets/README.md).

## Device Support

| Device | Display | TinyGo target | Validation |
| :--- | :--- | :--- | :--- |
| Adafruit Reverse TFT Feather | ST7789, 240×135 | [Local target](./devices/feather-reverse-tft/target.json) | Hello demo confirmed on hardware |
| Raspberry Pi Pico | SSD1306, 128×64 | `pico` | Compiled |
| Raspberry Pi Pico 2 | SSD1306, 128×64 | `pico2` | Compiled |
| Adafruit CLUE | ST7789, 240×240 | `clue` | Compiled |

The core also has an ESP32-S3 compile check. Other boards can use any initialized
`drivers.Displayer`; concrete peripherals stay under `devices/<name>/main.go`.
See [adding a device](./DEVICES.md#devices-and-wiring).

The default font uses 6×8 pixel cells. Two cell grids take about **2.6 KiB** on a
128×64 OLED, **10 KiB** on a 240×135 TFT, or **24.8 KiB** on a 320×240 display.
Composed screens also retain child buffers; viewports retain text and line offsets.
The core renderer allocates no pixel framebuffer; display drivers may use one.
See [memory and resolution](./API.md#memory-and-resolution) for the full budget.

## Development

See the [changelog](CHANGELOG.md) for the upcoming v0.1.0 release summary.

The [browser playground](web/README.md) uses the shared Go model, fonts, and
renderer. Booba turns the model's Bubbles help map into HTML controls: Pause
becomes Resume, and Color is disabled on monochrome screens. Add working demos
to the Go `web.Config.Demos` list in `web/hello/cmd/demo/main.go`; the [browser guide](web/README.md#adding-a-gallery-entry)
explains how to build and register them.

Hello World has a **Bubble Tea / TinyTea** runtime selector. Switching restarts
the demo and keeps the display preset. Bubble Tea uses commands and timers;
TinyTea runs cooperatively with host-supplied ticks. Both share the same visual
model and help bindings. Go versus TinyGo is a separate compiler choice. The
browser bundle includes both runtimes, so it does not compare firmware sizes.

| Command | Purpose |
| :--- | :--- |
| `make ci` | Run all CI checks locally, generate previews, and build firmware and the WASM site. |
| `make web` | Build the static browser demo in `build/web/`. |
| `make web-tinygo` | Build the browser demo with TinyGo v0.43 and its matching runtime shim. |
| `make -C web test` | Check web modules, Go tests, vet, and the JavaScript key encoder. |
| `make web-serve` | Build and serve the interactive demo on port 8080. |
| `make check` | Check formatting, module consistency, local documentation links, race tests, vet, and host builds. |
| `make test` | Run race-enabled Go tests with coverage and vet. |
| `make firmware` | Compile device examples and core, widget, and layout checks. |
| `make hello` | Build the primary Bubble Tea Reverse TFT Feather firmware. |
| `make hello-tinytea` | Build the original cooperative Feather demo. |
| `make hello-preview` | Generate the hello PNG and animated GIF in `build/`. |
| `make snapshot` | Render the counter to `snapshot.png`. |

[CI](https://github.com/NimbleTerminal/TinyBubbles/actions/workflows/ci.yml)
runs on pushes, pull requests, and manual dispatch. It checks code and docs,
renders previews, and builds the firmware matrix and both Go and TinyGo WASM
sites. Artifacts are kept for 14 days. CI does not flash hardware or deploy the
site. `make ci` runs the same checks locally.

There are four Go modules. Root checks do not run the nested modules' tests:

| Module | Toolchain and role |
| :--- | :--- |
| Root | Go 1.23 language version; renderer, widgets, layout, and TinyTea; no Bubble Tea dependency. |
| `bubbletea/` | Go 1.26.0 and TinyGo 0.43; Bubble Tea v2.1 firmware with portability patches. |
| `web/` | Go 1.26.8; reusable canvas, LEDs, Booba actions, and scaffold tool. |
| `web/hello/` | Demo gallery consumer of the harness. TinyGo WASM uses a 128 KiB stack. |

Other TinyGo display projects can use the [browser harness](web/README.md) without
TinyBubbles cells or fonts. Presets come from Go; the tool generates the page
and build assets. Hidden tabs throttle timers to roughly once per second,
slowing every `time.Sleep`; keep previews visible when testing timing.

Documentation and web asset builds need Python 3. Web tests need Node.js 22 or
newer. See the [backend](bubbletea/README.md) and [browser](web/README.md) guides
for dependency details.

To refresh the README animation after a demo change:

```sh
make hello-preview
cp build/hello.gif docs/assets/hello.gif
```

## TODO

The next milestone is a **three-button settings menu on the Feather** with
scrolling options, editable values, and a confirmation dialog.

- [x] **Layout helpers:** cell regions, padding, row/column splits, alignment,
  and clipped buffer composition.
- [ ] **Button interactions:** optional hold-to-repeat, long-press, and release
  events, suitable for devices with only a few buttons.
- [ ] **TinyTea cooperative timers:** reusable ticks and delayed messages with cancellation,
  keeping commands short and avoiding application-specific scheduling boilerplate.
- [x] **Interactive browser simulator:** WASM hello demo with OLED/TFT presets,
  keyboard input, and HTML controls generated from the model's help map.
- [x] **Browser widget gallery:** Hello World, seven widgets, Settings, and the layout playground.
- [ ] **More widget demos:** add more composed screens.
- [ ] **Rendering benchmarks:** measure frame time, allocations, and display writes
  on the Feather; use the results to evaluate glyph caching and batched transfers.
- [ ] **OLED hardware validation:** verify the Pico SSD1306 example on a physical
  display, including monochrome styling and button navigation.
- [ ] **First release:** document remaining API limitations and publish versioned modules.
- [x] **Vanity imports:** use `nimbleterminal.dev/tinybubbles` for Go packages
  and `NimbleTerminal/TinyBubbles` for the GitHub repository.

## Open Collaboration

Contributions and feedback are welcome. Run checks for the modules you change;
use `make ci` for the full suite. Include your board,
display, TinyGo version, and reproduction steps when reporting hardware issues.

- [GitHub Issues](https://github.com/NimbleTerminal/TinyBubbles/issues)
- [GitHub Pull Requests](https://github.com/NimbleTerminal/TinyBubbles/pulls)

## Acknowledgements

Thanks to [Charm.sh](https://charm.sh) for sharing
[Bubble Tea](https://github.com/charmbracelet/bubbletea) and its approach to building UIs.
Thanks to the [TinyGo](https://tinygo.org) community for the compiler,
[drivers](https://github.com/tinygo-org/drivers), and
[TinyFont](https://github.com/tinygo-org/tinyfont).

Thanks to tinyterm, tinypod, and tinygui for their embedded display work, and
[Booba](https://github.com/NimbleMarkets/go-booba) for browser I/O and generated
HTML controls.

## License

This project is released under the **MIT License**, see [LICENSE](./LICENSE).
Third-party dependencies and font faces retain their respective licenses.

Copyright (c) 2026 [Neomantra Corp](https://www.neomantra.com).

----

Made with :heart: and :fire: by the team behind [NimbleTerminal](https://NimbleTerminal.com).
