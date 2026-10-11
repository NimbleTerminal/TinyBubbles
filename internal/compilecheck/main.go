//go:build tinygo

// This hardware-free firmware checks the core on targets without example wiring.
package main

import (
	bubbles "github.com/NimbleTerminal/TinyBubbles"
	"github.com/NimbleTerminal/TinyBubbles/demo"
	"image/color"
)

type display struct{ pixels uint32 }

func (*display) Size() (int16, int16)                { return 128, 64 }
func (d *display) SetPixel(x, y int16, c color.RGBA) { d.pixels += uint32(c.R) + uint32(x) + uint32(y) }
func (*display) Display() error                      { return nil }
func main() {
	d := &display{}
	p, err := bubbles.NewProgram(demo.New(), bubbles.Config{Display: d, Monochrome: true})
	if err != nil {
		panic(err)
	}
	if err = p.Send(bubbles.KeyMsg{Key: bubbles.KeySelect}); err != nil {
		panic(err)
	}
	if err = p.Send(bubbles.KeyMsg{Key: bubbles.KeyQuit}); err != nil {
		panic(err)
	}
	if err = p.Run(); err != nil {
		panic(err)
	}
	println(d.pixels)
}
