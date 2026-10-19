// Render each shared widget to a small PNG on the host for widgets/README.md.
// The scenes use the default 6×8 font on a 128×64 display, enlarged twice.
package main

import (
	"flag"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"

	bubbles "nimbleterminal.dev/tinybubbles"
	"nimbleterminal.dev/tinybubbles/layout"
	"nimbleterminal.dev/tinybubbles/widgets/choice"
	"nimbleterminal.dev/tinybubbles/widgets/dialog"
	"nimbleterminal.dev/tinybubbles/widgets/list"
	"nimbleterminal.dev/tinybubbles/widgets/progress"
	"nimbleterminal.dev/tinybubbles/widgets/status"
	"nimbleterminal.dev/tinybubbles/widgets/value"
	"nimbleterminal.dev/tinybubbles/widgets/viewport"
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

// scene builds a view for a cell grid of the given size.
type scene struct {
	name string
	view func(columns, rows int) *bubbles.Buffer
}

func main() {
	out := flag.String("out", filepath.Join("docs", "assets", "widgets"), "output directory")
	flag.Parse()
	must(os.MkdirAll(*out, 0755))
	theme := bubbles.DefaultTheme(false)
	// Semantic status colors, as in the browser demo: Ready, Warning, Error.
	for i, c := range []color.RGBA{{80, 220, 120, 255}, {255, 190, 60, 255}, {255, 90, 90, 255}} {
		theme.Styles[4+i] = bubbles.Style{Foreground: c, Background: color.RGBA{A: 255}}
	}
	for _, s := range scenes() {
		d := display{image.NewRGBA(image.Rect(0, 0, width, height))}
		r, err := bubbles.NewRenderer(d, nil, theme)
		must(err)
		must(r.Render(s.view(r.Size())))
		large := image.NewRGBA(image.Rect(0, 0, width*scale, height*scale))
		for y := 0; y < height*scale; y++ {
			for x := 0; x < width*scale; x++ {
				large.SetRGBA(x, y, d.RGBAAt(x/scale, y/scale))
			}
		}
		f, err := os.Create(filepath.Join(*out, s.name+".png"))
		must(err)
		must(png.Encode(f, large))
		must(f.Close())
	}
}

func scenes() []scene {
	return []scene{
		{"list", func(w, h int) *bubbles.Buffer {
			m := list.New([]list.Item{
				{ID: "music", Label: "Music"},
				{ID: "settings", Label: "Settings"},
				{ID: "radio", Label: "Radio", Disabled: true},
				{ID: "alarms", Label: "Alarms"},
				{ID: "clock", Label: "Clock"},
				{ID: "lights", Label: "Lights"},
				{ID: "weather", Label: "Weather forecast"},
				{ID: "bluetooth", Label: "Bluetooth"},
				{ID: "about", Label: "About"},
			})
			m.SetSize(w, h)
			m.Move(1)
			return m.CellView()
		}},
		{"value", func(w, h int) *bubbles.Buffer {
			m, err := value.New(value.Config{Label: "Volume", Unit: "%", Min: 0, Max: 100, Step: 5, Value: 50})
			must(err)
			m.SetSize(w, h)
			m.Adjust(1)
			return m.CellView()
		}},
		{"dialog", func(w, h int) *bubbles.Buffer {
			m := dialog.New(dialog.Config{
				Title: "Reset settings?", Body: "Restore the default volume and brightness?",
				Confirm: "Reset", Cancel: "Keep settings",
			})
			m.SetSize(w, h)
			return m.CellView()
		}},
		{"choice", func(w, h int) *bubbles.Buffer {
			m, err := choice.New(choice.Config{Label: "Playback", Options: []string{"Single", "Repeat", "Shuffle"}})
			must(err)
			m.SetSize(w, h)
			m.Move(1)
			return m.CellView()
		}},
		{"progress", func(w, h int) *bubbles.Buffer {
			m := progress.New(progress.Config{Label: "Loading", Done: 35, Total: 100})
			m.SetSize(w, h)
			return m.CellView()
		}},
		{"status", func(w, h int) *bubbles.Buffer {
			b := bubbles.NewBuffer(w, h)
			rows := [...]struct {
				label, detail string
				state         status.State
			}{
				{"Wi-Fi", "Connected", status.Ready},
				{"Battery", "Low charge", status.Warning},
				{"Sensor", "No response", status.Error},
			}
			for i, row := range rows {
				m := status.New(status.Config{Label: row.label, Detail: row.detail, State: row.state})
				m.Styles[status.Ready], m.Styles[status.Warning], m.Styles[status.Error] = 4, 5, 6
				m.SetSize(w, 2)
				layout.Draw(b, layout.Rect{Y: 3 * i, Width: w, Height: 2}, m.CellView())
			}
			return b
		}},
		{"viewport", func(w, h int) *bubbles.Buffer {
			m := viewport.New("Device guide\n\nD0 pops a bubble.\nD1 sets the palette.\nD2 pauses motion.\n\nHold D2 to go back.\n\nLog\n12:00 boot ok\n12:01 wifi ready")
			m.SetSize(w, h)
			m.Scroll(2)
			return m.CellView()
		}},
	}
}
