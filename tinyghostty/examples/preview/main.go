// Run a TinyBubbles Bubble Tea program with an embedded Ghostty terminal and
// save the final frame as a PNG. The parent model draws a title row and lays
// the terminal out below it; scripted VT output arrives as OutputMsg values,
// as a PTY pump using tinyghostty.Sender would deliver them.
package main

import (
	"flag"
	"image"
	"image/color"
	"image/png"
	"os"

	tea "charm.land/bubbletea/v2"
	bubbles "nimbleterminal.dev/tinybubbles"
	displaytea "nimbleterminal.dev/tinybubbles/bubbletea"
	"nimbleterminal.dev/tinybubbles/layout"
	"nimbleterminal.dev/tinybubbles/tinyghostty"
	"nimbleterminal.dev/tinybubbles/tinyghostty/render"
)

// The Feather Reverse TFT panel, enlarged for the preview.
const width, height, scale = 240, 135, 2

type display struct{ *image.RGBA }

func (d display) Size() (int16, int16)              { return width, height }
func (d display) SetPixel(x, y int16, c color.RGBA) { d.SetRGBA(int(x), int(y), c) }
func (d display) Display() error                    { return nil }

// script is what a small shell session might print.
var script = []string{
	"\x1b[1;32mtiny\x1b[0m@\x1b[34mbubbles\x1b[0m:~$ ls\r\n",
	"\x1b[34mdemo\x1b[0m  \x1b[34mlayout\x1b[0m  \x1b[32mrun.sh\x1b[0m  README.md\r\n",
	"tiny@bubbles:~$ status\r\n",
	"  vt      \x1b[32mok\x1b[0m\r\n  render  \x1b[32mok\x1b[0m\r\n  heap    \x1b[33m64 KiB\x1b[0m\r\n",
	"  \x1b[7m reverse \x1b[0m \x1b[2mfaint\x1b[0m \x1b[1mbold\x1b[0m \x1b[31merror\x1b[0m\r\n",
	"tiny@bubbles:~$ ",
}

type doneMsg struct{}

// shell composes a title row and the terminal. It owns sizing and focus.
type shell struct {
	term *tinyghostty.Model
	buf  *bubbles.Buffer
	body layout.Rect
}

func (s *shell) Init() tea.Cmd {
	cmds := make([]tea.Cmd, 0, len(script)+1)
	for _, chunk := range script {
		cmds = append(cmds, func() tea.Msg { return tinyghostty.OutputMsg(chunk) })
	}
	return tea.Sequence(append(cmds, func() tea.Msg { return doneMsg{} })...)
}

func (s *shell) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		s.buf.Resize(msg.Width, msg.Height)
		_, s.body = layout.Bounds(s.buf).Rows(1, 0)
		s.term.SetSize(s.body.Width, s.body.Height)
		return s, nil
	case doneMsg:
		return s, tea.Quit
	}
	// The terminal has focus, so keys and paste go to it as well.
	s.term.Update(msg)
	return s, nil
}

func (s *shell) View() tea.View { return tea.NewView("") }

func (s *shell) CellView() *bubbles.Buffer {
	s.buf.Clear(bubbles.Normal)
	title, _ := layout.Bounds(s.buf).Rows(1, 0)
	layout.ClipText(s.buf, title, " tinyghostty", bubbles.Selected, layout.Start)
	layout.Draw(s.buf, s.body, s.term.CellView())
	return s.buf
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	out := flag.String("out", "tinyghostty.png", "output PNG")
	flag.Parse()
	term := tinyghostty.New(40, 16)
	must(term.Err())
	defer term.Close()

	d := display{image.NewRGBA(image.Rect(0, 0, width, height))}
	p, err := displaytea.NewProgram(&shell{term: term, buf: bubbles.NewBuffer(0, 0)}, d, nil, render.Theme(false))
	must(err)
	_, err = p.Run()
	must(err)
	must(term.Err())

	large := image.NewRGBA(image.Rect(0, 0, width*scale, height*scale))
	for y := 0; y < height*scale; y++ {
		for x := 0; x < width*scale; x++ {
			large.SetRGBA(x, y, d.RGBAAt(x/scale, y/scale))
		}
	}
	f, err := os.Create(*out)
	must(err)
	must(png.Encode(f, large))
	must(f.Close())
}
