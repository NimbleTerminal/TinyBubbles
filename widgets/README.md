# Small-screen widgets

Widgets share cell layouts across Bubble Tea and TinyTea. They use the display's
font and theme. They do not draw ANSI text or own the display driver.

## Scrolling list

The list draws one item per row, skips disabled items, and wraps at either end.
The highlight stays visible when the viewport shrinks. Long labels truncate with
a dot. A `>` marker and inverse selection work on monochrome screens; disabled
rows have a `-` marker. A scrollbar appears when the items exceed the viewport.

Use the Bubble Tea adapter by default:

```go
import (
    listtea "nimbleterminal.dev/tinybubbles/bubbletea/list"
    "nimbleterminal.dev/tinybubbles/widgets/list"
)

menu := listtea.New([]list.Item{
    {ID: "music", Label: "Music"},
    {ID: "settings", Label: "Settings"},
    {ID: "radio", Label: "Radio", Disabled: true},
})
```

The model implements `tea.Model` and `CellView`. Forward window size and key
messages to it and return its command from your parent Update. When the parent
receives `listtea.SelectedMsg`, use `msg.Item.ID` to open a screen or run an action.
Moving the highlight does not activate an item. The command captures the item
at activation, even if selection changes before the command runs.

`menu.Keys()` implements the Bubbles help map used by browser controls.
`menu.KeyMap` lets you change or disable bindings. Defaults are Up, Down, and
Enter. Forward input only while the widget has focus. The parent owns Back and
Quit. For three buttons, map Next, Select, and Back; Next wraps to the first item.

`SetSize(width, height)` takes cells. `SetItems` copies the slice and preserves
the selected enabled ID, falling back to the first enabled item. Use unique,
stable IDs. Empty and all-disabled lists have no selection and disable actions.
`Selected()` returns the highlighted item and a boolean; `SelectedIndex()` returns
-1 when there is no selection.

The shared model lives in `widgets/list`. It exposes `Move`, `SetSize`, `SetItems`,
and `CellView` without any Bubble Tea dependency. `tinytea/list` supplies the same
basic model and `SelectedMsg` using TinyTea types. Its keys are fixed to Up, Down,
and Enter; Bubbles help bindings stay in the Bubble Tea adapter.

The list copies item records once per `SetItems` and allocates a cell buffer only
when the size changes. Drawing and navigation allocate no additional buffers.
The buffer costs about eight bytes per cell; the renderer keeps its own grid.
Labels follow TinyBubbles' one-rune-per-cell rules and the selected font's glyph
coverage. There is no filtering, multi-selection, or item description layout yet.

## Try it

Run `make web-serve` and choose **Scrolling menu** in the gallery. Try the
128×64 OLED preset to see scrolling, then the TFT presets for longer labels.
The demo reports the chosen item rather than opening settings screens.
It runs Bubble Tea; Hello World retains its runtime selector.

The shared widget and both adapters have host tests. `make ci` also compiles
the gallery with TinyGo 0.43 and checks the TinyTea adapter in firmware.
