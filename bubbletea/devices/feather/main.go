//go:build tinygo && feather_reverse_tft

// Adafruit product 5691: ESP32-S3 Reverse TFT Feather, 240x135 ST7789.
package main

import (
	tea "charm.land/bubbletea/v2"
	bubbles "nimbleterminal.dev/tinybubbles"
	displaytea "nimbleterminal.dev/tinybubbles/bubbletea"
	teamodel "nimbleterminal.dev/tinybubbles/bubbletea/hello"
	"nimbleterminal.dev/tinybubbles/demo/hello"
	"nimbleterminal.dev/tinybubbles/devices/feather-reverse-tft/board"
	"time"
)

func main() {
	device, err := board.New()
	if err != nil {
		panic(err)
	}
	buttons := [3]bubbles.Button{
		{Key: bubbles.KeySelect, Debounce: 25 * time.Millisecond},
		{Key: bubbles.KeyRight, Debounce: 25 * time.Millisecond},
		{Key: bubbles.KeyBack, Debounce: 25 * time.Millisecond},
	}
	p, err := displaytea.NewProgram(teamodel.New(), device.Display, hello.NewFont(), hello.Theme())
	if err != nil {
		panic(err)
	}
	// Only this goroutine touches GPIO; Bubble Tea serializes model updates.
	done := make(chan struct{})
	go func() {
		keys := [3]rune{tea.KeyEnter, tea.KeyRight, tea.KeyEscape}
		for {
			select {
			case <-done:
				return
			default:
			}
			now := time.Now()
			for i := range buttons {
				pressed := device.Pressed(i)
				if _, ok := buttons[i].Update(pressed, now); ok {
					p.Send(tea.KeyPressMsg{Code: keys[i]})
				}
			}
			time.Sleep(5 * time.Millisecond)
		}
	}()
	println("Starting Bubble Tea v2.1 on Displayer")
	_, err = p.Run()
	close(done)
	if err != nil {
		panic(err)
	}
}
