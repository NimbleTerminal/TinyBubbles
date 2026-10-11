# Browser demo

The hello demo runs the shared Bubble Tea model and TinyBubbles cell renderer
against a canvas-backed `drivers.Displayer`. HTML controls come from the model's
Bubbles `help.KeyMap`, through Booba Terminal Actions. No help renderer appears
inside the simulated display.

## Run locally

From the repository root:

```sh
make web-serve
```

Open <http://127.0.0.1:8080>. Requires Go (the module selects Go 1.26.8) and
Python 3. `make web` builds only; `make -C web test` also requires Node.js 22 or
newer for the key encoder tests. No npm dependencies or bundler are needed.

Choose Feather TFT (240×135), monochrome OLED (128×64), or color TFT (320×240).
Pop increments the enlarged counter; Color changes palette; Pause becomes
Resume. Color is disabled on the OLED. More reveals Quit; Restart reloads the
program. Focus the display for Enter, Right, Escape, and Q keyboard controls.

The Runtime selector switches Hello World between **Bubble Tea** and **TinyTea**.
The URL stores `runtime=bubbletea` or `runtime=tinytea`, alongside the display
preset. Switching reloads the page and resets the counter and animation.
Bubble Tea is the default. TinyTea runs the real cooperative program with
host-supplied 100 ms ticks; Bubble Tea schedules its own `tea.Tick` commands.
Both use the same hello state, fonts, palettes, and help bindings.

This is a runtime behavior comparison. The WASM binary includes both runtimes
and the browser protocol dependencies, so its size does not represent a minimal
TinyTea firmware build. Go versus TinyGo remains a separate compiler choice.

## How it works

The Bubble Tea path has two outputs from one Go model:

```text
hello.CellView → DisplayModel → TinyBubbles renderer → canvas Displayer
hello.Keys → Booba bubblehelp → tea.Raw(OSC 7501) → Booba → HTML action bar
HTML action → key encoder → Booba input → tea.KeyPressMsg → hello.Update
```

`displaytea.NewModel` allows Booba to own the Bubble Tea program and its I/O.
Pass the wrapper's `Context()` to the program, use `WithoutRenderer()`, supply
its `Size()` as the initial window size, and call `Close()` after `Run()`.
Check `Err()` for display failures. Terminal resizes are clamped to the physical
cell grid, so resizing the page does not change the simulated device resolution.

`WithoutRenderer` suppresses terminal rendering but **does not suppress
`tea.Raw` commands**. Booba's input/output buffers carry the capability query,
reply, and action updates. Do not replace that output with `io.Discard` or
disable input. `View.ProgramStatus` requires terminal rendering; this demo uses
the `bubblehelp` wrapper's raw protocol path instead.

`bubbletea/hello/keys.go` is the single definition of available actions. Both
`Update` and the help metadata use those bindings. Hardware imports only the
small Bubbles `key` package. The Booba wrapper and `help` adapter live here in
the separate web module. `ShortHelp` supplies the compact buttons; `FullHelp`
supplies More. Labels and enabled state follow model updates automatically.

The browser explicitly opts into Terminal Actions and supplies a restricted
key encoder. It respects application-cursor mode for arrow keys, rejects
unsupported gestures, and clears controls when the program stops. There is no
parallel JavaScript implementation of the counter or animation.

### TinyTea bridge

`tinyhello.go` adapts hello to `tinytea.Model`. It forwards state updates to the
shared hello model, replaces its scheduled ticks with host ticks, and returns
TinyTea's Quit command. It does not execute Bubble Tea commands or support
arbitrary Bubble Tea models. The active runtime is also printed on the display.

The WASM host registers the same Booba read/write callbacks. Callbacks enqueue
complete gestures from this page's restricted encoder; one loop calls TinyTea's
`Send`, `Step`, and tick handler. Booba's publisher turns the shared Bubbles help
map into the same OSC action snapshots. Display size stays fixed. Quit stops
ticks and clears the HTML controls through the existing page lifecycle.

## Versions and hosting

This module pins Booba v0.8.0 and the Bubble Tea WASM fork used by that release
(`c397c32a5074`). The fork is required for browser I/O; the embedded backend
continues using upstream Bubble Tea v2.1 with its bare-metal portability patches.
Root and firmware dependencies do not inherit this module's replacements.

`make web` uses standard Go's `js/wasm` target. `make web-tinygo` builds the same
demo with TinyGo v0.43, a 128 KiB goroutine stack, and its matching `wasm_exec.js`.
The default TinyGo WASM stack overflowed during startup; the larger stack was
validated in Safari with live actions and pixel rendering. Always rebuild when
changing compiler so the runtime shim matches the binary.

The static site is generated in `build/web/`. All asset URLs are relative, so
it can be hosted under a repository subpath. CI tests the bridge and builds the
standard Go site, then uploads **only `build/web/`** as the `web-demo` artifact.
A separate TinyGo v0.43 job builds and uploads `web-demo-tinygo`.
Publishing to a public host is a separate step. Never publish all of `build/`,
which may contain private device backups.

The browser allocates a 4-byte-per-pixel RGBA framebuffer plus canvas ImageData
(about 253 KiB combined for 240×135). This is a browser driver cost, not an
embedded renderer requirement. See [the core memory budget](../API.md#memory-and-resolution).

To add a widget demo, create a Bubble Tea model with `CellView()` and a Bubbles
help map, then wrap it as in [demo.go](./demo.go). Keep widget state in Go,
peripheral setup in device mains, and browser transport in this module.

Generated distributions include `THIRD_PARTY_NOTICES.txt`. Dependencies retain their own licenses: [Booba](https://github.com/NimbleMarkets/go-booba),
[Ghostty Web](https://github.com/coder/ghostty-web), and the Go/TinyGo runtime.

## Adding a gallery entry

`site/demos.js` lists the available demos: stable ID, category, title, description,
control instructions, source URL, supported runtimes, and WASM asset path. The gallery groups entries
by category and links them with `?demo=<id>`, preserving the selected display.
Unknown demo IDs fall back to hello; unsupported runtimes fall back to the entry’s default. Only add demos when their WASM assets are ready.

For a new widget, add its Go entry point, build its WASM into `build/web/`, and
register that asset in `demos.js`. Use the same compiler and runtime shim for all
entries in a build. Selecting a demo reloads the page, giving each Go program a
fresh lifecycle. Its controls still come from its own Go help map; the gallery
registry contains presentation metadata only.
