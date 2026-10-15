// Package tinyhal provides a Hardware Abstraction Layer for running TinyGo
// machine peripheral code inside WebAssembly (WASM/WASI) or host Go tests.
//
// When compiled with TinyGo on non-MCU targets (!baremetal, such as -target=wasm
// or -target=wasip1), TinyGo's machine package delegates GPIO, PWM, ADC, SPI,
// I2C, and UART operations to external //export __tinygo_* symbols. tinyhal
// defines those symbols inside the Go/WASM module and routes them to an
// in-memory peripheral netlist and device state machines.
//
// All bus protocols, pin levels, and peripheral models stay in Go. A thin
// JavaScript host only needs to load the WASM binary, paint high-level visual
// events (LED changes, display framebuffers), and inject high-level input
// events (button presses, analog values, serial bytes).
package tinyhal

import (
	"errors"
	"sync"
)

// Pin identifies a microcontroller pin number, matching machine.Pin.
type Pin uint8

// NoPin indicates an unassigned pin, matching machine.NoPin.
const NoPin Pin = 0xff

// PinMode configures pin direction and pull resistors, matching machine.PinMode.
type PinMode uint8

const (
	PinInput PinMode = iota
	PinOutput
	PinInputPullup
	PinInputPulldown
)

// PinConfig holds GPIO pin configuration, matching machine.PinConfig.
type PinConfig struct {
	Mode PinMode
}

// PinState reports the current state of a simulated GPIO pin.
type PinState struct {
	Configured       bool
	Mode             PinMode
	Output           bool
	ExternalInput    bool
	HasExternalInput bool
	PWM              uint8
	ADC              uint16
}

// Level returns the effective logic level on the pin, accounting for direction,
// external input drivers, and internal pull resistors.
func (s PinState) Level() bool {
	if s.Mode == PinOutput {
		return s.Output
	}
	if s.HasExternalInput {
		return s.ExternalInput
	}
	if s.Mode == PinInputPullup {
		return true
	}
	return false
}

// SPIConfig describes an SPI bus configuration, matching machine.SPIConfig.
type SPIConfig struct {
	Frequency uint32
	SCK       Pin
	SDO       Pin
	SDI       Pin
	Mode      uint8
}

// I2CConfig describes an I2C bus configuration, matching machine.I2CConfig.
type I2CConfig struct {
	Frequency uint32
	SCL       Pin
	SDA       Pin
}

// UARTConfig describes a UART bus configuration, matching machine.UARTConfig.
type UARTConfig struct {
	TX Pin
	RX Pin
}

// I2C transfer return codes used by TinyGo's machine_generic.go.
const (
	I2COK int = iota
	I2CErrNoDevices
	I2CErrMultipleDevices
	I2CErrWrongAddress
	I2CErrBus
)

var (
	ErrNoI2CDevice = errors.New("tinyhal: no I2C device at address")
	ErrUnknownID   = errors.New("tinyhal: unknown input target ID")
)

// PinChangeFunc is notified while holding no internal locks when a pin's state changes.
type PinChangeFunc func(h *HAL, pin Pin, state PinState)

type pwmKey struct {
	instance int32
	channel  uint8
}

type spiBusState struct {
	configured bool
	config     SPIConfig
	devices    []SPIDevice
}

type i2cBusState struct {
	configured bool
	config     I2CConfig
	devices    map[uint16]I2CDevice
}

type uartBusState struct {
	configured bool
	config     UARTConfig
	rx         []byte
	tx         []byte
}

// HAL maintains simulated pins, buses, attached device models, and high-level events.
type HAL struct {
	mu           sync.Mutex
	pins         [256]PinState
	pinListeners [256][]PinChangeFunc
	pwmTops      map[int32]uint32
	pwmPins      map[pwmKey]Pin
	spiBuses     map[uint8]*spiBusState
	i2cBuses     map[uint8]*i2cBusState
	uartBuses    map[uint8]*uartBusState
	buttons      map[string]*Button
	events       []Event
	maxEvents    int
	onEvent      func(Event)
	tracer       *TraceRecorder
}

var (
	defaultMu  sync.RWMutex
	defaultHAL = New()
)

// Default returns the process-wide HAL instance used by TinyGo's __tinygo_* hooks.
func Default() *HAL {
	defaultMu.RLock()
	h := defaultHAL
	defaultMu.RUnlock()
	return h
}

// SetDefault replaces the process-wide HAL instance used by TinyGo's __tinygo_* hooks.
func SetDefault(h *HAL) {
	if h == nil {
		h = New()
	}
	defaultMu.Lock()
	defaultHAL = h
	defaultMu.Unlock()
}

// New creates an initialized HAL instance with an event queue capacity of 64.
func New() *HAL {
	h := &HAL{maxEvents: 64}
	h.Reset()
	return h
}

// SetTracer attaches or detaches a bus and pin trace recorder on h.
func (h *HAL) SetTracer(tr *TraceRecorder) {
	h.mu.Lock()
	h.tracer = tr
	h.mu.Unlock()
}

// Tracer returns the currently attached TraceRecorder, or nil if none is set.
func (h *HAL) Tracer() *TraceRecorder {
	h.mu.Lock()
	tr := h.tracer
	h.mu.Unlock()
	return tr
}

// Reset clears all pin states, buses, registered peripherals, and queued events.
func (h *HAL) Reset() {
	h.mu.Lock()
	h.pins = [256]PinState{}
	h.pinListeners = [256][]PinChangeFunc{}
	h.pwmTops = make(map[int32]uint32)
	h.pwmPins = make(map[pwmKey]Pin)
	h.spiBuses = make(map[uint8]*spiBusState)
	h.i2cBuses = make(map[uint8]*i2cBusState)
	h.uartBuses = make(map[uint8]*uartBusState)
	h.buttons = make(map[string]*Button)
	h.events = h.events[:0]
	if h.tracer != nil {
		h.tracer.Reset()
	}
	h.mu.Unlock()
}

// WatchPin registers a listener that fires whenever pin is configured, toggled, or PWM-adjusted.
func (h *HAL) WatchPin(pin Pin, fn PinChangeFunc) {
	if pin == NoPin || fn == nil {
		return
	}
	h.mu.Lock()
	h.pinListeners[pin] = append(h.pinListeners[pin], fn)
	state := h.pins[pin]
	h.mu.Unlock()
	if state.Configured {
		fn(h, pin, state)
	}
}

// GPIOConfigure configures a pin's direction and pull mode.
func (h *HAL) GPIOConfigure(pin Pin, cfg PinConfig) {
	if pin == NoPin {
		return
	}
	h.mu.Lock()
	s := h.pins[pin]
	s.Configured = true
	s.Mode = cfg.Mode
	h.pins[pin] = s
	listeners := append([]PinChangeFunc(nil), h.pinListeners[pin]...)
	h.mu.Unlock()

	for _, fn := range listeners {
		fn(h, pin, s)
	}
}

// GPIOSet sets the output level of a GPIO pin.
func (h *HAL) GPIOSet(pin Pin, high bool) {
	if pin == NoPin {
		return
	}
	h.mu.Lock()
	s := h.pins[pin]
	s.Configured = true
	s.Output = high
	if high {
		s.PWM = 255
	} else {
		s.PWM = 0
	}
	h.pins[pin] = s
	listeners := append([]PinChangeFunc(nil), h.pinListeners[pin]...)
	tr := h.tracer
	h.mu.Unlock()

	if tr != nil {
		tr.recordGPIO(pin, s.Level())
	}
	h.Emit(Event{
		Kind: EventPin,
		Pin:  pin,
		High: s.Level(),
	})
	for _, fn := range listeners {
		fn(h, pin, s)
	}
}

// GPIOGet returns the effective logic level of a GPIO pin.
func (h *HAL) GPIOGet(pin Pin) bool {
	return h.PinLevel(pin)
}

// PinLevel returns the effective logic level of a pin (output or external input/pull).
func (h *HAL) PinLevel(pin Pin) bool {
	if pin == NoPin {
		return false
	}
	h.mu.Lock()
	s := h.pins[pin]
	h.mu.Unlock()
	return s.Level()
}

// PinState returns a snapshot of a pin's current configuration and levels.
func (h *HAL) PinState(pin Pin) PinState {
	if pin == NoPin {
		return PinState{}
	}
	h.mu.Lock()
	s := h.pins[pin]
	h.mu.Unlock()
	return s
}

// SetPinInput drives a pin from the outside world (e.g. a button or external signal).
func (h *HAL) SetPinInput(pin Pin, high bool) {
	if pin == NoPin {
		return
	}
	h.mu.Lock()
	s := h.pins[pin]
	s.ExternalInput = high
	s.HasExternalInput = true
	h.pins[pin] = s
	listeners := append([]PinChangeFunc(nil), h.pinListeners[pin]...)
	tr := h.tracer
	h.mu.Unlock()

	if tr != nil {
		tr.recordGPIO(pin, s.Level())
	}
	h.Emit(Event{
		Kind: EventPin,
		Pin:  pin,
		High: s.Level(),
	})
	for _, fn := range listeners {
		fn(h, pin, s)
	}
}

// ClearPinInput disconnects an external input driver so pull resistors take effect.
func (h *HAL) ClearPinInput(pin Pin) {
	if pin == NoPin {
		return
	}
	h.mu.Lock()
	s := h.pins[pin]
	s.HasExternalInput = false
	s.ExternalInput = false
	h.pins[pin] = s
	listeners := append([]PinChangeFunc(nil), h.pinListeners[pin]...)
	h.mu.Unlock()

	for _, fn := range listeners {
		fn(h, pin, s)
	}
}

// PWMConfigure records the counter top for a PWM timer instance.
func (h *HAL) PWMConfigure(instance int32, _ float64, top uint32) {
	h.mu.Lock()
	if top == 0 {
		top = 0xffff
	}
	h.pwmTops[instance] = top
	h.mu.Unlock()
}

// PWMChannelConfigure binds a PWM timer instance and channel to a GPIO pin.
func (h *HAL) PWMChannelConfigure(instance, channel int32, pin Pin) {
	if pin == NoPin {
		return
	}
	h.mu.Lock()
	h.pwmPins[pwmKey{instance: instance, channel: uint8(channel)}] = pin
	h.mu.Unlock()
}

// PWMChannelSet updates a PWM channel duty cycle and notifies listeners on its bound pin.
func (h *HAL) PWMChannelSet(instance int32, channel uint8, value uint32) {
	h.mu.Lock()
	pin, ok := h.pwmPins[pwmKey{instance: instance, channel: channel}]
	if !ok || pin == NoPin {
		h.mu.Unlock()
		return
	}
	top := h.pwmTops[instance]
	if top == 0 {
		top = 0xffff
	}
	var duty uint8
	if value >= top {
		duty = 255
	} else {
		duty = uint8((uint64(value) * 255) / uint64(top))
	}
	s := h.pins[pin]
	s.Configured = true
	s.Mode = PinOutput
	s.PWM = duty
	s.Output = duty > 0
	h.pins[pin] = s
	listeners := append([]PinChangeFunc(nil), h.pinListeners[pin]...)
	tr := h.tracer
	h.mu.Unlock()

	if tr != nil {
		tr.recordPWM(pin, duty)
	}
	for _, fn := range listeners {
		fn(h, pin, s)
	}
}

// SetADC sets the simulated 16-bit analog reading on an ADC pin.
func (h *HAL) SetADC(pin Pin, value uint16) {
	if pin == NoPin {
		return
	}
	h.mu.Lock()
	s := h.pins[pin]
	s.ADC = value
	h.pins[pin] = s
	h.mu.Unlock()
}

// ADCRead returns the simulated 16-bit analog reading for an ADC pin.
func (h *HAL) ADCRead(pin Pin) uint16 {
	if pin == NoPin {
		return 0
	}
	h.mu.Lock()
	v := h.pins[pin].ADC
	h.mu.Unlock()
	return v
}

func (h *HAL) spiBusLocked(bus uint8) *spiBusState {
	b, ok := h.spiBuses[bus]
	if !ok {
		b = &spiBusState{}
		h.spiBuses[bus] = b
	}
	return b
}

// AttachSPI attaches a peripheral device model to an SPI bus.
func (h *HAL) AttachSPI(bus uint8, dev SPIDevice) {
	if dev == nil {
		return
	}
	h.mu.Lock()
	b := h.spiBusLocked(bus)
	b.devices = append(b.devices, dev)
	configured := b.configured
	cfg := b.config
	h.mu.Unlock()
	if configured {
		dev.Configure(cfg)
	}
}

// SPIConfigure configures an SPI bus and notifies attached SPI devices.
func (h *HAL) SPIConfigure(bus uint8, cfg SPIConfig) {
	h.mu.Lock()
	b := h.spiBusLocked(bus)
	b.configured = true
	b.config = cfg
	devs := append([]SPIDevice(nil), b.devices...)
	h.mu.Unlock()

	for _, dev := range devs {
		dev.Configure(cfg)
	}
}

// SPITransfer performs a single-byte full-duplex SPI transfer on bus.
func (h *HAL) SPITransfer(bus uint8, w byte) byte {
	var tx [1]byte
	var rx [1]byte
	tx[0] = w
	h.SPITx(bus, tx[:], rx[:])
	return rx[0]
}

// SPITx performs a multi-byte SPI transaction across all devices attached to bus.
func (h *HAL) SPITx(bus uint8, w, r []byte) {
	var stackDevs [4]SPIDevice
	h.mu.Lock()
	b := h.spiBusLocked(bus)
	var devs []SPIDevice
	if len(b.devices) <= len(stackDevs) {
		n := copy(stackDevs[:], b.devices)
		devs = stackDevs[:n]
	} else {
		devs = append([]SPIDevice(nil), b.devices...)
	}
	tr := h.tracer
	h.mu.Unlock()

	for i := range r {
		r[i] = 0
	}
	for _, dev := range devs {
		dev.Transfer(h, w, r)
	}
	if tr != nil {
		tr.recordSPI(bus, w, r)
	}
}

// SPIBus adapts a HAL SPI bus to tinygo.org/x/drivers.SPI.
type SPIBus struct {
	hal *HAL
	bus uint8
}

// SPI returns a drivers.SPI-compatible handle for the given bus number.
func (h *HAL) SPI(bus uint8) *SPIBus {
	return &SPIBus{hal: h, bus: bus}
}

// Configure configures the underlying HAL SPI bus.
func (b *SPIBus) Configure(cfg SPIConfig) error {
	b.hal.SPIConfigure(b.bus, cfg)
	return nil
}

// Transfer writes and reads a single byte on the SPI bus.
func (b *SPIBus) Transfer(w byte) (byte, error) {
	return b.hal.SPITransfer(b.bus, w), nil
}

// Tx performs a read/write transaction on the SPI bus.
func (b *SPIBus) Tx(w, r []byte) error {
	b.hal.SPITx(b.bus, w, r)
	return nil
}

func (h *HAL) i2cBusLocked(bus uint8) *i2cBusState {
	b, ok := h.i2cBuses[bus]
	if !ok {
		b = &i2cBusState{devices: make(map[uint16]I2CDevice)}
		h.i2cBuses[bus] = b
	}
	return b
}

// AttachI2C attaches an I2C peripheral device to bus at dev.Address().
func (h *HAL) AttachI2C(bus uint8, dev I2CDevice) {
	if dev == nil {
		return
	}
	h.mu.Lock()
	b := h.i2cBusLocked(bus)
	b.devices[dev.Address()] = dev
	h.mu.Unlock()
}

// I2CConfigure configures an I2C bus.
func (h *HAL) I2CConfigure(bus uint8, cfg I2CConfig) {
	h.mu.Lock()
	b := h.i2cBusLocked(bus)
	b.configured = true
	b.config = cfg
	h.mu.Unlock()
}

// I2CSetBaudRate updates the clock frequency of an I2C bus.
func (h *HAL) I2CSetBaudRate(bus uint8, br uint32) {
	h.mu.Lock()
	b := h.i2cBusLocked(bus)
	b.config.Frequency = br
	h.mu.Unlock()
}

// I2CTransfer executes an I2C transaction and returns a TinyGo machine_generic error code.
func (h *HAL) I2CTransfer(bus uint8, addr uint16, w, r []byte) int {
	h.mu.Lock()
	b := h.i2cBusLocked(bus)
	dev, ok := b.devices[addr]
	tr := h.tracer
	h.mu.Unlock()
	if !ok {
		return I2CErrNoDevices
	}
	for i := range r {
		r[i] = 0
	}
	if err := dev.Tx(h, w, r); err != nil {
		return I2CErrBus
	}
	if tr != nil {
		tr.recordI2C(bus, addr, w, r)
	}
	return I2COK
}

// I2CBus adapts a HAL I2C bus to tinygo.org/x/drivers.I2C.
type I2CBus struct {
	hal *HAL
	bus uint8
}

// I2C returns a drivers.I2C-compatible handle for the given bus number.
func (h *HAL) I2C(bus uint8) *I2CBus {
	return &I2CBus{hal: h, bus: bus}
}

// Configure configures the underlying HAL I2C bus.
func (b *I2CBus) Configure(cfg I2CConfig) error {
	b.hal.I2CConfigure(b.bus, cfg)
	return nil
}

// SetBaudRate updates the I2C bus frequency.
func (b *I2CBus) SetBaudRate(br uint32) error {
	b.hal.I2CSetBaudRate(b.bus, br)
	return nil
}

// Tx performs an I2C transaction at addr.
func (b *I2CBus) Tx(addr uint16, w, r []byte) error {
	code := b.hal.I2CTransfer(b.bus, addr, w, r)
	if code != I2COK {
		return ErrNoI2CDevice
	}
	return nil
}

func (h *HAL) uartBusLocked(bus uint8) *uartBusState {
	u, ok := h.uartBuses[bus]
	if !ok {
		u = &uartBusState{}
		h.uartBuses[bus] = u
	}
	return u
}

// UARTConfigure configures a simulated UART bus.
func (h *HAL) UARTConfigure(bus uint8, cfg UARTConfig) {
	h.mu.Lock()
	u := h.uartBusLocked(bus)
	u.configured = true
	u.config = cfg
	h.mu.Unlock()
}

// UARTRead reads up to len(buf) bytes from the simulated UART RX buffer.
func (h *HAL) UARTRead(bus uint8, buf []byte) int {
	if len(buf) == 0 {
		return 0
	}
	h.mu.Lock()
	u := h.uartBusLocked(bus)
	n := copy(buf, u.rx)
	u.rx = u.rx[n:]
	h.mu.Unlock()
	return n
}

// UARTWrite appends data to the simulated UART TX buffer and emits an EventUART.
func (h *HAL) UARTWrite(bus uint8, data []byte) int {
	if len(data) == 0 {
		return 0
	}
	cp := append([]byte(nil), data...)
	h.mu.Lock()
	u := h.uartBusLocked(bus)
	u.tx = append(u.tx, cp...)
	tr := h.tracer
	h.mu.Unlock()

	if tr != nil {
		tr.recordUART(bus, cp)
	}
	h.Emit(Event{
		Kind: EventUART,
		Bus:  bus,
		Data: cp,
	})
	return len(data)
}

// UARTTX returns a copy of all bytes written to the simulated UART TX buffer.
func (h *HAL) UARTTX(bus uint8) []byte {
	h.mu.Lock()
	u := h.uartBusLocked(bus)
	out := append([]byte(nil), u.tx...)
	h.mu.Unlock()
	return out
}

// InjectUARTRX queues incoming serial bytes into the simulated UART RX buffer.
func (h *HAL) InjectUARTRX(bus uint8, data []byte) {
	if len(data) == 0 {
		return
	}
	h.mu.Lock()
	u := h.uartBusLocked(bus)
	u.rx = append(u.rx, data...)
	h.mu.Unlock()
}
