// Render the composed settings menu to a small PNG on the host for widgets/README.md.
package main

import (
	"flag"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"

	tea "charm.land/bubbletea/v2"
	bubbles "nimbleterminal.dev/tinybubbles"
	"nimbleterminal.dev/tinybubbles/bubbletea/settings"
)

const width, height, scale = 128, 64, 2

type display struct{ *image.RGBA }

func (d display) Size() (int16, int16)              { return width, height }
func (d display) SetPixel(x, y int16, c color.RGBA) { d.SetRGBA(int(x), int(y), c) }
func (d display) Display() error                    { return nil }

func must(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	out := flag.String("out", filepath.Join("..", "docs", "assets", "widgets"), "output directory")
	flag.Parse()
	must(os.MkdirAll(*out, 0755))
	d := display{image.NewRGBA(image.Rect(0, 0, width, height))}
	r, err := bubbles.NewRenderer(d, nil, bubbles.DefaultTheme(false))
	must(err)
	m := settings.New()
	columns, rows := r.Size()
	m.Update(tea.WindowSizeMsg{Width: columns, Height: rows})
	m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	must(r.Render(m.CellView()))
	large := image.NewRGBA(image.Rect(0, 0, width*scale, height*scale))
	for y := 0; y < height*scale; y++ {
		for x := 0; x < width*scale; x++ {
			large.SetRGBA(x, y, d.RGBAAt(x/scale, y/scale))
		}
	}
	f, err := os.Create(filepath.Join(*out, "settings.png"))
	must(err)
	must(png.Encode(f, large))
	must(f.Close())
}
