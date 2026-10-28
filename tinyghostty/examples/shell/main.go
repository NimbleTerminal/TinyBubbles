//go:build unix

// Run your shell inside an emulated TinyBubbles display, drawn in this
// terminal with half-block pixels. Your shell runs in a PTY; its output
// feeds an embedded tinyghostty terminal, and keys travel through Bubble Tea
// and the widget's key encoder back to the PTY. Press ctrl+] to quit.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"image"
	"image/color"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/term"
	"github.com/creack/pty"
	bubbles "nimbleterminal.dev/tinybubbles"
	displaytea "nimbleterminal.dev/tinybubbles/bubbletea"
	"nimbleterminal.dev/tinybubbles/layout"
	"nimbleterminal.dev/tinybubbles/tinyghostty"
	"nimbleterminal.dev/tinybubbles/tinyghostty/render"
)

// Display presets in pixels. The default 6×8 font sets the cell grid.
var displays = map[string]image.Point{
	"feather": {240, 135}, // Adafruit ESP32-S3 Reverse TFT Feather: 40×16 cells
	"oled":    {128, 64},  // SSD1306 OLED: 21×8 cells
	"lcd":     {320, 240}, // 320×240 TFT: 53×30 cells
}

// screen is a Displayer that paints changed pixel pairs as "▀" cells: the
// foreground is the upper pixel and the background the lower one.
type screen struct {
	*image.RGBA
	out      *bufio.Writer
	top, bot []color.RGBA // last painted pixels for each terminal cell
	drew     bool
}

func newScreen(size image.Point, out io.Writer) *screen {
	return &screen{
		RGBA: image.NewRGBA(image.Rectangle{Max: size}),
		out:  bufio.NewWriterSize(out, 64<<10),
		top:  make([]color.RGBA, size.X*(size.Y+1)/2),
		bot:  make([]color.RGBA, size.X*(size.Y+1)/2),
	}
}

func (s *screen) Size() (int16, int16) {
	b := s.Bounds()
	return int16(b.Dx()), int16(b.Dy())
}

func (s *screen) SetPixel(x, y int16, c color.RGBA) { s.SetRGBA(int(x), int(y), c) }

func (s *screen) Display() error {
	w, h := s.Bounds().Dx(), s.Bounds().Dy()
	var fg, bg color.RGBA
	styled := false
	for cy := 0; cy < (h+1)/2; cy++ {
		next := -1 // column the terminal cursor sits at, or -1
		for x := 0; x < w; x++ {
			top, bot := s.RGBAAt(x, 2*cy), s.RGBAAt(x, 2*cy+1)
			i := cy*w + x
			if s.drew && s.top[i] == top && s.bot[i] == bot {
				next = -1
				continue
			}
			s.top[i], s.bot[i] = top, bot
			if next != x {
				fmt.Fprintf(s.out, "\x1b[%d;%dH", cy+1, x+1)
			}
			if !styled || top != fg {
				s.sgr(38, top)
			}
			if !styled || bot != bg {
				s.sgr(48, bot)
			}
			fg, bg, styled = top, bot, true
			s.out.WriteString("▀")
			next = x + 1
		}
	}
	s.drew = true
	s.out.WriteString("\x1b[0m")
	return s.out.Flush()
}

func (s *screen) sgr(kind int, c color.RGBA) {
	b := s.out.AvailableBuffer()
	b = append(b, "\x1b["...)
	b = strconv.AppendInt(b, int64(kind), 10)
	b = append(b, ";2;"...)
	b = strconv.AppendInt(b, int64(c.R), 10)
	b = append(b, ';')
	b = strconv.AppendInt(b, int64(c.G), 10)
	b = append(b, ';')
	b = strconv.AppendInt(b, int64(c.B), 10)
	s.out.Write(append(b, 'm'))
}

type exitMsg struct{}

// shell composes a title row and the terminal, and owns sizing and focus.
type shell struct {
	title string
	term  *tinyghostty.Model
	buf   *bubbles.Buffer
	body  layout.Rect
}

func (s *shell) Init() tea.Cmd { return nil }

func (s *shell) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		s.buf.Resize(msg.Width, msg.Height)
		_, s.body = layout.Bounds(s.buf).Rows(1, 0)
		s.term.SetSize(s.body.Width, s.body.Height)
		return s, nil
	case tea.KeyPressMsg:
		if msg.String() == "ctrl+]" {
			return s, tea.Quit
		}
	case exitMsg:
		return s, tea.Quit
	}
	s.term.Update(msg)
	return s, nil
}

func (s *shell) View() tea.View { return tea.NewView("") }

func (s *shell) CellView() *bubbles.Buffer {
	s.buf.Clear(bubbles.Normal)
	title, _ := layout.Bounds(s.buf).Rows(1, 0)
	layout.ClipText(s.buf, title, s.title, bubbles.Selected, layout.Start)
	layout.Draw(s.buf, s.body, s.term.CellView())
	return s.buf
}

func main() {
	preset := flag.String("display", "feather", "display preset: feather, oled, or lcd")
	mono := flag.Bool("mono", false, "monochrome theme")
	flag.Parse()
	size, ok := displays[*preset]
	if !ok {
		fail(fmt.Errorf("unknown display %q", *preset))
	}
	if !term.IsTerminal(os.Stdin.Fd()) || !term.IsTerminal(os.Stdout.Fd()) {
		fail(fmt.Errorf("run this in an interactive terminal"))
	}
	if w, h, err := term.GetSize(os.Stdout.Fd()); err == nil && w > 0 && h > 0 && (w < size.X || h < (size.Y+1)/2+1) {
		hint := "shrink the font"
		if *preset != "oled" {
			hint += " or try -display oled"
		}
		fail(fmt.Errorf("the %s display needs a %d×%d terminal; this one is %d×%d: %s",
			*preset, size.X, (size.Y+1)/2+1, w, h, hint))
	}

	argv := flag.Args()
	if len(argv) == 0 {
		argv = []string{os.Getenv("SHELL")}
		if argv[0] == "" {
			argv[0] = "/bin/sh"
		}
	}

	theme := render.Theme(*mono)
	vt := tinyghostty.New(1, 1)
	if err := vt.Err(); err != nil {
		fail(err)
	}
	defer vt.Close()
	vt.Renderer().SetTheme(theme)
	sh := &shell{title: " tinyghostty - " + filepath.Base(argv[0]), term: vt, buf: bubbles.NewBuffer(0, 0)}

	scr := newScreen(size, os.Stdout)
	model, err := displaytea.NewModel(sh, scr, nil, theme)
	if err != nil {
		fail(err)
	}
	defer model.Close()
	cols, rows := model.Size()

	cmd := exec.Command(argv[0], argv[1:]...)
	cmd.Env = append(os.Environ(), "TERM=xterm-256color")
	ptmx, err := pty.StartWithSize(cmd, &pty.Winsize{Cols: uint16(cols), Rows: uint16(rows - 1)})
	if err != nil {
		fail(err)
	}
	defer ptmx.Close()
	vt.Input = ptmx

	// Bubble Tea decodes keys from stdin; frames go to stdout through screen.
	// With its output discarded it leaves the terminal cooked, so set raw mode.
	saved, err := term.MakeRaw(os.Stdin.Fd())
	if err != nil {
		fail(err)
	}
	p := tea.NewProgram(model, tea.WithContext(model.Context()), tea.WithInput(os.Stdin),
		tea.WithOutput(io.Discard), tea.WithoutRenderer(), tea.WithoutSignalHandler(),
		tea.WithEnvironment([]string{}), tea.WithWindowSize(cols, rows))
	go func() {
		io.Copy(tinyghostty.Sender(p.Send), ptmx)
		p.Send(exitMsg{})
	}()

	// Alternate screen, hidden cursor, bracketed paste; a status line below.
	fmt.Printf("\x1b[?1049h\x1b[?25l\x1b[?2004h\x1b[2J\x1b[%d;1H\x1b[2m%s %d×%d px, %d×%d cells · ctrl+] quits\x1b[0m",
		(size.Y+1)/2+1, *preset, size.X, size.Y, cols, rows-1)
	_, runErr := p.Run()
	fmt.Print("\x1b[?2004l\x1b[?25h\x1b[?1049l")
	term.Restore(os.Stdin.Fd(), saved)

	cmd.Process.Kill()
	cmd.Wait()
	if runErr == nil {
		runErr = model.Err()
	}
	if runErr == nil {
		runErr = vt.Err()
	}
	if runErr != nil {
		fail(runErr)
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "shell:", err)
	os.Exit(1)
}
