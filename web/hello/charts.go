package web

import (
	"fmt"
	"image/color"
	"math"
	"time"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	bubbles "nimbleterminal.dev/tinybubbles"
	displaytea "nimbleterminal.dev/tinybubbles/bubbletea"
	"nimbleterminal.dev/tinybubbles/layout"
	"nimbleterminal.dev/tinybubbles/tinycharts"
	browser "nimbleterminal.dev/tinybubbles/web"
	"tinygo.org/x/drivers"
)

const (
	chartViewAll = iota
	chartViewStream
	chartViewSpark
	chartViewBars
	chartViewLogic
	chartViewCount
)

const (
	chartStyleGreen bubbles.StyleID = 4
	chartStyleAmber bubbles.StyleID = 5
)

type chartsTickMsg struct{ generation uint64 }

type chartsDemo struct {
	buffer     *bubbles.Buffer
	stream     *tinycharts.Streamline
	spark      *tinycharts.Sparkline
	bars       *tinycharts.BarChart
	logic      *tinycharts.LogicChart
	header     layout.Rect
	footer     layout.Rect
	panels     [4]layout.Rect
	viewMode   int
	streamMode tinycharts.StreamMode
	horizBars  bool
	paused     bool
	step       int
	burstTTL   int
	generation uint64
	title      string
	caption    string
}

func newChartsDemo() *chartsDemo {
	m := &chartsDemo{
		buffer: bubbles.NewBuffer(0, 0),
		stream: tinycharts.NewStreamline(
			1, 1,
			tinycharts.WithStreamRange(0, 100),
			tinycharts.WithStreamMode(tinycharts.StreamModeBraille),
			tinycharts.WithStreamStyle(bubbles.Accent),
		),
		spark: tinycharts.NewSparkline(
			1, 1,
			tinycharts.WithSparklineMax(100),
			tinycharts.WithSparklineAutoMax(false),
			tinycharts.WithSparklineStyle(chartStyleGreen),
		),
		bars: tinycharts.NewBarChart(
			1, 1,
			tinycharts.WithBarMax(100),
			tinycharts.WithBarHorizontal(true),
		),
		logic: tinycharts.NewLogicChart(1, 1, []tinycharts.LogicChannel{
			{Label: "CLK", Kind: tinycharts.LogicBit, Style: chartStyleGreen},
			{Label: "PWM", Kind: tinycharts.LogicPWM, Style: bubbles.Accent},
			{Label: "ACT", Kind: tinycharts.LogicBit, Style: chartStyleAmber},
			{Label: "SPI", Kind: tinycharts.LogicBus, Style: bubbles.Normal},
		}),
		horizBars: true,
	}
	m.resetData()
	return m
}

func (m *chartsDemo) tick() tea.Cmd {
	g := m.generation
	return tea.Tick(160*time.Millisecond, func(time.Time) tea.Msg { return chartsTickMsg{g} })
}

func (m *chartsDemo) Init() tea.Cmd  { return m.tick() }
func (m *chartsDemo) View() tea.View { return tea.NewView("") }

func (m *chartsDemo) resetData() {
	m.step = 0
	m.burstTTL = 0
	m.stream.Clear()
	m.spark.Clear()
	m.logic.Clear()
	for i := 0; i < 48; i++ {
		m.advanceSample(false)
	}
	m.reflow()
}

func (m *chartsDemo) advanceSample(burst bool) {
	m.step++
	if burst {
		m.burstTTL = 3
	}
	rad := float64(m.step) * 0.28
	sine := 50 + 38*math.Sin(rad)
	harm := 45 + 30*math.Cos(rad*0.7)
	if m.burstTTL > 0 {
		sine = min(100, sine+25)
		harm = 96
	}
	m.stream.Push(sine)
	m.spark.Push(harm)

	clk := uint16(m.step & 1)
	pwm := uint16(math.Round((sine / 100.0) * 255.0))
	var act, bus uint16
	if m.burstTTL > 0 {
		act = 1
		bus = 1
		m.burstTTL--
	} else if m.step%7 == 0 {
		bus = 1
	}
	m.logic.PushStep(clk, pwm, act, bus)

	m.bars.SetBars([]tinycharts.BarData{
		{Label: "ADC", Value: sine, Style: bubbles.Accent},
		{Label: "PWM", Value: harm, Style: chartStyleGreen},
		{Label: "BUS", Value: float64(25 + (m.step*9)%75), Style: chartStyleAmber},
	})
	m.updateLabels()
}

func (m *chartsDemo) reflow() {
	var rest, body layout.Rect
	m.header, rest = layout.Bounds(m.buffer).Rows(1, 0)
	body, m.footer = rest.Rows(max(0, rest.Height-1), 0)
	m.panels = [4]layout.Rect{}

	switch m.viewMode {
	case chartViewStream:
		m.panels[0] = body
	case chartViewSpark:
		m.panels[1] = body
	case chartViewBars:
		m.panels[2] = body
	case chartViewLogic:
		m.panels[3] = body
	default:
		// Split dashboard: 2x2 grid on wide screens, 2 stacked rows on compact screens.
		topH := max(1, (body.Height-1)/2)
		top, bottom := body.Rows(topH, 1)
		if body.Width >= 32 && body.Height >= 10 {
			leftW := max(1, (body.Width-1)/2)
			m.panels[0], m.panels[1] = top.Columns(leftW, 1)
			m.panels[2], m.panels[3] = bottom.Columns(leftW, 1)
		} else {
			m.panels[0] = top
			m.panels[3] = bottom
		}
	}

	if r := m.panels[0]; r.Width > 0 && r.Height > 0 {
		m.stream.Resize(r.Width, r.Height)
	}
	if r := m.panels[1]; r.Width > 0 && r.Height > 0 {
		m.spark.Resize(r.Width, r.Height)
	}
	if r := m.panels[2]; r.Width > 0 && r.Height > 0 {
		m.bars.Resize(r.Width, r.Height)
	}
	if r := m.panels[3]; r.Width > 0 && r.Height > 0 {
		m.logic.Resize(r.Width, r.Height)
	}
	m.updateLabels()
}

func (m *chartsDemo) updateLabels() {
	viewNames := [chartViewCount]string{"ALL", "STREAM", "SPARK", "BARS", "LOGIC"}
	modeNames := [3]string{"BRAILLE", "ARC", "BOX"}
	barDir := "H"
	if !m.horizBars {
		barDir = "V"
	}
	state := "LIVE"
	if m.paused {
		state = "HOLD"
	}
	m.title = fmt.Sprintf("CHARTS %s · %s", viewNames[m.viewMode], modeNames[m.streamMode])
	m.caption = fmt.Sprintf("%s BAR:%s STEP:%d", state, barDir, m.step)
}

func (m *chartsDemo) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		w, h := m.buffer.Size()
		if w != msg.Width || h != msg.Height {
			m.buffer.Resize(msg.Width, msg.Height)
		}
		m.reflow()
		// Reseed if resizing expanded the chart history capacity.
		if m.step < msg.Width*2 {
			for i := 0; i < msg.Width; i++ {
				m.advanceSample(false)
			}
		}
		return m, nil

	case chartsTickMsg:
		if msg.generation != m.generation || m.paused {
			return m, nil
		}
		m.advanceSample(false)
		return m, m.tick()

	case tea.KeyPressMsg:
		k := m.keys()
		switch {
		case key.Matches(msg, k.view):
			m.viewMode = (m.viewMode + 1) % chartViewCount
			m.reflow()
			return m, nil
		case key.Matches(msg, k.mode):
			m.streamMode = (m.streamMode + 1) % 3
			m.stream.SetMode(m.streamMode)
			m.horizBars = !m.horizBars
			m.bars.Horizontal = m.horizBars
			m.updateLabels()
			return m, nil
		case key.Matches(msg, k.burst):
			m.advanceSample(true)
			return m, nil
		case key.Matches(msg, k.pause):
			m.paused = !m.paused
			m.generation++
			m.updateLabels()
			if !m.paused {
				return m, m.tick()
			}
			return m, nil
		case key.Matches(msg, k.reset):
			m.viewMode = chartViewAll
			m.streamMode = tinycharts.StreamModeBraille
			m.stream.SetMode(m.streamMode)
			m.horizBars = true
			m.bars.Horizontal = true
			m.paused = false
			m.generation++
			m.resetData()
			return m, m.tick()
		case key.Matches(msg, k.quit):
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m *chartsDemo) CellView() *bubbles.Buffer {
	b := m.buffer
	b.Clear(bubbles.Normal)
	layout.ClipText(b, m.header, m.title, bubbles.Accent, layout.Start)
	if r := m.panels[0]; r.Width > 0 && r.Height > 0 {
		m.stream.Draw(b, r)
	}
	if r := m.panels[1]; r.Width > 0 && r.Height > 0 {
		m.spark.Draw(b, r)
	}
	if r := m.panels[2]; r.Width > 0 && r.Height > 0 {
		m.bars.Draw(b, r)
	}
	if r := m.panels[3]; r.Width > 0 && r.Height > 0 {
		m.logic.Draw(b, r)
	}
	layout.ClipText(b, m.footer, m.caption, bubbles.Muted, layout.Start)
	return b
}

type chartsDemoKeys struct {
	view, mode, burst, pause, reset, quit key.Binding
}

func (k chartsDemoKeys) ShortHelp() []key.Binding {
	return []key.Binding{k.view, k.mode, k.burst}
}

func (k chartsDemoKeys) FullHelp() [][]key.Binding {
	return [][]key.Binding{k.ShortHelp(), {k.pause, k.reset, k.quit}}
}

func (m *chartsDemo) keys() chartsDemoKeys {
	pauseLabel := "Pause"
	if m.paused {
		pauseLabel = "Resume"
	}
	return chartsDemoKeys{
		view:  key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "Next chart")),
		mode:  key.NewBinding(key.WithKeys("m"), key.WithHelp("m", "Line/Bar style")),
		burst: key.NewBinding(key.WithKeys("b"), key.WithHelp("b", "Spike/Burst")),
		pause: key.NewBinding(key.WithKeys("p"), key.WithHelp("p", pauseLabel)),
		reset: key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "Reset")),
		quit:  key.NewBinding(key.WithKeys("q"), key.WithHelp("q", "Quit")),
	}
}

// NewChartsDemo constructs the tinycharts gallery demo wrapped with browser Terminal Actions.
func NewChartsDemo(display drivers.Displayer, mono bool) (tea.Model, *displaytea.DisplayModel, error) {
	theme := bubbles.DefaultTheme(mono)
	if !mono {
		theme.Styles[chartStyleGreen] = bubbles.Style{
			Foreground: color.RGBA{R: 80, G: 235, B: 130, A: 255},
		}
		theme.Styles[chartStyleAmber] = bubbles.Style{
			Foreground: color.RGBA{R: 255, G: 195, B: 65, A: 255},
		}
	}
	pixels, err := displaytea.NewModel(newChartsDemo(), display, bubbles.DefaultFont(), theme)
	if err != nil {
		return nil, nil, err
	}
	wrapped, err := browser.WrapActions(pixels, "charts", func(model tea.Model) help.KeyMap {
		return model.(*displaytea.DisplayModel).Model().(*chartsDemo).keys()
	}, nil)
	if err != nil {
		pixels.Close()
		return nil, nil, err
	}
	return wrapped, pixels, nil
}
