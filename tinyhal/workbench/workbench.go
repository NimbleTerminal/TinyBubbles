// Package workbench implements a standalone virtual MCU hardware bench powered
// purely by tinyhal, tinygo.org/x/drivers, and tinygo.org/x/tinyfont.
//
// It does not use TinyBubbles cell buffers, widgets, or Bubble Tea / TinyTea
// display renderers. Instead, embedded firmware code reads simulated GPIO
// buttons, an ADC channel, and an I2C temperature sensor, drives GPIO and PWM
// LEDs, accesses an SPI register device and UART bus, and renders a live
// oscilloscope and bus dashboard over SPI (ST7789) or I2C (SSD1306).
package workbench

import (
	"fmt"
	"image/color"
	"math"

	"nimbleterminal.dev/tinybubbles/tinyhal"
	"tinygo.org/x/drivers"
	"tinygo.org/x/tinyfont"
	"tinygo.org/x/tinyfont/proggy"
)

// Simulated MCU pin and bus assignments.
const (
	PinButtonMode  tinyhal.Pin = 0  // D0: active-low mode button (pull-up)
	PinButtonLED   tinyhal.Pin = 1  // D1: active-high LED toggle button (pull-down)
	PinButtonBurst tinyhal.Pin = 2  // D2: active-high SPI/I2C/UART burst button (pull-down)
	PinSPICS       tinyhal.Pin = 10 // CS for SPI register peripheral
	PinLEDStatus   tinyhal.Pin = 13 // GPIO digital output LED (Green)
	PinLEDPWM      tinyhal.Pin = 14 // PWM0 channel 0 dimmer LED (Cyan)
	PinLEDActivity tinyhal.Pin = 15 // GPIO bus-activity pulse LED (Amber)
	PinADCPot      tinyhal.Pin = 26 // 12-bit ADC input (0..4095)
	PinSPISDO      tinyhal.Pin = 35
	PinSPISCK      tinyhal.Pin = 36
	PinTFTDC       tinyhal.Pin = 40
	PinTFTReset    tinyhal.Pin = 41
	PinTFTCS       tinyhal.Pin = 42
	PinTFTLite     tinyhal.Pin = 45

	I2COLEDAddr   uint16 = 0x3c
	I2CSensorAddr uint16 = 0x48
)

// Mode selects the analog signal source for ADC pin 26.
type Mode uint8

const (
	ModeManual Mode = iota
	ModeSine
	ModeSaw
)

func (m Mode) String() string {
	switch m {
	case ModeSine:
		return "SINE"
	case ModeSaw:
		return "SAW"
	default:
		return "ADC"
	}
}

// Config configures the virtual MCU workbench and its display/LED sinks.
type Config struct {
	Width      int16
	Height     int16
	Monochrome bool
	Target     drivers.Displayer
	LEDs       tinyhal.LEDWriter
}

type ledGroupWriter struct {
	sink   tinyhal.LEDWriter
	colors [3]color.RGBA
}

func (g *ledGroupWriter) slot(index int) tinyhal.LEDWriter {
	return ledSlotWriter{group: g, index: index}
}

type ledSlotWriter struct {
	group *ledGroupWriter
	index int
}

func (s ledSlotWriter) WriteColors(colors []color.RGBA) error {
	if s.index >= 0 && s.index < len(s.group.colors) && len(colors) > s.index {
		s.group.colors[s.index] = colors[s.index]
	}
	if s.group.sink != nil {
		return s.group.sink.WriteColors(s.group.colors[:])
	}
	return nil
}

// Workbench runs the simulated microcontroller board and peripheral netlist.
type Workbench struct {
	cfg        Config
	hal        *tinyhal.HAL
	bus        mcuBus
	ledGroup   *ledGroupWriter
	ledStatus  *tinyhal.LED
	ledPWM     *tinyhal.LED
	ledAct     *tinyhal.LED
	btnMode    *tinyhal.Button
	btnLED     *tinyhal.Button
	btnBurst   *tinyhal.Button
	spiReg     *tinyhal.SPIRegisterDevice
	i2cSensor  *tinyhal.I2CRegisterDevice
	tft        *tinyhal.ST7789
	oled       *tinyhal.SSD1306
	frameBuf   *frameCanvas
	mode       Mode
	manualADC  uint16
	adcReading uint16
	tempC      int8
	spiVal     byte
	burstCount uint8
	uartCount  uint32
	lastUART   string
	actTTL     int
	phase      int
	history    [96]uint16
	histIdx    int
	prevD0     bool
	prevD1     bool
	prevD2     bool
}

// New creates and initializes a virtual MCU workbench.
func New(cfg Config) *Workbench {
	if cfg.Width <= 0 {
		cfg.Width = 240
	}
	if cfg.Height <= 0 {
		cfg.Height = 135
	}
	hal := newWorkbenchHAL()
	group := &ledGroupWriter{sink: cfg.LEDs}
	w := &Workbench{
		cfg:      cfg,
		hal:      hal,
		bus:      newMCUBus(hal),
		ledGroup: group,
		frameBuf: newFrameCanvas(cfg.Width, cfg.Height, cfg.Monochrome),
	}
	w.attachPeripherals()
	w.Reset()
	return w
}

func (w *Workbench) attachPeripherals() {
	h := w.hal
	w.ledStatus = tinyhal.AttachLED(h, tinyhal.LEDConfig{
		ID:     "gpio13",
		Pin:    PinLEDStatus,
		Color:  color.RGBA{R: 80, G: 235, B: 130, A: 255},
		Writer: w.ledGroup.slot(0),
		Index:  0,
	})
	w.ledPWM = tinyhal.AttachLED(h, tinyhal.LEDConfig{
		ID:     "pwm14",
		Pin:    PinLEDPWM,
		Color:  color.RGBA{R: 64, G: 220, B: 255, A: 255},
		Writer: w.ledGroup.slot(1),
		Index:  1,
	})
	w.ledAct = tinyhal.AttachLED(h, tinyhal.LEDConfig{
		ID:     "act15",
		Pin:    PinLEDActivity,
		Color:  color.RGBA{R: 255, G: 195, B: 65, A: 255},
		Writer: w.ledGroup.slot(2),
		Index:  2,
	})

	w.btnMode = tinyhal.AttachButton(h, tinyhal.ButtonConfig{
		ID:        "d0",
		Pin:       PinButtonMode,
		ActiveLow: true,
	})
	w.btnLED = tinyhal.AttachButton(h, tinyhal.ButtonConfig{
		ID:        "d1",
		Pin:       PinButtonLED,
		ActiveLow: false,
	})
	w.btnBurst = tinyhal.AttachButton(h, tinyhal.ButtonConfig{
		ID:        "d2",
		Pin:       PinButtonBurst,
		ActiveLow: false,
	})

	w.spiReg = tinyhal.NewSPIRegisterDevice(PinSPICS)
	h.AttachSPI(0, w.spiReg)

	w.i2cSensor = tinyhal.NewI2CRegisterDevice(I2CSensorAddr)
	w.i2cSensor.SetRegister(0x02, 0xa5) // sensor chip ID
	h.AttachI2C(0, w.i2cSensor)

	if w.cfg.Monochrome {
		w.oled = tinyhal.AttachSSD1306(h, tinyhal.SSD1306Config{
			ID:      "oled",
			Bus:     0,
			Address: I2COLEDAddr,
			Width:   w.cfg.Width,
			Height:  w.cfg.Height,
			Target:  w.cfg.Target,
		})
	} else {
		w.tft = tinyhal.AttachST7789(h, tinyhal.ST7789Config{
			ID:        "tft",
			Bus:       0,
			Width:     w.cfg.Width,
			Height:    w.cfg.Height,
			Reset:     PinTFTReset,
			DC:        PinTFTDC,
			CS:        PinTFTCS,
			Backlight: PinTFTLite,
			Target:    w.cfg.Target,
		})
	}
}

// Reset restores initial pin, bus, sensor, and waveform states and renders a frame.
func (w *Workbench) Reset() {
	w.mode = ModeManual
	w.manualADC = 2048
	w.tempC = 24
	w.burstCount = 1
	w.uartCount = 0
	w.actTTL = 0
	w.phase = 0
	w.prevD0 = false
	w.prevD1 = false
	w.prevD2 = false

	w.bus.configurePinsAndBuses(w.cfg.Width, w.cfg.Height, w.cfg.Monochrome)
	w.i2cSensor.SetRegister(0x00, byte(w.tempC))
	_ = w.hal.InjectInput(tinyhal.InputEvent{Kind: tinyhal.InputADC, Pin: PinADCPot, Value: w.manualADC})

	// Turn status LED on initially and run one SPI/I2C/UART transaction so registers are populated.
	w.bus.pinSet(PinLEDStatus, true)
	w.runBusBurst()
	for i := range w.history {
		w.history[i] = w.manualADC
	}
	w.sampleAndRender()
}

// HAL returns the underlying tinyhal.HAL instance.
func (w *Workbench) HAL() *tinyhal.HAL { return w.hal }

// Mode returns the active ADC signal source mode.
func (w *Workbench) Mode() Mode { return w.mode }

// ADC returns the most recent 12-bit ADC reading (0..4095).
func (w *Workbench) ADC() uint16 { return w.adcReading }

// TempC returns the most recent I2C temperature sensor reading in Celsius.
func (w *Workbench) TempC() int8 { return w.tempC }

// LEDStatusOn reports whether the digital status LED (GPIO13) is illuminated.
func (w *Workbench) LEDStatusOn() bool { return w.ledStatus.On() }

// PWMPercent returns the current PWM duty percentage (0..100) on LED 1 (GPIO14).
func (w *Workbench) PWMPercent() int {
	return (int(w.ledPWM.Brightness()) * 100) / 255
}

// SPIRegister returns the latest byte read back from SPI register 0x01.
func (w *Workbench) SPIRegister() byte { return w.spiVal }

// Frames returns the number of display frames flushed over SPI or I2C.
func (w *Workbench) Frames() int {
	if w.cfg.Monochrome {
		return w.oled.Frames()
	}
	return w.tft.Frames()
}

// PressMode simulates pressing and releasing active-low GPIO button D0.
func (w *Workbench) PressMode() {
	_ = w.hal.InjectInput(tinyhal.InputEvent{Kind: tinyhal.InputButton, ID: "d0", Pressed: true})
	w.pollButtons()
	_ = w.hal.InjectInput(tinyhal.InputEvent{Kind: tinyhal.InputButton, ID: "d0", Pressed: false})
	w.pollButtons()
	w.sampleAndRender()
}

// ToggleLED simulates pressing and releasing active-high GPIO button D1.
func (w *Workbench) ToggleLED() {
	_ = w.hal.InjectInput(tinyhal.InputEvent{Kind: tinyhal.InputButton, ID: "d1", Pressed: true})
	w.pollButtons()
	_ = w.hal.InjectInput(tinyhal.InputEvent{Kind: tinyhal.InputButton, ID: "d1", Pressed: false})
	w.pollButtons()
	w.sampleAndRender()
}

// TriggerBurst simulates pressing and releasing active-high GPIO button D2.
func (w *Workbench) TriggerBurst() {
	_ = w.hal.InjectInput(tinyhal.InputEvent{Kind: tinyhal.InputButton, ID: "d2", Pressed: true})
	w.pollButtons()
	_ = w.hal.InjectInput(tinyhal.InputEvent{Kind: tinyhal.InputButton, ID: "d2", Pressed: false})
	w.pollButtons()
	w.sampleAndRender()
}

// AdjustADC changes the simulated analog input on Pin 26 by delta (-4095..+4095).
func (w *Workbench) AdjustADC(delta int) {
	next := int(w.manualADC) + delta
	if next < 0 {
		next = 0
	} else if next > 4095 {
		next = 4095
	}
	w.manualADC = uint16(next)
	_ = w.hal.InjectInput(tinyhal.InputEvent{Kind: tinyhal.InputADC, Pin: PinADCPot, Value: w.manualADC})
	w.sampleAndRender()
}

// AdjustTemp changes the simulated I2C sensor temperature register by delta Celsius.
func (w *Workbench) AdjustTemp(delta int8) {
	next := int(w.tempC) + int(delta)
	if next < -20 {
		next = -20
	} else if next > 85 {
		next = 85
	}
	w.i2cSensor.SetRegister(0x00, byte(int8(next)))
	w.actTTL = 2
	w.bus.pinSet(PinLEDActivity, true)
	w.sampleAndRender()
}

// Tick advances the workbench simulation by one timer step.
func (w *Workbench) Tick() {
	w.phase = (w.phase + 1) % 64
	switch w.mode {
	case ModeSine:
		rad := (float64(w.phase) / 64.0) * 2.0 * math.Pi
		val := uint16(math.Round((math.Sin(rad)*0.5 + 0.5) * 4095.0))
		_ = w.hal.InjectInput(tinyhal.InputEvent{Kind: tinyhal.InputADC, Pin: PinADCPot, Value: val})
	case ModeSaw:
		val := uint16((w.phase * 4095) / 63)
		_ = w.hal.InjectInput(tinyhal.InputEvent{Kind: tinyhal.InputADC, Pin: PinADCPot, Value: val})
	}
	w.pollButtons()
	if w.actTTL > 0 {
		w.actTTL--
		if w.actTTL == 0 {
			w.bus.pinSet(PinLEDActivity, false)
		}
	}
	w.sampleAndRender()
}

func (w *Workbench) pollButtons() {
	// D0 is active-low (pull-up); D1 and D2 are active-high (pull-down).
	d0 := !w.bus.pinGet(PinButtonMode)
	d1 := w.bus.pinGet(PinButtonLED)
	d2 := w.bus.pinGet(PinButtonBurst)

	if d0 && !w.prevD0 {
		w.mode = (w.mode + 1) % 3
		if w.mode == ModeManual {
			_ = w.hal.InjectInput(tinyhal.InputEvent{Kind: tinyhal.InputADC, Pin: PinADCPot, Value: w.manualADC})
		}
	}
	if d1 && !w.prevD1 {
		w.bus.pinSet(PinLEDStatus, !w.bus.pinGet(PinLEDStatus))
	}
	if d2 && !w.prevD2 {
		w.runBusBurst()
	}
	w.prevD0, w.prevD1, w.prevD2 = d0, d1, d2
}

func (w *Workbench) runBusBurst() {
	w.burstCount++
	val := byte(0xa0 + (w.burstCount & 0x1f))

	// 1. Write and read back register 0x01 on the SPI register peripheral.
	w.bus.pinSet(PinSPICS, false)
	w.bus.spiTx([]byte{0x80 | 0x01, val}, nil)
	var rx [2]byte
	w.bus.spiTx([]byte{0x01, 0x00}, rx[:])
	w.bus.pinSet(PinSPICS, true)
	w.spiVal = rx[1]

	// 2. Read temperature register 0x00 from the I2C sensor at 0x48.
	var tempBuf [1]byte
	_ = w.bus.i2cTx(I2CSensorAddr, []byte{0x00}, tempBuf[:])
	w.tempC = int8(tempBuf[0])

	// 3. Emit a compact telemetry packet over UART0.
	w.uartCount++
	volts := (float64(w.adcReading) * 3.30) / 4095.0
	w.lastUART = fmt.Sprintf("TX#%d %.2fV %dC", w.uartCount, volts, w.tempC)
	w.bus.uartWrite([]byte(w.lastUART + "\n"))

	// 4. Pulse the bus-activity GPIO LED (Pin 15).
	w.actTTL = 3
	w.bus.pinSet(PinLEDActivity, true)
}

func (w *Workbench) sampleAndRender() {
	// Read 12-bit ADC on Pin 26 and drive PWM0 channel 0 on Pin 14.
	w.adcReading = w.bus.adcRead(PinADCPot)
	w.bus.pwmSet(uint32(w.adcReading))

	// Poll I2C temperature register 0x00.
	var tempBuf [1]byte
	_ = w.bus.i2cTx(I2CSensorAddr, []byte{0x00}, tempBuf[:])
	w.tempC = int8(tempBuf[0])

	w.history[w.histIdx] = w.adcReading
	w.histIdx = (w.histIdx + 1) % len(w.history)

	w.drawDashboard()
	w.bus.flushDisplay(w.frameBuf)
}

func (w *Workbench) drawDashboard() {
	c := w.frameBuf
	width, height := c.width, c.height

	bg := color.RGBA{R: 10, G: 18, B: 28, A: 255}
	panelBg := color.RGBA{R: 18, G: 31, B: 46, A: 255}
	headerBg := color.RGBA{R: 24, G: 48, B: 68, A: 255}
	white := color.RGBA{R: 235, G: 245, B: 250, A: 255}
	muted := color.RGBA{R: 135, G: 165, B: 180, A: 255}
	cyan := color.RGBA{R: 64, G: 220, B: 255, A: 255}
	green := color.RGBA{R: 80, G: 235, B: 130, A: 255}
	amber := color.RGBA{R: 255, G: 195, B: 65, A: 255}
	gridCol := color.RGBA{R: 32, G: 54, B: 74, A: 255}

	if c.mono {
		bg = color.RGBA{A: 255}
		panelBg = color.RGBA{A: 255}
		headerBg = color.RGBA{A: 255}
		white = color.RGBA{R: 255, G: 255, B: 255, A: 255}
		muted = white
		cyan = white
		green = white
		amber = white
		gridCol = white
	}

	c.fillRect(0, 0, width, height, bg)

	volts := (float64(w.adcReading) * 3.30) / 4095.0
	ledState := "OFF"
	if w.ledStatus.On() {
		ledState = "ON"
	}

	if height <= 64 {
		// Compact 128x64 monochrome OLED layout over I2C0.
		tinyfont.WriteLine(c, &proggy.TinySZ8pt7b, 2, 8, fmt.Sprintf("HAL I2C0 %s %dC", w.mode, w.tempC), white)
		tinyfont.WriteLine(c, &proggy.TinySZ8pt7b, 2, 18, fmt.Sprintf("L13:%s PWM:%d%%", ledState, w.PWMPercent()), white)
		tinyfont.WriteLine(c, &proggy.TinySZ8pt7b, 2, 28, fmt.Sprintf("ADC:%.2fV SPI:%02X", volts, w.spiVal), white)
		w.drawScope(2, 32, width-4, height-34, panelBg, gridCol, cyan)
		return
	}

	// Color TFT layout (240x135 Feather or 320x240 Large TFT) over SPI0.
	c.fillRect(0, 0, width, 16, headerBg)
	tinyfont.WriteLine(c, &proggy.TinySZ8pt7b, 6, 11, "TINYHAL · SPI0 ST7789", cyan)
	tinyfont.WriteLine(c, &proggy.TinySZ8pt7b, width-72, 11, fmt.Sprintf("MODE:%s", w.mode), amber)

	// Pin & LED status strip.
	ledCol := muted
	if w.ledStatus.On() {
		ledCol = green
	}
	actCol := muted
	if w.ledAct.On() {
		actCol = amber
	}
	tinyfont.WriteLine(c, &proggy.TinySZ8pt7b, 6, 28, fmt.Sprintf("GPIO13:%s", ledState), ledCol)
	tinyfont.WriteLine(c, &proggy.TinySZ8pt7b, 78, 28, fmt.Sprintf("PWM14:%3d%%", w.PWMPercent()), cyan)
	tinyfont.WriteLine(c, &proggy.TinySZ8pt7b, 156, 28, fmt.Sprintf("ACT15:%v", w.ledAct.On()), actCol)

	// ADC voltage gauge bar.
	tinyfont.WriteLine(c, &proggy.TinySZ8pt7b, 6, 42, fmt.Sprintf("ADC26: %4d (%0.2fV)", w.adcReading, volts), white)
	barX, barY, barW, barH := int16(136), int16(34), width-142, int16(9)
	c.fillRect(barX, barY, barW, barH, panelBg)
	fillW := int16((int(w.adcReading) * int(barW)) / 4095)
	if fillW > 0 {
		c.fillRect(barX, barY, fillW, barH, cyan)
	}

	// Oscilloscope waveform area.
	scopeY := int16(48)
	scopeH := height - 68
	w.drawScope(6, scopeY, width-12, scopeH, panelBg, gridCol, green)

	// Bottom bus telemetry bar (SPI, I2C, UART).
	footerY := height - 6
	tinyfont.WriteLine(c, &proggy.TinySZ8pt7b, 6, footerY,
		fmt.Sprintf("SPI[01]:0x%02X I2C[48]:%dC %s", w.spiVal, w.tempC, w.lastUART), muted)
}

func (w *Workbench) drawScope(x, y, width, height int16, bg, border, trace color.RGBA) {
	if width <= 4 || height <= 4 {
		return
	}
	c := w.frameBuf
	c.fillRect(x, y, width, height, bg)
	// Top and bottom scope rails.
	c.fillRect(x, y, width, 1, border)
	c.fillRect(x, y+height-1, width, 1, border)

	innerH := height - 4
	samples := int(width - 2)
	if samples > len(w.history) {
		samples = len(w.history)
	}
	var prevY int16
	for i := 0; i < samples; i++ {
		idx := (w.histIdx - samples + i + len(w.history)) % len(w.history)
		val := w.history[idx]
		py := y + height - 2 - int16((int(val)*int(innerH))/4095)
		px := x + 1 + int16(i)
		c.SetPixel(px, py, trace)
		if i > 0 {
			y0, y1 := prevY, py
			if y0 > y1 {
				y0, y1 = y1, y0
			}
			for yy := y0; yy <= y1; yy++ {
				c.SetPixel(px, yy, trace)
			}
		}
		prevY = py
	}
}

// frameCanvas holds the local firmware frame before flushing over SPI (RGB565BE) or I2C (SSD1306 GDDRAM).
type frameCanvas struct {
	width  int16
	height int16
	mono   bool
	rgb565 []byte
	gddram []byte
}

func newFrameCanvas(w, h int16, mono bool) *frameCanvas {
	pages := max(int16(1), h/8)
	return &frameCanvas{
		width:  w,
		height: h,
		mono:   mono,
		rgb565: make([]byte, int(w)*int(h)*2),
		gddram: make([]byte, int(w)*int(pages)+1),
	}
}

func (f *frameCanvas) Size() (int16, int16) { return f.width, f.height }

func (f *frameCanvas) Display() error { return nil }

func (f *frameCanvas) SetPixel(x, y int16, c color.RGBA) {
	if x < 0 || y < 0 || x >= f.width || y >= f.height {
		return
	}
	if f.mono {
		idx := 1 + int(x) + int(y/8)*int(f.width)
		if idx < len(f.gddram) {
			if c.R != 0 || c.G != 0 || c.B != 0 {
				f.gddram[idx] |= 1 << uint8(y%8)
			} else {
				f.gddram[idx] &^= 1 << uint8(y%8)
			}
		}
		return
	}
	v := (uint16(c.R>>3) << 11) | (uint16(c.G>>2) << 5) | uint16(c.B>>3)
	i := (int(y)*int(f.width) + int(x)) * 2
	f.rgb565[i] = byte(v >> 8)
	f.rgb565[i+1] = byte(v)
}

func (f *frameCanvas) fillRect(x, y, w, h int16, c color.RGBA) {
	left, top := max(int16(0), x), max(int16(0), y)
	right, bottom := min(f.width, x+w), min(f.height, y+h)
	for yy := top; yy < bottom; yy++ {
		for xx := left; xx < right; xx++ {
			f.SetPixel(xx, yy, c)
		}
	}
}
