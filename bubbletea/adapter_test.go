package displaytea

import (
	"bytes"
	"errors"
	"image/color"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	bubbles "nimbleterminal.dev/tinybubbles"
	"tinygo.org/x/drivers"
)

type rawModel struct{ testModel }

func (rawModel) Init() tea.Cmd { return tea.Sequence(tea.Raw("control metadata"), tea.Quit) }

func TestHostedProgramKeepsRawOutputAndPhysicalSize(t *testing.T) {
	d := &testDisplay{}
	m, err := NewModel(rawModel{}, d, nil, bubbles.DefaultTheme(false))
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	m.Update(tea.WindowSizeMsg{Width: 999, Height: 999})
	if m.Model().(testModel).width != 20 {
		t.Fatal("host resize changed the display grid")
	}
	// Restore the model whose Init publishes a control message.
	m.model = rawModel{}
	var out bytes.Buffer
	p := tea.NewProgram(m, tea.WithContext(m.Context()), tea.WithInput(nil), tea.WithOutput(&out),
		tea.WithoutRenderer(), tea.WithoutSignalHandler(), tea.WithoutCatchPanics(),
		tea.WithEnvironment([]string{}), tea.WithWindowSize(20, 8))
	timer := time.AfterFunc(3*time.Second, p.Kill)
	defer timer.Stop()
	if _, err := p.Run(); err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(out.Bytes(), []byte("control metadata")) {
		t.Fatalf("Raw output lost: %q", out.String())
	}
	if d.flushes == 0 {
		t.Fatal("hosted model did not render")
	}
}

type testDisplay struct {
	flushes int
	pixels  int
	err     error
}

func (*testDisplay) Size() (int16, int16)                { return 120, 64 }
func (d *testDisplay) SetPixel(int16, int16, color.RGBA) { d.pixels++ }
func (d *testDisplay) Display() error                    { d.flushes++; return d.err }

type countMsg struct{}
type testModel struct {
	count int
	width int
}

func (testModel) Init() tea.Cmd {
	return tea.Batch(func() tea.Msg { return countMsg{} }, tea.Tick(time.Millisecond, func(time.Time) tea.Msg { return countMsg{} }))
}
func (m testModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
	case countMsg:
		m.count++
	}
	if m.count == 2 && m.width > 0 {
		return m, tea.Quit
	}
	return m, nil
}
func (m testModel) View() tea.View { return tea.NewView("Real Bubble Tea") }
func run(t *testing.T, p *Program) (tea.Model, error) {
	t.Helper()
	timer := time.AfterFunc(3*time.Second, p.Kill)
	defer timer.Stop()
	return p.Run()
}
func TestRealProgramCommandsAndDisplay(t *testing.T) {
	d := &testDisplay{}
	p, err := NewProgram(testModel{}, d, nil, bubbles.DefaultTheme(false))
	if err != nil {
		t.Fatal(err)
	}
	final, err := run(t, p)
	if err != nil {
		t.Fatal(err)
	}
	m := final.(testModel)
	if m.count != 2 || m.width != 20 {
		t.Fatalf("final model: %+v", m)
	}
	if d.flushes == 0 || d.pixels == 0 {
		t.Fatal("display was not rendered")
	}
}
func TestDisplayErrorStopsProgram(t *testing.T) {
	want := errors.New("display disconnected")
	p, err := NewProgram(testModel{}, &testDisplay{err: want}, nil, bubbles.DefaultTheme(false))
	if err != nil {
		t.Fatal(err)
	}
	_, err = run(t, p)
	var displayErr *bubbles.DisplayError
	if !errors.As(err, &displayErr) || !errors.Is(err, want) {
		t.Fatalf("got %v, want %v", err, want)
	}
}

type cellModel struct {
	testModel
	cells *bubbles.Buffer
}

func (m *cellModel) Init() tea.Cmd                       { return tea.Quit }
func (m *cellModel) Update(tea.Msg) (tea.Model, tea.Cmd) { return m, nil }
func (m *cellModel) CellView() *bubbles.Buffer           { return m.cells }

type recordingFont struct {
	glyph      rune
	foreground color.RGBA
}

func (*recordingFont) Size() (int16, int16) { return 6, 8 }
func (f *recordingFont) DrawGlyph(_ drivers.Displayer, _, _ int16, r rune, c color.RGBA) {
	f.glyph = r
	f.foreground = c
}
func TestCellViewPreservesTilesAndStyles(t *testing.T) {
	cells := bubbles.NewBuffer(20, 8)
	cells.Set(0, 0, bubbles.Cell{Rune: 'Z', Style: bubbles.Accent})
	font := &recordingFont{}
	theme := bubbles.DefaultTheme(false)
	p, err := NewProgram(&cellModel{cells: cells}, &testDisplay{}, font, theme)
	if err != nil {
		t.Fatal(err)
	}
	_, err = run(t, p)
	if err != nil {
		t.Fatal(err)
	}
	if font.glyph != 'Z' || font.foreground != theme.Styles[bubbles.Accent].Foreground {
		t.Fatalf("custom glyph/style lost: %+v", font)
	}
}

type keyModel struct{ presses int }

func (keyModel) Init() tea.Cmd { return nil }
func (m keyModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyPressMsg); ok {
		if key.Code == 'q' {
			return m, tea.Quit
		}
		if key.Code == tea.KeyRight {
			m.presses++
		}
	}
	return m, nil
}
func (keyModel) View() tea.View { return tea.NewView("Buttons") }
func TestPeripheralMessages(t *testing.T) {
	p, err := NewProgram(keyModel{}, &testDisplay{}, nil, bubbles.DefaultTheme(false))
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		p.Send(tea.KeyPressMsg{Code: tea.KeyRight})
		p.Send(tea.KeyPressMsg{Code: 'q'})
	}()
	final, err := run(t, p)
	if err != nil {
		t.Fatal(err)
	}
	<-done
	if final.(keyModel).presses != 1 {
		t.Fatalf("lost button message: %+v", final)
	}
}
