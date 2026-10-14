//go:build !tinygo || baremetal

package main

import (
	"fmt"

	bubbles "nimbleterminal.dev/tinybubbles"
	"nimbleterminal.dev/tinybubbles/tinyhal"
	"nimbleterminal.dev/tinybubbles/tinytea"
	"tinygo.org/x/drivers/ssd1306"
)

type hostModel struct {
	presses int
	regVal  byte
}

func (m hostModel) Init() tinytea.Cmd { return nil }

func (m hostModel) Update(msg tinytea.Msg) (tinytea.Model, tinytea.Cmd) {
	if k, ok := msg.(tinytea.KeyPressMsg); ok && k.Code == tinytea.KeyEnter {
		m.presses++
	}
	return m, nil
}

func (m hostModel) View() tinytea.View {
	return tinytea.NewView(fmt.Sprintf("tinyhal press=%d spi=0x%02x", m.presses, m.regVal))
}

func main() {
	hal := tinyhal.New()

	const (
		pinButtonD0  tinyhal.Pin = 0
		pinStatusLED tinyhal.Pin = 13
		pinSPICS     tinyhal.Pin = 10
	)

	led := tinyhal.AttachLED(hal, tinyhal.LEDConfig{ID: "status", Pin: pinStatusLED})
	tinyhal.AttachButton(hal, tinyhal.ButtonConfig{ID: "d0", Pin: pinButtonD0, ActiveLow: true})
	spiSensor := tinyhal.NewSPIRegisterDevice(pinSPICS)
	hal.AttachSPI(0, spiSensor)
	oled := tinyhal.AttachSSD1306(hal, tinyhal.SSD1306Config{ID: "oled", Bus: 0, Address: 0x3c, Width: 128, Height: 64})

	hal.GPIOConfigure(pinStatusLED, tinyhal.PinConfig{Mode: tinyhal.PinOutput})
	hal.GPIOConfigure(pinButtonD0, tinyhal.PinConfig{Mode: tinyhal.PinInputPullup})
	hal.GPIOConfigure(pinSPICS, tinyhal.PinConfig{Mode: tinyhal.PinOutput})
	hal.GPIOSet(pinSPICS, true)

	hal.GPIOSet(pinSPICS, false)
	hal.SPITx(0, []byte{0x80 | 0x01, 0xa5}, nil)
	rx := make([]byte, 2)
	hal.SPITx(0, []byte{0x01, 0x00}, rx)
	hal.GPIOSet(pinSPICS, true)

	display := ssd1306.NewI2C(hal.I2C(0))
	display.Configure(ssd1306.Config{Width: 128, Height: 64, Address: 0x3c})

	prog, err := tinytea.NewProgram(
		hostModel{regVal: rx[1]},
		display,
		bubbles.DefaultFont(),
		bubbles.DefaultTheme(true),
	)
	if err != nil {
		panic(err)
	}
	if err := prog.Start(); err != nil {
		panic(err)
	}

	_ = hal.InjectInput(tinyhal.InputEvent{Kind: tinyhal.InputButton, ID: "d0", Pressed: true})
	if !hal.GPIOGet(pinButtonD0) {
		hal.GPIOSet(pinStatusLED, true)
		_ = prog.Send(tinytea.KeyPressMsg{Code: tinytea.KeyEnter})
	}
	_ = hal.InjectInput(tinyhal.InputEvent{Kind: tinyhal.InputButton, ID: "d0", Pressed: false})

	fmt.Printf("tinyhal host demo complete: led_on=%v spi_reg=0x%02x oled_frames=%d events=%d\n",
		led.On(), spiSensor.Register(0x01), oled.Frames(), len(hal.DrainEvents()))
}
