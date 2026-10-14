package tinyhal

import (
	"image/color"
	"sync"
)

// SPIDevice models a peripheral attached to a simulated SPI bus.
type SPIDevice interface {
	Configure(cfg SPIConfig)
	Transfer(h *HAL, w, r []byte)
}

// I2CDevice models an addressable peripheral attached to a simulated I2C bus.
type I2CDevice interface {
	Address() uint16
	Tx(h *HAL, w, r []byte) error
}

// LEDWriter matches the WriteColors interface implemented by nimbleterminal.dev/tinybubbles/web.LEDs.
type LEDWriter interface {
	WriteColors(colors []color.RGBA) error
}

// LEDConfig configures a simulated GPIO/PWM LED indicator.
type LEDConfig struct {
	ID        string
	Pin       Pin
	ActiveLow bool
	Color     color.RGBA
	Writer    LEDWriter
	Index     int
}

// LED models an indicator LED driven by a GPIO output or PWM channel.
type LED struct {
	hal        *HAL
	id         string
	pin        Pin
	activeLow  bool
	baseColor  color.RGBA
	writer     LEDWriter
	index      int
	mu         sync.Mutex
	on         bool
	brightness uint8
}

// AttachLED attaches a simulated LED model to cfg.Pin on h.
func AttachLED(h *HAL, cfg LEDConfig) *LED {
	c := cfg.Color
	if c == (color.RGBA{}) {
		c = color.RGBA{R: 0, G: 220, B: 120, A: 255}
	}
	l := &LED{
		hal:       h,
		id:        cfg.ID,
		pin:       cfg.Pin,
		activeLow: cfg.ActiveLow,
		baseColor: c,
		writer:    cfg.Writer,
		index:     cfg.Index,
	}
	h.WatchPin(cfg.Pin, l.onPinChange)
	return l
}

func (l *LED) onPinChange(h *HAL, _ Pin, s PinState) {
	if !s.Configured || s.Mode != PinOutput {
		return
	}
	duty := s.PWM
	if l.activeLow {
		duty = 255 - duty
	}
	on := duty > 0

	l.mu.Lock()
	changed := l.on != on || l.brightness != duty
	l.on = on
	l.brightness = duty
	scaled := scaleColor(l.baseColor, duty)
	l.mu.Unlock()

	if !changed {
		return
	}
	if l.writer != nil && l.index >= 0 {
		colors := make([]color.RGBA, l.index+1)
		colors[l.index] = scaled
		_ = l.writer.WriteColors(colors)
	}
	h.Emit(Event{
		Kind:       EventLED,
		ID:         l.id,
		Pin:        l.pin,
		High:       on,
		Brightness: duty,
		Color:      scaled,
	})
}

// On reports whether the LED is currently illuminated.
func (l *LED) On() bool {
	l.mu.Lock()
	v := l.on
	l.mu.Unlock()
	return v
}

// Brightness returns the LED brightness in the range 0..255.
func (l *LED) Brightness() uint8 {
	l.mu.Lock()
	v := l.brightness
	l.mu.Unlock()
	return v
}

// CurrentColor returns the brightness-scaled RGBA color of the LED.
func (l *LED) CurrentColor() color.RGBA {
	l.mu.Lock()
	c := scaleColor(l.baseColor, l.brightness)
	l.mu.Unlock()
	return c
}

func scaleColor(c color.RGBA, brightness uint8) color.RGBA {
	if brightness == 0 {
		return color.RGBA{A: 255}
	}
	return color.RGBA{
		R: uint8((uint16(c.R) * uint16(brightness)) / 255),
		G: uint8((uint16(c.G) * uint16(brightness)) / 255),
		B: uint8((uint16(c.B) * uint16(brightness)) / 255),
		A: 255,
	}
}

// ButtonConfig configures a simulated pushbutton connected to a GPIO pin.
type ButtonConfig struct {
	ID        string
	Pin       Pin
	ActiveLow bool
}

// Button models a momentary pushbutton that drives a GPIO input pin.
type Button struct {
	hal       *HAL
	id        string
	pin       Pin
	activeLow bool
	mu        sync.Mutex
	pressed   bool
}

// AttachButton attaches a simulated pushbutton to cfg.Pin and registers its ID on h.
func AttachButton(h *HAL, cfg ButtonConfig) *Button {
	b := &Button{
		hal:       h,
		id:        cfg.ID,
		pin:       cfg.Pin,
		activeLow: cfg.ActiveLow,
	}
	if cfg.ID != "" {
		h.mu.Lock()
		h.buttons[cfg.ID] = b
		h.mu.Unlock()
	}
	// Set initial released pin state.
	h.SetPinInput(cfg.Pin, b.pinLevelFor(false))
	return b
}

func (b *Button) pinLevelFor(pressed bool) bool {
	if b.activeLow {
		return !pressed
	}
	return pressed
}

// SetPressed updates the button state, drives the GPIO input pin, and emits EventButton.
func (b *Button) SetPressed(pressed bool) {
	b.mu.Lock()
	changed := b.pressed != pressed
	b.pressed = pressed
	b.mu.Unlock()

	b.hal.SetPinInput(b.pin, b.pinLevelFor(pressed))
	if changed {
		b.hal.Emit(Event{
			Kind:    EventButton,
			ID:      b.id,
			Pin:     b.pin,
			Pressed: pressed,
		})
	}
}

// Press depresses the button.
func (b *Button) Press() { b.SetPressed(true) }

// Release releases the button.
func (b *Button) Release() { b.SetPressed(false) }

// Pressed reports whether the button is currently depressed.
func (b *Button) Pressed() bool {
	b.mu.Lock()
	v := b.pressed
	b.mu.Unlock()
	return v
}

// SPIRegisterDevice models a simple stateful SPI register/loopback peripheral.
//
// When CS is NoPin or driven low:
//   - Writing [0x80 | reg, val, ...] stores val into register reg (and subsequent registers).
//   - Writing [reg & 0x7f, 0x00, ...] reads starting at register reg into r[1:].
type SPIRegisterDevice struct {
	cs           Pin
	mu           sync.Mutex
	config       SPIConfig
	regs         [128]byte
	lastTx       []byte
	transactions int
}

// NewSPIRegisterDevice creates a register-mapped SPI peripheral gated by cs (or NoPin).
func NewSPIRegisterDevice(cs Pin) *SPIRegisterDevice {
	return &SPIRegisterDevice{cs: cs}
}

// Configure records the SPI bus configuration.
func (d *SPIRegisterDevice) Configure(cfg SPIConfig) {
	d.mu.Lock()
	d.config = cfg
	d.mu.Unlock()
}

// Transfer processes a full-duplex SPI transaction when CS is active (low or NoPin).
func (d *SPIRegisterDevice) Transfer(h *HAL, w, r []byte) {
	if d.cs != NoPin && h.PinLevel(d.cs) {
		return
	}
	n := len(w)
	if len(r) > n {
		n = len(r)
	}
	if n == 0 {
		return
	}

	d.mu.Lock()
	defer d.mu.Unlock()
	d.transactions++
	d.lastTx = append(d.lastTx[:0], w...)

	if len(w) == 0 {
		for i := range r {
			r[i] = d.regs[i&0x7f]
		}
		return
	}

	cmd := w[0]
	reg := cmd & 0x7f
	writeMode := (cmd & 0x80) != 0
	if len(r) > 0 {
		r[0] = d.regs[reg]
	}
	for i := 1; i < n; i++ {
		idx := (reg + uint8(i-1)) & 0x7f
		if writeMode && i < len(w) {
			d.regs[idx] = w[i]
		}
		if i < len(r) {
			r[i] = d.regs[idx]
		}
	}
}

// Register returns the current byte stored at reg (0..127).
func (d *SPIRegisterDevice) Register(reg uint8) byte {
	d.mu.Lock()
	v := d.regs[reg&0x7f]
	d.mu.Unlock()
	return v
}

// SetRegister sets the byte stored at reg (0..127).
func (d *SPIRegisterDevice) SetRegister(reg uint8, val byte) {
	d.mu.Lock()
	d.regs[reg&0x7f] = val
	d.mu.Unlock()
}

// Transactions returns the number of active SPI transactions processed by the device.
func (d *SPIRegisterDevice) Transactions() int {
	d.mu.Lock()
	n := d.transactions
	d.mu.Unlock()
	return n
}

// LastTx returns a copy of the most recent MOSI payload received while selected.
func (d *SPIRegisterDevice) LastTx() []byte {
	d.mu.Lock()
	out := append([]byte(nil), d.lastTx...)
	d.mu.Unlock()
	return out
}
