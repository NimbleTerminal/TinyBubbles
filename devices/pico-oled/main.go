//go:build tinygo && (pico || pico2)

// Pico/Pico 2 with an external 128x64 I2C SSD1306 and two active-low buttons.
package main

import (
	"machine"
	bubbles "nimbleterminal.dev/tinybubbles"
	"nimbleterminal.dev/tinybubbles/demo"
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
	poller := bubbles.ButtonPoller{Buttons: []bubbles.Button{
		{Key: bubbles.KeyLeft, Debounce: 25 * time.Millisecond},
		{Key: bubbles.KeyRight, Debounce: 25 * time.Millisecond},
	}}
	pins := [2]machine.Pin{machine.GP14, machine.GP15}
	pressed := func(i int) bool { return !pins[i].Get() }
	input := func() bubbles.Msg { return poller.Poll(time.Now(), pressed) }
	p, err := bubbles.NewProgram(demo.New(), bubbles.Config{Display: display, Monochrome: true, Input: input})
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
