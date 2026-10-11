//go:build tinygo && clue_alpha

// Adafruit CLUE (nRF52840), built-in 240x240 ST7789 and A/B buttons.
package main

import (
	"machine"
	bubbles "nimbleterminal.dev/tinybubbles"
	"nimbleterminal.dev/tinybubbles/demo"
	"time"
	"tinygo.org/x/drivers/st7789"
)

func main() {
	if err := machine.SPI1.Configure(machine.SPIConfig{Frequency: 8000000, SCK: machine.TFT_SCK, SDO: machine.TFT_SDO, SDI: machine.NoPin, Mode: 0}); err != nil {
		panic(err)
	}
	display := st7789.New(machine.SPI1, machine.TFT_RESET, machine.TFT_DC, machine.TFT_CS, machine.TFT_LITE)
	// Match the CLUE's native MADCTL=0xa0 and 80-column panel offset:
	// https://github.com/adafruit/circuitpython/blob/main/ports/nordic/boards/clue_nrf52840_express/board.c
	display.Configure(st7789.Config{Width: 240, Height: 240, Rotation: st7789.ROTATION_270, RowOffset: 80})
	machine.TFT_LITE.High()
	machine.BUTTON_LEFT.Configure(machine.PinConfig{Mode: machine.PinInputPullup})
	machine.BUTTON_RIGHT.Configure(machine.PinConfig{Mode: machine.PinInputPullup})
	left := bubbles.Button{Key: bubbles.KeyLeft, Debounce: 25 * time.Millisecond}
	right := bubbles.Button{Key: bubbles.KeyRight, Debounce: 25 * time.Millisecond}
	nextRight := false
	input := func() bubbles.Msg {
		now := time.Now()
		if nextRight {
			if msg, ok := right.Update(!machine.BUTTON_RIGHT.Get(), now); ok {
				nextRight = false
				return msg
			}
			if msg, ok := left.Update(!machine.BUTTON_LEFT.Get(), now); ok {
				nextRight = true
				return msg
			}
		} else {
			if msg, ok := left.Update(!machine.BUTTON_LEFT.Get(), now); ok {
				nextRight = true
				return msg
			}
			if msg, ok := right.Update(!machine.BUTTON_RIGHT.Get(), now); ok {
				nextRight = false
				return msg
			}
		}
		return nil
	}
	p, err := bubbles.NewProgram(demo.New(), bubbles.Config{Display: &display, Input: input})
	if err != nil {
		panic(err)
	}
	if err = p.Run(); err != nil {
		panic(err)
	}
}
