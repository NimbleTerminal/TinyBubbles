// Command terminal runs the Bubble Tea hello model with the 240x135 Feather
// display drawn inside a terminal. ntcharts picture shows the framebuffer as
// Kitty graphics where the terminal supports them and as half-block cells
// elsewhere. It builds natively and for WASI preview 1 with Go or TinyGo; see
// README.md for the terminal setup each build needs.
package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/NimbleMarkets/ntcharts/v2/picture"
	"github.com/charmbracelet/colorprofile"
	displaytea "nimbleterminal.dev/tinybubbles/bubbletea"
	teamodel "nimbleterminal.dev/tinybubbles/bubbletea/hello"
	"nimbleterminal.dev/tinybubbles/demo/hello"
)

const width, height = 240, 135

// display is the Displayer the TinyBubbles renderer draws into: a plain RGBA
// framebuffer that picture maps onto terminal cells.
type display struct {
	*image.RGBA
	frames int
}

func (d *display) Size() (int16, int16)              { return width, height }
func (d *display) SetPixel(x, y int16, c color.RGBA) { d.SetRGBA(int(x), int(y), c) }
func (d *display) Display() error                    { d.frames++; return nil }

// app owns the display model and the picture that shows its framebuffer.
type app struct {
	inner     *displaytea.DisplayModel
	fb        *display
	pic       picture.Model
	wantKitty bool
	shown     int // fb.frames last handed to picture
	status    string
}

func (a *app) Init() tea.Cmd { return tea.Batch(a.inner.Init(), a.pic.Init()) }

func (a *app) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	if k, ok := msg.(tea.KeyPressMsg); ok {
		switch k.String() {
		case "ctrl+c", "ctrl+d":
			// The terminal is in raw mode, so Ctrl-C arrives as a key, not a signal.
			return a, tea.Quit
		case "k":
			if err := a.pic.ToggleBlocked(); err != nil && a.pic.Mode() == picture.PictureGlyph {
				a.status = err.Error()
			} else {
				cmds = append(cmds, a.pic.Toggle())
				a.status = ""
			}
		}
	}
	_, cmd := a.inner.Update(msg)
	cmds = append(cmds, cmd)
	if _, quitting := msg.(tea.QuitMsg); quitting {
		return a, tea.Batch(cmds...)
	}
	cmds = append(cmds, a.pic.Update(msg))
	// Picture's own messages (Kitty frames, probe replies) must not trigger a
	// new SetImage, or Kitty mode re-encodes a frame for every frame it sends.
	if picture.IsPictureMsg(msg) {
		return a, tea.Batch(cmds...)
	}
	if a.wantKitty && a.pic.Mode() == picture.PictureGlyph && a.pic.ToggleBlocked() == nil {
		cmds = append(cmds, a.pic.Toggle())
		a.wantKitty = false
	}
	// The adapter renders cells into the framebuffer in View. Do it here so
	// picture can schedule its Kitty encode command from Update, and only
	// hand over the image when the renderer flushed a new frame.
	a.inner.View()
	if a.fb.frames != a.shown {
		a.shown = a.fb.frames
		cmds = append(cmds, a.pic.SetImage(a.fb.RGBA))
	}
	return a, tea.Batch(cmds...)
}

func (a *app) View() tea.View {
	v := a.pic.View()
	if a.status != "" {
		v.Content += "\n" + a.status
	}
	v.AltScreen = true
	return v
}

func main() {
	cols := flag.Int("cols", 240, "terminal columns for the display")
	rows := flag.Int("rows", 68, "terminal rows for the display")
	glyph := flag.Bool("glyph", false, "stay in half-block mode instead of switching to Kitty graphics")
	stats := flag.Bool("stats", false, "print frame statistics on exit")
	snapshot := flag.String("snapshot", "", "write the final framebuffer to this PNG on exit")
	flag.Parse()

	fb := &display{RGBA: image.NewRGBA(image.Rect(0, 0, width, height))}
	inner, err := displaytea.NewModel(teamodel.New(), fb, hello.NewFont(), hello.Theme())
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer inner.Close()
	// Kitty placements are pre-scaled to cols*cellWidth by rows*cellHeight
	// pixels, about 1920x1088 here. The display is 240x135, so encode at 1/8
	// and let the terminal upscale: 16 times fewer pixels per frame.
	pic := picture.NewWithConfig(picture.Config{Fit: picture.FitFill, KittyFormat: picture.KittyFormatZlib, KittyResolutionFactor: 0.125})
	pic.SetSize(*cols, *rows)
	a := &app{inner: inner, fb: fb, pic: pic, wantKitty: !*glyph}

	p := tea.NewProgram(a, tea.WithContext(inner.Context()), tea.WithInput(stdin()),
		// WASI cannot tell whether stdout is a terminal, and profile detection
		// would then strip every color.
		tea.WithColorProfile(colorprofile.TrueColor),
		tea.WithoutSignalHandler(), tea.WithoutCatchPanics(), tea.WithWindowSize(*cols, *rows+1))
	start := time.Now()
	_, err = p.Run()
	elapsed := time.Since(start)
	if err != nil {
		fmt.Fprintln(os.Stderr, "run error:", err)
	}
	if inner.Err() != nil {
		fmt.Fprintln(os.Stderr, "display error:", inner.Err())
	}
	if *stats {
		fmt.Fprintf(os.Stderr, "frames=%d elapsed=%s fps=%.1f mode=%d kitty=%d stdin=%s\r\n",
			fb.frames, elapsed.Round(time.Millisecond), float64(fb.frames)/elapsed.Seconds(), a.pic.Mode(), picture.KittySupported(), stdinMode)
	}
	if *snapshot != "" {
		if err := writePNG(*snapshot, fb.RGBA); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
	if err != nil || inner.Err() != nil {
		os.Exit(1)
	}
}

func writePNG(path string, img image.Image) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	if err := png.Encode(f, img); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}
