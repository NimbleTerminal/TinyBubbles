# Bubble Tea and TinyTea

[Back to the README](README.md)

TinyBubbles supplies display rendering, fonts, themes, and device-focused UI.
**Bubble Tea v2.1 is the primary runtime.** TinyTea is an optional cooperative
replacement for smaller memory budgets. Both support small-screen widgets and
custom cell views; choosing a runtime does not limit font sizes or layout.

## Choosing a runtime

| Capability | Bubble Tea display backend | TinyTea |
| --- | --- | --- |
| Model API | Upstream `tea.Model`, `tea.Cmd`, `tea.Msg`, `tea.View` | Source-compatible basic signatures, separate Go types |
| Plain view | `tea.NewView(text)`, clipped to cells | Same supported usage |
| Custom fonts/styles | Optional `CellView() *tinybubbles.Buffer` | Same interface |
| Commands | Upstream concurrent command execution, Batch, Sequence | One short command per Step; queue capacity 16 |
| Timers | `tea.Tick` and `tea.Every` | Device schedules custom messages; no Tick/Every helpers |
| Input | `Program.Send` from peripheral goroutines | Single-owner synchronous Send; errors returned to caller |
| Keys | Upstream key types and fields | `KeyPressMsg{Code: ...}`; directions, Enter, Escape, ordinary runes |
| Window size | Cell dimensions from display; upstream delivery order | Cell dimensions delivered before the first view and Init command execution |
| Main loop | `Run() (tea.Model, error)` | `Run() (tinytea.Model, error)` or Start/Step/Stopped |
| Quit | Upstream Quit command | Quit command; directly sent QuitMsg queues shutdown for Step |
| Errors | Adapter cancels on display failure | Synchronous errors; display retries on a later Step |
| Terminal behavior | Terminal features unsupported by display backend | No terminal features |
| Toolchains | TinyGo 0.43, Go 1.26; vendored portability patches | TinyGo 0.43; lightweight root module uses Go 1.23 language level |

TinyTea is not binary-compatible with `tea.Model`. Change imports and recompile;
models that use unsupported Bubble Tea helpers need adaptation. Libraries that
import Bubble Tea directly remain Bubble Tea libraries. TinyTea does not pretend
to implement their complete API. Never replace a timer with a sleeping TinyTea
command: it would block input and rendering.

## Switching a simple model

Write the supported model subset using the `tea` import alias:

```go
func (m model) Init() tea.Cmd { return nil }
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    if key, ok := msg.(tea.KeyPressMsg); ok && key.Code == 'q' {
        return m, tea.Quit
    }
    return m, nil
}
func (m model) View() tea.View { return tea.NewView("Hello, tiny world!") }
```

For Bubble Tea:

```go
import (
    tea "charm.land/bubbletea/v2"
    displaytea "github.com/NimbleTerminal/TinyBubbles/bubbletea"
)
// After initializing display, font, and theme:
p, err := displaytea.NewProgram(model{}, display, font, theme)
```

For TinyTea, change the model import and program constructor:

```go
import tea "github.com/NimbleTerminal/TinyBubbles/tinytea"

p, err := tea.NewProgram(model{}, display, font, theme)
```

Check construction errors before calling `Run`. Both return the final model and
an error. For physical buttons on TinyTea, use a single device loop: poll buttons,
call `Send` and check its error, then call `Step` and check its error. Stop when
`Stopped()` is true. Interrupts must hand data to this loop rather than calling
Send directly. Root-package `Button` press messages are translated into TinyTea
key presses; Select maps to Enter, Back to Escape, and Quit to 'q'.

Keep view methods free of state changes: view-call counts and ordering are not
portable between runtimes.

Both runtimes accept the same optional `CellView()` method and the same renderer
font/theme arguments. Keep reusable widgets focused on state, layout, and cells;
let the owning model return runtime-specific commands. Small-screen UX—focus,
three-button menus, large text, monochrome styles—belongs in widgets and does
not depend on a particular scheduler.

## Existing cooperative API

The root package's original `Model` (`View() *Buffer`), `Config`, and `Program`
remain available for existing demos. They implement the underlying TinyTea
scheduler; new alternate-runtime applications should use the `tinytea` package's
Bubble Tea-shaped API. See the [legacy lifecycle reference](API.md) for command
queue and Step behavior, and the [Bubble Tea backend guide](bubbletea/README.md)
for preparation, firmware builds, and measured resource costs.

The existing Feather measurements are about 33 KiB flash / 13 KiB static RAM
for the original cooperative demo versus 260 KiB / 91 KiB for Bubble Tea.
They exclude heap allocations and task stacks; the new TinyTea compatibility
adapter adds code and a fallback grid, so these are not its exact footprint.
