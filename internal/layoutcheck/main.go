//go:build tinygo

// Compile composition helpers for an embedded target.
package main

import (
	bubbles "nimbleterminal.dev/tinybubbles"
	"nimbleterminal.dev/tinybubbles/layout"
)

var result *bubbles.Buffer

func main() {
	result = bubbles.NewBuffer(21, 8)
	child := bubbles.NewBuffer(8, 2)
	header, body := layout.Bounds(result).Rows(1, 1)
	left, _ := body.Inset(0, 1, 0, 1).Columns(8, 1)
	layout.Text(result, header, "DEVICE", bubbles.Accent)
	layout.Text(child, layout.Bounds(child), "Ready", bubbles.Normal)
	layout.Draw(result, left.Align(8, 2, layout.Center, layout.End), child)
}
