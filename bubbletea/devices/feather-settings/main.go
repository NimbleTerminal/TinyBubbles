//go:build tinygo && feather_reverse_tft

// Adafruit product 5691: ESP32-S3 Reverse TFT Feather, 240x135 ST7789.
// Three-button settings menu: D0 up/decrease (hold to repeat), D1 down/increase
// (hold to repeat), D2 short-tap select/apply and long-press cancel.
package main

import (
	"time"

	bubbles "nimbleterminal.dev/tinybubbles"
	displaytea "nimbleterminal.dev/tinybubbles/bubbletea"
	"nimbleterminal.dev/tinybubbles/bubbletea/settings"
	"nimbleterminal.dev/tinybubbles/devices/feather-reverse-tft/board"
)

func main() {
	device, err := board.New()
	if err != nil {
		panic(err)
	}
	buttons := settings.FeatherButtons()
	poller := bubbles.ButtonPoller{Buttons: buttons[:]}
	p, err := displaytea.NewProgram(settings.New(), device.Display, bubbles.DefaultFont(), bubbles.DefaultTheme(false))
	if err != nil {
		panic(err)
	}
	// Only this goroutine touches GPIO; Bubble Tea serializes model updates.
	done := make(chan struct{})
	go func() {
		for {
			select {
			case <-done:
				return
			default:
			}
			if msg := poller.Poll(time.Now(), device.Pressed); msg != nil {
				p.Send(msg)
			}
			time.Sleep(5 * time.Millisecond)
		}
	}()
	println("Starting Bubble Tea v2.1 settings on Displayer")
	_, err = p.Run()
	close(done)
	if err != nil {
		panic(err)
	}
}
