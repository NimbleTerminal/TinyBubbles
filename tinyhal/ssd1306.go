package tinyhal

import (
	"image/color"
	"sync"

	"tinygo.org/x/drivers"
)

// SSD1306 command constants matching tinygo.org/x/drivers/ssd1306.
const (
	ssd1306DefaultAddr   = 0x3c
	ssd1306ColumnAddr    = 0x21
	ssd1306PageAddr      = 0x22
	ssd1306NormalDisplay = 0xA6
	ssd1306InvertDisplay = 0xA7
	ssd1306DisplayOff    = 0xAE
	ssd1306DisplayOn     = 0xAF
)

// SSD1306Config configures a virtual SSD1306 I2C monochrome OLED display peripheral.
type SSD1306Config struct {
	ID       string
	Bus      uint8
	Address  uint16
	Width    int16
	Height   int16
	OnColor  color.RGBA
	OffColor color.RGBA
	Target   drivers.Displayer
}

// SSD1306 models an I2C SSD1306 OLED controller (default 128x64 at 0x3c).
// It decodes command (0x00) and GDDRAM data (0x40) transactions produced by
// tinygo.org/x/drivers/ssd1306 into an RGBA8888 framebuffer and optional Target Displayer.
type SSD1306 struct {
	hal      *HAL
	id       string
	addr     uint16
	width    int16
	height   int16
	onColor  color.RGBA
	offColor color.RGBA
	target   drivers.Displayer

	mu        sync.Mutex
	gddram    []byte
	pixels    []byte
	snapshot  []byte
	colStart  int16
	colEnd    int16
	pageStart int16
	pageEnd   int16
	curCol    int16
	curPage   int16
	pending   byte
	paramStep int
	displayOn bool
	inverted  bool
	frames    int
}

// AttachSSD1306 attaches a virtual SSD1306 I2C display peripheral to cfg.Bus on h.
func AttachSSD1306(h *HAL, cfg SSD1306Config) *SSD1306 {
	addr := cfg.Address
	if addr == 0 {
		addr = ssd1306DefaultAddr
	}
	w, height := cfg.Width, cfg.Height
	if w <= 0 {
		w = 128
	}
	if height <= 0 {
		height = 64
	}
	on := cfg.OnColor
	if on == (color.RGBA{}) {
		on = color.RGBA{R: 255, G: 255, B: 255, A: 255}
	}
	off := cfg.OffColor
	if off.A == 0 {
		off = color.RGBA{R: 0, G: 0, B: 0, A: 255}
	}
	pages := max(int16(1), height/8)
	d := &SSD1306{
		hal:       h,
		id:        cfg.ID,
		addr:      addr,
		width:     w,
		height:    height,
		onColor:   on,
		offColor:  off,
		target:    cfg.Target,
		gddram:    make([]byte, int(w)*int(pages)),
		pixels:    make([]byte, int(w)*int(height)*4),
		snapshot:  make([]byte, int(w)*int(height)*4),
		colStart:  0,
		colEnd:    w - 1,
		pageStart: 0,
		pageEnd:   pages - 1,
	}
	d.renderLocked()
	h.AttachI2C(cfg.Bus, d)
	return d
}

// Address returns the 7-bit I2C address of the virtual SSD1306 controller.
func (d *SSD1306) Address() uint16 {
	return d.addr
}

// Tx processes an I2C transaction directed to the SSD1306 controller.
func (d *SSD1306) Tx(h *HAL, w, _ []byte) error {
	if len(w) < 2 {
		return nil
	}
	control := w[0]
	payload := w[1:]

	d.mu.Lock()
	switch control {
	case 0x00:
		for _, b := range payload {
			d.handleCommandByteLocked(b)
		}
		d.mu.Unlock()
		return nil
	case 0x40:
		pages := max(int16(1), d.height/8)
		for _, b := range payload {
			if d.curCol >= 0 && d.curCol < d.width && d.curPage >= 0 && d.curPage < pages {
				idx := int(d.curPage)*int(d.width) + int(d.curCol)
				if idx < len(d.gddram) {
					d.gddram[idx] = b
				}
			}
			d.curCol++
			if d.curCol > d.colEnd {
				d.curCol = d.colStart
				d.curPage++
				if d.curPage > d.pageEnd {
					d.curPage = d.pageStart
				}
			}
		}
		d.renderLocked()
		d.frames++
		snapshot := d.snapshot
		copy(snapshot, d.pixels)
		width, height := d.width, d.height
		target := d.target
		id := d.id
		d.mu.Unlock()

		if target != nil {
			for y := int16(0); y < height; y++ {
				row := int(y) * int(width) * 4
				for x := int16(0); x < width; x++ {
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
			Width:  width,
			Height: height,
			Pixels: snapshot,
		})
		return nil
	default:
		d.mu.Unlock()
		return nil
	}
}

func (d *SSD1306) handleCommandByteLocked(b byte) {
	if d.paramStep > 0 {
		switch d.pending {
		case ssd1306ColumnAddr:
			if d.paramStep == 1 {
				d.colStart = int16(b)
				d.curCol = d.colStart
				d.paramStep = 2
			} else {
				d.colEnd = int16(b)
				d.paramStep = 0
			}
			return
		case ssd1306PageAddr:
			if d.paramStep == 1 {
				d.pageStart = int16(b)
				d.curPage = d.pageStart
				d.paramStep = 2
			} else {
				d.pageEnd = int16(b)
				d.paramStep = 0
			}
			return
		}
	}

	switch b {
	case ssd1306ColumnAddr, ssd1306PageAddr:
		d.pending = b
		d.paramStep = 1
	case ssd1306DisplayOff:
		d.displayOn = false
	case ssd1306DisplayOn:
		d.displayOn = true
	case ssd1306NormalDisplay:
		d.inverted = false
	case ssd1306InvertDisplay:
		d.inverted = true
	}
}

func (d *SSD1306) renderLocked() {
	for y := int16(0); y < d.height; y++ {
		page := y / 8
		bit := uint8(y % 8)
		for x := int16(0); x < d.width; x++ {
			idx := int(page)*int(d.width) + int(x)
			lit := false
			if idx < len(d.gddram) {
				lit = ((d.gddram[idx] >> bit) & 1) == 1
			}
			if d.inverted {
				lit = !lit
			}
			c := d.offColor
			if lit {
				c = d.onColor
			}
			pi := (int(y)*int(d.width) + int(x)) * 4
			d.pixels[pi] = c.R
			d.pixels[pi+1] = c.G
			d.pixels[pi+2] = c.B
			d.pixels[pi+3] = 255
		}
	}
}

// Size returns the dimensions of the virtual SSD1306 display.
func (d *SSD1306) Size() (int16, int16) {
	return d.width, d.height
}

// Pixel returns the RGBA color at (x, y) in the virtual framebuffer.
func (d *SSD1306) Pixel(x, y int16) color.RGBA {
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
func (d *SSD1306) Pixels() []byte {
	d.mu.Lock()
	out := make([]byte, len(d.pixels))
	copy(out, d.pixels)
	d.mu.Unlock()
	return out
}

// Frames returns the number of GDDRAM frame flushes received over I2C.
func (d *SSD1306) Frames() int {
	d.mu.Lock()
	n := d.frames
	d.mu.Unlock()
	return n
}
