// Command terminal runs the Bubble Tea hello model with the 240x135 Feather
// display drawn inside a terminal. ntpicture shows the framebuffer as Kitty
// graphics where the terminal supports them and as half-block cells
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

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/colorprofile"
	"nimbleterminal.dev/ntpicture"
	displaytea "nimbleterminal.dev/tinybubbles/bubbletea"
	teamodel "nimbleterminal.dev/tinybubbles/bubbletea/hello"
	"nimbleterminal.dev/tinybubbles/demo/hello"
)

const width, height = 240, 135

// unpacedHello leaves scheduling to the host's picture completion messages.
// The shared hello model retains its 100 ms timer on every other target.
type unpacedHello struct{ *teamodel.Model }

func (*unpacedHello) Init() tea.Cmd {
	return func() tea.Msg { return hello.TickMsg{} }
}

func (m *unpacedHello) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	_, cmd := m.Model.Update(msg)
	if _, tick := msg.(hello.TickMsg); tick {
		cmd = nil
	}
	return m, cmd
}

// display is the Displayer the TinyBubbles renderer draws into: a plain RGBA
// framebuffer that the picture maps onto terminal cells.
type display struct {
	*image.RGBA
	frames int
}

func (d *display) Size() (int16, int16)              { return width, height }
func (d *display) SetPixel(x, y int16, c color.RGBA) { d.SetRGBA(int(x), int(y), c) }
func (d *display) Display() error                    { d.frames++; return nil }

// app owns the display model and the picture that shows its framebuffer.
type app struct {
	inner      *displaytea.DisplayModel
	keys       teamodel.KeyMap
	fb         *display
	pic        *ntpicture.Picture
	cols, rows int
	wantKitty  bool
	shown      int // fb.frames last handed to the picture
	status     string
	uncapped   bool
	paused     bool
	presented  int
}

func (a *app) Init() tea.Cmd {
	return tea.Batch(a.inner.Init(), a.pic.Init(), a.pic.SetSize(a.cols, a.rows))
}

func (a *app) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	innerMsg := msg
	advance := false
	switch v := msg.(type) {
	case ntpicture.PresentedMsg:
		if v.Owner == a.pic && v.Generation == a.pic.Status().Generation {
			a.presented++
			if a.uncapped && !a.paused {
				innerMsg = hello.TickMsg{}
				advance = true
			}
		}
	case tea.KeyPressMsg:
		switch {
		case v.String() == "ctrl+c", v.String() == "ctrl+d", key.Matches(v, a.keys.Quit):
			// The terminal is in raw mode, so Ctrl-C arrives as a key. QuitMsg
			// never reaches Update, so delete the terminal placements first.
			return a, tea.Sequence(a.pic.Close(), tea.Quit)
		case v.String() == "k":
			// An explicit choice wins over a late capability-probe reply.
			a.wantKitty = false
			cmds = append(cmds, a.toggle())
		case key.Matches(v, a.keys.Pause):
			a.paused = !a.paused
		}
	case ntpicture.ErrorMsg:
		if v.Owner == a.pic && v.Generation == a.pic.Status().Generation && v.Err != nil {
			a.status = v.Err.Error()
		}
	}
	_, cmd := a.inner.Update(innerMsg)
	cmds = append(cmds, cmd, a.pic.Update(msg))
	if a.wantKitty && ntpicture.KittySupported() == ntpicture.KittyCapabilitySupported {
		a.wantKitty = false
		cmds = append(cmds, a.pic.SetMode(ntpicture.PictureKitty))
	}
	// The picture's own messages never carry a new frame, so skip rendering.
	if ntpicture.IsPictureMsg(msg) && !advance {
		return a, tea.Batch(cmds...)
	}
	// The adapter renders cells into the framebuffer in View. Do it here so
	// the picture can schedule its encode from Update, and only hand over the
	// image when the renderer flushed a new frame. SetImage copies the pixels,
	// so the renderer may keep drawing into the framebuffer.
	a.inner.View()
	// Some animation steps leave the pixels unchanged. Still complete a picture
	// in uncapped mode so the next step has a completion to advance from.
	if a.fb.frames != a.shown || advance {
		a.shown = a.fb.frames
		cmds = append(cmds, a.pic.SetImage(a.fb.RGBA))
	}
	return a, tea.Batch(cmds...)
}

// toggle switches between Kitty graphics and half-blocks, reporting why Kitty
// is unavailable instead of switching.
func (a *app) toggle() tea.Cmd {
	a.status = ""
	if a.pic.Mode() == ntpicture.PictureKitty {
		return a.pic.SetMode(ntpicture.PictureGlyph)
	}
	if err := ntpicture.KittyUnavailable(); err != nil {
		a.status = err.Error()
		return nil
	}
	return a.pic.SetMode(ntpicture.PictureKitty)
}

func (a *app) View() tea.View {
	content := a.pic.View()
	if a.status != "" {
		content += "\n" + a.status
	}
	v := tea.NewView(content)
	v.AltScreen = true
	return v
}

func main() {
	cols := flag.Int("cols", 240, "terminal columns for the display")
	rows := flag.Int("rows", 68, "terminal rows for the display")
	glyph := flag.Bool("glyph", false, "stay in half-block mode instead of switching to Kitty graphics")
	uncapped := flag.Bool("uncapped", false, "advance animation after each picture completion instead of waiting 100 ms")
	stats := flag.Bool("stats", false, "print frame statistics on exit")
	snapshot := flag.String("snapshot", "", "write the final framebuffer to this PNG on exit")
	flag.Parse()

	fb := &display{RGBA: image.NewRGBA(image.Rect(0, 0, width, height))}
	model := teamodel.New()
	var visual tea.Model = model
	if *uncapped {
		visual = &unpacedHello{model}
	}
	inner, err := displaytea.NewModel(visual, fb, hello.NewFont(), hello.Theme())
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer inner.Close()
	// Kitty placements are pre-scaled to cols*cellWidth by rows*cellHeight
	// pixels, about 1920x1088 here. The display is 240x135, so encode at 1/8
	// and let the terminal upscale: 16 times fewer pixels per frame.
	// StableID keeps one image ID across frames, so the placeholder grid does
	// not change and Bubble Tea sends only the new bitmap each frame.
	pic := ntpicture.New(ntpicture.Options{StableID: true, Render: ntpicture.Config{
		Fit: ntpicture.FitFill, KittyFormat: ntpicture.KittyFormatZlib, KittyResolutionFactor: 0.125}})
	defer pic.Close()
	a := &app{inner: inner, keys: model.Keys(), fb: fb, pic: pic, cols: *cols, rows: *rows, wantKitty: !*glyph}
	a.uncapped = *uncapped
	fps := 60
	if *uncapped {
		fps = 120 // Bubble Tea's maximum text-renderer refresh rate.
	}

	p := tea.NewProgram(a, tea.WithContext(inner.Context()), tea.WithInput(stdin()),
		// WASI cannot tell whether stdout is a terminal, and profile detection
		// would then strip every color.
		tea.WithColorProfile(colorprofile.TrueColor),
		tea.WithoutSignalHandler(), tea.WithoutCatchPanics(), tea.WithFPS(fps), tea.WithWindowSize(*cols, *rows+1))
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
			fb.frames, elapsed.Round(time.Millisecond), float64(fb.frames)/elapsed.Seconds(), a.pic.Mode(), ntpicture.KittySupported(), stdinMode)
		fmt.Fprintf(os.Stderr, "pictures=%d picture_fps=%.1f uncapped=%t\r\n",
			a.presented, float64(a.presented)/elapsed.Seconds(), a.uncapped)
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
