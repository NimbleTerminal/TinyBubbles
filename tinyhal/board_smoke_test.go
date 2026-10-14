package tinyhal

import (
	"image/color"
	"testing"
	"time"

	bubbles "nimbleterminal.dev/tinybubbles"
	"nimbleterminal.dev/tinybubbles/demo"
	"nimbleterminal.dev/tinybubbles/demo/hello"
	"tinygo.org/x/drivers/ssd1306"
)

// hostST7789Driver is a minimal host-side drivers.Displayer that speaks the
// ST7789 4-wire SPI protocol (CASET, RASET, RAMWR, MADCTL) over tinyhal's
// SPI and GPIO buses without importing TinyGo's machine package in go test.
type hostST7789Driver struct {
	hal               *HAL
	bus               uint8
	cs, dc, rst, bl   Pin
	width, height     int16
	colOffset, rowOff int16
	madctl            byte
	dirty             bool
}

func (d *hostST7789Driver) configure() {
	for _, p := range []Pin{d.cs, d.dc, d.rst, d.bl} {
		d.hal.GPIOConfigure(p, PinConfig{Mode: PinOutput})
	}
	d.hal.GPIOSet(d.cs, true)
	d.hal.GPIOSet(d.rst, true)
	d.hal.GPIOSet(d.bl, true)

	d.hal.GPIOSet(d.cs, false)
	d.cmd(st7789SLPOUT)
	d.cmd(st7789COLMOD, 0x55)
	d.cmd(st7789MADCTL, d.madctl)
	d.cmd(st7789DISPON)
	d.hal.GPIOSet(d.cs, true)
}

func (d *hostST7789Driver) cmd(c byte, data ...byte) {
	d.hal.GPIOSet(d.dc, false)
	d.hal.SPITx(d.bus, []byte{c}, nil)
	d.hal.GPIOSet(d.dc, true)
	if len(data) > 0 {
		d.hal.SPITx(d.bus, data, nil)
	}
}

func (d *hostST7789Driver) Size() (int16, int16) {
	return d.width, d.height
}

func (d *hostST7789Driver) SetPixel(x, y int16, c color.RGBA) {
	_ = d.FillRectangle(x, y, 1, 1, c)
}

func (d *hostST7789Driver) FillRectangle(x, y, width, height int16, c color.RGBA) error {
	if width <= 0 || height <= 0 {
		return nil
	}
	x0 := uint16(x + d.colOffset)
	x1 := uint16(x + width - 1 + d.colOffset)
	y0 := uint16(y + d.rowOff)
	y1 := uint16(y + height - 1 + d.rowOff)

	rgb565 := (uint16(c.R&0xF8) << 8) | (uint16(c.G&0xFC) << 3) | (uint16(c.B) >> 3)
	hi, lo := byte(rgb565>>8), byte(rgb565)
	n := int(width) * int(height)
	buf := make([]byte, n*2)
	for i := 0; i < len(buf); i += 2 {
		buf[i] = hi
		buf[i+1] = lo
	}

	d.hal.GPIOSet(d.cs, false)
	d.cmd(st7789CASET, byte(x0>>8), byte(x0), byte(x1>>8), byte(x1))
	d.cmd(st7789RASET, byte(y0>>8), byte(y0), byte(y1>>8), byte(y1))
	d.cmd(st7789RAMWR, buf...)
	d.hal.GPIOSet(d.cs, true)
	d.dirty = true
	return nil
}

func (d *hostST7789Driver) Display() error {
	return nil
}

// TestBoardSmokePicoOLED validates the devices/pico-oled setup against tinyhal:
// 128x64 I2C0 SSD1306 (0x3c), active-low GP14/GP15 buttons, ButtonPoller, and demo.New().
func TestBoardSmokePicoOLED(t *testing.T) {
	h := New()
	const (
		pinSDA  Pin = 4
		pinSCL  Pin = 5
		pinGP14 Pin = 14
		pinGP15 Pin = 15
	)

	target := newRecordDisplayer(128, 64)
	oled := AttachSSD1306(h, SSD1306Config{
		ID:      "pico-oled",
		Bus:     0,
		Address: 0x3c,
		Width:   128,
		Height:  64,
		Target:  target,
	})

	h.I2CConfigure(0, I2CConfig{Frequency: 400_000, SDA: pinSDA, SCL: pinSCL})
	display := ssd1306.NewI2C(h.I2C(0))
	display.Configure(ssd1306.Config{Width: 128, Height: 64, Address: 0x3c})

	h.GPIOConfigure(pinGP14, PinConfig{Mode: PinInputPullup})
	h.GPIOConfigure(pinGP15, PinConfig{Mode: PinInputPullup})
	AttachButton(h, ButtonConfig{ID: "left", Pin: pinGP14, ActiveLow: true})
	AttachButton(h, ButtonConfig{ID: "right", Pin: pinGP15, ActiveLow: true})

	poller := bubbles.ButtonPoller{Buttons: []bubbles.Button{
		{Key: bubbles.KeyLeft, Debounce: 25 * time.Millisecond},
		{Key: bubbles.KeyRight, Debounce: 25 * time.Millisecond},
	}}
	pins := [2]Pin{pinGP14, pinGP15}
	pressed := func(i int) bool { return !h.GPIOGet(pins[i]) }

	now := time.Now()
	input := func() bubbles.Msg { return poller.Poll(now, pressed) }

	p, err := bubbles.NewProgram(demo.New(), bubbles.Config{
		Display:    display,
		Monochrome: true,
		Input:      input,
	})
	if err != nil {
		t.Fatalf("bubbles.NewProgram failed: %v", err)
	}

	// Initial step renders frame 1.
	if err := p.Step(); err != nil {
		t.Fatalf("initial p.Step failed: %v", err)
	}

	// Press Right (GP15 active-low) and advance clock past debounce.
	_ = h.InjectInput(InputEvent{Kind: InputButton, ID: "right", Pressed: true})
	if err := p.Step(); err != nil {
		t.Fatalf("p.Step on button press failed: %v", err)
	}
	now = now.Add(30 * time.Millisecond)
	if err := p.Step(); err != nil {
		t.Fatalf("p.Step after debounce failed: %v", err)
	}
	_ = h.InjectInput(InputEvent{Kind: InputButton, ID: "right", Pressed: false})

	if oled.Frames() < 2 {
		t.Fatalf("expected at least 2 SSD1306 frame flushes on Pico setup, got %d", oled.Frames())
	}
	if len(target.pixels) == 0 {
		t.Fatalf("expected Pico OLED smoke test to light pixels")
	}
}

// TestBoardSmokeCLUE validates the devices/clue setup against tinyhal:
// 240x240 SPI1 ST7789 (Rotation270 / MADCTL=0xa0, RowOffset=80), active-low A/B buttons, and demo.New().
func TestBoardSmokeCLUE(t *testing.T) {
	h := New()
	const (
		pinSCK   Pin = 13
		pinSDO   Pin = 15
		pinCS    Pin = 31
		pinDC    Pin = 32
		pinReset Pin = 33
		pinLite  Pin = 34
		pinBtnL  Pin = 5
		pinBtnR  Pin = 11
	)

	target := newRecordDisplayer(240, 240)
	tft := AttachST7789(h, ST7789Config{
		ID:        "clue-tft",
		Bus:       1,
		Width:     240,
		Height:    240,
		RowOffset: 80,
		CS:        pinCS,
		DC:        pinDC,
		Reset:     pinReset,
		Backlight: pinLite,
		Target:    target,
	})

	h.SPIConfigure(1, SPIConfig{Frequency: 8_000_000, SCK: pinSCK, SDO: pinSDO, SDI: NoPin})
	// Match CLUE's native MADCTL=0xa0 (MV|MY) and 80-column offset after MV swap.
	display := &hostST7789Driver{
		hal:       h,
		bus:       1,
		cs:        pinCS,
		dc:        pinDC,
		rst:       pinReset,
		bl:        pinLite,
		width:     240,
		height:    240,
		colOffset: 80,
		madctl:    st7789MADCTLMV | 0x80,
	}
	display.configure()

	h.GPIOConfigure(pinBtnL, PinConfig{Mode: PinInputPullup})
	h.GPIOConfigure(pinBtnR, PinConfig{Mode: PinInputPullup})
	AttachButton(h, ButtonConfig{ID: "left", Pin: pinBtnL, ActiveLow: true})
	AttachButton(h, ButtonConfig{ID: "right", Pin: pinBtnR, ActiveLow: true})

	poller := bubbles.ButtonPoller{Buttons: []bubbles.Button{
		{Key: bubbles.KeyLeft, Debounce: 25 * time.Millisecond},
		{Key: bubbles.KeyRight, Debounce: 25 * time.Millisecond},
	}}
	pins := [2]Pin{pinBtnL, pinBtnR}
	pressed := func(i int) bool { return !h.GPIOGet(pins[i]) }

	now := time.Now()
	input := func() bubbles.Msg { return poller.Poll(now, pressed) }

	p, err := bubbles.NewProgram(demo.New(), bubbles.Config{Display: display, Input: input})
	if err != nil {
		t.Fatalf("bubbles.NewProgram failed: %v", err)
	}
	if err := p.Step(); err != nil {
		t.Fatalf("initial p.Step failed: %v", err)
	}

	_ = h.InjectInput(InputEvent{Kind: InputButton, ID: "right", Pressed: true})
	_ = p.Step()
	now = now.Add(30 * time.Millisecond)
	if err := p.Step(); err != nil {
		t.Fatalf("p.Step after debounce failed: %v", err)
	}
	_ = h.InjectInput(InputEvent{Kind: InputButton, ID: "right", Pressed: false})

	if tft.Frames() < 2 {
		t.Fatalf("expected at least 2 ST7789 frame flushes on CLUE setup, got %d", tft.Frames())
	}
	if len(target.pixels) == 0 {
		t.Fatalf("expected CLUE ST7789 smoke test to light pixels")
	}
}

// TestBoardSmokeFeatherReverseTFT validates the devices/feather-reverse-tft setup against tinyhal:
// 240x135 SPI0 ST7789 (RowOffset=40, ColumnOffset=53, BGR), D0 active-low + D1/D2 active-high, and hello.New().
func TestBoardSmokeFeatherReverseTFT(t *testing.T) {
	h := New()
	const (
		tftPower     Pin = 7
		tftClock     Pin = 36
		tftMOSI      Pin = 35
		tftCS        Pin = 42
		tftDC        Pin = 40
		tftReset     Pin = 41
		tftBacklight Pin = 45
		pinD0        Pin = 0
		pinD1        Pin = 1
		pinD2        Pin = 2
	)

	h.GPIOConfigure(tftPower, PinConfig{Mode: PinOutput})
	h.GPIOSet(tftPower, true)

	target := newRecordDisplayer(240, 135)
	tft := AttachST7789(h, ST7789Config{
		ID:           "feather-tft",
		Bus:          0,
		Width:        240,
		Height:       135,
		RowOffset:    40,
		ColumnOffset: 53,
		CS:           tftCS,
		DC:           tftDC,
		Reset:        tftReset,
		Backlight:    tftBacklight,
		Target:       target,
	})

	h.SPIConfigure(0, SPIConfig{Frequency: 20_000_000, SCK: tftClock, SDO: tftMOSI, SDI: NoPin})
	display := &hostST7789Driver{
		hal:       h,
		bus:       0,
		cs:        tftCS,
		dc:        tftDC,
		rst:       tftReset,
		bl:        tftBacklight,
		width:     240,
		height:    135,
		colOffset: 40,
		rowOff:    53,
		madctl:    st7789MADCTLMV | 0x48,
	}
	display.configure()

	// D0 is active-low (pull-up); D1 and D2 are active-high (pull-down).
	h.GPIOConfigure(pinD0, PinConfig{Mode: PinInputPullup})
	h.GPIOConfigure(pinD1, PinConfig{Mode: PinInputPulldown})
	h.GPIOConfigure(pinD2, PinConfig{Mode: PinInputPulldown})
	AttachButton(h, ButtonConfig{ID: "d0", Pin: pinD0, ActiveLow: true})
	AttachButton(h, ButtonConfig{ID: "d1", Pin: pinD1, ActiveLow: false})
	AttachButton(h, ButtonConfig{ID: "d2", Pin: pinD2, ActiveLow: false})

	pins := [3]Pin{pinD0, pinD1, pinD2}
	boardPressed := func(index int) bool {
		v := h.GPIOGet(pins[index])
		if index == 0 {
			return !v
		}
		return v
	}

	buttons := [3]bubbles.Button{
		{Key: bubbles.KeySelect, Debounce: 25 * time.Millisecond},
		{Key: bubbles.KeyRight, Debounce: 25 * time.Millisecond},
		{Key: bubbles.KeyBack, Debounce: 25 * time.Millisecond},
	}
	now := time.Now()
	cursor := 0
	nextFrame := now.Add(100 * time.Millisecond)
	poll := func() bubbles.Msg {
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
			if msg, ok := buttons[i].Update(boardPressed(i), now); ok {
				cursor = (i + 1) % 4
				return msg
			}
		}
		return nil
	}

	theme := hello.Theme()
	p, err := bubbles.NewProgram(hello.New(), bubbles.Config{
		Display: display,
		Font:    hello.NewFont(),
		Theme:   &theme,
		Input:   poll,
	})
	if err != nil {
		t.Fatalf("bubbles.NewProgram failed: %v", err)
	}
	if err := p.Step(); err != nil {
		t.Fatalf("initial p.Step failed: %v", err)
	}

	// Press D0 (active-low pop) and D1 (active-high palette cycle).
	for _, id := range []string{"d0", "d1"} {
		_ = h.InjectInput(InputEvent{Kind: InputButton, ID: id, Pressed: true})
		_ = p.Step()
		now = now.Add(30 * time.Millisecond)
		if err := p.Step(); err != nil {
			t.Fatalf("p.Step(%s) failed: %v", id, err)
		}
		_ = h.InjectInput(InputEvent{Kind: InputButton, ID: id, Pressed: false})
		now = now.Add(30 * time.Millisecond)
		_ = p.Step()
	}

	if tft.Frames() < 3 {
		t.Fatalf("expected at least 3 ST7789 frame flushes on Feather setup, got %d", tft.Frames())
	}
	if len(target.pixels) == 0 {
		t.Fatalf("expected Feather Reverse TFT smoke test to light pixels")
	}
}
