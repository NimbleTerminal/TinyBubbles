//go:build js && wasm

package main

import (
	"image/color"
	"syscall/js"

	tea "charm.land/bubbletea/v2"
	booba "github.com/NimbleMarkets/go-booba/wasm"
	demo "nimbleterminal.dev/tinybubbles/web"
)

// canvasDisplay is a browser implementation of the same hardware contract.
type canvasDisplay struct {
	width, height      int16
	pixels             []byte
	context, imageData js.Value
	frames             int
	canvas             js.Value
}

func newDisplay(width, height int16) *canvasDisplay {
	canvas := js.Global().Get("document").Call("getElementById", "display")
	canvas.Set("width", int(width))
	canvas.Set("height", int(height))
	ctx := canvas.Call("getContext", "2d")
	return &canvasDisplay{width: width, height: height, pixels: make([]byte, int(width)*int(height)*4), context: ctx, imageData: ctx.Call("createImageData", int(width), int(height)), canvas: canvas}
}
func (d *canvasDisplay) Size() (int16, int16) { return d.width, d.height }
func (d *canvasDisplay) SetPixel(x, y int16, c color.RGBA) {
	if x < 0 || y < 0 || x >= d.width || y >= d.height {
		return
	}
	i := (int(y)*int(d.width) + int(x)) * 4
	d.pixels[i] = c.R
	d.pixels[i+1] = c.G
	d.pixels[i+2] = c.B
	d.pixels[i+3] = 255
}
func (d *canvasDisplay) FillRectangle(x, y, w, h int16, c color.RGBA) error {
	for yy := y; yy < y+h; yy++ {
		for xx := x; xx < x+w; xx++ {
			d.SetPixel(xx, yy, c)
		}
	}
	return nil
}
func (d *canvasDisplay) Display() error {
	js.CopyBytesToJS(d.imageData.Get("data"), d.pixels)
	d.context.Call("putImageData", d.imageData, 0, 0)
	d.frames++
	d.canvas.Get("dataset").Set("frames", d.frames)
	return nil
}
func main() {
	width, height := int16(240), int16(135)
	mono := false
	switch js.Global().Get("tinybubblesScreen").String() {
	case "oled":
		width, height, mono = 128, 64, true
	case "large":
		width, height = 320, 240
	}
	model, pixels, err := demo.NewDemo(newDisplay(width, height), mono)
	if err != nil {
		panic(err)
	}
	cols, rows := pixels.Size()
	p := booba.NewProgram(model, tea.WithContext(pixels.Context()), tea.WithoutRenderer(), tea.WithoutSignalHandler(), tea.WithoutCatchPanics(), tea.WithWindowSize(cols, rows))
	_, err = p.Run()
	if pixels.Err() != nil {
		err = pixels.Err()
	}
	pixels.Close()
	message := "Demo stopped. Reload to start again."
	if err != nil {
		message = "Demo stopped: " + err.Error()
	}
	js.Global().Call("tinybubblesStopped", message)
	// Booba owns browser callbacks. Keep the Go runtime alive after disconnect.
	select {}
}
