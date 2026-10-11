# Changelog

This file lists what changed in each TinyBubbles release, newest first. It
follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and
[Semantic Versioning](https://semver.org/spec/v2.0.0.html). Before 1.0, a minor
release may change public APIs; the notes will say so.

## Unreleased

## v0.1.0 — 2026-10-11

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

### Rendering

- **Cell buffers.** Text lives in a grid of character cells, each holding one
  rune and a style ID. Buffers are reused between frames. Writes are clipped,
  so oversized content never corrupts memory.
- **Changed-cell renderer.** Only cells that changed are redrawn, and the
  display is flushed only when something changed. A failed flush is retried
  on the next frame. Drivers that offer `FillRectangle` get faster background
  fills; others fall back to pixel writes.
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
  indicator when the total is unknown. The parent owns the timer.
- **Status indicator.** Offline, Ready, Busy, Warning, and Error with distinct
  ASCII symbols, so the state is clear without color.
- **Text viewport.** Scrolls and pages through plain text. It wraps at cell
  boundaries and allocates only the visible grid.
- **Settings example.** Combines the list, value, choice, and dialog widgets
  and shows how a parent routes focus and consumes child results.

### Layout

- **Layout helpers.** Split a grid into rows and columns, add padding, align a
  child, and copy child buffers into a parent. They allocate nothing and work
  with either runtime. See the [layout guide](layout/README.md).

### Browser

- **Demo gallery.** `make web-serve` runs Hello World, every widget, the
  Settings screen, and a layout playground in a browser with OLED and TFT
  presets. Hello World can switch between Bubble Tea and TinyTea. Links keep
  the chosen demo and preset.
- **Reusable browser harness.** The `web` module gives any TinyGo display
  project a canvas display, simulated LEDs, keyboard input, and HTML buttons
  generated from the model's Bubbles help map. Disabled actions show as
  disabled buttons. `go tool tinybubbles-web` scaffolds a new project. See the
  [browser guide](web/README.md).

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

### Reliability and shared helpers

- Cooperative firmware retries transient display failures through paced Step
  loops. Both runtimes document Run as fail-fast and identify display errors.
- Browser scaffolds keep harness JavaScript and support assets out of consumer
  source trees. Builds override only consumer HTML and CSS.
- Scaffold refreshes preserve existing go.mod files, including with --force.
- Bubble Tea adapter tests refresh vendored dependencies before running.

- Consolidated widget label clipping, buffer resizing, and result message types.
  One-cell labels retain their first character; centered labels ignore controls.
- Shared Feather board initialization between runtimes and button polling
  between Pico and CLUE.

- Updated the browser harness, demo, and scaffold to go-booba v0.9.1.

- Preserved viewport position across collapsed layouts and kept bottom-pinned
  viewports at the end after reflow.
- Allocated fallback text grids only for plain-text models and removed redundant
  background fills during full repaints.
- Pruned obsolete generated browser assets during forced refreshes.
- Kept narrow indeterminate progress bars animated and made nil buffers behave
  consistently as empty grids.
