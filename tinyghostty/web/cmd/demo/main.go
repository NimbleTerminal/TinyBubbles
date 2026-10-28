//go:build js && wasm

// Command demo runs tinyghostty in the browser harness. Build it with TinyGo;
// Go's js/wasm has no cgo for libghostty-vt.
package main

import (
	tea "charm.land/bubbletea/v2"

	demo "nimbleterminal.dev/tinybubbles/tinyghostty/web"
	browser "nimbleterminal.dev/tinybubbles/web"
)

const source = "https://github.com/NimbleTerminal/TinyBubbles/tree/main/tinyghostty"

func main() {
	selected := browser.Configure(browser.Config{
		Title:  "tinyghostty",
		Source: source,
		Introduction: []browser.TextPart{
			{Text: "A live "}, {Text: "Ghostty", URL: "https://ghostty.org"},
			{Text: " VT terminal inside a TinyBubbles display. TinyGo compiles Go and Ghostty's freestanding C library into one WASM binary; a small shell runs in it, so nothing leaves your browser."},
		},
		Note: "Click the display and type. Buttons run sample commands.",
		Screens: []browser.Screen{
			{ID: "feather", Label: "Feather TFT 240×135", Width: 240, Height: 135},
			{ID: "lcd", Label: "TFT 320×240", Width: 320, Height: 240},
			{ID: "oled", Label: "OLED 128×64", Width: 128, Height: 64, Monochrome: true},
		},
		Demos: []browser.Demo{{
			ID: "tinyghostty", Title: "TinyGhostty", Source: source, TextInput: true,
			Description: "Type commands: help, colors, styles, box, ls, cat. Colors snap to the display's eight theme styles.",
		}},
	})
	display := browser.NewCanvasDisplay("display", selected.Screen.Width, selected.Screen.Height)
	model, pixels, err := demo.NewDemo(display, selected.Screen.Monochrome)
	if err != nil {
		browser.Stopped(err)
		select {}
	}
	cols, rows := pixels.Size()
	browser.Run(model, browser.RunOptions{
		Program: []tea.ProgramOption{tea.WithContext(pixels.Context()), tea.WithWindowSize(cols, rows)},
		Error:   func() error { pixels.Close(); return pixels.Err() },
	})
}
