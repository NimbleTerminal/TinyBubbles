// Snapshot renders the shared demo to a PNG with ordinary Go, without a terminal.
package main

import (
	"image"
	"image/color"
	"image/png"
	bubbles "nimbleterminal.dev/tinybubbles"
	"nimbleterminal.dev/tinybubbles/demo"
	"os"
)

type display struct{ *image.RGBA }

func (d display) Size() (int16, int16)              { return int16(d.Rect.Dx()), int16(d.Rect.Dy()) }
func (d display) SetPixel(x, y int16, c color.RGBA) { d.SetRGBA(int(x), int(y), c) }
func (d display) Display() error                    { return nil }

func main() {
	d := display{image.NewRGBA(image.Rect(0, 0, 240, 135))}
	p, err := bubbles.NewProgram(demo.New(), bubbles.Config{Display: d})
	if err != nil {
		panic(err)
	}
	if err = p.Send(bubbles.KeyMsg{Key: bubbles.KeySelect}); err != nil {
		panic(err)
	}
	path := "snapshot.png"
	if len(os.Args) > 1 {
		path = os.Args[1]
	}
	f, err := os.Create(path)
	if err != nil {
		panic(err)
	}
	if err = png.Encode(f, d.RGBA); err != nil {
		panic(err)
	}
	if err = f.Close(); err != nil {
		panic(err)
	}
}
