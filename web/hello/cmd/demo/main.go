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
		Title:       "TinyBubbles",
		Source:      "https://github.com/NimbleTerminal/TinyBubbles",
		Description: "TinyGo compiles Go for microcontrollers. Bubble Tea is a Go framework for interactive terminal apps. TinyBubbles brings Bubble Tea to small embedded displays, so you can try a Go interface in your browser and run the same UI on hardware.",
		Screens: []browser.Screen{
			{ID: "feather", Label: "Feather TFT · 240 × 135", Width: 240, Height: 135},
			{ID: "oled", Label: "Monochrome OLED · 128 × 64", Width: 128, Height: 64, Monochrome: true},
			{ID: "large", Label: "Large TFT · 320 × 240", Width: 320, Height: 240},
		},
		Demos: []browser.Demo{{ID: "hello", Title: "Hello, tiny world", Description: "Animation, custom fonts, and a counter on a tiny display.", Source: "https://github.com/NimbleTerminal/TinyBubbles/tree/main/bubbletea/hello", Runtimes: []browser.Runtime{
			{ID: "bubbletea", Label: "Bubble Tea", Description: "Commands and timers run through Bubble Tea’s event loop."},
			{ID: "tinytea", Label: "TinyTea", Description: "Cooperative commands; the host supplies animation ticks."},
		}}},
	})
	display := browser.NewCanvasDisplay("display", selected.Screen.Width, selected.Screen.Height)
	if selected.Runtime.ID == "tinytea" {
		browser.Stopped(runTiny(display, selected.Screen.Monochrome))
		select {}
	}
	model, pixels, err := demo.NewDemo(display, selected.Screen.Monochrome)
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
