# Bubble Tea v2.1 display backend

This primary backend module runs the **actual `charm.land/bubbletea/v2` v2.1.0 Program**
with TinyBubbles' Displayer renderer. Bubble Tea owns the event loop, commands,
goroutines, timers, and message delivery. The rendering core and optional [TinyTea replacement](../RUNTIMES.md) remain
independent of Bubble Tea dependencies.

The adapter wraps a `tea.Model`, disables terminal I/O, and renders during the
wrapped model's `View` call. Bubble Tea's renderer interface is private, so this
uses `WithoutRenderer()` and public model/program APIs.

- Plain `tea.View.Content` becomes a clipped character grid. ANSI escapes,
  terminal cursor modes, mouse regions, and terminal graphics are unsupported.
- An optional `CellView() *tinybubbles.Buffer` supplies styled cells directly.
  This preserves custom fonts, large glyph tiles, and the enlarged hello counter.
- GPIO code sends `tea.KeyPressMsg` through `Program.Send`. Custom messages work
  the same way. The greeting uses real `tea.Tick` commands for animation.
- Display errors cancel the program and are returned by `Run`.

## Build and test

Use **TinyGo 0.43.0 and Go 1.26.x**. Go 1.27 currently fails to compile the
Ultraviolet dependency's `hash/maphash` import with TinyGo 0.43. This module needs
Go 1.26 because Bubble Tea v2.1 requires it; the root module remains Go 1.23.

From this directory:

```sh
make prepare       # vendor pinned dependencies and apply bare-metal changes
make test          # actual Bubble Tea loop, commands, timer, and display tests
make firmware      # build/bubbletea-feather.bin in the repository root
```

To flash the Reverse TFT Feather from this directory after identifying its port:

```sh
GOROOT="$(GOTOOLCHAIN=go1.26.0 go env GOROOT)" tinygo flash \
  -target=../devices/feather-reverse-tft/target.json \
  -port=<current-serial-port> ./devices/feather
```

The Makefile selects `GOTOOLCHAIN=go1.26.0` and passes its `GOROOT` to TinyGo.
Go downloads that toolchain if needed. Override `GOTOOLCHAIN` to select another
Go 1.26 patch release, or `TINYGO` to select the compiler binary.

Firmware uses the same Reverse TFT Feather configuration and controls as the
[working demo](../devices/feather-reverse-tft/README.md): D0 pop, D1 color,
D2 pause. The top-left header reads **BUBBLE TEA 2.1**, identifying this runtime
on the physical screen. It retains the existing fonts and large counter.
Building does not flash.

## Portability changes

Unmodified upstream Bubble Tea v2.1 does **not** currently compile for this
bare-metal target. `prepare.py` patches a disposable, ignored vendor directory;
module caches and the user's Bubble Tea checkout are untouched. Run `make prepare`
again after editing local dependencies. Third-party license files stay in vendor.

The patch selects non-terminal fallbacks for Bubble Tea, Ultraviolet, x/term,
and cancelreader when TinyGo supplies the `baremetal` build tag. It also:

1. Supplies no-op terminal initialization, resize watching, and suspension.
2. Substitutes the no-op renderer for the terminal renderer on bare metal.
3. Makes terminal input initialization return an explicit unsupported error;
   embedded applications must use `WithInput(nil)` and `Send`.

The adapter sets these options automatically. Host builds retain upstream
terminal implementations. `tea.Program.Run`, its event loop, command handling,
and the Model/Cmd/Msg contracts are unchanged. This is an experimental port,
not a claim that all Bubble Tea APIs or terminal libraries work on microcontrollers.
Process execution, OS signals, and terminal-only commands are outside its scope.

## Resource expectations

Measured with TinyGo 0.43 / Go 1.26 on ESP32-S3:

| Firmware | Flash bytes | Static RAM bytes |
| --- | ---: | ---: |
| Headless Bubble Tea probe | 242,285 | 88,192 |
| Feather greeting with ST7789 and custom fonts | 265,718 | 93,464 |

These figures exclude dynamic heap use and task stacks and are not a minimum RAM
guarantee. Keeping the terminal renderer and
reader reachable exceeded internal RAM by approximately 1.9 MiB.

At 40x16 with 6x8 cells, the adapter fallback grid and renderer history cost
roughly 10 KiB. A model supplying its own cell grid adds another 5 KiB, alongside
driver buffers, fonts, and the Bubble Tea runtime. ESP32-S3 is the current compile target;
small nRF52/RP2040 targets need separate measurements. The Feather firmware was flashed with MD5 verification and emitted its startup
message over USB serial. Screen/button behavior and long-running heap use still
need hardware validation before treating this as production-ready.
