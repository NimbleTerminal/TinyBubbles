//go:build tinygo && feather_reverse_tft

// Adafruit product 5691: ESP32-S3 Reverse TFT Feather, 240x135 ST7789.
package main

import (
	bubbles "github.com/NimbleTerminal/TinyBubbles"
	"github.com/NimbleTerminal/TinyBubbles/demo/hello"
	"machine"
	"time"
	"tinygo.org/x/drivers/st7789"
)

// GPIO assignments and panel settings follow Adafruit's CircuitPython board:
// ports/espressif/boards/adafruit_feather_esp32s3_reverse_tft/{pins.c,board.c}.
const (
	tftPower     = machine.GPIO7
	tftClock     = machine.GPIO36
	tftMOSI      = machine.GPIO35
	tftCS        = machine.GPIO42
	tftDC        = machine.GPIO40
	tftReset     = machine.GPIO41
	tftBacklight = machine.GPIO45
)

func main() {
	tftPower.Configure(machine.PinConfig{Mode: machine.PinOutput})
	tftPower.High()
	time.Sleep(10 * time.Millisecond)
	if err := machine.SPI0.Configure(machine.SPIConfig{Frequency: 20000000, SCK: tftClock, SDO: tftMOSI, SDI: machine.NoPin, CS: machine.NoPin, Mode: 0}); err != nil {
		panic(err)
	}
	display := st7789.New(machine.SPI0, tftReset, tftDC, tftCS, tftBacklight)
	// IsBGR must precede Configure so MADCTL includes BGR. Rotation90 swaps
	// the native 135x240 size and offsets into 240x135, column 40 / row 53.
	display.IsBGR(true)
	display.Configure(st7789.Config{
		Width: 135, Height: 240, Rotation: st7789.ROTATION_90, RowOffset: 40, ColumnOffset: 53,
		PVGAMCTRL: []uint8{0xd0, 0x00, 0x02, 0x07, 0x0a, 0x28, 0x32, 0x44, 0x42, 0x06, 0x0e, 0x12, 0x14, 0x17},
		NVGAMCTRL: []uint8{0xd0, 0x00, 0x02, 0x07, 0x0a, 0x28, 0x31, 0x54, 0x47, 0x0e, 0x1c, 0x17, 0x1b, 0x1e},
	})
	pins := [3]machine.Pin{machine.GPIO0, machine.GPIO1, machine.GPIO2}
	pins[0].Configure(machine.PinConfig{Mode: machine.PinInputPullup})
	for _, pin := range pins[1:] {
		pin.Configure(machine.PinConfig{Mode: machine.PinInputPulldown})
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
			pressed := pins[i].Get()
			if i == 0 {
				pressed = !pressed
			}
			if msg, ok := buttons[i].Update(pressed, now); ok {
				cursor = (i + 1) % 4
				return msg
			}
		}
		return nil
	}
	theme := hello.Theme()
	p, err := bubbles.NewProgram(hello.New(), bubbles.Config{Display: &display, Font: hello.NewFont(), Theme: &theme, Input: poll})
	if err != nil {
		panic(err)
	}
	if err = p.Run(); err != nil {
		panic(err)
	}
}
