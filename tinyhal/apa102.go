package tinyhal

import (
	"image/color"
	"sync"
)

// APA102Order specifies the subpixel byte order for APA102 / DotStar SPI LEDs,
// matching tinygo.org/x/drivers/apa102.
type APA102Order int

const (
	APA102OrderBGR APA102Order = iota
	APA102OrderBRG
	APA102OrderGRB
)

// APA102Config configures a virtual APA102 / DotStar addressable RGB LED strip on an SPI bus.
type APA102Config struct {
	ID     string
	Bus    uint8
	CS     Pin
	Count  int
	Order  APA102Order
	Writer LEDWriter
}

// APA102 models a chain of APA102 / DotStar addressable RGB LEDs over SPI.
//
// It decodes the 4-byte 0x00 start frame followed by 4-byte LED frames
// ([0xE0 | brightness5, c1, c2, c3]) emitted by tinygo.org/x/drivers/apa102,
// updates an optional LEDWriter (such as web.LEDs), and emits EventLEDStrip.
type APA102 struct {
	hal    *HAL
	id     string
	bus    uint8
	csPin  Pin
	count  int
	order  APA102Order
	writer LEDWriter

	mu        sync.Mutex
	colors    []color.RGBA
	staging   []color.RGBA
	zeroRun   int
	inFrame   bool
	ledIndex  int
	frameByte [4]byte
	bytePos   int
	updates   int
}

// AttachAPA102 attaches a virtual APA102 addressable RGB LED strip to cfg.Bus on h.
func AttachAPA102(h *HAL, cfg APA102Config) *APA102 {
	count := cfg.Count
	if count <= 0 {
		count = 1
	}
	cs := cfg.CS
	if cs == 0 {
		cs = NoPin
	}
	d := &APA102{
		hal:     h,
		id:      cfg.ID,
		bus:     cfg.Bus,
		csPin:   cs,
		count:   count,
		order:   cfg.Order,
		writer:  cfg.Writer,
		colors:  make([]color.RGBA, count),
		staging: make([]color.RGBA, count),
	}
	h.AttachSPI(cfg.Bus, d)
	return d
}

// Configure implements SPIDevice.
func (d *APA102) Configure(_ SPIConfig) {}

// Transfer implements SPIDevice and decodes APA102 start, pixel, and end frames.
func (d *APA102) Transfer(h *HAL, w, _ []byte) {
	if d.csPin != NoPin && h.PinLevel(d.csPin) {
		return
	}
	if len(w) == 0 {
		return
	}

	var flushed []color.RGBA
	d.mu.Lock()
	for _, b := range w {
		if !d.inFrame {
			if b == 0x00 {
				d.zeroRun++
				if d.zeroRun >= 4 {
					d.inFrame = true
					d.ledIndex = 0
					d.bytePos = 0
					d.zeroRun = 0
				}
			} else {
				d.zeroRun = 0
			}
			continue
		}

		// Inside LED data stream: allow a new 4-zero start frame if re-synchronized early.
		if d.bytePos == 0 && b == 0x00 {
			d.zeroRun = 1
			d.inFrame = false
			continue
		}
		// Ignore 0xFF end-frame clock bytes once all LEDs in the strip have been latched.
		if d.ledIndex >= d.count {
			if b == 0x00 {
				d.zeroRun = 1
				d.inFrame = false
			}
			continue
		}

		d.frameByte[d.bytePos] = b
		d.bytePos++
		if d.bytePos == 4 {
			d.bytePos = 0
			d.staging[d.ledIndex] = d.decodeLED(d.frameByte)
			d.ledIndex++
			if d.ledIndex == d.count {
				copy(d.colors, d.staging)
				d.updates++
				flushed = make([]color.RGBA, d.count)
				copy(flushed, d.colors)
			}
		}
	}
	writer := d.writer
	id := d.id
	bus := d.bus
	d.mu.Unlock()

	if flushed != nil {
		if writer != nil {
			_ = writer.WriteColors(flushed)
		}
		h.Emit(Event{
			Kind:   EventLEDStrip,
			ID:     id,
			Bus:    bus,
			Colors: flushed,
		})
	}
}

func (d *APA102) decodeLED(f [4]byte) color.RGBA {
	bri5 := f[0] & 0x1f
	alpha := (bri5 << 3) | (bri5 >> 2)
	var r, g, b byte
	switch d.order {
	case APA102OrderBRG:
		b, r, g = f[1], f[2], f[3]
	case APA102OrderGRB:
		g, r, b = f[1], f[2], f[3]
	default:
		b, g, r = f[1], f[2], f[3]
	}
	return scaleColor(color.RGBA{R: r, G: g, B: b, A: 255}, alpha)
}

// Colors returns a copy of the latched RGBA colors across the strip.
func (d *APA102) Colors() []color.RGBA {
	d.mu.Lock()
	out := make([]color.RGBA, len(d.colors))
	copy(out, d.colors)
	d.mu.Unlock()
	return out
}

// Updates returns the number of complete strip frames latched over SPI.
func (d *APA102) Updates() int {
	d.mu.Lock()
	n := d.updates
	d.mu.Unlock()
	return n
}
