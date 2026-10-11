//go:build tinygo && feather_reverse_tft

// Adafruit product 5691: ESP32-S3 Reverse TFT Feather, 240x135 ST7789.
package main

import (
	bubbles "nimbleterminal.dev/tinybubbles"
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
	// Four round-robin sources (three buttons plus timer) prevent button
	// bursts starving animation and preserve simultaneous button presses.
	cursor := 0
	nextFrame := time.Now().Add(100 * time.Millisecond)
	poll := func() bubbles.Msg {
		now := time.Now()
		for n := 0; n < 4; n++ {
			i := (cursor + n) % 4
			if i == 3 {
				if !now.Before(nextFrame) {
					nextFrame = now.Add(100 * time.Millisecond)
					cursor = 0
					return hello.TickMsg{}
				}
				continue
			}
			pressed := device.Pressed(i)
			if msg, ok := buttons[i].Update(pressed, now); ok {
				cursor = (i + 1) % 4
				return msg
			}
		}
		return nil
	}
	theme := hello.Theme()
	p, err := bubbles.NewProgram(hello.New(), bubbles.Config{Display: device.Display, Font: hello.NewFont(), Theme: &theme, Input: poll})
	if err != nil {
		panic(err)
	}
	// Retry transient bus failures on the next frame; keep polling buttons.
	for !p.Stopped() {
		if err := p.Step(); err != nil {
			if _, retry := err.(*bubbles.DisplayError); !retry {
				panic(err)
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
}
