// Command terminal runs the Bubble Tea hello model with the 240x135 Feather
// display drawn inside a terminal. ntpicture shows the framebuffer as Kitty
// graphics where the terminal supports them and as half-block cells
// elsewhere. The key bindings are published as Booba Terminal Actions; in a
// terminal without that support the program hosts them itself as a row of
// clickable buttons. It builds natively and for WASI preview 1 with Go or
// TinyGo; see README.md for the terminal setup each build needs.
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
	"github.com/NimbleMarkets/go-booba/actions"
	"github.com/NimbleMarkets/go-booba/actions/bubblehelp"
	"github.com/NimbleMarkets/go-booba/actions/termhost"
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

// kittyKey toggles between Kitty graphics and half-blocks. Its help label and
// enabled state follow the picture; see helpKeys.
var kittyKey = key.NewBinding(key.WithKeys("k"), key.WithHelp("k", "Kitty graphics"))

// keyMap is the help map behind the keyboard, Booba's action buttons, and the
// hosted bar: the hello bindings plus the picture toggle, with quit in full help.
type keyMap struct {
	hello teamodel.KeyMap
	kitty key.Binding
}

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.hello.Pop, k.hello.Color, k.hello.Pause, k.kitty}
}
func (k keyMap) FullHelp() [][]key.Binding { return [][]key.Binding{k.ShortHelp(), {k.hello.Quit}} }

// app owns the display model and the picture that shows its framebuffer.
type app struct {
	inner      *displaytea.DisplayModel
	model      *teamodel.Model // nil in tests that only need key matching
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
		case key.Matches(v, kittyKey):
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
	if a.wantKitty && a.pic.KittySupported() == ntpicture.KittyCapabilitySupported {
		a.wantKitty = false
		cmds = append(cmds, a.pic.SetMode(ntpicture.PictureKitty))
	}
	// Picture notifications never carry a new frame. Internal picture messages
	// also leave the framebuffer unchanged and are filtered by the frame count below.
	switch msg.(type) {
	case ntpicture.PresentedMsg, ntpicture.ErrorMsg, ntpicture.KittyCapabilityMsg, ntpicture.TerminalSessionMsg:
		if !advance {
			return a, tea.Batch(cmds...)
		}
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

// helpKeys is the live help map for the action publisher: the hello model
// relabels Pause itself, and the toggle reflects the picture's mode.
func (a *app) helpKeys() keyMap {
	keys := keyMap{hello: a.keys, kitty: kittyKey}
	if a.model != nil {
		keys.hello = a.model.Keys()
	}
	if a.pic.Mode() == ntpicture.PictureKitty {
		keys.kitty.SetHelp("k", "Half-blocks")
	} else {
		keys.kitty.SetEnabled(a.pic.KittySupported() == ntpicture.KittyCapabilitySupported)
	}
	return keys
}

// toggle switches between Kitty graphics and half-blocks, reporting why Kitty
// is unavailable instead of switching.
func (a *app) toggle() tea.Cmd {
	a.status = ""
	if a.pic.Mode() == ntpicture.PictureKitty {
		return a.pic.SetMode(ntpicture.PictureGlyph)
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
	a := &app{inner: inner, model: model, keys: model.Keys(), fb: fb, pic: pic, cols: *cols, rows: *rows, wantKitty: !*glyph}
	a.uncapped = *uncapped
	// Publish the help map as Terminal Actions. Booba's browser terminal turns
	// it into buttons; elsewhere termhost answers the capability query itself
	// and draws the same buttons on the row under the picture.
	published, err := bubblehelp.NewModel(a, func(tea.Model) bubblehelp.Description {
		return bubblehelp.Description{Context: "hello", Keys: a.helpKeys(),
			Status: actions.ProgramStatus{App: "tinybubbles", Summary: "TinyBubbles hello"}}
	}, bubblehelp.ModelOptions{Instance: "tinybubbles-hello", Adapter: bubblehelp.Options{IncludeDisabled: true},
		OnError: func(err error) { fmt.Fprintln(os.Stderr, "actions:", err) }})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	host, output := termhost.New(published, os.Stdout, termhost.Options{})
	fps := 60
	if *uncapped {
		fps = 120 // Bubble Tea's maximum text-renderer refresh rate.
	}

	p := tea.NewProgram(host, tea.WithContext(inner.Context()), tea.WithInput(stdin()), tea.WithOutput(output),
		// WASI cannot tell whether stdout is a terminal, and profile detection
		// would then strip every color.
		tea.WithColorProfile(colorprofile.TrueColor),
		// One row under the picture for the status line, one for the action bar.
		tea.WithoutSignalHandler(), tea.WithoutCatchPanics(), tea.WithFPS(fps), tea.WithWindowSize(*cols, *rows+2))
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
		fmt.Fprintf(os.Stderr, "frames=%d elapsed=%s fps=%.1f mode=%d kitty=%d actions=%d stdin=%s\r\n",
			fb.frames, elapsed.Round(time.Millisecond), float64(fb.frames)/elapsed.Seconds(), a.pic.Mode(), a.pic.KittySupported(), host.Mode(), stdinMode)
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
