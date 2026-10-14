package web

import (
	"time"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"nimbleterminal.dev/tinybubbles/tinyhal"
	"nimbleterminal.dev/tinybubbles/tinyhal/workbench"
	browser "nimbleterminal.dev/tinybubbles/web"
	"tinygo.org/x/drivers"
)

type tinyhalTick struct{}

type tinyhalDemo struct {
	wb *workbench.Workbench
}

func newTinyHALDemo(display drivers.Displayer, leds tinyhal.LEDWriter, monochrome bool) *tinyhalDemo {
	w, h := display.Size()
	wb := workbench.New(workbench.Config{
		Width:      w,
		Height:     h,
		Monochrome: monochrome,
		Target:     display,
		LEDs:       leds,
	})
	return &tinyhalDemo{wb: wb}
}

func (m *tinyhalDemo) tick() tea.Cmd {
	return tea.Tick(150*time.Millisecond, func(time.Time) tea.Msg { return tinyhalTick{} })
}

func (m *tinyhalDemo) Init() tea.Cmd {
	return m.tick()
}

func (m *tinyhalDemo) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tinyhalTick:
		m.wb.Tick()
		return m, m.tick()
	case tea.KeyPressMsg:
		keys := m.keys()
		switch {
		case key.Matches(msg, keys.quit):
			return m, tea.Quit
		case key.Matches(msg, keys.led):
			m.wb.ToggleLED()
		case key.Matches(msg, keys.burst):
			m.wb.TriggerBurst()
		case key.Matches(msg, keys.mode):
			m.wb.PressMode()
		case key.Matches(msg, keys.view):
			m.wb.ToggleView()
		case key.Matches(msg, keys.vcd):
			exportVCD("tinyhal-trace.vcd", m.wb.VCD())
		case key.Matches(msg, keys.adcUp):
			m.wb.AdjustADC(256)
		case key.Matches(msg, keys.adcDown):
			m.wb.AdjustADC(-256)
		case key.Matches(msg, keys.tempUp):
			m.wb.AdjustTemp(1)
		case key.Matches(msg, keys.tempDown):
			m.wb.AdjustTemp(-1)
		case key.Matches(msg, keys.reset):
			m.wb.Reset()
		}
		return m, nil
	}
	return m, nil
}

func (m *tinyhalDemo) View() tea.View {
	return tea.NewView("")
}

type tinyhalKeys struct {
	led      key.Binding
	burst    key.Binding
	mode     key.Binding
	view     key.Binding
	vcd      key.Binding
	adcDown  key.Binding
	adcUp    key.Binding
	tempDown key.Binding
	tempUp   key.Binding
	reset    key.Binding
	quit     key.Binding
}

func (k tinyhalKeys) ShortHelp() []key.Binding {
	return []key.Binding{k.led, k.burst, k.mode, k.view, k.vcd, k.adcDown, k.adcUp, k.tempDown, k.tempUp, k.reset}
}

func (k tinyhalKeys) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.led, k.burst, k.mode, k.view, k.vcd},
		{k.adcDown, k.adcUp, k.tempDown, k.tempUp, k.reset, k.quit},
	}
}

func (m *tinyhalDemo) keys() tinyhalKeys {
	ledLabel := "D1: LED on"
	if m.wb.LEDStatusOn() {
		ledLabel = "D1: LED off"
	}
	modeLabel := "D0: Sine wave"
	switch m.wb.Mode() {
	case workbench.ModeSine:
		modeLabel = "D0: Saw wave"
	case workbench.ModeSaw:
		modeLabel = "D0: Manual ADC"
	}
	viewLabel := "View: Logic"
	if m.wb.ViewMode() == workbench.ViewLogic {
		viewLabel = "View: Scope"
	}
	k := tinyhalKeys{
		led:      key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", ledLabel)),
		burst:    key.NewBinding(key.WithKeys("space"), key.WithHelp("space", "D2: Bus burst")),
		mode:     key.NewBinding(key.WithKeys("m"), key.WithHelp("m", modeLabel)),
		view:     key.NewBinding(key.WithKeys("v"), key.WithHelp("v", viewLabel)),
		vcd:      key.NewBinding(key.WithKeys("x"), key.WithHelp("x", "Export .vcd")),
		adcDown:  key.NewBinding(key.WithKeys("down"), key.WithHelp("down", "ADC -")),
		adcUp:    key.NewBinding(key.WithKeys("up"), key.WithHelp("up", "ADC +")),
		tempDown: key.NewBinding(key.WithKeys("left"), key.WithHelp("left", "I2C temp -")),
		tempUp:   key.NewBinding(key.WithKeys("right"), key.WithHelp("right", "I2C temp +")),
		reset:    key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "Reset HAL")),
		quit:     key.NewBinding(key.WithKeys("q"), key.WithHelp("q", "Quit")),
	}
	manual := m.wb.Mode() == workbench.ModeManual
	k.adcDown.SetEnabled(manual && m.wb.ADC() > 0)
	k.adcUp.SetEnabled(manual && m.wb.ADC() < 4095)
	k.tempDown.SetEnabled(m.wb.TempC() > -20)
	k.tempUp.SetEnabled(m.wb.TempC() < 85)
	k.vcd.SetEnabled(len(m.wb.Tracer().Samples()) > 0)
	return k
}

// NewTinyHALDemo constructs the standalone tinyhal hardware workbench demo.
// It renders directly from tinyhal's virtual ST7789 (SPI0) or SSD1306 (I2C0)
// and GPIO/PWM LED models without using displaytea or TinyBubbles cell buffers.
func NewTinyHALDemo(display drivers.Displayer, leds tinyhal.LEDWriter, monochrome bool) (tea.Model, error) {
	m := newTinyHALDemo(display, leds, monochrome)
	return browser.WrapActions(m, "tinyhal", func(model tea.Model) help.KeyMap {
		return model.(*tinyhalDemo).keys()
	}, nil)
}
