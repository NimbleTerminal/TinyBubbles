package tinyhal

import (
	"bytes"
	"image/color"
	"testing"

	bubbles "nimbleterminal.dev/tinybubbles"
	"nimbleterminal.dev/tinybubbles/tinytea"
	"tinygo.org/x/drivers/ssd1306"
)

type recordLEDWriter struct {
	colors []color.RGBA
}

func (w *recordLEDWriter) WriteColors(colors []color.RGBA) error {
	w.colors = append(w.colors[:0], colors...)
	return nil
}

type recordDisplayer struct {
	w, h    int16
	flushes int
	pixels  map[[2]int16]color.RGBA
}

func newRecordDisplayer(w, h int16) *recordDisplayer {
	return &recordDisplayer{
		w:      w,
		h:      h,
		pixels: make(map[[2]int16]color.RGBA),
	}
}

func (d *recordDisplayer) Size() (int16, int16) { return d.w, d.h }

func (d *recordDisplayer) SetPixel(x, y int16, c color.RGBA) {
	if c.R != 0 || c.G != 0 || c.B != 0 {
		d.pixels[[2]int16{x, y}] = c
	} else {
		delete(d.pixels, [2]int16{x, y})
	}
}

func (d *recordDisplayer) Display() error {
	d.flushes++
	return nil
}

func TestGPIOPullsAndLED(t *testing.T) {
	h := New()

	// Pull-up input defaults to high; pull-down defaults to low.
	const inUp Pin = 4
	const inDown Pin = 5
	h.GPIOConfigure(inUp, PinConfig{Mode: PinInputPullup})
	h.GPIOConfigure(inDown, PinConfig{Mode: PinInputPulldown})
	if !h.GPIOGet(inUp) {
		t.Fatalf("expected pull-up pin %d to read high", inUp)
	}
	if h.GPIOGet(inDown) {
		t.Fatalf("expected pull-down pin %d to read low", inDown)
	}

	// External input overrides pull-up until cleared.
	h.SetPinInput(inUp, false)
	if h.GPIOGet(inUp) {
		t.Fatalf("expected external low input on pin %d", inUp)
	}
	h.ClearPinInput(inUp)
	if !h.GPIOGet(inUp) {
		t.Fatalf("expected pull-up pin %d to return high after ClearPinInput", inUp)
	}

	// Attach an LED with a writer and verify digital and PWM updates.
	writer := &recordLEDWriter{}
	const ledPin Pin = 13
	led := AttachLED(h, LEDConfig{
		ID:     "status",
		Pin:    ledPin,
		Color:  color.RGBA{R: 200, G: 100, B: 50, A: 255},
		Writer: writer,
		Index:  0,
	})
	h.GPIOConfigure(ledPin, PinConfig{Mode: PinOutput})
	h.GPIOSet(ledPin, true)
	if !led.On() || led.Brightness() != 255 {
		t.Fatalf("expected LED on at full brightness, got on=%v brightness=%d", led.On(), led.Brightness())
	}
	if len(writer.colors) != 1 || writer.colors[0].R != 200 {
		t.Fatalf("unexpected LED writer colors: %+v", writer.colors)
	}

	// Drive the same LED pin with 50% PWM duty cycle.
	h.PWMConfigure(0, 1000, 1000)
	h.PWMChannelConfigure(0, 1, ledPin)
	h.PWMChannelSet(0, 1, 500)
	if !led.On() || led.Brightness() != 127 {
		t.Fatalf("expected PWM brightness 127, got %d", led.Brightness())
	}
	if c := led.CurrentColor(); c.R == 0 || c.R >= 200 {
		t.Fatalf("expected scaled color between 1 and 199, got %+v", c)
	}

	h.GPIOSet(ledPin, false)
	if led.On() || led.Brightness() != 0 {
		t.Fatalf("expected LED off, got on=%v brightness=%d", led.On(), led.Brightness())
	}
}

func TestButtonAndHostInputInjection(t *testing.T) {
	h := New()
	const btnPin Pin = 0

	// Active-low button (like Feather D0): released=high, pressed=low.
	h.GPIOConfigure(btnPin, PinConfig{Mode: PinInputPullup})
	btn := AttachButton(h, ButtonConfig{
		ID:        "d0",
		Pin:       btnPin,
		ActiveLow: true,
	})
	if !h.GPIOGet(btnPin) || btn.Pressed() {
		t.Fatalf("expected released active-low button pin to read high")
	}

	if err := h.InjectInput(InputEvent{Kind: InputButton, ID: "d0", Pressed: true}); err != nil {
		t.Fatalf("InjectInput failed: %v", err)
	}
	if h.GPIOGet(btnPin) || !btn.Pressed() {
		t.Fatalf("expected pressed active-low button pin to read low")
	}

	btn.Release()
	if !h.GPIOGet(btnPin) || btn.Pressed() {
		t.Fatalf("expected released button pin to return high")
	}

	// ADC and UART host input injection.
	if err := h.InjectInput(InputEvent{Kind: InputADC, Pin: 26, Value: 4095}); err != nil {
		t.Fatalf("ADC InjectInput failed: %v", err)
	}
	if got := h.ADCRead(26); got != 4095 {
		t.Fatalf("expected ADC reading 4095, got %d", got)
	}

	h.UARTConfigure(0, UARTConfig{TX: 1, RX: 2})
	if err := h.InjectInput(InputEvent{Kind: InputUART, Bus: 0, Data: []byte("ping")}); err != nil {
		t.Fatalf("UART InjectInput failed: %v", err)
	}
	rx := make([]byte, 8)
	n := h.UARTRead(0, rx)
	if string(rx[:n]) != "ping" {
		t.Fatalf("expected UART RX %q, got %q", "ping", string(rx[:n]))
	}
	h.UARTWrite(0, []byte("pong"))
	if !bytes.Equal(h.UARTTX(0), []byte("pong")) {
		t.Fatalf("expected UART TX %q, got %q", "pong", string(h.UARTTX(0)))
	}
}

func TestSPIRegisterDeviceWithCS(t *testing.T) {
	h := New()
	const csPin Pin = 10
	h.GPIOConfigure(csPin, PinConfig{Mode: PinOutput})
	h.GPIOSet(csPin, true) // deselected

	dev := NewSPIRegisterDevice(csPin)
	h.AttachSPI(0, dev)
	h.SPIConfigure(0, SPIConfig{Frequency: 8_000_000, SCK: 18, SDO: 19, SDI: 16})

	// Transaction while CS is high must be ignored.
	h.SPITx(0, []byte{0x80 | 0x05, 0x42, 0x99}, nil)
	if dev.Transactions() != 0 || dev.Register(0x05) != 0 {
		t.Fatalf("expected SPI device to ignore transfer when CS is high")
	}

	// Select CS low, write registers 0x05 and 0x06, then deselect.
	h.GPIOSet(csPin, false)
	h.SPITx(0, []byte{0x80 | 0x05, 0x42, 0x99}, nil)
	h.GPIOSet(csPin, true)

	if dev.Register(0x05) != 0x42 || dev.Register(0x06) != 0x99 {
		t.Fatalf("unexpected register values: reg[5]=0x%02x reg[6]=0x%02x", dev.Register(0x05), dev.Register(0x06))
	}

	// Read back registers 0x05 and 0x06 over MISO.
	rx := make([]byte, 3)
	h.GPIOSet(csPin, false)
	h.SPITx(0, []byte{0x05, 0x00, 0x00}, rx)
	h.GPIOSet(csPin, true)

	if rx[1] != 0x42 || rx[2] != 0x99 {
		t.Fatalf("expected MISO readback [_, 0x42, 0x99], got %v", rx)
	}
}

func TestVirtualST7789SPIProtocol(t *testing.T) {
	h := New()
	const (
		csPin  Pin = 42
		dcPin  Pin = 40
		rstPin Pin = 41
		blPin  Pin = 45
	)
	for _, p := range []Pin{csPin, dcPin, rstPin, blPin} {
		h.GPIOConfigure(p, PinConfig{Mode: PinOutput})
	}
	h.GPIOSet(csPin, true)
	h.GPIOSet(rstPin, true)
	h.GPIOSet(blPin, true)

	target := newRecordDisplayer(240, 135)
	panel := AttachST7789(h, ST7789Config{
		ID:           "tft",
		Bus:          0,
		Width:        240,
		Height:       135,
		RowOffset:    40,
		ColumnOffset: 53,
		CS:           csPin,
		DC:           dcPin,
		Reset:        rstPin,
		Backlight:    blPin,
		Target:       target,
	})

	sendCmd := func(cmd byte, data ...byte) {
		h.GPIOSet(dcPin, false)
		h.SPITx(0, []byte{cmd}, nil)
		h.GPIOSet(dcPin, true)
		if len(data) > 0 {
			h.SPITx(0, data, nil)
		}
	}

	// Configure MADCTL with MV (Rotation90, which swaps RowOffset=40 into X and ColumnOffset=53 into Y)
	// and write a 2x1 red/blue window at logical (10, 20) -> wire CASET (50..51), RASET (73..73).
	h.GPIOSet(csPin, false)
	sendCmd(st7789SLPOUT)
	sendCmd(st7789COLMOD, 0x55) // RGB565
	sendCmd(st7789MADCTL, st7789MADCTLMV)
	sendCmd(st7789CASET, 0x00, 50, 0x00, 51)
	sendCmd(st7789RASET, 0x00, 73, 0x00, 73)
	sendCmd(st7789RAMWR,
		0xF8, 0x00, // pure red in RGB565BE
		0x00, 0x1F, // pure blue in RGB565BE
	)
	sendCmd(st7789DISPON)
	h.GPIOSet(csPin, true)

	if panel.Frames() != 1 {
		t.Fatalf("expected 1 flushed frame on CS rising edge, got %d", panel.Frames())
	}
	if got := panel.Pixel(10, 20); got.R != 255 || got.G != 0 || got.B != 0 {
		t.Fatalf("expected red pixel at (10,20), got %+v", got)
	}
	if got := panel.Pixel(11, 20); got.R != 0 || got.G != 0 || got.B != 255 {
		t.Fatalf("expected blue pixel at (11,20), got %+v", got)
	}
	if target.flushes != 1 || len(target.pixels) != 2 {
		t.Fatalf("expected target displayer to receive 1 flush and 2 lit pixels, got flushes=%d pixels=%d", target.flushes, len(target.pixels))
	}
}

type tinyTeaStatusModel struct {
	presses int
}

func (m tinyTeaStatusModel) Init() tinytea.Cmd { return nil }

func (m tinyTeaStatusModel) Update(msg tinytea.Msg) (tinytea.Model, tinytea.Cmd) {
	if k, ok := msg.(tinytea.KeyPressMsg); ok && k.Code == tinytea.KeyEnter {
		m.presses++
	}
	return m, nil
}

func (m tinyTeaStatusModel) View() tinytea.View {
	if m.presses > 0 {
		return tinytea.NewView("HAL OK")
	}
	return tinytea.NewView("READY")
}

func TestVirtualSSD1306WithTinyTea(t *testing.T) {
	h := New()
	target := newRecordDisplayer(128, 64)
	oled := AttachSSD1306(h, SSD1306Config{
		ID:      "oled",
		Bus:     0,
		Address: 0x3c,
		Width:   128,
		Height:  64,
		Target:  target,
	})

	// Use tinygo.org/x/drivers/ssd1306 directly over tinyhal's I2C(0) bus.
	drv := ssd1306.NewI2C(h.I2C(0))
	drv.Configure(ssd1306.Config{Width: 128, Height: 64, Address: 0x3c})

	var fbEvents int
	h.OnEvent(func(ev Event) {
		if ev.Kind == EventFramebuffer && ev.ID == "oled" {
			fbEvents++
		}
	})

	prog, err := tinytea.NewProgram(
		tinyTeaStatusModel{},
		drv,
		bubbles.DefaultFont(),
		bubbles.DefaultTheme(true),
	)
	if err != nil {
		t.Fatalf("tinytea.NewProgram failed: %v", err)
	}
	if err := prog.Start(); err != nil {
		t.Fatalf("prog.Start failed: %v", err)
	}
	if err := prog.Send(tinytea.KeyPressMsg{Code: tinytea.KeyEnter}); err != nil {
		t.Fatalf("prog.Send failed: %v", err)
	}

	if oled.Frames() < 2 || fbEvents < 2 {
		t.Fatalf("expected at least 2 SSD1306 frame flushes, got frames=%d events=%d", oled.Frames(), fbEvents)
	}
	if len(target.pixels) == 0 {
		t.Fatalf("expected TinyTea text rendering to light pixels on virtual SSD1306")
	}
}
