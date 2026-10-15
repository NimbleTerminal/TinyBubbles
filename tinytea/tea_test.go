package tinytea

import (
	"image/color"
	"reflect"
	"testing"
	"time"

	bubbles "nimbleterminal.dev/tinybubbles"
	"tinygo.org/x/drivers"
)

type display struct{ flushes int }

func (*display) Size() (int16, int16)              { return 120, 64 }
func (*display) SetPixel(int16, int16, color.RGBA) {}
func (d *display) Display() error                  { d.flushes++; return nil }

type countMsg struct{}
type model struct {
	width, count int
	ready        bool
}

func (m *model) Init() Cmd { return func() Msg { m.ready = m.width == 20; return countMsg{} } }
func (m *model) Update(msg Msg) (Model, Cmd) {
	switch msg := msg.(type) {
	case WindowSizeMsg:
		m.width = msg.Width
	case countMsg:
		m.count++
	case KeyPressMsg:
		if msg.Code == KeyEnter {
			m.count++
		}
		if msg.Code == 'q' {
			return m, Quit
		}
	}
	return m, nil
}
func (*model) View() View { return NewView("TinyTea") }
func TestCooperativeModelAndKeys(t *testing.T) {
	m := &model{}
	d := &display{}
	p, err := NewProgram(m, d, nil, bubbles.DefaultTheme(false))
	if err != nil {
		t.Fatal(err)
	}
	if err = p.Start(); err != nil {
		t.Fatal(err)
	}
	if m.count != 0 || m.width != 20 {
		t.Fatalf("unexpected Start: %+v", m)
	}
	if err = p.Step(); err != nil {
		t.Fatal(err)
	}
	if !m.ready || m.count != 1 {
		t.Fatalf("command ran before dimensions: %+v", m)
	}
	if err = p.Send(bubbles.KeyMsg{Key: bubbles.KeySelect}); err != nil {
		t.Fatal(err)
	}
	if m.count != 2 {
		t.Fatal("button not mapped to Enter")
	}
	if err = p.Send(KeyPressMsg{Code: 'q'}); err != nil {
		t.Fatal(err)
	}
	final, err := p.Run()
	if err != nil {
		t.Fatal(err)
	}
	if final != m || !p.Stopped() || d.flushes == 0 {
		t.Fatal("incorrect final state")
	}
}

type cellModel struct {
	model
	cells *bubbles.Buffer
}

func (m *cellModel) Init() Cmd                 { return nil }
func (m *cellModel) Update(Msg) (Model, Cmd)   { return m, nil }
func (m *cellModel) CellView() *bubbles.Buffer { return m.cells }

type font struct {
	glyph      rune
	foreground color.RGBA
}

func (*font) Size() (int16, int16) { return 6, 8 }
func (f *font) DrawGlyph(_ drivers.Displayer, _, _ int16, r rune, c color.RGBA) {
	f.glyph = r
	f.foreground = c
}
func TestCustomCellRendering(t *testing.T) {
	cells := bubbles.NewBuffer(20, 8)
	cells.Set(0, 0, bubbles.Cell{Rune: 'X', Style: bubbles.Accent})
	f := &font{}
	theme := bubbles.DefaultTheme(false)
	p, err := NewProgram(&cellModel{cells: cells}, &display{}, f, theme)
	if err != nil {
		t.Fatal(err)
	}
	if err = p.Step(); err != nil {
		t.Fatal(err)
	}
	if f.glyph != 'X' || f.foreground != theme.Styles[bubbles.Accent].Foreground {
		t.Fatalf("custom font/style lost: %+v", f)
	}
}

func TestFallbackGridIsLazyAndReusable(t *testing.T) {
	a := &adapter{model: &cellModel{cells: bubbles.NewBuffer(20, 8)}}
	a.Update(bubbles.WindowSizeMsg{Width: 20, Height: 8})
	a.View()
	if a.buffer != nil {
		t.Fatal("cell model allocated fallback grid")
	}
	a.model = &model{}
	b := a.View()
	if b.At(0, 0).Rune != 'T' {
		t.Fatal("plain view missing")
	}
	if a.View() != b {
		t.Fatal("fallback grid not reused")
	}
	a.Update(bubbles.WindowSizeMsg{Width: 10, Height: 4})
	if w, h := a.View().Size(); w != 10 || h != 4 {
		t.Fatal(w, h)
	}
}

type timerTestModel struct {
	events []string
	at     []time.Time
	timer  Timer
	next   func(Msg) Cmd
}

func (m *timerTestModel) Init() Cmd {
	if m.next != nil {
		return m.next("init")
	}
	return nil
}

func (m *timerTestModel) Update(msg Msg) (Model, Cmd) {
	switch msg := msg.(type) {
	case string:
		m.events = append(m.events, msg)
	case time.Time:
		m.events = append(m.events, "time")
		m.at = append(m.at, msg)
	}
	if m.next != nil {
		return m, m.next(msg)
	}
	return m, nil
}

func (*timerTestModel) View() View { return NewView("timers") }

func TestCooperativeTimersTickEveryAfter(t *testing.T) {
	if Tick(time.Second, nil) != nil || Every(time.Second, nil) != nil || After(time.Second, nil) != nil {
		t.Fatal("expected nil Cmd for nil timer callback/message")
	}

	now := time.Unix(100, 12*int64(time.Millisecond))
	m := &timerTestModel{}
	m.next = func(msg Msg) Cmd {
		switch msg {
		case "init":
			return Tick(50*time.Millisecond, func(ts time.Time) Msg { return ts })
		case "schedule-every":
			return Every(50*time.Millisecond, func(ts time.Time) Msg { return ts })
		case "schedule-after":
			return After(30*time.Millisecond, "delayed")
		case "schedule-zero":
			return Tick(0, func(time.Time) Msg { return "zero-1" })
		case "zero-1":
			return After(0, "zero-2")
		}
		return nil
	}

	p, err := NewProgram(m, &display{}, nil, bubbles.DefaultTheme(false))
	if err != nil {
		t.Fatal(err)
	}
	p.adapter.now = func() time.Time { return now }

	if err := p.Start(); err != nil {
		t.Fatal(err)
	}
	// Step at +10ms (22ms): timer registered in adapter, not due until 62ms.
	now = now.Add(10 * time.Millisecond)
	if err := p.Step(); err != nil {
		t.Fatal(err)
	}
	if len(m.events) != 0 {
		t.Fatalf("timer fired early: %v", m.events)
	}

	// Immediate command via Send must not be blocked behind waiting timer.
	m.next = func(msg Msg) Cmd {
		if msg == "trigger-immediate" {
			return func() Msg { return "immediate-ran" }
		}
		return nil
	}
	if err := p.Send("trigger-immediate"); err != nil {
		t.Fatal(err)
	}
	if err := p.Step(); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(m.events, []string{"trigger-immediate", "immediate-ran"}) {
		t.Fatalf("immediate command blocked by timer: %v", m.events)
	}
	m.events = nil

	// Advance to 62ms: initial 50ms Tick fires.
	now = time.Unix(100, 62*int64(time.Millisecond))
	if err := p.Step(); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(m.events, []string{"time"}) || !m.at[0].Equal(now) {
		t.Fatalf("unexpected Tick firing: events=%v at=%v", m.events, m.at)
	}
	m.events = nil
	m.at = nil

	// Schedule Every(50ms) at 62ms -> should align to 100ms (next 50ms multiple).
	m.next = func(msg Msg) Cmd {
		if msg == "schedule-every" {
			return Every(50*time.Millisecond, func(ts time.Time) Msg { return ts })
		}
		return nil
	}
	if err := p.Send("schedule-every"); err != nil {
		t.Fatal(err)
	}
	if err := p.Step(); err != nil {
		t.Fatal(err)
	}
	now = time.Unix(100, 99*int64(time.Millisecond))
	if err := p.Step(); err != nil {
		t.Fatal(err)
	}
	if len(m.at) != 0 {
		t.Fatalf("Every fired before aligned boundary: %v", m.at)
	}
	now = time.Unix(100, 100*int64(time.Millisecond))
	if err := p.Step(); err != nil {
		t.Fatal(err)
	}
	if len(m.at) != 1 || !m.at[0].Equal(now) {
		t.Fatalf("Every did not fire at 100ms boundary: %v", m.at)
	}

	// Schedule After(30ms) and zero-duration chained timers.
	m.events = nil
	m.next = func(msg Msg) Cmd {
		switch msg {
		case "schedule-after":
			return After(30*time.Millisecond, "delayed")
		case "delayed":
			return Tick(0, func(time.Time) Msg { return "zero-1" })
		case "zero-1":
			return After(0, "zero-2")
		}
		return nil
	}
	if err := p.Send("schedule-after"); err != nil {
		t.Fatal(err)
	}
	if err := p.Step(); err != nil {
		t.Fatal(err)
	}
	now = now.Add(30 * time.Millisecond)
	// Step 1 fires "delayed" and queues "zero-1" without firing it in the same Step.
	if err := p.Step(); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(m.events, []string{"schedule-after", "delayed"}) {
		t.Fatalf("expected one timer firing per Step, got %v", m.events)
	}
	// Step 2 fires "zero-1" and queues "zero-2".
	if err := p.Step(); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(m.events, []string{"schedule-after", "delayed", "zero-1"}) {
		t.Fatalf("unexpected events after second Step: %v", m.events)
	}
	// Step 3 fires "zero-2".
	if err := p.Step(); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(m.events, []string{"schedule-after", "delayed", "zero-1", "zero-2"}) {
		t.Fatalf("unexpected events after third Step: %v", m.events)
	}
}

type valueTimerModel struct {
	timer  Timer
	events []string
}

func (m valueTimerModel) Init() Cmd {
	return m.timer.After(40*time.Millisecond, "from-init")
}

func (m valueTimerModel) Update(msg Msg) (Model, Cmd) {
	switch msg := msg.(type) {
	case string:
		m.events = append(m.events, msg)
		if msg == "cancel" {
			return m, m.timer.Cancel()
		}
	}
	return m, nil
}

func (valueTimerModel) View() View { return NewView("value-timer") }

func TestTimerReuseCancelAndRepeat(t *testing.T) {
	now := time.Unix(200, 0)
	m := &timerTestModel{}
	p, err := NewProgram(m, &display{}, nil, bubbles.DefaultTheme(false))
	if err != nil {
		t.Fatal(err)
	}
	p.adapter.now = func() time.Time { return now }
	if err := p.Start(); err != nil {
		t.Fatal(err)
	}

	// Rescheduling the same Timer replaces the earlier schedule without firing it.
	m.next = func(msg Msg) Cmd {
		switch msg {
		case "arm-1":
			return m.timer.After(50*time.Millisecond, "stale")
		case "arm-2":
			return m.timer.After(50*time.Millisecond, "fresh")
		}
		return nil
	}
	if err := p.Send("arm-1"); err != nil {
		t.Fatal(err)
	}
	if !m.timer.Active() {
		t.Fatal("expected timer to be active after After")
	}
	if err := p.Step(); err != nil {
		t.Fatal(err)
	}
	now = now.Add(30 * time.Millisecond)
	if err := p.Send("arm-2"); err != nil {
		t.Fatal(err)
	}
	if err := p.Step(); err != nil {
		t.Fatal(err)
	}
	// At +60ms (past arm-1's 50ms deadline, before arm-2's 80ms deadline): nothing fires.
	now = time.Unix(200, 60*int64(time.Millisecond))
	if err := p.Step(); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(m.events, []string{"arm-1", "arm-2"}) {
		t.Fatalf("stale timer should not have fired: %v", m.events)
	}
	// At +80ms: "fresh" fires and marks one-shot timer inactive.
	now = time.Unix(200, 80*int64(time.Millisecond))
	if err := p.Step(); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(m.events, []string{"arm-1", "arm-2", "fresh"}) || m.timer.Active() {
		t.Fatalf("expected fresh timer firing and inactive state: events=%v active=%v", m.events, m.timer.Active())
	}

	// Direct statement cancellation in Update while returning another command.
	m.events = nil
	m.next = func(msg Msg) Cmd {
		switch msg {
		case "arm-cancel":
			return m.timer.After(40*time.Millisecond, "never")
		case "stop-now":
			m.timer.Cancel()
			return func() Msg { return "stopped" }
		}
		return nil
	}
	_ = p.Send("arm-cancel")
	_ = p.Step()
	_ = p.Send("stop-now")
	_ = p.Step()
	now = now.Add(100 * time.Millisecond)
	_ = p.Step()
	if !reflect.DeepEqual(m.events, []string{"arm-cancel", "stop-now", "stopped"}) || m.timer.Active() {
		t.Fatalf("direct Cancel failed: events=%v active=%v", m.events, m.timer.Active())
	}

	// Repeating timer fires periodically, coalesces long pauses, and stops on Cancel.
	m.events = nil
	m.next = func(msg Msg) Cmd {
		switch msg {
		case "start-repeat":
			return m.timer.Repeat(25*time.Millisecond, "tick")
		case "cancel-repeat":
			return m.timer.Cancel()
		}
		return nil
	}
	_ = p.Send("start-repeat")
	_ = p.Step()
	for i := 0; i < 3; i++ {
		now = now.Add(25 * time.Millisecond)
		if err := p.Step(); err != nil {
			t.Fatal(err)
		}
	}
	if !reflect.DeepEqual(m.events, []string{"start-repeat", "tick", "tick", "tick"}) || !m.timer.Active() {
		t.Fatalf("unexpected repeat events: %v", m.events)
	}
	// Long pause (+250ms) should coalesce to a single tick on the next Step.
	now = now.Add(250 * time.Millisecond)
	_ = p.Step()
	_ = p.Step()
	if !reflect.DeepEqual(m.events, []string{"start-repeat", "tick", "tick", "tick", "tick"}) {
		t.Fatalf("long pause burst-fired ticks: %v", m.events)
	}
	_ = p.Send("cancel-repeat")
	_ = p.Step()
	now = now.Add(100 * time.Millisecond)
	_ = p.Step()
	if !reflect.DeepEqual(m.events, []string{"start-repeat", "tick", "tick", "tick", "tick", "cancel-repeat"}) || m.timer.Active() {
		t.Fatalf("cancelled repeat still fired: %v", m.events)
	}

	// Value-receiver model using NewTimer() can cancel an Init timer via returned Cmd.
	vm := valueTimerModel{timer: NewTimer()}
	vp, err := NewProgram(vm, &display{}, nil, bubbles.DefaultTheme(false))
	if err != nil {
		t.Fatal(err)
	}
	vp.adapter.now = func() time.Time { return now }
	_ = vp.Start()
	_ = vp.Step()
	_ = vp.Send("cancel")
	_ = vp.Step()
	now = now.Add(100 * time.Millisecond)
	_ = vp.Step()
	final, err := vp.adapter.model.(valueTimerModel), error(nil)
	if err != nil || !reflect.DeepEqual(final.events, []string{"cancel"}) {
		t.Fatalf("value-receiver NewTimer cancel failed: %v", final.events)
	}

	// Nil *Timer receiver fallbacks and nil callback/msg cancellation on Timer.
	var nilTimer *Timer
	if nilTimer.Cancel() != nil || nilTimer.Active() {
		t.Fatal("expected nil Timer Cancel/Active to be no-ops")
	}
	if nilTimer.Tick(time.Millisecond, func(time.Time) Msg { return "x" }) == nil ||
		nilTimer.Every(time.Millisecond, func(time.Time) Msg { return "x" }) == nil ||
		nilTimer.After(time.Millisecond, "x") == nil ||
		nilTimer.Repeat(time.Millisecond, "x") == nil {
		t.Fatal("expected nil *Timer scheduling methods to fall back to package helpers")
	}
	var tmr Timer
	_ = tmr.Tick(10*time.Millisecond, func(time.Time) Msg { return "t" })
	if !tmr.Active() {
		t.Fatal("expected Timer.Tick to mark timer active")
	}
	_ = tmr.Tick(10*time.Millisecond, nil)
	if tmr.Active() {
		t.Fatal("expected Timer.Tick(..., nil) to cancel timer")
	}
	_ = tmr.Every(10*time.Millisecond, func(time.Time) Msg { return "e" })
	_ = tmr.Every(10*time.Millisecond, nil)
	_ = tmr.After(10*time.Millisecond, nil)
	_ = tmr.Repeat(10*time.Millisecond, nil)

	// Schedule >4 concurrent timers and stop via QuitMsg from a timer.
	m.events = nil
	for i := 0; i < 5; i++ {
		label := string(rune('A' + i))
		m.next = func(Msg) Cmd { return After(time.Duration(i+1)*10*time.Millisecond, label) }
		_ = p.Send("q")
		_ = p.Step()
	}
	m.next = func(Msg) Cmd { return After(60*time.Millisecond, QuitMsg{}) }
	_ = p.Send("q")
	_ = p.Step()
	now = now.Add(100 * time.Millisecond)
	for i := 0; i < 6 && !p.Stopped(); i++ {
		if err := p.Step(); err != nil {
			t.Fatal(err)
		}
	}
	if !p.Stopped() {
		t.Fatal("expected timer QuitMsg to stop program")
	}
}
