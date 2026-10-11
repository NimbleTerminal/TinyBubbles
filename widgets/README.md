# Small-screen widgets

Widgets share cell layouts across Bubble Tea and TinyTea. They use the display's
font and theme. They do not draw ANSI text or own the display driver.
Use [layout helpers](../layout/README.md) to size and compose their cell buffers.

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

## Confirmation dialog

The dialog asks for a decision with two choices. It defaults to Cancel, wraps
the prompt, and keeps both choices visible in separate rows on a small display.

```go
import (
    dialogtea "nimbleterminal.dev/tinybubbles/bubbletea/dialog"
    "nimbleterminal.dev/tinybubbles/widgets/dialog"
)

prompt := dialogtea.New(dialog.Config{
    Title: "Reset settings?",
    Body: "Restore the default volume and brightness?",
    Confirm: "Reset", Cancel: "Keep settings",
})
```

Forward messages and return the command. The parent handles
`dialogtea.ResultMsg{Confirmed: ...}` and performs any confirmed action.
Left or Right changes the selected choice, Enter resolves it, and Escape always
cancels. The short help map exposes Next, Select, and Cancel for three-button
navigation. The Bubble Tea `KeyMap` is configurable.

A decision sets `Done()` immediately and disables all dialog bindings. Further
input produces no second result. `Reset()` reopens the prompt with Cancel
selected; it preserves the text and size. Result commands capture the decision,
so resetting does not change an already returned command. The parent should
consume the result before reopening the dialog.

Give the dialog the focused input while it is open. It provides its own cell
view; the parent owns any overlay, screen transition, or background view.
The widget never resets hardware or writes settings itself.

Use at least four cell rows for a title, prompt, and both choices. Body text
wraps at words, splits long words at rune boundaries, and truncates with a dot
when it runs out of rows. Whitespace in the body is collapsed. Shorter layouts
prioritize choices; a one-row layout shows only the selected choice. Check that
important prompt text fits your smallest supported display.

The core is `widgets/dialog`. `tinytea/dialog` uses the same state and rendering
with fixed Left/Right/Enter/Escape keys and a TinyTea `ResultMsg`. Text layout is
cached when the size changes, and `CellView` allocates nothing. The dialog owns
one cell grid plus its wrapped text; the renderer owns another grid.

## Choice picker and toggles

`widgets/choice` edits an index into a fixed set of named options. Left and Right
wrap, Enter applies, and Escape restores the applied option. Use
`bubbletea/choice.New(choice.Config{Label: "Playback", Options: []string{"Single",
"Repeat", "Shuffle"}, Value: 0})` and check its error. Options must be nonempty
and the initial index must be valid. Construction copies the options.

`Value()` and `Applied()` return indices; `Label()` returns the draft option text.
`SetValue(index)` replaces both values and returns false for an invalid index.
The adapter emits `AppliedMsg` or `CancelledMsg` with the captured `Index` and
`Label`. It exposes configurable bindings through `KeyMap` and `Keys()`; a single
option disables navigation. For a toggle, use options `Off` and `On`. The core
also offers `NewToggle(label, enabled)`.

`tinytea/choice` supplies the same messages with fixed keys and no Bubbles help.
Rendering uses one reused grid, clips long labels, and allocates nothing in
`CellView`. Labels use the display font, including its glyph limitations.

## Composing a settings screen

[bubbletea/settings](../bubbletea/settings/model.go) combines the list, value
picker, choice picker, and dialog. The parent forwards input only to the focused
widget and exposes that widget's help bindings. Selecting a menu item opens its
editor; Apply or Cancel returns to the same menu selection. Input waits until a
child's result command is consumed, preventing repeated presses from crossing
screen transitions. Reset defaults asks for confirmation and starts on Cancel.

`settings.New()` implements `tea.Model` and `CellView`; `Values()` returns a copy
of the applied settings. This is an in-memory example: it does not write storage
or change hardware. Each child retains a cell grid, so composition uses more RAM
than a single widget. Device apps can create editors on demand to reduce memory.

## Progress bar

`widgets/progress` displays completed work as a bar and an integer percentage.
A zero total selects an activity indicator for work with no known total.

```go
import (
    progresstea "nimbleterminal.dev/tinybubbles/bubbletea/progress"
    "nimbleterminal.dev/tinybubbles/widgets/progress"
)

bar := progresstea.New(progress.Config{Label: "Loading", Total: 100})
// Forward this message to the bar in the parent's Update:
msg := progresstea.UpdateMsg{Done: 35, Total: 100}
```

Counts are `uint32`. `SetProgress(done, total)` clamps completed work to the total;
an unknown total discards the completed count. `Complete()` is true only for a
known total that has been reached. The widget emits no completion command and
binds no keys. Parents decide what finishing, cancelling, or restarting means.

For unknown work, send `PulseMsg{}` or call `Advance()` from the parent's timer.
Each pulse moves a short segment one cell, wrapping at the end. The widget owns
no clock or goroutine; TinyTea parents must supply pulses from their device loop.
`tinytea/progress` accepts the same count and pulse messages. Bubble Tea parents
can schedule pulses with `tea.Tick`.

`SetSize` takes cells. One row displays only the bar; two rows add the percentage
or Working text, and taller views add the label. ASCII `=` and `-` distinguish
filled and empty cells even on monochrome displays. The buffer is reused, drawing
allocates nothing, and percentage arithmetic uses 64 bits to avoid count overflow.
Memory is one cell grid (about eight bytes per cell), plus the renderer's grid.

## Status indicator

`widgets/status` shows Offline, Ready, Busy, Warning, or Error with a distinct
ASCII symbol (`-`, `+`, `~`, `!`, `x`) and explicit state text. Color is optional;
the state remains readable on monochrome displays.

```go
import (
    statustea "nimbleterminal.dev/tinybubbles/bubbletea/status"
    "nimbleterminal.dev/tinybubbles/widgets/status"
)

indicator := statustea.New(status.Config{
    Label: "Wi-Fi", State: status.Ready, Detail: "Connected",
})
// In the parent's Update, forward state changes from your device:
indicator.Update(statustea.UpdateMsg{State: status.Warning, Detail: "Weak signal"})
```

`SetStatus(state, detail)` replaces state and detail, retaining the label.
`State()` and `Detail()` read them. Invalid states become Offline. The widget
has no keys, timers, animation, or device effects. `tinytea/status` accepts the
same messages with TinyTea's model interface.

Give the widget one cell row for its heading, or two for optional detail. The
heading puts the symbol and state before the device label so narrow screens keep
the most useful information. Text clips with a dot and never wraps; a one-cell
view keeps only the symbol. Extra rows are cleared. Drawing reuses the grid and
allocates nothing; the grid uses about eight bytes per cell.

`Styles[state]` selects a TinyBubbles style ID. Map those IDs in the parent theme
to add semantic colors. The browser demo uses green, amber, and red for Ready,
Warning, and Error on color displays. These colors supplement the text.

## Text viewport

`widgets/viewport` displays a scrollable plain-text document. It preserves blank
lines and indentation, wraps at rune boundaries, and reserves the rightmost
column for a scrollbar when the width is at least two cells. It does not parse
ANSI, wrap at words, or support horizontal scrolling. Each rune occupies one cell.

```go
import viewporttea "nimbleterminal.dev/tinybubbles/bubbletea/viewport"

view := viewporttea.New("Device information\n\nFirmware ready.")
```

Forward size and key messages to the focused viewport. Up/Down scroll one row;
Left/Right move a page with one row of overlap. Home/End
jump to the top/bottom. `KeyMap` is configurable, and `Keys()` disables actions
at the boundaries. Scrolling clamps and never wraps. Parents own Back and Quit;
three-button devices can use Up, Down, and Back.

`SetContent(text)` replaces the document and returns to the top. CRLF becomes a
newline, tabs become four spaces, and other ASCII controls are removed. It is
not a terminal emulator or a bounded log store: parents must limit retained text.
`Scroll(rows)`, `Page(direction)`, `GotoTop()`, and `GotoBottom()` also work without
a runtime. `Offset()` reports the first visible wrapped row; `TotalLines()` counts
wrapped rows. `AtTop()` and `AtBottom()` report navigation boundaries.

Resizing reflows text and keeps the previous top line's first character visible
where possible, then clamps to the final page. A zero-size viewport cannot scroll.
`tinytea/viewport` supports fixed Up/Down and Left/Right keys; Home/End and Bubbles
help bindings are available through the Bubble Tea adapter only.

Content and size changes cache line boundaries. Scrolling and `CellView` allocate
nothing. Memory includes normalized UTF-8 text, two integers per wrapped row
(about eight bytes per row on 32-bit targets), and one visible cell grid. The
renderer owns a second grid. No grid is allocated for offscreen content.

## Try it

Run `make web-serve` and choose **Scrolling menu** in the gallery. Try the
128×64 OLED preset to see scrolling, then the TFT presets for longer labels.
The demo reports the chosen item rather than opening settings screens.
Choose **Value picker** to adjust a volume setting, apply it, and cancel edits.
The picker is a UI demo and does not control audio. Choose **Confirmation dialog**
to confirm or cancel a mock settings reset. Choose **Settings** to try all four
widgets together, including on/off and playback choices. The widget demos run Bubble Tea;
Hello World retains its runtime selector. **Progress bar** simulates a load with
Pause/Resume, Start over, and Known/Unknown total controls. Its timer belongs to
the demo; switching browser tabs can slow it down. **Status indicator** cycles
through simulated connection states; it does not read real network status.
**Text viewport** contains a device guide and sample log; try paging on the OLED
preset and use More for top/bottom controls.

The shared widget and both adapters have host tests. `make ci` also compiles
the gallery with TinyGo 0.43 and checks the TinyTea widget adapters in firmware.

## Shared implementation

Widget labels use `layout.ClipText` for consistent clipping and alignment, and
widget grids use `Buffer.Resize`. Result messages are defined in `widgets/<name>`
and aliased by both runtime adapters. Existing adapter-qualified message names
still work and now identify the same Go type across runtimes. Parents handling
both runtimes should use one case for the shared type in a type switch.

Viewport position survives a temporary zero-size layout. A viewport at the bottom
stays at the bottom after reflow; replacing content still returns to the top.
An indeterminate progress bar blinks when only one interior cell is available.
