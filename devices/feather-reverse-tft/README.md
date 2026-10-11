# Hello, Tiny World — Reverse TFT Feather

For the primary **Bubble Tea** firmware, use the
[Bubble Tea backend guide](../../bubbletea/README.md). This page describes the
original cooperative demo; build it with `make hello-tinytea`.

For the **Adafruit ESP32-S3 Reverse TFT Feather, product 5691**. Uses the built-in
240x135 ST7789 and three buttons; no extra wiring, network, or PSRAM is needed.
Requires TinyGo **v0.43.0 or newer**.

The screen has a large greeting, rising bubble sprites, four color palettes,
a pop counter, and pause/resume. The custom font draws doubled text and bubble
tiles through the ordinary cell renderer. Animation is a custom `TickMsg` every
100 ms, scheduled by the device input loop without blocking commands.

| Button | Action |
| --- | --- |
| D0 | Pop the next bubble; its sparkle remains until animation advances |
| D1 | Cycle lagoon, blossom, sunshine, and mint colors |
| D2 | Pause / resume motion; other controls remain usable |

## Build and preview

From the repository root:

```sh
make hello-tinytea                       # build/feather-reverse-tft.bin
make hello-preview               # build/hello.png and build/hello.gif
```

The preview renders the same model/font, enlarged 3× with nearest-neighbor
scaling. The GIF demonstrates popping, palette changes, and pausing. It does not
simulate SPI transfer time or prove hardware behavior.

## Flash

**Flashing installs TinyGo in place of the existing firmware/bootloader.** Copy
any files you want to retain from `CIRCUITPY` before flashing. TinyGo's ESP32 image
is not a CircuitPython UF2 application and must not be copied to a mounted drive.
The Adafruit factory-reset procedure can restore the original firmware later.

1. Connect the board with a USB data cable.
2. Hold **D0/BOOT**, press and release **RESET**, then release D0. This enters the
   chip's ROM download mode. Use this sequence for the first flash from CircuitPython
   or TinyUF2 and whenever automatic reset fails.
3. Identify its serial port, such as `/dev/cu.usbmodem…` on macOS, then run:

```sh
tinygo flash \
  -target=./devices/feather-reverse-tft/target.json \
  -port=/dev/cu.usbmodemYOUR_PORT \
  ./devices/feather-reverse-tft
```

4. Tap RESET if the program does not start automatically. Release D0 before
   reset so the chip starts the demo instead of re-entering download mode.

This compiles and uploads using TinyGo's built-in ESP32 flasher. The local target
inherits `esp32s3-generic`, with a dedicated build tag; all physical pins are
configured in the shared [board package](board/board.go). Use this target JSON, not `xiao-esp32s3` or a target
for the original ESP32. The 4 MB flash is ample; this demo uses internal SRAM
and does not initialize the board's external PSRAM.

## Board details

| Signal | ESP32-S3 GPIO |
| --- | --- |
| TFT / STEMMA power enable | 7, high |
| SPI clock / MOSI | 36 / 35 |
| TFT CS / DC / reset | 42 / 40 / 41 |
| Backlight | 45, high |
| D0 | 0, active-low, pull-up |
| D1 / D2 | 1 / 2, active-high, pull-down |

The native panel is 135x240. TinyGo's `Rotation90`, `RowOffset: 40`, and
`ColumnOffset: 53` produce the visible 240x135 landscape window. BGR is selected
before `Configure`; the gamma tables match Adafruit's initialization. SPI runs
at a conservative 20 MHz. The hello demo has been confirmed working on a physical
Reverse TFT Feather. The enlarged counter update was also flashed and verified
by the flasher; the desktop preview shows its updated layout.

The font uses 6x8 cells: 40x16 cells, with seven spare vertical pixels. Reusable
model and renderer grids occupy about 10 KiB combined; sprites need no separate
pixel framebuffer. TinyGo v0.43.0 reports 34,218 bytes flash and 13,776 bytes static
RAM for the initial build, excluding the grids, driver buffer, and other live
heap allocations. These sizes can change with compiler settings.

Sources:

- [Adafruit product 5691](https://www.adafruit.com/product/5691)
- [Pinouts and button polarity](https://learn.adafruit.com/esp32-s3-reverse-tft-feather/pinouts)
- [Adafruit pin assignments](https://github.com/adafruit/circuitpython/blob/main/ports/espressif/boards/adafruit_feather_esp32s3_reverse_tft/pins.c)
- [Adafruit display initialization](https://github.com/adafruit/circuitpython/blob/main/ports/espressif/boards/adafruit_feather_esp32s3_reverse_tft/board.c)
- [Factory reset / recovery](https://learn.adafruit.com/esp32-s3-reverse-tft-feather/factory-reset)
