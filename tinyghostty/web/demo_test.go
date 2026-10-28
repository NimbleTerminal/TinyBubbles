package demo

import (
	"image"
	"image/color"
	"strings"
	"testing"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"nimbleterminal.dev/tinybubbles/tinyghostty/render"
)

func started(t *testing.T, cols, rows int) *Model {
	t.Helper()
	m := New(render.Theme(false))
	if m.Err() != nil {
		t.Fatal(m.Err())
	}
	t.Cleanup(m.Close)
	m.Update(tea.WindowSizeMsg{Width: cols, Height: rows})
	return m
}

func typeText(m *Model, text string) {
	for _, r := range text {
		k := tea.KeyPressMsg{Code: r, Text: string(r)}
		if r == '\r' {
			k = tea.KeyPressMsg{Code: tea.KeyEnter}
		}
		m.Update(k)
	}
}

func TestBannerWaitsForSize(t *testing.T) {
	m := New(render.Theme(false))
	defer m.Close()
	if strings.TrimSpace(m.Text()) != "" {
		t.Fatal("banner before size")
	}
	m.Update(tea.WindowSizeMsg{Width: 40, Height: 16})
	if !strings.Contains(m.Text(), "tinyghostty") || !strings.Contains(m.Text(), "tiny:~$") {
		t.Fatalf("banner %q", m.Text())
	}
}

func TestTypingRunsCommands(t *testing.T) {
	m := started(t, 40, 16)
	typeText(m, "echo hi there\r")
	if !strings.Contains(m.Text(), "\nhi there\n") {
		t.Fatalf("echo %q", m.Text())
	}
	typeText(m, "nope\r")
	if !strings.Contains(m.Text(), "sh: nope: command not found") {
		t.Fatalf("unknown %q", m.Text())
	}
	// Backspace edits the line; Ctrl+L clears; Up recalls history.
	typeText(m, "echo ab")
	m.Update(tea.KeyPressMsg{Code: tea.KeyBackspace})
	typeText(m, "c\r")
	if !strings.Contains(m.Text(), "\nac\n") {
		t.Fatalf("backspace %q", m.Text())
	}
	m.Update(tea.KeyPressMsg{Code: 'l', Mod: tea.ModCtrl})
	m.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	if lines := strings.Split(m.Text(), "\n"); lines[0] != "tiny:~$ echo ac" {
		t.Fatalf("clear and recall %q", lines[0])
	}
}

func TestTabCompletes(t *testing.T) {
	m := started(t, 40, 16)
	typeText(m, "col")
	m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	if !strings.Contains(m.Text(), "tiny:~$ colors") {
		t.Fatalf("completion %q", m.Text())
	}
}

func TestButtonsTypeCommands(t *testing.T) {
	m := started(t, 40, 16)
	m.Update(tea.KeyPressMsg{Code: tea.KeyF1})
	if !strings.Contains(m.Text(), "tiny:~$ help") || !strings.Contains(m.Text(), "colors") {
		t.Fatalf("help button %q", m.Text())
	}
	for _, b := range m.Keys.ShortHelp() {
		if !b.Enabled() || len(b.Keys()) != 1 {
			t.Fatalf("binding %v", b.Help())
		}
	}
	if !key.Matches(tea.KeyPressMsg{Code: tea.KeyF4}, m.Keys.Clear) {
		t.Fatal("clear binding")
	}
}

func TestEveryCommandRuns(t *testing.T) {
	for _, name := range names() {
		for _, size := range [][2]int{{21, 7}, {40, 15}, {53, 29}} {
			m := started(t, size[0], size[1])
			typeText(m, name+" README.md\r")
			if strings.Contains(m.Text(), "command not found") || m.Err() != nil {
				t.Fatalf("%s at %v: %q %v", name, size, m.Text(), m.Err())
			}
		}
	}
}

func TestSplitInputWaits(t *testing.T) {
	var out strings.Builder
	s := NewShell(&out, func() (int, int) { return 40, 10 })
	s.Write([]byte("\x1b["))
	s.Process()
	s.Write([]byte("A\xe7"))
	s.Process()
	s.Write([]byte("\x95\x8c"))
	s.Process()
	if out.String() != "界" {
		t.Fatalf("output %q", out.String())
	}
}

type display struct{ *image.RGBA }

func (d display) Size() (int16, int16)              { return 240, 135 }
func (d display) SetPixel(x, y int16, c color.RGBA) { d.SetRGBA(int(x), int(y), c) }
func (d display) Display() error                    { return nil }

func TestNewDemoDraws(t *testing.T) {
	d := display{image.NewRGBA(image.Rect(0, 0, 240, 135))}
	model, pixels, err := NewDemo(d, false)
	if err != nil {
		t.Fatal(err)
	}
	defer pixels.Close()
	cols, rows := pixels.Size()
	model.Update(tea.WindowSizeMsg{Width: cols, Height: rows})
	model.View()
	if pixels.Err() != nil {
		t.Fatal(pixels.Err())
	}
	lit := 0
	for i := 0; i < len(d.Pix); i += 4 {
		if d.Pix[i] > 0 || d.Pix[i+1] > 0 {
			lit++
		}
	}
	if lit == 0 {
		t.Fatal("nothing drawn")
	}
}
