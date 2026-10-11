# TinyBubbles API Guide

[Back to the README](./README.md) · [Device guide](./DEVICES.md)

This guide covers shared rendering and the original cooperative runtime API.
Start with the [Bubble Tea backend](bubbletea/README.md) for new applications;
see [TinyTea and runtime differences](RUNTIMES.md) for the lightweight replacement.

## Model, commands, and a ready display

```go
import (
    "strconv"
    bubbles "github.com/NimbleTerminal/TinyBubbles"
)

type counter struct {
    value int
    grid  *bubbles.Buffer
}

func (m *counter) Init() bubbles.Cmd { return nil }

func (m *counter) Update(msg bubbles.Msg) (bubbles.Model, bubbles.Cmd) {
    switch msg := msg.(type) {
    case bubbles.WindowSizeMsg:
        m.grid = bubbles.NewBuffer(msg.Width, msg.Height)
    case bubbles.KeyMsg:
        switch msg.Key {
        case bubbles.KeySelect:
            m.value++
        case bubbles.KeyQuit:
            return m, bubbles.Quit
        }
    }
    return m, nil
}

func (m *counter) View() *bubbles.Buffer {
    if m.grid == nil { return nil }
    m.grid.Clear(bubbles.Normal)
    m.grid.Text(0, 0, "Count: "+strconv.Itoa(m.value), bubbles.Accent)
    return m.grid
}
```

In a device entry point, initialize the bus,
display and buttons, then pass them into the framework:

```go
p, err := bubbles.NewProgram(&counter{}, bubbles.Config{
    Display:    display, // an initialized drivers.Displayer
    Monochrome: true,    // false for a color display
    Input:      poll,    // func() bubbles.Msg; return nil when idle
})
if err != nil { panic(err) }
if err := p.Run(); err != nil { panic(err) }
```

The complete shared model is in [`demo/model.go`](demo/model.go). Left/A
decrements; Right/B increments. It also handles Up/Down, Select, Back to reset,
and Quit. Compact screens show a short layout; wider/taller screens have a
centered layout and extra help. The Pico and CLUE examples use this same model;
the Reverse TFT Feather runs the separate `demo/hello` playground.

`Model` has `Init() Cmd`, `Update(Msg) (Model, Cmd)`, and `View() *Buffer`.
A `Cmd` is `func() Msg`; any Go value can be a custom message. `nil` means no work.
For example, a short sensor read can return a `temperatureMsg` to `Update`.

The lifecycle is deliberately cooperative:

1. `Start` calls `Init` once, queues its command, delivers `WindowSizeMsg`, and
   renders the first view. Size contains both cell and pixel dimensions.
2. `Step` polls input once, executes at most one FIFO command, delivers its
   result to `Update`, then renders. Commands returned by `Update` join the queue.
3. `Run` repeatedly calls `Step` and sleeps for `Config.PollInterval` (default
   10 ms). Use `Step` in your own loop for custom peripheral/timer scheduling.

`Send(msg)` synchronously updates and renders, initializing the program if
necessary. It queues returned commands; call `Step` or `Run` to execute them.
`QuitMsg` stops the program and releases queued commands. `KeyQuit` is an ordinary
key delivered to your model, which can return `bubbles.Quit` or ask for confirmation.

Unlike BubbleTea's concurrent command execution, TinyBubbles does not create
goroutines or channels. Commands must be short and must not wait for a button,
network response, or timer. Advance long operations in your device loop and feed
completion messages through `Send` or `Input`. A fixed 16-command queue bounds RAM;
overflow returns `ErrCommandQueueFull` rather than silently growing or dropping
work. The triggering message has already updated the model when that error is
returned, so treat it as backpressure or an application scheduling error.

All program methods belong to one main loop. Do not call `Send` from an interrupt
or a second goroutine. Transfer hardware events through a device-owned safe queue
and consume them from `Input`. Renderer/bus errors propagate to the caller; a
later `Step` retries the full frame. `Run` returns on errors so the application
can choose whether to retry, report, or stop.

## Displayer and rendering

The existing drivers interface is used directly:

```go
type Displayer interface {
    Size() (width, height int16)
    SetPixel(x, y int16, c color.RGBA)
    Display() error
}
```

`Size` is the visible pixel area after orientation is configured. `SetPixel`
accepts RGBA colors; use opaque colors for display output. Drivers may buffer writes or send them directly;
`Display` flushes a buffered driver and can be a no-op on a direct-write driver.
The renderer calls it once per changed frame, and never for an unchanged frame.
The driver contract cannot report `SetPixel` failures; TinyBubbles propagates
errors from `Display` and optional rectangle fills.

If the concrete driver also implements
`FillRectangle(x, y, width, height int16, c color.RGBA) error`, the renderer uses it
for backgrounds. Otherwise it fills with `SetPixel`. SSD1306 and ST7789 both
provide this optional method. No custom replacement Displayer interface is needed.

The renderer maintains one previous-cell grid and compares it with the current
view. Changed cells have their background erased before glyphs are drawn, so
shorter text does not leave old pixels. The first frame and theme changes clear
the whole display, including right/bottom margins outside the character grid.
Smaller and nil views erase omitted cells. The renderer clips oversized views
and glyphs, and retries a full repaint after a flush or rectangle error.
Applications should give it exclusive ownership of the display, or use a standalone
`Renderer` and call `Invalidate` after drawing outside the renderer.

`Buffer.Text` accepts UTF-8 and newlines, clips horizontally, and does not wrap.
One rune occupies one cell; there is no shaping, wide-character layout, or
combining-character composition. Missing glyphs fall back to `?` in the TinyFont
adapter. `Text` is **not an ANSI parser**. To port BubbleTea code, replace string
views with buffer writes and move terminal styling into the theme palette. ANSI
compatibility is intentionally left for a future optional package.

## Fonts and themes

The default is TinyFont's Org01 face in fixed **6x8 pixel cells**, with baseline 5.
It is small, with ASCII coverage. Set `Config.Font` to `bubbles.TinyFont` to use
another face and explicit cell width, height, and baseline, or implement:

```go
type Font interface {
    Size() (width, height int16)
    DrawGlyph(drivers.Displayer, x, y int16, r rune, c color.RGBA)
}
```

`DrawGlyph` receives the top-left pixel position and a clipping display; it must
not retain that display. Choose dimensions that include glyph ascenders and
descenders. Check the license of the particular TinyFont face you distribute.

Cells store a palette ID instead of two RGBA colors. `Theme` has eight styles:
`Normal`, `Accent`, `Selected`, `Muted`, and four application slots. Unknown IDs
fall back to `Normal`. `DefaultTheme(true)` uses black/white and inverse selection;
`DefaultTheme(false)` adds cyan accents and gray help text. Selection remains
visible without relying on hue. Pass a custom `*Theme` in `Config` or call
`Program.SetTheme` to repaint on the next step.

Displayer cannot identify color depth, so set `Monochrome` in the device config.
The model chooses its layout from `WindowSizeMsg`; it is not tied to a board name.
Configure rotation before creating a program. Runtime physical resizing requires
a new renderer/program; changing only a model's cell grid does not resize hardware.

## Memory and resolution

With the default 6x8 font, dimensions use integer division (unused edge pixels
are background). A cell is typically 8 bytes, including alignment, on 32-bit
TinyGo targets. One reusable model buffer plus one renderer cache costs roughly
**16 × columns × rows bytes**, before slice headers, fonts, driver buffers, model
state, stacks, and runtime overhead.

| Pixels | Grid | Two cell grids | Typical separate pixel framebuffer |
| --- | --- | --- | --- |
| 128x64 | 21x8 | 2,688 B | SSD1306: 1,024 B (1 bit/pixel) |
| 240x135 | 40x16 | 10,240 B | RGB565 if buffered: 64,800 B |
| 240x240 | 40x30 | 19,200 B | RGB565 if buffered: 115,200 B |
| 320x240 | 53x30 | 25,440 B | RGB565 if buffered: 153,600 B |

TinyBubbles allocates **no pixel framebuffer**. The pinned ST7789 driver writes
directly to the panel with a small transfer buffer; SSD1306 maintains its own
monochrome buffer. Larger cells reduce grid RAM. Reuse your `Buffer` and clear it
in `View`; repeated buffer allocation will cause heap churn. Text formatting,
messages and font implementations can still allocate; this is not an
allocation-free API. Font glyph tables also consume flash/data depending on the
font and compiler. The renderer's cell comparison is O(columns × rows), even
when the bus receives no writes. Glyph pixels on direct-write TFT drivers can
cost many small SPI transactions; full-screen animation is not the primary use.

Use `tinygo build -size=short` for code/static RAM figures, and profile heap/stack
headroom on your actual board. Those compiler figures do not include live heap
allocations such as cell and display buffers.
