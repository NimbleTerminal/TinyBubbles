# TinyBubbles Device Guide

[Back to the README](./README.md) · [API and memory guide](./API.md)

The Reverse TFT Feather hello demo has been confirmed working on hardware.
The Pico, Pico 2, and CLUE examples have been compiled with TinyGo v0.43.0;
their physical display and button operation has not yet been verified.

For the Feather, see the [hello demo and flashing instructions](./devices/feather-reverse-tft/README.md).

The primary Bubble Tea Feather entry point lives in
[bubbletea/devices/feather](bubbletea/devices/feather). The root device examples
use the original cooperative TinyTea scheduler. See [runtime differences](RUNTIMES.md).

## Devices and wiring

Device-specific imports stay under `devices/<name>/`, following the
Config + device-entry-point separation used by tinypod. Build tags keep `machine`
out of normal host builds.

| Example | TinyGo target | Display | Input |
| --- | --- | --- | --- |
| `devices/pico-oled` | `pico`, `pico2` | External SSD1306, 128x64, I2C address `0x3c` | GP14 / GP15 buttons |
| `devices/clue` | `clue` | Built-in ST7789, 240x240, SPI1 | Built-in A / B buttons |
| `devices/feather-reverse-tft` | Local `target.json` | Built-in ST7789, 240x135, SPI0 | Built-in D0 / D1 / D2; animated hello demo |

For Pico/Pico 2, connect SSD1306 VCC to **3V3**, GND to GND, SDA to **GP4**, SCL
to **GP5**. Use a 3.3 V-compatible module with I2C pull-ups (add them if the board
lacks them). Connect each normally-open button between **GP14/GP15 and GND**;
internal pull-ups are enabled. Pins are GPIO names, not physical header numbers.
The SSD1306 example uses 400 kHz I2C. Change the address if your panel uses `0x3d`.
The CLUE needs no external wiring and uses 8 MHz SPI with its board pin constants.
Its 270-degree driver rotation and 80-pixel offset match the panel's native
orientation in [Adafruit's board initialization](https://github.com/adafruit/circuitpython/blob/main/ports/nordic/boards/clue_nrf52840_express/board.c).

The examples use 25 ms press/release debounce and no auto-repeat. `Button.Update`
takes a logical pressed state and a timestamp, so the same mapper works with
GPIO, an I2C expander, or a gamepad. Input returns one message per poll; queue
multiple events if your hardware produces them together. The examples alternate
button polling priority to preserve simultaneous presses. Standard keys are
Up, Down, Left, Right, Select, Back, and Quit; custom messages need no registration.

To add a board:

1. Create `devices/<name>/main.go` with appropriate TinyGo/board build tags.
2. Initialize I2C/SPI, configure the concrete display's resolution, rotation,
   offsets and backlight, then initialize your buttons/peripherals.
3. Pass the ready display pointer, `Monochrome`, optional font/theme, and a
   nonblocking input function in `bubbles.Config` to `NewProgram`.
4. Call `Run`, or integrate `Step` with your existing main loop. Build with
   `tinygo build -target=<board> ./devices/<name>` and add it to the firmware checks.

Panel offsets are a driver/device concern: a 240x135 ST7789 module may need
different offsets from the CLUE's 240x240 panel. Never copy offsets between
unrelated panels without checking their specifications.

## Build and flash

Run commands from the repository root with TinyGo v0.43.0 or newer:

```sh
make firmware
tinygo flash -target=pico ./devices/pico-oled
tinygo flash -target=pico2 ./devices/pico-oled
tinygo flash -target=clue ./devices/clue
```

`make firmware` builds the cooperative device examples and ESP32-S3 compile checks.
For the primary Bubble Tea Feather image, use `make hello` and follow the
[backend flashing guide](bubbletea/README.md#build-and-test).
`make hello-tinytea` builds the original cooperative Feather image.

The two Feather runtimes share [the board package](devices/feather-reverse-tft/board/board.go)
for power, SPI, panel configuration, and button polarity. Each main keeps its
runtime loop and control mapping. Pico and CLUE use the shared `ButtonPoller`
with device-local pin reads.

## Terminal target

[`devices/terminal`](devices/terminal/README.md) draws the hello display inside a
terminal instead of on a panel. It uses the same model, fonts, and renderer as the
Feather, with ntcharts picture showing the framebuffer as Kitty graphics or
half-blocks. It builds natively and for WASI preview 1 with Go or TinyGo:

```sh
make -C devices/terminal build build-wasip1 build-tinygo
devices/terminal/run.sh            # native
devices/terminal/run.sh wasip1     # Go WASI under wasmtime
devices/terminal/run.sh tinygo     # TinyGo WASI under wasmtime
```
