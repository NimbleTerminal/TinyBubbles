# Terminal target

Runs the Bubble Tea hello model with its 240×135 Feather display drawn inside a
terminal. The TinyBubbles renderer draws into an RGBA framebuffer, and
[ntcharts picture](https://github.com/NimbleMarkets/ntcharts) shows that image as
Kitty graphics where the terminal supports them, or as half-block cells elsewhere.
It is a development target: the same model and fonts as the Feather, no hardware.

The program builds natively and for WASI preview 1 with Go or TinyGo. The WASI
builds need Bubble Tea's wasip1 stubs, so this module replaces Bubble Tea with the
same fork commit that `web/go.mod` pins.

## Build and run

Run from this directory. The WASI builds run under [wasmtime](https://wasmtime.dev).

```sh
make build          # build/terminal/terminal (native)
make build-wasip1   # build/terminal/terminal-go.wasm
make build-tinygo   # build/terminal/terminal-tinygo.wasm, TinyGo 0.43 with Go 1.26.8
./run.sh            # native
./run.sh wasip1
./run.sh tinygo
./run.sh wasip1 -glyph   # stay in half-block mode
```

Enter pops a bubble, Right changes the palette, Esc pauses, k toggles between Kitty
graphics and half-blocks, and q or Ctrl-C quits. The script reads the terminal size
with `stty size` because `tput` fails for `TERM` values without local terminfo, such
as `xterm-ghostty`. A terminal of at least 240×70 shows the display at one column per
pixel; smaller terminals get a 120×34 picture. `COLUMNS` and `LINES` override detection.

`make check` tidies, vets, and builds the native and Go WASI targets. `make smoke`
also builds the TinyGo target and drives all three through a pseudo-terminal with
`smoke.py`, checking that each draws, handles keys, and quits.

## How the WASI builds differ

- **Input.** A stdin read blocks the whole single-threaded runtime under WASI, which
  starves Bubble Tea's timers, and wasmtime rejects non-blocking stdin. `stdin_wasi.go`
  polls readiness with `poll_oneoff` and sleeps between polls. It works unchanged in
  Go and TinyGo.
- **Terminal modes.** Bubble Tea cannot see the tty under WASI, so `run.sh` puts the
  input side in raw mode and leaves output post-processing on. Bubble Tea assumes a
  cooked output side there and emits bare line feeds; a full `stty raw` makes rows drift.
- **Colors.** Color profile detection finds no terminal and would strip every color,
  so the program forces true color.
- **Size.** Bubble Tea cannot query the window size, so the program passes it.

## Performance

TinyBubbles' own rendering costs about 50 µs per frame. The hello model ticks every
100 ms, so 10 frames per second is the ceiling on every target. Half-block mode
builds a 150 KB styled string per frame for 240×68 cells, which costs about 20 ms
natively, 55 ms in Go WASI, and 100 ms in TinyGo WASI, and Bubble Tea parses it
back each frame. Kitty mode sends a small bitmap instead and costs 1 to 2 ms, so it
is the default. Two details keep it cheap: the image is encoded at display
resolution rather than the cell rectangle's pixel size, and the picture only
receives a new image when the renderer flushed a frame, never in response to
picture's own messages.

Kitty mode has been exercised through a pseudo-terminal with the capability forced.
The probe round trip in a real Kitty-capable terminal is not covered by `make smoke`.
