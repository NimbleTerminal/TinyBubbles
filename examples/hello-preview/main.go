// Render the actual hello model and font to a PNG and an animated GIF on the host.
package main

import (
	"flag"
	bubbles "github.com/NimbleTerminal/TinyBubbles"
	"github.com/NimbleTerminal/TinyBubbles/demo/hello"
	"image"
	"image/color"
	"image/draw"
	"image/gif"
	"image/png"
	"os"
	"path/filepath"
)

type display struct{ *image.RGBA }

func (d display) Size() (int16, int16)              { return 240, 135 }
func (d display) SetPixel(x, y int16, c color.RGBA) { d.SetRGBA(int(x), int(y), c) }
func (d display) Display() error                    { return nil }
func must(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	out := flag.String("out", "build", "output directory")
	flag.Parse()
	must(os.MkdirAll(*out, 0755))
	d := display{image.NewRGBA(image.Rect(0, 0, 240, 135))}
	theme := hello.Theme()
	p, err := bubbles.NewProgram(hello.New(), bubbles.Config{Display: d, Font: hello.NewFont(), Theme: &theme})
	must(err)
	must(p.Start())
	palette := color.Palette{theme.Styles[0].Background}
	for _, style := range theme.Styles {
		palette = append(palette, style.Foreground)
	}
	animation := &gif.GIF{LoopCount: 0}
	for frame := 0; frame < 120; frame++ {
		switch frame {
		case 15, 25, 100:
			must(p.Send(bubbles.KeyMsg{Key: bubbles.KeySelect}))
		case 35, 85:
			must(p.Send(bubbles.KeyMsg{Key: bubbles.KeyRight}))
		case 55, 70:
			must(p.Send(bubbles.KeyMsg{Key: bubbles.KeyBack}))
		}
		must(p.Send(hello.TickMsg{}))
		// Nearest-neighbor enlargement preserves the hardware pixel grid.
		large := image.NewRGBA(image.Rect(0, 0, 720, 405))
		for y := 0; y < 405; y++ {
			for x := 0; x < 720; x++ {
				large.SetRGBA(x, y, d.RGBAAt(x/3, y/3))
			}
		}
		if frame == 0 {
			f, err := os.Create(filepath.Join(*out, "hello.png"))
			must(err)
			must(png.Encode(f, large))
			must(f.Close())
		}
		indexed := image.NewPaletted(large.Bounds(), palette)
		draw.Draw(indexed, indexed.Bounds(), large, image.Point{}, draw.Src)
		animation.Image = append(animation.Image, indexed)
		animation.Delay = append(animation.Delay, 10)
	}
	f, err := os.Create(filepath.Join(*out, "hello.gif"))
	must(err)
	must(gif.EncodeAll(f, animation))
	must(f.Close())
}
