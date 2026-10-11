# TinyBubbles Device Guide

[Back to the README](./README.md) · [API and memory guide](./API.md)

The Reverse TFT Feather hello demo has been confirmed working on hardware.
The Pico, Pico 2, and CLUE examples have been compiled with TinyGo v0.43.0;
their physical display and button operation has not yet been verified.

For the Feather, see the [hello demo and flashing instructions](./devices/feather-reverse-tft/README.md).

## Devices and wiring

Device-specific imports stay under `devices/<name>/main.go`, following the
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

`make firmware` builds all device examples plus an ESP32-S3 core compile check.
For the Feather, `make hello` builds its image; follow its device guide to enter
ROM download mode and choose the correct serial port before flashing.
