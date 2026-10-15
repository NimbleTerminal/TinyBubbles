package tinyhal

import (
	"image/color"
	"sync"

	"tinygo.org/x/drivers"
)

// ST7789 command opcodes from the ST7789 datasheet and tinygo.org/x/drivers/st7789.
const (
	st7789NOP     = 0x00
	st7789SWRESET = 0x01
	st7789SLPIN   = 0x10
	st7789SLPOUT  = 0x11
	st7789NORON   = 0x13
	st7789INVOFF  = 0x20
	st7789INVON   = 0x21
	st7789DISPOFF = 0x28
	st7789DISPON  = 0x29
	st7789CASET   = 0x2A
	st7789RASET   = 0x2B
	st7789RAMWR   = 0x2C
	st7789MADCTL  = 0x36
	st7789COLMOD  = 0x3A
	st7789GSCAN   = 0x45

	st7789MADCTLMV = 0x20

	st7789ColorRGB444 = 0x03
	st7789ColorRGB565 = 0x05
)

// ST7789Config configures a virtual ST7789 SPI display peripheral.
type ST7789Config struct {
	ID           string
	Bus          uint8
	Width        int16
	Height       int16
	RowOffset    int16
	ColumnOffset int16
	Reset        Pin
	DC           Pin
	CS           Pin
	Backlight    Pin
	Target       drivers.Displayer
}

// ST7789 models an ST7789 TFT display connected over SPI and control GPIO pins.
// It decodes CASET, RASET, RAMWR, MADCTL, and COLMOD commands emitted by
// tinygo.org/x/drivers/st7789 into an RGBA8888 framebuffer and optional Target Displayer.
type ST7789 struct {
	hal          *HAL
	id           string
	width        int16
	height       int16
	rowOffsetCfg int16
	colOffsetCfg int16
	resetPin     Pin
	dcPin        Pin
	csPin        Pin
	blPin        Pin
	target       drivers.Displayer

	mu         sync.Mutex
	pixels     []byte
	snapshot   []byte
	cmd        byte
	paramBuf   [6]byte
	paramLen   int
	xs, xe     int16
	ys, ye     int16
	curX, curY int16
	madctl     byte
	colmod     byte
	sleeping   bool
	displayOn  bool
	inverted   bool
	backlight  bool
	dirty      bool
	halfByte   byte
	hasHalf    bool
	frames     int
}

// AttachST7789 attaches a virtual ST7789 display to cfg.Bus and its control pins on h.
func AttachST7789(h *HAL, cfg ST7789Config) *ST7789 {
	w, height := cfg.Width, cfg.Height
	if w <= 0 {
		w = 240
	}
	if height <= 0 {
		height = 135
	}
	d := &ST7789{
		hal:          h,
		id:           cfg.ID,
		width:        w,
		height:       height,
		rowOffsetCfg: cfg.RowOffset,
		colOffsetCfg: cfg.ColumnOffset,
		resetPin:     cfg.Reset,
		dcPin:        cfg.DC,
		csPin:        cfg.CS,
		blPin:        cfg.Backlight,
		target:       cfg.Target,
		pixels:       make([]byte, int(w)*int(height)*4),
		snapshot:     make([]byte, int(w)*int(height)*4),
		colmod:       st7789ColorRGB565,
		xe:           w - 1,
		ye:           height - 1,
		backlight:    true,
	}
	for i := 3; i < len(d.pixels); i += 4 {
		d.pixels[i] = 255
	}
	h.AttachSPI(cfg.Bus, d)
	if cfg.CS != NoPin {
		h.WatchPin(cfg.CS, d.onCSChange)
	}
	if cfg.Reset != NoPin {
		h.WatchPin(cfg.Reset, d.onResetChange)
	}
	if cfg.Backlight != NoPin {
		h.WatchPin(cfg.Backlight, d.onBacklightChange)
	}
	return d
}

// Configure implements SPIDevice.
func (d *ST7789) Configure(_ SPIConfig) {}

func (d *ST7789) onCSChange(h *HAL, _ Pin, s PinState) {
	// Flush buffered framebuffer updates when CS is deasserted (rising edge).
	if s.Level() {
		d.flushIfDirty(h)
	}
}

func (d *ST7789) onResetChange(_ *HAL, _ Pin, s PinState) {
	if !s.Level() {
		d.mu.Lock()
		d.resetLocked()
		d.mu.Unlock()
	}
}

func (d *ST7789) onBacklightChange(_ *HAL, _ Pin, s PinState) {
	d.mu.Lock()
	d.backlight = s.Level()
	d.mu.Unlock()
}

func (d *ST7789) resetLocked() {
	d.cmd = st7789NOP
	d.paramLen = 0
	d.xs, d.xe = 0, d.width-1
	d.ys, d.ye = 0, d.height-1
	d.curX, d.curY = 0, 0
	d.colmod = st7789ColorRGB565
	d.sleeping = true
	d.displayOn = false
	d.hasHalf = false
}

// Transfer implements SPIDevice and decodes ST7789 command and data streams.
func (d *ST7789) Transfer(h *HAL, w, r []byte) {
	if d.csPin != NoPin && h.PinLevel(d.csPin) {
		return
	}
	isData := true
	if d.dcPin != NoPin {
		isData = h.PinLevel(d.dcPin)
	}

	d.mu.Lock()
	if !isData {
		for _, b := range w {
			d.cmd = b
			d.paramLen = 0
			d.hasHalf = false
			switch b {
			case st7789SWRESET:
				d.resetLocked()
			case st7789SLPIN:
				d.sleeping = true
			case st7789SLPOUT:
				d.sleeping = false
			case st7789INVOFF:
				d.inverted = false
			case st7789INVON:
				d.inverted = true
			case st7789DISPOFF:
				d.displayOn = false
			case st7789DISPON:
				d.displayOn = true
			case st7789RAMWR:
				d.curX = d.xs
				d.curY = d.ys
			}
		}
		d.mu.Unlock()
		return
	}

	if d.cmd == st7789GSCAN && len(r) > 0 {
		for i := range r {
			r[i] = 0
		}
		d.mu.Unlock()
		return
	}

	switch d.cmd {
	case st7789CASET:
		for _, b := range w {
			if d.paramLen < 4 {
				d.paramBuf[d.paramLen] = b
				d.paramLen++
			}
			if d.paramLen == 4 {
				colOff, _ := d.offsetsLocked()
				d.xs = int16(uint16(d.paramBuf[0])<<8|uint16(d.paramBuf[1])) - colOff
				d.xe = int16(uint16(d.paramBuf[2])<<8|uint16(d.paramBuf[3])) - colOff
			}
		}
	case st7789RASET:
		for _, b := range w {
			if d.paramLen < 4 {
				d.paramBuf[d.paramLen] = b
				d.paramLen++
			}
			if d.paramLen == 4 {
				_, rowOff := d.offsetsLocked()
				d.ys = int16(uint16(d.paramBuf[0])<<8|uint16(d.paramBuf[1])) - rowOff
				d.ye = int16(uint16(d.paramBuf[2])<<8|uint16(d.paramBuf[3])) - rowOff
			}
		}
	case st7789MADCTL:
		if len(w) > 0 {
			d.madctl = w[len(w)-1]
		}
	case st7789COLMOD:
		if len(w) > 0 {
			d.colmod = w[len(w)-1] & 0x07
		}
	case st7789RAMWR:
		d.writePixelsLocked(w)
	}
	noCS := d.csPin == NoPin
	d.mu.Unlock()

	if noCS {
		d.flushIfDirty(h)
	}
}

func (d *ST7789) offsetsLocked() (colOff, rowOff int16) {
	if d.madctl == 0 {
		return 0, 0
	}
	if (d.madctl & st7789MADCTLMV) != 0 {
		return d.rowOffsetCfg, d.colOffsetCfg
	}
	return d.colOffsetCfg, d.rowOffsetCfg
}

func (d *ST7789) writePixelsLocked(data []byte) {
	if len(data) == 0 {
		return
	}
	if d.colmod == st7789ColorRGB444 {
		// RGB444 packs two 12-bit pixels into 3 bytes (or 2 bytes per pixel in 16-bit transfers).
		for _, b := range data {
			if !d.hasHalf {
				d.halfByte = b
				d.hasHalf = true
				continue
			}
			hi, lo := d.halfByte, b
			d.hasHalf = false
			r4 := (hi >> 4) & 0x0f
			g4 := hi & 0x0f
			b4 := (lo >> 4) & 0x0f
			d.putWindowPixelLocked(color.RGBA{
				R: r4 * 17,
				G: g4 * 17,
				B: b4 * 17,
				A: 255,
			})
		}
		return
	}

	// Default RGB565BE: 2 bytes per pixel (big-endian).
	for _, b := range data {
		if !d.hasHalf {
			d.halfByte = b
			d.hasHalf = true
			continue
		}
		v := uint16(d.halfByte)<<8 | uint16(b)
		d.hasHalf = false
		r5 := uint8((v >> 11) & 0x1f)
		g6 := uint8((v >> 5) & 0x3f)
		b5 := uint8(v & 0x1f)
		d.putWindowPixelLocked(color.RGBA{
			R: (r5 << 3) | (r5 >> 2),
			G: (g6 << 2) | (g6 >> 4),
			B: (b5 << 3) | (b5 >> 2),
			A: 255,
		})
	}
}

func (d *ST7789) putWindowPixelLocked(c color.RGBA) {
	if d.curX >= 0 && d.curX < d.width && d.curY >= 0 && d.curY < d.height {
		i := (int(d.curY)*int(d.width) + int(d.curX)) * 4
		d.pixels[i] = c.R
		d.pixels[i+1] = c.G
		d.pixels[i+2] = c.B
		d.pixels[i+3] = 255
		d.dirty = true
	}
	d.curX++
	if d.curX > d.xe {
		d.curX = d.xs
		d.curY++
		if d.curY > d.ye {
			d.curY = d.ys
		}
	}
}

func (d *ST7789) flushIfDirty(h *HAL) {
	d.mu.Lock()
	if !d.dirty {
		d.mu.Unlock()
		return
	}
	d.dirty = false
	d.frames++
	snapshot := d.snapshot
	copy(snapshot, d.pixels)
	w, height := d.width, d.height
	target := d.target
	id := d.id
	d.mu.Unlock()

	if target != nil {
		for y := int16(0); y < height; y++ {
			row := int(y) * int(w) * 4
			for x := int16(0); x < w; x++ {
				i := row + int(x)*4
				target.SetPixel(x, y, color.RGBA{
					R: snapshot[i],
					G: snapshot[i+1],
					B: snapshot[i+2],
					A: 255,
				})
			}
		}
		_ = target.Display()
	}

	h.Emit(Event{
		Kind:   EventFramebuffer,
		ID:     id,
		Width:  w,
		Height: height,
		Pixels: snapshot,
	})
}

// Size returns the dimensions of the virtual ST7789 display.
func (d *ST7789) Size() (int16, int16) {
	return d.width, d.height
}

// SetPixel writes a pixel directly to the virtual framebuffer.
func (d *ST7789) SetPixel(x, y int16, c color.RGBA) {
	if x < 0 || y < 0 || x >= d.width || y >= d.height {
		return
	}
	d.mu.Lock()
	i := (int(y)*int(d.width) + int(x)) * 4
	d.pixels[i] = c.R
	d.pixels[i+1] = c.G
	d.pixels[i+2] = c.B
	d.pixels[i+3] = 255
	d.dirty = true
	d.mu.Unlock()
}

// FillRectangle fills a rectangle directly in the virtual framebuffer.
func (d *ST7789) FillRectangle(x, y, width, height int16, c color.RGBA) error {
	left, top := max(int16(0), x), max(int16(0), y)
	right, bottom := min(d.width, x+width), min(d.height, y+height)
	d.mu.Lock()
	for yy := top; yy < bottom; yy++ {
		row := int(yy) * int(d.width) * 4
		for xx := left; xx < right; xx++ {
			i := row + int(xx)*4
			d.pixels[i] = c.R
			d.pixels[i+1] = c.G
			d.pixels[i+2] = c.B
			d.pixels[i+3] = 255
		}
	}
	d.dirty = true
	d.mu.Unlock()
	d.flushIfDirty(d.hal)
	return nil
}

// Display flushes any pending framebuffer changes to Target and emits EventFramebuffer.
func (d *ST7789) Display() error {
	d.flushIfDirty(d.hal)
	return nil
}

// Pixel returns the RGBA color at (x, y) in the virtual framebuffer.
func (d *ST7789) Pixel(x, y int16) color.RGBA {
	if x < 0 || y < 0 || x >= d.width || y >= d.height {
		return color.RGBA{}
	}
	d.mu.Lock()
	i := (int(y)*int(d.width) + int(x)) * 4
	c := color.RGBA{
		R: d.pixels[i],
		G: d.pixels[i+1],
		B: d.pixels[i+2],
		A: d.pixels[i+3],
	}
	d.mu.Unlock()
	return c
}

// Pixels returns a copy of the raw RGBA8888 framebuffer (4 * Width * Height bytes).
func (d *ST7789) Pixels() []byte {
	d.mu.Lock()
	out := make([]byte, len(d.pixels))
	copy(out, d.pixels)
	d.mu.Unlock()
	return out
}

// Frames returns the number of framebuffer flushes emitted by the display.
func (d *ST7789) Frames() int {
	d.mu.Lock()
	n := d.frames
	d.mu.Unlock()
	return n
}

// Backlight reports whether the display backlight pin is high.
func (d *ST7789) Backlight() bool {
	d.mu.Lock()
	b := d.backlight
	d.mu.Unlock()
	return b
}
