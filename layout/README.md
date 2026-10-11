# Cell layouts

`nimbleterminal.dev/tinybubbles/layout` divides a display's character grid and
copies child buffers into a parent. It has no runtime or device dependency.
All coordinates and sizes are in cells. The helpers allocate nothing.

```go
bounds := layout.Bounds(parentBuffer)
header, rest := bounds.Rows(1, 0)
body, footer := rest.Rows(max(0, rest.Height-1), 0)

// In Update, after a size change:
menu.SetSize(body.Width, body.Height)

// In CellView, clear stale cells before composing:
parentBuffer.Clear(tinybubbles.Normal)
layout.Text(parentBuffer, header, "SETTINGS", tinybubbles.Accent)
layout.Draw(parentBuffer, body, menu.CellView())
layout.Text(parentBuffer, footer, "Choose a setting", tinybubbles.Muted)
```

Import `layout` from `nimbleterminal.dev/tinybubbles/layout` and the root package
as `tinybubbles`. See [the Settings model](../bubbletea/settings/model.go) for a
complete example. Widget size changes belong in Update, not CellView.

## Regions

- `Rect{X, Y, Width, Height}` describes a region. Nonpositive extents are empty.
- `Bounds(buffer)` returns its full grid, or an empty region for nil.
- `Rows(first, gap)` returns a top region and the remaining bottom region.
- `Columns(first, gap)` returns a left region and the remaining right region.
- `Inset(top, right, bottom, left)` removes padding.
- `Align(width, height, horizontal, vertical)` places a child using `Start`,
  `Center`, or `End`. Oversized children are cropped to the region size.

Negative sizes and gaps become zero. Splits clamp to available space, giving the
first region priority. Insets give the left and top priority over the opposite
sides. Centering puts an odd spare cell after the child.

Compose splits to make stacks or columns. For two equal columns with a one-cell
gap, use `left, right := bounds.Columns(max(0, bounds.Width-1)/2, 1)`. The second
column gets any extra cell. Parents decide when a narrow screen should use rows
instead. There is no automatic sizing, flex layout, or widget tree.

## Drawing

`Draw(destination, region, source)` copies the source's top-left cells, preserving
runes and styles. It clips to both buffers and the destination region. Negative
origins crop the corresponding leading source cells. Blank source cells overwrite
the destination; cells beyond the source stay unchanged. Overlapping self-copies
are supported. Buffers must not be mutated concurrently.

`Text(destination, region, text, style)` clips text to that region. Newlines move
to the next row; carriage returns reset the column. Other ASCII controls are
ignored. Text does not wrap automatically; use the viewport for wrapped content.

Clear the parent before each composition to remove stale content. Keep and reuse
child buffers; these helpers create none. Each existing grid still costs about
eight bytes per cell, and the renderer has its own grid. Clipping a child into a
smaller region does not reduce its allocation: size the widget for its region.

Focus, input routing, timers, and device effects stay with the parent model.
These helpers work with Bubble Tea, TinyTea, and the root renderer alike.

## Try the playground

Run `make web-serve` and choose **Layout playground**. Arrangement cycles through
Auto, Rows, and Columns. Auto uses columns when the padded body is at least 32
cells wide; otherwise it stacks rows. Padding and Gap toggle between zero and
one cell. Alignment moves both child tiles through Start, Center, and End on
both axes. Gap and Reset are under More.

The patterned panels show region bounds. Blank space shows padding and gaps.
The header names the arrangement; the footer shows padding (`P`), gap (`G`), and
alignment. Try the OLED and TFT presets to see the same layout adapt. This sizing
policy belongs to [the demo model](../web/hello/layout.go), not the layout package.
