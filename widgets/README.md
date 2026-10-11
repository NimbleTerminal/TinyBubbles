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

## Value picker

The value picker edits a bounded integer setting such as volume or brightness.
It keeps a draft separate from the last applied value. Left and Right adjust one
step, Enter applies, and Escape cancels back to the applied value. Bounds clamp;
they never wrap. A step that crosses a bound lands on that bound.

```go
import (
    valuetea "nimbleterminal.dev/tinybubbles/bubbletea/value"
    "nimbleterminal.dev/tinybubbles/widgets/value"
)

picker, err := valuetea.New(value.Config{
    Label: "Volume", Unit: "%",
    Min: 0, Max: 100, Step: 5, Value: 50,
})
```

Check the construction error. `Min` must not exceed `Max`, and `Step` must be
positive. A fixed range is allowed. All numbers are `int32`; adjustments use
wider arithmetic to avoid overflow. The initial value is clamped to the range.

Forward messages and return commands as with the list. The parent receives
`valuetea.AppliedMsg` or `valuetea.CancelledMsg`, each containing `Value` captured
at the key press. Apply and Cancel remain available even when the value hasn't
changed, so a parent can use them to close the editor. Neither writes settings
or touches hardware. The widget updates its own applied value immediately;
use `SetValue` to restore it if a parent rejects a change or persistence fails.

`Value()` reads the draft, `Applied()` reads the last applied value, and `Dirty()`
reports a difference. `SetValue` clamps an external value and replaces both.
The range and step are fixed at construction; create another picker to change
those constraints. `KeyMap` is configurable, and `Keys()` disables adjustment
buttons at the bounds. Forward input only to the focused widget.

The core is `widgets/value`. `tinytea/value` provides the same behavior and
messages with TinyTea types, fixed Left/Right/Enter/Escape keys, and no Bubbles
help dependency. A three-button device can map Decrease, Increase, and Apply;
Cancel is optional and can be mapped by the containing screen.

Rendering uses the selected display font. The numeric value gets an inverse
highlight; larger layouts add a label, meter, bounds, and Applied/Unsaved state.
Layouts below three rows keep only the value. Units are display text; for decimal
values, store scaled integers in your model rather than relying on floating point.
The picker does not currently format those integers as decimal fractions.

The grid is reused while drawing. Number strings are cached outside
`CellView`, which allocates nothing. The cell grid uses about eight bytes per
cell, plus the renderer's own grid.

## Try it

Run `make web-serve` and choose **Scrolling menu** in the gallery. Try the
128×64 OLED preset to see scrolling, then the TFT presets for longer labels.
The demo reports the chosen item rather than opening settings screens.
Choose **Value picker** to adjust a volume setting, apply it, and cancel edits.
The picker is a UI demo and does not control audio. Both widget demos run Bubble
Tea; Hello World retains its runtime selector.

The shared widget and both adapters have host tests. `make ci` also compiles
the gallery with TinyGo 0.43 and checks both TinyTea adapters in firmware.
