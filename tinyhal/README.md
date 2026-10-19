# `tinyhal`: In-WASM Hardware Abstraction Layer for TinyGo

[Back to the README](../README.md) · [Runtimes](../RUNTIMES.md) · [Browser harness](../web/README.md)

`nimbleterminal.dev/tinybubbles/tinyhal` implements the host side of TinyGo's
`machine` peripheral calls inside Go/WASM (or standard Go host tests) so
embedded-style code using `machine.Pin`, `machine.SPI`, `machine.I2C`,
`machine.PWM`, `machine.ADC`, `machine.UART`, and `tinygo.org/x/drivers` can run
in a browser or WASI runtime while keeping the JavaScript layer minimal.

## How it works with TinyGo (`!baremetal`)

On non-MCU targets (`!baremetal`, including `-target=wasm` and `-target=wasip1`),
TinyGo's `src/machine/machine_generic.go` routes `machine` calls to 16 external
symbol declarations:

| Peripheral | Exported symbol declarations in `machine_generic.go` |
| --- | --- |
| GPIO | `__tinygo_gpio_configure`, `__tinygo_gpio_set`, `__tinygo_gpio_get` |
| PWM | `__tinygo_pwm_configure`, `__tinygo_pwm_channel_configure`, `__tinygo_pwm_channel_set` |
| ADC | `__tinygo_adc_read` |
| SPI | `__tinygo_spi_configure`, `__tinygo_spi_transfer`, `__tinygo_spi_tx` |
| I2C | `__tinygo_i2c_configure`, `__tinygo_i2c_set_baud_rate`, `__tinygo_i2c_transfer` |
| UART | `__tinygo_uart_configure`, `__tinygo_uart_read`, `__tinygo_uart_write` |

`tinyhal` defines those symbols in [`hooks_tinygo.go`](./hooks_tinygo.go)
(`//go:build tinygo && !baremetal`) and routes them to `tinyhal.Default()`.
Because LLVM resolves the symbols inside the WASM module at link time, pin
toggles and SPI/I2C transfers execute **entirely inside Go/WASM** without
crossing the JavaScript boundary per byte or pin change. JavaScript only
receives high-level visual events (such as LED updates or flushed display
framebuffers) and injects high-level user inputs (such as button presses).

## Package layout

| File | Build tags | Purpose |
| --- | --- | --- |
| [`hal.go`](./hal.go) | *(none)* | Core `HAL`, `Pin`, `PinMode`, GPIO/PWM/ADC/SPI/I2C/UART state, and `drivers.SPI` / `drivers.I2C` adapters |
| [`events.go`](./events.go) | *(none)* | High-level output `Event` (`pin`, `led`, `led_strip`, `button`, `framebuffer`, `uart`) and host `InputEvent` injection |
| [`devices.go`](./devices.go) | *(none)* | `SPIDevice`, `I2CDevice`, `LED`, `Button`, `SPIRegisterDevice`, and `I2CRegisterDevice` models |
| [`apa102.go`](./apa102.go) | *(none)* | Virtual `APA102` / DotStar SPI RGB LED strip model compatible with `tinygo.org/x/drivers/apa102` |
| [`st7789.go`](./st7789.go) | *(none)* | Virtual `ST7789` SPI + GPIO TFT display model (`CASET`, `RASET`, `RAMWR`, `MADCTL`, `COLMOD`) |
| [`ssd1306.go`](./ssd1306.go) | *(none)* | Virtual `SSD1306` I2C monochrome OLED display model |
| [`trace.go`](./trace.go) | *(none)* | `TraceRecorder` capturing GPIO, PWM, SPI, I2C, and UART transitions and exporting IEEE 1364 `.vcd` waveforms |
| [`hooks_tinygo.go`](./hooks_tinygo.go) | `tinygo && !baremetal` | `//export __tinygo_*` symbol implementations bridged to `tinyhal.Default()` |
| [`bridge_js.go`](./bridge_js.go) | `js && wasm` | Minimal browser JS host bridge (`BindJS`) |
| [`board_smoke_test.go`](./board_smoke_test.go) | *(none)* | Host smoke tests for the Pico OLED (`SSD1306` + `demo.New`), CLUE (`ST7789` + `demo.New`), and Feather Reverse TFT (`ST7789` + `hello.New`) board flows |
| [`workbench/`](./workbench) | `tinygo && !baremetal` / host | Standalone MCU workbench demo (GPIO, PWM LEDs, 12-bit ADC oscilloscope, SPI ST7789/registers, I2C SSD1306/sensor, UART) without TinyBubbles cell buffers or widgets |
| [`cmd/tinyhal-demo/`](./cmd/tinyhal-demo) | `tinygo && !baremetal` / host | Runnable example combining `machine`, `st7789`/`ssd1306`, `apa102`, `TraceRecorder`, `tinyhal`, and `tinytea` |

The core package has no `machine` or `syscall/js` imports in untagged files, so
`go test ./tinyhal/...` tests all bus logic, virtual peripherals, `.vcd` waveform
export, and board smoke setups with standard Go on the host.

## Minimal JavaScript host interface

When using [`tinyhal.BindJS(hal)`](./bridge_js.go) in a `js && wasm` build, the
JavaScript host needs only three hooks on `globalThis` / `window`:

1. **Receive high-level visual events (`window.tinyhalOnEvent`)**:
   ```js
   window.tinyhalOnEvent = (eventJSON, pixels) => {
     const ev = JSON.parse(eventJSON);
     switch (ev.kind) {
       case "led":
         // ev.id, ev.pin, ev.high, ev.brightness (0..255), ev.color ({R,G,B,A})
         document.getElementById(`led-${ev.id}`).style.opacity = ev.brightness / 255;
         break;
       case "framebuffer":
         // ev.id, ev.width, ev.height; pixels is a Uint8Array of RGBA8888 bytes
         const ctx = document.getElementById(ev.id).getContext("2d");
         const img = new ImageData(new Uint8ClampedArray(pixels.buffer), ev.width, ev.height);
         ctx.putImageData(img, 0, 0);
         break;
       case "button":
         // ev.id, ev.pin, ev.pressed
         break;
       case "uart":
         // ev.bus, ev.data (base64 in JSON)
         break;
     }
   };
   ```
2. **Inject button presses (`window.tinyhalSetButton`)**:
   ```js
   buttonElement.addEventListener("pointerdown", () => window.tinyhalSetButton("d0", true));
   buttonElement.addEventListener("pointerup", () => window.tinyhalSetButton("d0", false));
   ```
3. **Inject general host inputs (`window.tinyhalInjectInput`)**:
   ```js
   window.tinyhalInjectInput(JSON.stringify({ kind: "adc", pin: 26, value: 2048 }));
   window.tinyhalInjectInput(JSON.stringify({ kind: "pin", pin: 4, high: true }));
   ```

When paired with [`nimbleterminal.dev/tinybubbles/web`](../web/README.md), you
can also pass `web.NewCanvasDisplay(...)` directly as `ST7789Config.Target` (or
`SSD1306Config.Target`) and `web.NewLEDs(...)` as `LEDConfig.Writer` so
`tinyhal` paints directly to the shared TinyBubbles browser harness without
writing custom canvas code.

## Coexisting with `tinytea` and `bubbletea`

`tinyhal` is designed to work alongside `tinytea` and `bubbletea` in two ways:

1. **Rendering a `tinytea` or `bubbletea` UI through simulated hardware buses**:
   Pass a `tinygo.org/x/drivers/st7789.Device` (backed by `machine.SPI0`) or
   `ssd1306.Device` (backed by `machine.I2C0` or `hal.I2C(0)`) as the
   `drivers.Displayer` to `tinytea.NewProgram` or `displaytea.NewProgram`. Cell
   diffs flow through the real driver over SPI/I2C into `tinyhal.ST7789` or
   `tinyhal.SSD1306`.
2. **Inspecting HAL state from a companion status/debug model**:
   A `tinytea` or `bubbletea` model running in the same WASM module can call
   `hal.PinState(pin)`, `led.On()`, `led.Brightness()`, `spiDev.Register(reg)`,
   or `hal.DrainEvents()` to render a live pin/bus inspector panel.

## Build and run

Run the unit and integration tests with standard Go:

```sh
go test -race -cover ./tinyhal/...
```

Run the example program under TinyGo WASI (`-target=wasip1`), which compiles
`machine.Pin`, `machine.SPI0`, `st7789`, `tinytea`, and `tinyhal` into a single
WASM module and executes it:

```sh
tinygo run -target=wasip1 ./tinyhal/cmd/tinyhal-demo
```

Compile the same example to a browser WASM binary (`-target=wasm`):

```sh
mkdir -p build
tinygo build -target=wasm -o build/tinyhal-demo.wasm ./tinyhal/cmd/tinyhal-demo
```

Or run the host simulation fallback with standard Go:

```sh
go run ./tinyhal/cmd/tinyhal-demo
```

## How `tinyhal` differs from the [TinyGo Playground](https://tinygo.org/playground/)

Both let TinyGo `machine`-peripheral code run in a browser, but with nearly
opposite architectures:

| | [TinyGo Playground](https://tinygo.org/playground/) | `tinyhal` |
| --- | --- | --- |
| What it is | Hosted web app (code editor + circuit simulator) | Go library you compile into your own app or page |
| Compiling | Server-side cloud compile to WASM (board tags, e.g. `-tags=circuitplay_express`) | Local `tinygo build -target=wasm` (or `-target=wasip1`); no server |
| Peripheral emulation | JavaScript "simulator core" running in a web worker | Go device models (`ST7789`, `SSD1306`, `APA102`, `LED`, `Button`, ...) linked into the same WASM module |
| `__tinygo_*` symbol implementations | Exported from JavaScript into the WASM module | Defined in `hooks_tinygo.go`, resolved by LLVM inside the module at link time |
| WASM ↔ JS traffic | Per-pin toggle / per-byte SPI-I2C transfer crosses the host boundary | JS receives only batched high-level events (LED state/brightness/color, flushed framebuffers, UART data) and injects high-level inputs |
| Hardware wiring | Interactive drag-and-drop SVG breadboard | Devices configured in Go code (`tinyhal.Default()`, config structs) |
| Headless / WASI | Browser-only product | Runs under `wasip1`; host-simulatable with standard `go test` (untagged core has no `machine` or `syscall/js` imports) |
| Extras | Power-consumption estimates, download binary for physical boards | `TraceRecorder` exporting IEEE 1364 `.vcd` waveforms; `tinytea`/`bubbletea` UIs rendered through real display drivers over the simulated buses |

In short: [TinyGo Playground](https://tinygo.org/playground/) emulates the *hardware in JavaScript* around a
cloud-compiled program, while `tinyhal` emulates the *hardware in Go inside
the WASM module itself*, keeping the JavaScript layer to three minimal hooks
(`window.tinyhalOnEvent`, `window.tinyhalSetButton`,
`window.tinyhalInjectInput`).
