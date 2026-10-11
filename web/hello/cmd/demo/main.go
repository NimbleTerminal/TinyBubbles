//go:build js && wasm

package main

import (
	"sync"
	"syscall/js"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/NimbleMarkets/go-booba/actions"
	browser "nimbleterminal.dev/tinybubbles/web"
	demo "nimbleterminal.dev/tinybubbles/web/hello"
)

func main() {
	selected := browser.Configure(browser.Config{
		Title:  "TinyBubbles",
		Logo:   "bubbles.svg",
		Source: "https://github.com/NimbleTerminal/TinyBubbles",
		Introduction: []browser.TextPart{
			{Text: "TinyGo", URL: "https://tinygo.org"},
			{Text: " compiles Go for microcontrollers. "},
			{Text: "Bubble Tea", URL: "https://github.com/charmbracelet/bubbletea"},
			{Text: " is a Go framework for interactive terminal apps. TinyBubbles brings Bubble Tea to small embedded displays, so you can try a Go interface in your browser and run the same UI on hardware."},
		},
		Screens: []browser.Screen{
			{ID: "feather", Label: "Feather TFT · 240 × 135", Width: 240, Height: 135},
			{ID: "oled", Label: "Monochrome OLED · 128 × 64", Width: 128, Height: 64, Monochrome: true},
			{ID: "large", Label: "Large TFT · 320 × 240", Width: 320, Height: 240},
		},
		Demos: []browser.Demo{{ID: "hello", Title: "Hello, tiny world", Description: "Animation, custom fonts, and a counter on a tiny display.", Source: "https://github.com/NimbleTerminal/TinyBubbles/tree/main/bubbletea/hello", Runtimes: []browser.Runtime{
			{ID: "bubbletea", Label: "Bubble Tea", Description: "Commands and timers run through Bubble Tea’s event loop."},
			{ID: "tinytea", Label: "TinyTea", Description: "Cooperative commands; the host supplies animation ticks."},
		}}, {ID: "layout", Title: "Layout playground", Description: "Arrange two panels, adjust padding and gaps, and align their contents. Auto stacks narrow displays. P and G show padding and gap in cells.", Source: "https://github.com/NimbleTerminal/TinyBubbles/tree/main/web/hello/layout.go"}, {ID: "settings", Title: "Settings", Description: "Compose a menu, volume, on/off and playback choices, and a reset dialog.", Source: "https://github.com/NimbleTerminal/TinyBubbles/tree/main/bubbletea/settings"}, {ID: "list", Title: "Scrolling menu", Description: "A small-screen list with selection, disabled items, and wraparound navigation.", Source: "https://github.com/NimbleTerminal/TinyBubbles/tree/main/bubbletea/list"}, {ID: "value", Title: "Value picker", Description: "Adjust a bounded setting, apply it, or cancel changes.", Source: "https://github.com/NimbleTerminal/TinyBubbles/tree/main/bubbletea/value"}, {ID: "viewport", Title: "Text viewport", Description: "Scroll instructions, device information, and logs on a small display.", Source: "https://github.com/NimbleTerminal/TinyBubbles/tree/main/widgets/viewport"}, {ID: "status", Title: "Status indicator", Description: "Compact device states with symbols, labels, and optional color.", Source: "https://github.com/NimbleTerminal/TinyBubbles/tree/main/widgets/status"}, {ID: "progress", Title: "Progress bar", Description: "Track completed work or show activity when the total is unknown.", Source: "https://github.com/NimbleTerminal/TinyBubbles/tree/main/widgets/progress"}, {ID: "dialog", Title: "Confirmation dialog", Description: "A small-screen prompt with a safe default and one result per decision.", Source: "https://github.com/NimbleTerminal/TinyBubbles/tree/main/bubbletea/dialog"}},
	})
	display := browser.NewCanvasDisplay("display", selected.Screen.Width, selected.Screen.Height)
	if selected.Runtime.ID == "tinytea" {
		browser.Stopped(runTiny(display, selected.Screen.Monochrome))
		select {}
	}
	newDemo := demo.NewDemo
	switch selected.Demo.ID {
	case "list":
		newDemo = demo.NewListDemo
	case "value":
		newDemo = demo.NewValueDemo
	case "layout":
		newDemo = demo.NewLayoutDemo
	case "settings":
		newDemo = demo.NewSettingsDemo
	case "viewport":
		newDemo = demo.NewViewportDemo
	case "status":
		newDemo = demo.NewStatusDemo
	case "progress":
		newDemo = demo.NewProgressDemo
	case "dialog":
		newDemo = demo.NewDialogDemo
	}
	model, pixels, err := newDemo(display, selected.Screen.Monochrome)
	if err != nil {
		browser.Stopped(err)
		select {}
	}
	cols, rows := pixels.Size()
	browser.Run(model, browser.RunOptions{Program: []tea.ProgramOption{tea.WithContext(pixels.Context()), tea.WithWindowSize(cols, rows)}, Error: func() error { pixels.Close(); return pixels.Err() }})
}

// JS callbacks enqueue input. Only this loop operates TinyTea.
func runTiny(display *browser.CanvasDisplay, mono bool) error {
	input := make(chan string, 64)
	var mu sync.Mutex
	output := actions.QueryProgramStatus
	session, err := demo.NewTinyDemo(display, mono, func(wire string) { mu.Lock(); output += wire; mu.Unlock() })
	if err != nil {
		return err
	}
	write := js.FuncOf(func(_ js.Value, args []js.Value) any {
		if len(args) > 0 {
			select {
			case input <- args[0].String():
			default:
			}
		}
		return nil
	})
	read := js.FuncOf(func(js.Value, []js.Value) any {
		mu.Lock()
		defer mu.Unlock()
		result := output
		output = ""
		return result
	})
	resize := js.FuncOf(func(js.Value, []js.Value) any { return nil })
	js.Global().Set("bubbletea_write", write)
	js.Global().Set("bubbletea_read", read)
	js.Global().Set("bubbletea_resize", resize)
	defer write.Release()
	defer read.Release()
	defer resize.Release()
	if err := session.Program.Start(); err != nil {
		return err
	}
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for !session.Program.Stopped() {
		select {
		case value := <-input:
			err = session.Input(value)
		case <-ticker.C:
			err = session.Tick()
		}
		if err != nil {
			return err
		}
	}
	return nil
}
