//go:build js && wasm

package web

import (
	"encoding/json"
	"fmt"
	"image/color"
	"strconv"
	"syscall/js"

	tea "charm.land/bubbletea/v2"
	booba "github.com/NimbleMarkets/go-booba/wasm"
)

type CanvasDisplay struct {
	width, height          int16
	pixels                 []byte
	canvas, context, image js.Value
	frames                 int
}

func NewCanvasDisplay(id string, width, height int16) *CanvasDisplay {
	if width <= 0 || height <= 0 {
		panic("web: display size must be positive")
	}
	canvas := js.Global().Get("document").Call("getElementById", id)
	if canvas.IsNull() {
		panic("web: missing canvas " + id)
	}
	canvas.Set("width", int(width))
	canvas.Set("height", int(height))
	ctx := canvas.Call("getContext", "2d")
	return &CanvasDisplay{width: width, height: height, pixels: make([]byte, int(width)*int(height)*4), canvas: canvas, context: ctx, image: ctx.Call("createImageData", int(width), int(height))}
}
func (d *CanvasDisplay) Size() (int16, int16) { return d.width, d.height }
func (d *CanvasDisplay) SetPixel(x, y int16, c color.RGBA) {
	if x < 0 || y < 0 || x >= d.width || y >= d.height {
		return
	}
	i := (int(y)*int(d.width) + int(x)) * 4
	d.pixels[i], d.pixels[i+1], d.pixels[i+2], d.pixels[i+3] = c.R, c.G, c.B, 255
}
func (d *CanvasDisplay) FillRectangle(x, y, w, h int16, c color.RGBA) error {
	left, top := max(0, int(x)), max(0, int(y))
	right, bottom := min(int(d.width), int(x)+int(w)), min(int(d.height), int(y)+int(h))
	for yy := top; yy < bottom; yy++ {
		for xx := left; xx < right; xx++ {
			d.SetPixel(int16(xx), int16(yy), c)
		}
	}
	return nil
}
func (d *CanvasDisplay) Display() error {
	js.CopyBytesToJS(d.image.Get("data"), d.pixels)
	d.context.Call("putImageData", d.image, 0, 0)
	d.frames++
	d.canvas.Get("dataset").Set("frames", d.frames)
	return nil
}

type LEDs struct{ elements js.Value }

func NewLEDs(class string) *LEDs {
	return &LEDs{js.Global().Get("document").Call("getElementsByClassName", class)}
}
func (l *LEDs) Len() int { return l.elements.Length() }
func (l *LEDs) WriteColors(colors []color.RGBA) error {
	for i, c := range colors {
		if i >= l.Len() {
			break
		}
		l.elements.Index(i).Get("style").Set("background", "rgb("+strconv.Itoa(int(c.R))+","+strconv.Itoa(int(c.G))+","+strconv.Itoa(int(c.B))+")")
	}
	return nil
}

// Configure fills the page and returns its validated URL selections.
func Configure(config Config) Selection {
	if len(config.Screens) == 0 {
		panic("web: at least one screen is required")
	}
	for _, screen := range config.Screens {
		if screen.Width <= 0 || screen.Height <= 0 {
			panic("web: invalid screen size")
		}
	}
	data, err := json.Marshal(config)
	if err != nil {
		panic(err)
	}
	result := js.Global().Call("tinybubblesConfigure", string(data))
	var selection Selection
	if err := json.Unmarshal([]byte(result.String()), &selection); err != nil {
		panic(err)
	}
	return selection
}

type RunOptions struct {
	Program []tea.ProgramOption
	Error   func() error
}

// Run keeps the WASM instance alive after notifying the page that it stopped.
func Run(model tea.Model, config RunOptions) {
	options := append([]tea.ProgramOption{}, config.Program...)
	options = append(options, tea.WithoutRenderer(), tea.WithoutSignalHandler(), tea.WithoutCatchPanics())
	_, err := booba.NewProgram(model, options...).Run()
	if config.Error != nil {
		if appErr := config.Error(); appErr != nil {
			err = appErr
		}
	}
	Stopped(err)
	select {}
}
func Stopped(err error) {
	message := "Stopped. Restart to run again."
	if err != nil {
		message = fmt.Sprintf("Stopped: %v", err)
	}
	js.Global().Call("tinybubblesStopped", message, err != nil)
}
