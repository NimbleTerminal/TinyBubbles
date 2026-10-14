//go:build tinygo && !baremetal

// Command tinyhal-demo runs embedded TinyGo machine + st7789 + tinytea code
// against the in-WASM tinyhal hardware abstraction layer.
//
// Run with WASI:
//
//	tinygo run -target=wasip1 ./tinyhal/cmd/tinyhal-demo
//
// Build for browser WASM:
//
//	tinygo build -target=wasm -o build/tinyhal-demo.wasm ./tinyhal/cmd/tinyhal-demo
package main

import (
	"fmt"
	"machine"

	bubbles "nimbleterminal.dev/tinybubbles"
	"nimbleterminal.dev/tinybubbles/tinyhal"
	"nimbleterminal.dev/tinybubbles/tinytea"
	"tinygo.org/x/drivers/st7789"
)

const (
	pinButtonD0  = machine.Pin(0)
	pinStatusLED = machine.Pin(13)
	pinSPICS     = machine.Pin(10)
	pinSPISCK    = machine.Pin(36)
	pinSPISDO    = machine.Pin(35)
	pinTFTDC     = machine.Pin(40)
	pinTFTReset  = machine.Pin(41)
	pinTFTCS     = machine.Pin(42)
	pinTFTLite   = machine.Pin(45)
)

type demoModel struct {
	presses int
	regVal  byte
}

func (m demoModel) Init() tinytea.Cmd { return nil }

func (m demoModel) Update(msg tinytea.Msg) (tinytea.Model, tinytea.Cmd) {
	if k, ok := msg.(tinytea.KeyPressMsg); ok && k.Code == tinytea.KeyEnter {
		m.presses++
	}
	return m, nil
}

func (m demoModel) View() tinytea.View {
	return tinytea.NewView(fmt.Sprintf("tinyhal press=%d spi=0x%02x", m.presses, m.regVal))
}

func main() {
	hal := tinyhal.Default()

	// 1. Register virtual peripherals on the HAL netlist before running machine code.
	led := tinyhal.AttachLED(hal, tinyhal.LEDConfig{
		ID:  "status",
		Pin: tinyhal.Pin(pinStatusLED),
	})
	tinyhal.AttachButton(hal, tinyhal.ButtonConfig{
		ID:        "d0",
		Pin:       tinyhal.Pin(pinButtonD0),
		ActiveLow: true,
	})
	spiSensor := tinyhal.NewSPIRegisterDevice(tinyhal.Pin(pinSPICS))
	hal.AttachSPI(0, spiSensor)

	tft := tinyhal.AttachST7789(hal, tinyhal.ST7789Config{
		ID:        "tft",
		Bus:       0,
		Width:     240,
		Height:    135,
		Reset:     tinyhal.Pin(pinTFTReset),
		DC:        tinyhal.Pin(pinTFTDC),
		CS:        tinyhal.Pin(pinTFTCS),
		Backlight: tinyhal.Pin(pinTFTLite),
	})

	// 2. Standard TinyGo embedded code using machine.Pin and machine.SPI0.
	pinStatusLED.Configure(machine.PinConfig{Mode: machine.PinOutput})
	pinButtonD0.Configure(machine.PinConfig{Mode: machine.PinInputPullup})
	pinSPICS.Configure(machine.PinConfig{Mode: machine.PinOutput})
	pinSPICS.High()

	if err := machine.SPI0.Configure(machine.SPIConfig{
		Frequency: 20_000_000,
		SCK:       pinSPISCK,
		SDO:       pinSPISDO,
		SDI:       machine.NoPin,
	}); err != nil {
		panic(err)
	}

	// Write and read back a register on the SPI peripheral using machine.SPI0.
	pinSPICS.Low()
	_ = machine.SPI0.Tx([]byte{0x80 | 0x01, 0xa5}, nil)
	rx := make([]byte, 2)
	_ = machine.SPI0.Tx([]byte{0x01, 0x00}, rx)
	pinSPICS.High()

	// Configure a real tinygo.org/x/drivers/st7789 display over machine.SPI0.
	display := st7789.New(machine.SPI0, pinTFTReset, pinTFTDC, pinTFTCS, pinTFTLite)
	display.Configure(st7789.Config{Width: 240, Height: 135})

	// 3. Coexist with TinyTea rendering onto the ST7789 SPI display.
	prog, err := tinytea.NewProgram(
		demoModel{regVal: rx[1]},
		&display,
		bubbles.DefaultFont(),
		bubbles.DefaultTheme(false),
	)
	if err != nil {
		panic(err)
	}
	if err := prog.Start(); err != nil {
		panic(err)
	}

	// 4. Inject a simulated host button press and poll the GPIO pin via machine.Pin.Get().
	_ = hal.InjectInput(tinyhal.InputEvent{
		Kind:    tinyhal.InputButton,
		ID:      "d0",
		Pressed: true,
	})
	if !pinButtonD0.Get() {
		pinStatusLED.High()
		_ = prog.Send(tinytea.KeyPressMsg{Code: tinytea.KeyEnter})
	}
	_ = hal.InjectInput(tinyhal.InputEvent{
		Kind:    tinyhal.InputButton,
		ID:      "d0",
		Pressed: false,
	})

	fmt.Printf("tinyhal demo complete: led_on=%v spi_reg=0x%02x tft_frames=%d events=%d\n",
		led.On(), spiSensor.Register(0x01), tft.Frames(), len(hal.DrainEvents()))
}
