//go:build js && wasm && tinyghostty

package main

import (
	ghostty "nimbleterminal.dev/tinybubbles/tinyghostty/web"
	browser "nimbleterminal.dev/tinybubbles/web"
)

// The Makefile sets the tinyghostty tag when tinyghostty/lib holds the
// wasm32 libghostty-vt archive. Go's js/wasm cannot link it.
func init() {
	optional = append(optional, optionalDemo{
		Demo: browser.Demo{
			ID: "tinyghostty", Title: "TinyGhostty", TextInput: true,
			Description: "A live Ghostty VT terminal with a small in-browser shell. Click the display and type: help, colors, styles, box, ls, cat. Colors snap to the theme's eight styles.",
			Source:      "https://github.com/NimbleTerminal/TinyBubbles/tree/main/tinyghostty",
		},
		After: "charts",
		New:   ghostty.NewDemo,
	})
}
