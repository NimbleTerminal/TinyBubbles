//go:build tinygo && (pico || pico2)

// Pico/Pico 2 with an external 128x64 I2C SSD1306 and two active-low buttons.
package main

import (
	bubbles "github.com/NimbleTerminal/TinyBubbles"
	"github.com/NimbleTerminal/TinyBubbles/demo"
	"machine"
	"time"
	"tinygo.org/x/drivers/ssd1306"
)

func main() {
	if err := machine.I2C0.Configure(machine.I2CConfig{Frequency: 400000, SDA: machine.GP4, SCL: machine.GP5}); err != nil {
		panic(err)
	}
	display := ssd1306.NewI2C(machine.I2C0)
	display.Configure(ssd1306.Config{Width: 128, Height: 64, Address: 0x3c})
	machine.GP14.Configure(machine.PinConfig{Mode: machine.PinInputPullup})
	machine.GP15.Configure(machine.PinConfig{Mode: machine.PinInputPullup})
	left := bubbles.Button{Key: bubbles.KeyLeft, Debounce: 25 * time.Millisecond}
	right := bubbles.Button{Key: bubbles.KeyRight, Debounce: 25 * time.Millisecond}
	// Rotate polling priority so simultaneous presses are both delivered; a
	// button is not sampled/consumed until its message can be returned.
	nextRight := false
	input := func() bubbles.Msg {
		now := time.Now()
		if nextRight {
			if msg, ok := right.Update(!machine.GP15.Get(), now); ok {
				nextRight = false
				return msg
			}
			if msg, ok := left.Update(!machine.GP14.Get(), now); ok {
				nextRight = true
				return msg
			}
		} else {
			if msg, ok := left.Update(!machine.GP14.Get(), now); ok {
				nextRight = true
				return msg
			}
			if msg, ok := right.Update(!machine.GP15.Get(), now); ok {
				nextRight = false
				return msg
			}
		}
		return nil
	}
	p, err := bubbles.NewProgram(demo.New(), bubbles.Config{Display: display, Monochrome: true, Input: input})
	if err != nil {
		panic(err)
	}
	if err = p.Run(); err != nil {
		panic(err)
	}
}
