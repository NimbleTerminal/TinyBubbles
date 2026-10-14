# Terminal target

Runs the Bubble Tea hello model with its 240×135 Feather display drawn inside a
terminal. The TinyBubbles renderer draws into an RGBA framebuffer, and
[ntpicture](https://github.com/NimbleTerminal/ntpicture) shows that image as
Kitty graphics where the terminal supports them, or as half-block cells elsewhere.
It is a development target: the same model and fonts as the Feather, no hardware.

The program builds natively and for WASI preview 1 with Go or TinyGo. The WASI
builds need Bubble Tea's wasip1 stubs, so this module replaces Bubble Tea with the
same fork commit that `web/go.mod` pins. ntpicture and go-booba's `termhost` are
not released yet, so the module replaces both with the checkouts next to this
repository.

The expected layout is sibling `TinyBubbles/`, `ntpicture/`, and `go-booba/`
directories (for example, under `~/projects`). The go-booba checkout must include
`actions/termhost`. A clean checkout and CI also need these sources; the local
replacements are not downloadable releases.
At release, replace `v0.0.0` with the published version and remove the local
`replace nimbleterminal.dev/ntpicture` directive.

This target no longer imports NTCharts. ntpicture renders the pixels directly;
pixterm remains a dependency of its comparison tests. ntcanvas provides terminal
cell grids and is not needed for this pixel framebuffer. The embedded renderer
and browser modules keep their existing dependencies.

## Build and run

Run from this directory. The WASI builds run under [wasmtime](https://wasmtime.dev).
TinyGo builds use the Boehm garbage collector (`-gc=boehm`).

```sh
make build          # build/terminal/terminal (native)
make build-wasip1   # build/terminal/terminal-go.wasm
make build-tinygo   # build/terminal/terminal-tinygo.wasm, TinyGo 0.43 with Go 1.26.8
./run.sh            # native
./run.sh wasip1
./run.sh tinygo
./run.sh wasip1 -glyph   # stay in half-block mode
./run.sh native -uncapped -stats  # measure maximum picture throughput
```

Enter pops a bubble, Right changes the palette, Esc pauses, k toggles between Kitty
graphics and half-blocks, and q or Ctrl-C quits. The same bindings appear as a row
of buttons under the picture that you can click with the mouse; More reveals Quit.
The script reads the terminal size with `stty size` because `tput` fails for `TERM` values without local terminfo, such
as `xterm-ghostty`. A terminal of at least 240×70 shows the display at one column per
pixel; smaller terminals get a 120×34 picture. `COLUMNS` and `LINES` override detection.

`make check` checks module tidiness, tests, vets, and builds the native and Go WASI targets. `make smoke`
also builds the TinyGo target and drives all three through a pseudo-terminal with
`smoke.py`, checking that each draws, handles keys, and quits in half-block and
forced Kitty modes. Kitty checks also require image transmission and deletion.

## Buttons from the help map

The program publishes its help map as Booba
[Terminal Actions](https://github.com/NimbleMarkets/go-booba/blob/main/docs/TERMINAL_ACTIONS.md),
the OSC 7501 convention that Booba's browser terminal turns into buttons, with
the same `bubblehelp` wrapper the web gallery uses. A plain terminal never answers
the capability query, so go-booba's `termhost` wrapper answers it itself, takes the
reports out of the output stream before they reach the terminal, and draws them as
the button row. Clicks become ordinary key presses through the program loop, which
is why the hello model needs no knowledge of the bar. Under Booba the wrapper steps
aside and the browser renders the buttons instead. The Kitty toggle's label and
enabled state follow the picture, so the buttons always say what k will do.

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

`-uncapped` removes the hello model's 100 ms animation wait for this terminal
target. Each completed picture advances the next animation step, keeping one
producer frame in flight. Animation therefore runs faster, rather than preserving
wall-clock movement speed. Esc still pauses and resumes it.

`-stats` reports framebuffer flushes and completed pictures separately.
Picture completion means the component finished preparing/outputting the frame,
not that a terminal painted it. Bubble Tea's text renderer is raised from its
default 60 Hz to its maximum 120 Hz in uncapped mode; it may skip intermediate
glyph pictures. Kitty image output uses raw protocol messages. A pseudo-terminal
benchmark measures the producer/output pipeline, not actual display refresh.

TinyBubbles' own rendering costs about 50 µs per frame. The hello model ticks every
100 ms by default, so 10 frames per second is the default ceiling. Half-block mode
fits the image at one by two pixels per cell and builds about 100 KB of styled
text per frame for 240×68 cells, in a few milliseconds on every target. Kitty mode
sends a small bitmap instead. Two details keep it cheap: the image is encoded at
display resolution rather than the cell rectangle's pixel size, and the picture
only receives a new image when the renderer flushed a frame, never in response
to the picture's own messages.

The picture runs with `StableID`, so every frame reuses one image ID, the
placeholder grid never changes, and Bubble Tea sends only the new bitmap, about
9 KB per frame. Without it each frame gets a fresh ID and the whole 240×68 grid
is resent, about 150 KB per frame.

Kitty mode has been exercised through a pseudo-terminal with the capability forced.
The probe round trip in a real Kitty-capable terminal is not covered by `make smoke`.
Quitting with q or Ctrl-C deletes the terminal's copies of the frames first.
