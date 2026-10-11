# TinyBubbles - TinyGo Display UIs

<p>
    <a href="https://github.com/NimbleTerminal/TinyBubbles/actions/workflows/ci.yml"><img src="https://github.com/NimbleTerminal/TinyBubbles/actions/workflows/ci.yml/badge.svg" alt="Build Status"></a>
    <a href="https://pkg.go.dev/nimbleterminal.dev/tinybubbles"><img src="https://pkg.go.dev/badge/nimbleterminal.dev/tinybubbles.svg" alt="Go Reference"></a>
    <a href="https://tinygo.org"><img src="https://img.shields.io/badge/TinyGo-0.43%2B-blue" alt="TinyGo 0.43 or newer"></a>
    <a href="./LICENSE"><img src="https://img.shields.io/badge/License-MIT-blue.svg" alt="MIT License"></a>
</p>

`TinyBubbles` brings the [Bubble Tea](https://github.com/charmbracelet/bubbletea)
v2.1 runtime to [TinyGo](https://tinygo.org) embedded displays, with cell rendering,
custom fonts, and a focus on small-screen widgets. **Bubble Tea is the primary
runtime; TinyTea is the optional lightweight replacement.**

Small screens, big possibilities!

<p align="center">
    <a href="./devices/feather-reverse-tft/README.md"><b>Hello, Tiny World</b></a>
    (<a href="./demo/hello/model.go">source</a>)<br>
    <a href="./devices/feather-reverse-tft/README.md"><img src="docs/assets/hello.gif" alt="Hello, Tiny World: floating bubbles, color changes, and an enlarged pop counter" width="600"></a>
</p>

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
| [Cell buffer](cell.go) | Reusable character grids with UTF-8 text and clipped writes. |
| [Renderer](API.md#displayer-and-rendering) | Changed-cell drawing and optional FillRectangle acceleration. |
| [Fonts and themes](API.md#fonts-and-themes) | TinyFont, custom glyphs, palettes, and monochrome/color defaults. |
| [Devices](DEVICES.md) | Hardware initialization, buttons, buses, and power controls. |

## Quickstart

Use **TinyGo 0.43.0 and Go 1.26** for the Bubble Tea backend. Its Makefile selects
the Go toolchain and prepares the required bare-metal portability patches.
The backend is the primary development path, but remains an early port with
[documented limitations](bubbletea/README.md#portability-changes).

```sh
make hello          # Bubble Tea Feather firmware: build/bubbletea-feather.bin
make hello-preview  # desktop PNG/GIF of the shared greeting and fonts
```

See the [Bubble Tea backend guide](bubbletea/README.md) for build and flash
instructions. The top-left header on that firmware reads **BUBBLE TEA 2.1**.
The [device guide](DEVICES.md) covers wiring and adding a board.

## Usage

The canonical Go import path is `nimbleterminal.dev/tinybubbles`; replace the old
`github.com/NimbleTerminal/TinyBubbles` prefix in existing imports. Source and
issues remain on GitHub.

Use Bubble Tea's model interface and the display backend from the `bubbletea`
module (a separate module keeps these dependencies out of TinyTea):

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
| Counter | [demo](./demo/model.go) | One model adapting to a monochrome OLED or color TFT. |
| Hello preview | [examples/hello-preview](./examples/hello-preview) | The hello demo rendered to PNG and an animated GIF with ordinary Go. |
| Counter snapshot | [examples/snapshot](./examples/snapshot) | A desktop Displayer implementation that writes a PNG. |

## Device Support

| Device | Display | TinyGo target | Validation |
| :--- | :--- | :--- | :--- |
| Adafruit Reverse TFT Feather | ST7789, 240×135 | [Local target](./devices/feather-reverse-tft/target.json) | Demo confirmed on hardware |
| Raspberry Pi Pico | SSD1306, 128×64 | `pico` | Compiled |
| Raspberry Pi Pico 2 | SSD1306, 128×64 | `pico2` | Compiled |
| Adafruit CLUE | ST7789, 240×240 | `clue` | Compiled |

The core also has an ESP32-S3 compile check. Other boards can use any initialized
`drivers.Displayer`; concrete peripherals stay under `devices/<name>/main.go`.
See [adding a device](./DEVICES.md#devices-and-wiring).

The default font uses 6×8 pixel cells. Two cell grids take about **2.6 KiB** on a
128×64 OLED, **10 KiB** on a 240×135 TFT, or **24.8 KiB** on a 320×240 display.
TinyBubbles allocates no pixel framebuffer; drivers may maintain their own.
See [memory and resolution](./API.md#memory-and-resolution) for the full budget.

## Development

### Interactive browser demo

Run `make web-serve`, then open <http://127.0.0.1:8080>. The shared hello model
runs as WASM with OLED and TFT presets. Booba builds HTML controls from its
Bubbles help map: Pause becomes Resume, and Color is disabled for monochrome.
The simulated screen retains the custom fonts and enlarged counter, without
on-screen help text. See [the browser guide](./web/README.md) for architecture,
dependencies, and static hosting.

| Command | Purpose |
| :--- | :--- |
| `make ci` | Run all CI checks locally, generate previews, and build firmware and the WASM site. |
| `make web` | Build the static browser demo in `build/web/`. |
| `make web-serve` | Build and serve the interactive demo on port 8080. |
| `make check` | Check formatting, module consistency, local documentation links, race tests, vet, and host builds. |
| `make test` | Run race-enabled Go tests with coverage and vet. |
| `make firmware` | Compile all device examples and the ESP32-S3 core check. |
| `make hello` | Build the primary Bubble Tea Reverse TFT Feather firmware. |
| `make hello-tinytea` | Build the original cooperative Feather demo. |
| `make hello-preview` | Generate the hello PNG and animated GIF in `build/`. |
| `make snapshot` | Render the counter to `snapshot.png`. |

[GitHub Actions](https://github.com/NimbleTerminal/TinyBubbles/actions/workflows/ci.yml)
runs on every push and pull request, and supports manual dispatch. The host job
runs `make check` and generates both desktop previews. The firmware job builds
Pico, Pico 2, CLUE, the ESP32-S3 core check, and the Reverse TFT Feather with
**TinyGo v0.43.0**. Successful jobs upload firmware and preview artifacts for
14 days. CI validates builds; it does not flash connected hardware or publish
releases. `make check` also requires Python 3 for local documentation checks.
The web job uses Go 1.26.8 and Node.js 22, tests the action bridge, and uploads
the static `web-demo` artifact. It does not automatically deploy a public site.
A separate job builds `web-demo-tinygo` with TinyGo v0.43 and the matching runtime shim.

The core module declares Go 1.23 language compatibility and pins drivers v0.36.0 and
TinyFont v0.7.0. The core uses the TEA architecture without depending on
Bubble Tea or accepting ANSI views. The primary Bubble Tea backend has its
own module, host tests, and Feather firmware CI job. To port a terminal model, use buffer writes and
palette styles in `View`; see the [rendering guide](./API.md#displayer-and-rendering).

To refresh the README animation after a demo change:

```sh
make hello-preview
cp build/hello.gif docs/assets/hello.gif
```

## TODO

The next milestone is a **three-button settings menu on the Feather** with
scrolling options, editable values, and a confirmation dialog.

- [ ] **Widgets and layout:** scrolling menu, progress bar, label, and dialog;
  add buffer composition and alignment helpers.
- [ ] **Button interactions:** optional hold-to-repeat, long-press, and release
  events, suitable for devices with only a few buttons.
- [ ] **TinyTea cooperative timers:** reusable ticks and delayed messages with cancellation,
  keeping commands short and avoiding application-specific scheduling boilerplate.
- [x] **Interactive browser simulator:** WASM hello demo with OLED/TFT presets,
  keyboard input, and HTML controls generated from the model's help map.
- [ ] **Browser widget gallery:** add widget demos as the embedded widgets land.
- [ ] **Rendering benchmarks:** measure frame time, allocations, and display writes
  on the Feather; use the results to evaluate glyph caching and batched transfers.
- [ ] **OLED hardware validation:** verify the Pico SSD1306 example on a physical
  display, including monochrome styling and button navigation.
- [ ] **First release:** validate CI on the new repository, document any remaining
  API limitations, and publish an initial version.
- [x] **Canonical repository:** align module imports, links, and the remote with
  `NimbleTerminal/TinyBubbles`.

## Open Collaboration

We welcome contributions and feedback! Run `make check` before submitting changes,
and `make firmware` when changing the core or device setup. Include your board,
display, TinyGo version, and reproduction steps when reporting hardware issues.

- [GitHub Issues](https://github.com/NimbleTerminal/TinyBubbles/issues)
- [GitHub Pull Requests](https://github.com/NimbleTerminal/TinyBubbles/pulls)

## Acknowledgements

Thanks to [Charm.sh](https://charm.sh) for sharing
[Bubble Tea](https://github.com/charmbracelet/bubbletea) and its approach to building UIs.
Thanks to the [TinyGo](https://tinygo.org) community for the compiler,
[drivers](https://github.com/tinygo-org/drivers), and
[TinyFont](https://github.com/tinygo-org/tinyfont).

Thanks also to tinyterm, tinypod, and tinygui for exploring embedded display
rendering, device configuration, and UI design. TinyBubbles brings those concerns
together around a reusable Model / Update / View framework.

## License

This project is released under the **MIT License**, see [LICENSE](./LICENSE).
Third-party dependencies and font faces retain their respective licenses.

Copyright (c) 2026 [Neomantra Corp](https://www.neomantra.com).

----

Made with :heart: and :fire: by the team behind [NimbleTerminal](https://NimbleTerminal.com).
