# Browser harness for TinyGo displays

`nimbleterminal.dev/tinybubbles/web` runs display applications in a browser through
[Booba](https://github.com/NimbleMarkets/go-booba). It provides a canvas display,
LEDs, HTML controls from a Bubbles help map, and a shared page and build scaffold.
It does not require TinyBubbles cells, fonts, or the displaytea adapter.

This is a separate Go 1.26.8 module. It pins Booba v0.9.1 and the Bubble Tea WASM
fork at `c397c32a5074`. The root TinyBubbles module stays Go 1.23 with no Bubble
Tea dependency. **Each application's web module must repeat the Bubble Tea
replace directive** because Go does not inherit dependency replacements.

## Try the gallery

From the repository root:

```sh
make web-serve
```

Open <http://127.0.0.1:8080>. The gallery lives in [hello](./hello), a separate
consumer module. It includes Hello World, the TinyHAL workbench, TinyCharts, layout, Settings,
list, value, dialog, progress, status, and viewport demos. Settings includes choice pickers.
`make web-serve-tinygo` (or `make web-tinygo`) also adds a TinyGhostty tab when
`tinyghostty/lib/wasm/libghostty-vt.a` exists (`make -C tinyghostty lib`);
Go builds and builds without the archive omit it.

Hello World shares the device model and renderer and supports Bubble Tea
and TinyTea loops; the other demos use Bubble Tea. Switching runtime restarts
the demo and keeps the display preset. Both runtimes are in the browser binary, so this does not compare
firmware sizes. `make web-tinygo` builds with TinyGo 0.43, the Boehm garbage
collector (`-gc=boehm`), and a 128 KiB stack. CI uses the same build for the
published demo.

## Start another project

For local development, run from `TinyBubbles/web`:

```sh
go tool tinybubbles-web --module example.org/player/web \
  --local-harness "$PWD" /path/to/player/web
```

This creates `go.mod`, a Makefile, `index.html`, and `style.css`. Add your Go
main under `cmd/demo`, then run `go mod tidy` and `make serve` in the new module.
Existing page and build files are left alone unless you pass `--force`.
An existing `go.mod` is always preserved, even with `--force`. For a release, use `--harness-version v0.2.0` instead of `--local-harness`.

In an existing web module, add these tools and the harness dependency:

```go
tool (
    nimbleterminal.dev/tinybubbles/web/cmd/tinybubbles-web
    github.com/NimbleMarkets/go-booba/cmd/booba-assets
)

replace charm.land/bubbletea/v2 => github.com/neomantra/bubbletea/v2 v2.0.0-20261009201241-c397c32a5074
```

Require `nimbleterminal.dev/tinybubbles/web v0.2.0`, or use a local replace
when developing the harness. Do not change your
hardware application's root module. To generate a site without keeping local
copies of the harness assets:

```sh
go tool tinybubbles-web --assets --force build
```

That command runs `booba-assets`, generates dependency notices, and writes the
shared page into `build/`. The application still builds its own `app.wasm`. A `.tinybubbles-assets.json`
manifest tracks generated files. `--assets --force` removes tracked files no
longer shipped by the harness, plus the legacy `demos.js`. It preserves app
binaries and untracked custom assets. Keep the manifest between local builds.
The consumer Makefile can include `application.mk` from the harness module; see
[hello/Makefile](./hello/Makefile). Set `MAIN`, `OUT`, and `PORT` as needed.
`make serve` builds before serving. Go and TinyGo builds install matching
`wasm_exec.js` files. Never mix their binaries and runtime shims.

Builds need Go 1.26.8 and Python 3. Harness JavaScript tests need Node.js 22+.
Only `index.html` and `style.css` are consumer-owned page files. `SITE` overrides
those two files; JavaScript, logos, licenses, and notices always come from the
current harness during the build. Old scaffolded copies of `main.js`, `keys.js`,
`package.json`, `bubbles.svg`, `notices.py`, and `licenses/` are no longer used
and can be removed from consumer source trees after checking for local changes.
Do not carry JavaScript customizations forward by copying them over generated
assets; make shared harness changes instead.

## Go API

The browser APIs have `js && wasm` build tags. Host tests can import `Config`,
`Screen`, and `WrapActions` without importing `syscall/js`.

In your WASM main, pass page content and display presets from Go. `Logo` is an
optional image URL. `Introduction` accepts `TextPart` entries with optional URLs
for inline links; omit it to use plain `Description` text. Content stays plain
text rather than accepting raw HTML.

For example:

```go
selected := web.Configure(web.Config{
    Title: "My player",
    Source: "https://github.com/example/player",
    Description: "The same player UI running in a browser.",
    Screens: []web.Screen{
        {ID: "color", Label: "Color TFT", Width: 320, Height: 240},
        {ID: "oled", Label: "OLED", Width: 128, Height: 64, Monochrome: true},
    },
    LEDCount: 2,
})
display := web.NewCanvasDisplay("display", selected.Screen.Width, selected.Screen.Height)
leds := web.NewLEDs("led")
```

`CanvasDisplay` implements `Size`, `SetPixel`, `Display`, and `FillRectangle`.
It clips writes and flushes an RGBA buffer through canvas `putImageData`.
`LEDs.WriteColors([]color.RGBA)` colors elements by class; `Len()` gives the count.
The browser driver and canvas ImageData use about eight bytes per pixel combined.
These buffers are separate from your embedded driver's memory requirements.

Create your model using the display and LEDs, then wrap its current help map:

```go
wrapped, err := web.WrapActions(model, "player", func(m tea.Model) help.KeyMap {
    return m.(*Model).Keys()
}, nil)
if err != nil { panic(err) }
web.Run(wrapped, web.RunOptions{Error: model.Err})
```

The wrapper uses Booba's eight-character hexadecimal action IDs. The initial
snapshot, including its summary, must fit the OSC 7501 2048-byte message limit
or construction fails with an error. Later errors go to the callback; a nil
callback logs them.

`Run` creates the Booba program with terminal rendering, signals, and panic
recovery disabled. `RunOptions.Program` accepts extra Bubble Tea options, such
as a context or initial cell size. `Error` can return an application or display
error after the program ends. The helper notifies the page, clears its controls,
and keeps WASM alive. Custom loops can call `Stopped(error)` themselves.

## Input and timing

Booba's hidden terminal carries input and action metadata. `WithoutRenderer()`
allows `tea.Raw` output; keep Booba's input and output connected. No terminal
help view is drawn. HTML controls and canvas keyboard input use the current
**enabled** help bindings. Unsupported keys and modifiers are declined.
Booba's key encoder supports named keys and printable Unicode characters and
follows the live cursor, backarrow, and newline modes. It declines input when
an extended keyboard protocol is active. Shift used to type a character such as `+`
is part of the character; Shift+Arrow is a modified command and is declined.

A demo that hosts a terminal or text editor can set `Demo.TextInput`. The
focused display then forwards typed characters (including space), Ctrl with a
letter or `[ \ ] ^ _ @`, every named key the harness maps, and bracketed paste,
whether or not the help map binds them. Alt and Meta combinations stay with the
browser. Action buttons still follow the help map, and other demos keep the
enabled-binding rule.

`application.mk` also accepts `TINYGO_TAGS` for the TinyGo build and
`EXTRA_NOTICES`, license files appended to `THIRD_PARTY_NOTICES.txt` for linked
code that is not a Go module.

Hidden browser tabs throttle timers to about once per second. This slows every
`time.Sleep` in the Go program, including tinypod playback and LED updates.
Keep the tab visible when checking timing. The harness provides no Web Audio output.

## Adding a gallery entry

Supply `Config.Demos` from Go. Each entry has an ID, title, description, source
link, and optional runtimes. The page stores demo, runtime, and screen selections
in the URL; invalid values fall back to the first available entry. All models
are compiled into the application's `app.wasm`; the selected entry tells your
main which model to start. Sizes and presets are never duplicated in JavaScript.

For a TinyBubbles demo, follow `hello/layout.go` or an existing widget demo:
size child buffers in Update, expose a current help map, and keep rendering in
CellView. Register its metadata and model construction in
[hello/cmd/demo/main.go](hello/cmd/demo/main.go). Use a distinct action context
and check both OLED and TFT presets, including disabled controls.

Edit shared HTML and CSS in `internal/scaffold/files`, then rebuild the consumer.
Generated files in `build/web` are overwritten. The shared footer includes the
NimbleTerminal attribution and dependency notices link.

## Checks and hosting

`make -C web test` from the repository root tests the harness, DOM key mapping, and
the demo gallery. Tests cover upstream action IDs, the status size limit, live help
updates, and preserving existing scaffold files. `make ci` also builds firmware
and Go/TinyGo WASM. CI uploads only `build/web/` for the site, not all of `build/`.

Generated assets use relative URLs and work under a hosting subpath. Public
hosting for this repository uses [GitHub Pages](https://NimbleTerminal.github.io/TinyBubbles/).
After all CI checks pass on `main`, the workflow publishes the TinyGo WASM
site, including its matching runtime shim and dependency notices. Pull requests
and tags do not deploy. Preserve `THIRD_PARTY_NOTICES.txt` when publishing elsewhere.
