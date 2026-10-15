// Package tinytea provides a cooperative subset of Bubble Tea v2's model API
// for memory-constrained displays. See RUNTIMES.md for behavioral differences.
package tinytea

import (
	"errors"
	"sync/atomic"
	"time"

	bubbles "nimbleterminal.dev/tinybubbles"
	"tinygo.org/x/drivers"
)

type Msg = any
type Cmd func() Msg
type View struct{ Content string }

func NewView(content string) View { return View{Content: content} }

type Model interface {
	Init() Cmd
	Update(Msg) (Model, Cmd)
	View() View
}
type CellViewer interface{ CellView() *bubbles.Buffer }
type WindowSizeMsg struct{ Width, Height int }
type QuitMsg struct{}

func Quit() Msg { return QuitMsg{} }

// KeyPressMsg supports basic button presses, without modifiers or release events.
type KeyPressMsg struct{ Code rune }

const (
	KeyEnter  rune = 13
	KeyEscape rune = 27
)
const (
	KeyUp rune = 0x10ffff + 2 + iota
	KeyDown
	KeyRight
	KeyLeft
)

var (
	nextTimerID uint32
	timerClock  func() int64
)

func nowNano() int64 {
	if timerClock == nil {
		timerClock = func() int64 { return time.Now().UnixNano() }
	}
	return timerClock()
}

// Tick schedules a one-shot cooperative timer that invokes fn once d has
// elapsed on a later Step, without sleeping inside the command or starting a
// goroutine. Nonpositive durations fire on the next Step.
func Tick(d time.Duration, fn func(time.Time) Msg) Cmd {
	if fn == nil {
		return nil
	}
	created := nowNano()
	cb := func(ns int64) Msg { return fn(time.Unix(0, ns)) }
	return func() Msg {
		return timerSpec{created: created, duration: d, fn: cb}
	}
}

// Every schedules a one-shot cooperative timer aligned to the next multiple of
// d on the system clock. Nonpositive durations fire on the next Step.
func Every(d time.Duration, fn func(time.Time) Msg) Cmd {
	if fn == nil {
		return nil
	}
	created := nowNano()
	cb := func(ns int64) Msg { return fn(time.Unix(0, ns)) }
	return func() Msg {
		return timerSpec{created: created, duration: d, align: true, fn: cb}
	}
}

// After schedules msg to be delivered once d has elapsed on a later Step.
func After(d time.Duration, msg Msg) Cmd {
	if msg == nil {
		return nil
	}
	created := nowNano()
	return func() Msg {
		return timerSpec{created: created, duration: d, msg: msg}
	}
}

// Timer is a reusable, cancellable cooperative timer owned by a model.
// Its zero value is ready to use; NewTimer pre-assigns an ID when a model
// copies Timer by value before Init. Scheduling a new operation on the same
// Timer replaces and cancels any earlier pending schedule without manual
// generation counters.
type Timer struct {
	id     uint32
	gen    uint32
	active bool
}

// NewTimer returns a Timer with a pre-assigned identity suitable for models
// that pass Timer by value. Pointer-receiver models can use the zero value.
func NewTimer() Timer {
	return Timer{id: atomic.AddUint32(&nextTimerID, 1)}
}

func (t *Timer) nextGen() (uint32, uint32) {
	if t.id == 0 {
		t.id = atomic.AddUint32(&nextTimerID, 1)
	}
	t.gen++
	return t.id, t.gen
}

// Active reports whether the timer currently has a pending one-shot or
// repeating schedule.
func (t *Timer) Active() bool {
	return t != nil && t.active
}

// Cancel stops any pending one-shot or repeating schedule on t. Calling
// t.Cancel() inside Update takes effect immediately on pointer-receiver
// models; returning the resulting Cmd also scrubs pending entries when Timer
// is copied by value.
func (t *Timer) Cancel() Cmd {
	if t == nil {
		return nil
	}
	id, gen := t.nextGen()
	t.active = false
	return func() Msg {
		return timerSpec{owner: t, id: id, gen: gen, cancel: true}
	}
}

// Tick cancels any pending schedule on t and schedules a one-shot callback fn
// once d has elapsed on a later Step.
func (t *Timer) Tick(d time.Duration, fn func(time.Time) Msg) Cmd {
	if t == nil {
		return Tick(d, fn)
	}
	if fn == nil {
		return t.Cancel()
	}
	id, gen := t.nextGen()
	t.active = true
	created := nowNano()
	cb := func(ns int64) Msg { return fn(time.Unix(0, ns)) }
	return func() Msg {
		return timerSpec{owner: t, id: id, gen: gen, created: created, duration: d, fn: cb}
	}
}

// Every cancels any pending schedule on t and schedules a one-shot callback fn
// aligned to the next multiple of d on the system clock.
func (t *Timer) Every(d time.Duration, fn func(time.Time) Msg) Cmd {
	if t == nil {
		return Every(d, fn)
	}
	if fn == nil {
		return t.Cancel()
	}
	id, gen := t.nextGen()
	t.active = true
	created := nowNano()
	cb := func(ns int64) Msg { return fn(time.Unix(0, ns)) }
	return func() Msg {
		return timerSpec{owner: t, id: id, gen: gen, created: created, duration: d, align: true, fn: cb}
	}
}

// After cancels any pending schedule on t and delivers msg once d has elapsed
// on a later Step.
func (t *Timer) After(d time.Duration, msg Msg) Cmd {
	if t == nil {
		return After(d, msg)
	}
	if msg == nil {
		return t.Cancel()
	}
	id, gen := t.nextGen()
	t.active = true
	created := nowNano()
	return func() Msg {
		return timerSpec{owner: t, id: id, gen: gen, created: created, duration: d, msg: msg}
	}
}

// Repeat cancels any pending schedule on t and delivers msg every d across
// subsequent Step calls until cancelled or rescheduled.
func (t *Timer) Repeat(d time.Duration, msg Msg) Cmd {
	if t == nil {
		return After(d, msg)
	}
	if msg == nil {
		return t.Cancel()
	}
	id, gen := t.nextGen()
	t.active = true
	created := nowNano()
	return func() Msg {
		return timerSpec{owner: t, id: id, gen: gen, created: created, duration: d, repeat: true, msg: msg}
	}
}

// Program embeds the cooperative Step/Send/Stopped API. It has one owner:
// never call its methods concurrently or from interrupt handlers.
type Program struct {
	*bubbles.Program
	adapter *adapter
}

// NewProgram has the same display arguments as the Bubble Tea display adapter.
// Run returns the final model. Send returns synchronous rendering errors.
func NewProgram(model Model, display drivers.Displayer, font bubbles.Font, theme bubbles.Theme) (*Program, error) {
	if model == nil {
		return nil, errors.New("tinytea: nil model")
	}
	a := &adapter{model: model}
	a.timers = a.timerStorage[:0]
	p, err := bubbles.NewProgram(a, bubbles.Config{Display: display, Font: font, Theme: &theme, Input: a.pollTimers})
	if err != nil {
		return nil, err
	}
	return &Program{Program: p, adapter: a}, nil
}
func (p *Program) Run() (Model, error) {
	err := p.Program.Run()
	return p.adapter.model, err
}

// timerSpec uses int64 unix nanoseconds rather than time.Time fields so
// boxing timerSpec into Msg does not retain time.Time formatting methods in
// TinyGo firmware builds.
type timerSpec struct {
	owner    *Timer
	id       uint32
	gen      uint32
	created  int64
	deadline int64
	duration time.Duration
	align    bool
	repeat   bool
	cancel   bool
	fn       func(int64) Msg
	msg      Msg
}

func (s *timerSpec) active() bool {
	if s.cancel || (s.fn == nil && s.msg == nil) {
		return false
	}
	if s.owner != nil && s.owner.gen != s.gen {
		return false
	}
	return true
}

func (s *timerSpec) fire(now int64) Msg {
	if !s.repeat && s.owner != nil && s.owner.gen == s.gen {
		s.owner.active = false
	}
	if s.fn != nil {
		return s.fn(now)
	}
	return s.msg
}

type timerFiredMsg struct{ msg Msg }

type adapter struct {
	model         Model
	buffer        *bubbles.Buffer
	width, height int
	now           func() time.Time
	timerStorage  [4]timerSpec
	timers        []timerSpec
	timerCursor   int
	firedInPoll   bool
}

func (a *adapter) clock() int64 {
	if a.now != nil {
		return a.now().UnixNano()
	}
	if timerClock != nil {
		return timerClock()
	}
	return 0
}

func (a *adapter) callCreated() int64 {
	if a.now != nil {
		return a.now().UnixNano()
	}
	return 0
}

func computeDeadline(base int64, d time.Duration, align bool) int64 {
	if d <= 0 {
		return base
	}
	ns := int64(d)
	if align {
		rem := base % ns
		if rem < 0 {
			rem += ns
		}
		return base + (ns - rem)
	}
	return base + ns
}

func (a *adapter) clearTimers() {
	for i := range a.timers {
		if a.timers[i].owner != nil && a.timers[i].owner.gen == a.timers[i].gen {
			a.timers[i].owner.active = false
		}
		a.timers[i] = timerSpec{}
	}
	a.timers = a.timerStorage[:0]
	a.timerCursor = 0
}

func (a *adapter) matchTimer(slot, spec *timerSpec) bool {
	if spec.id != 0 && slot.id == spec.id {
		return true
	}
	if spec.owner != nil && slot.owner == spec.owner {
		return true
	}
	return false
}

func (a *adapter) registerTimer(spec timerSpec) {
	if spec.cancel || !spec.active() {
		for i := range a.timers {
			if a.matchTimer(&a.timers[i], &spec) && a.timers[i].gen <= spec.gen {
				if a.timers[i].owner != nil && a.timers[i].owner.gen == a.timers[i].gen {
					a.timers[i].owner.active = false
				}
				a.timers[i] = timerSpec{}
			}
		}
		return
	}
	for i := range a.timers {
		if a.matchTimer(&a.timers[i], &spec) {
			if a.timers[i].gen > spec.gen {
				return
			}
			a.timers[i] = spec
			return
		}
	}
	for i := range a.timers {
		if !a.timers[i].active() {
			a.timers[i] = spec
			return
		}
	}
	a.timers = append(a.timers, spec)
}

func (a *adapter) pollTimers() bubbles.Msg {
	a.firedInPoll = false
	n := len(a.timers)
	if n == 0 {
		return nil
	}
	now := a.clock()
	best := -1
	for k := 0; k < n; k++ {
		i := (a.timerCursor + k) % n
		if !a.timers[i].active() {
			a.timers[i] = timerSpec{}
			continue
		}
		if now < a.timers[i].deadline {
			continue
		}
		if best == -1 || a.timers[i].deadline < a.timers[best].deadline {
			best = i
		}
	}
	if best == -1 {
		return nil
	}
	a.timerCursor = (best + 1) % n
	a.firedInPoll = true
	spec := a.timers[best]
	if spec.repeat {
		ns := int64(spec.duration)
		if ns > 0 {
			for a.timers[best].deadline <= now {
				a.timers[best].deadline += ns
			}
		} else {
			a.timers[best].deadline = now
		}
	} else {
		a.timers[best] = timerSpec{}
	}
	out := spec.fire(now)
	if out == nil {
		return nil
	}
	if _, ok := out.(QuitMsg); ok {
		a.clearTimers()
		return bubbles.Quit()
	}
	return timerFiredMsg{msg: out}
}

func (a *adapter) handleTimerSpec(spec timerSpec, created int64, firedInPoll bool) bubbles.Msg {
	if a.now != nil || spec.created == 0 {
		spec.created = created
	}
	spec.deadline = computeDeadline(spec.created, spec.duration, spec.align)
	if spec.cancel || !spec.active() {
		a.registerTimer(spec)
		return nil
	}
	now := a.clock()
	if !firedInPoll && now >= spec.deadline {
		if spec.repeat {
			next := spec
			ns := int64(next.duration)
			if ns > 0 {
				for next.deadline <= now {
					next.deadline += ns
				}
			} else {
				next.deadline = now
			}
			a.registerTimer(next)
		} else {
			// Clear any older schedule on the same reusable Timer.
			cancelOld := spec
			cancelOld.cancel = true
			a.registerTimer(cancelOld)
		}
		out := spec.fire(now)
		if out == nil {
			return nil
		}
		if _, ok := out.(QuitMsg); ok {
			a.clearTimers()
			return bubbles.Quit()
		}
		return timerFiredMsg{msg: out}
	}
	a.registerTimer(spec)
	return nil
}

func (a *adapter) command(cmd Cmd, created int64) bubbles.Cmd {
	if cmd == nil {
		return nil
	}
	return func() bubbles.Msg {
		firedInPoll := a.firedInPoll
		a.firedInPoll = false
		msg := cmd()
		if spec, ok := msg.(timerSpec); ok {
			return a.handleTimerSpec(spec, created, firedInPoll)
		}
		if _, ok := msg.(QuitMsg); ok {
			a.clearTimers()
			return bubbles.Quit()
		}
		return msg
	}
}
func (a *adapter) Init() bubbles.Cmd {
	created := a.callCreated()
	return a.command(a.model.Init(), created)
}
func (a *adapter) Update(msg bubbles.Msg) (bubbles.Model, bubbles.Cmd) {
	var event Msg = msg
	switch msg := msg.(type) {
	case QuitMsg:
		a.clearTimers()
		return a, bubbles.Quit
	case timerFiredMsg:
		event = msg.msg
	case bubbles.WindowSizeMsg:
		a.width, a.height = msg.Width, msg.Height
		if a.buffer != nil {
			a.buffer.Resize(a.width, a.height)
		}
		event = WindowSizeMsg{Width: msg.Width, Height: msg.Height}
	case bubbles.KeyMsg:
		switch msg.Key {
		case bubbles.KeyUp:
			event = KeyPressMsg{Code: KeyUp}
		case bubbles.KeyDown:
			event = KeyPressMsg{Code: KeyDown}
		case bubbles.KeyRight:
			event = KeyPressMsg{Code: KeyRight}
		case bubbles.KeyLeft:
			event = KeyPressMsg{Code: KeyLeft}
		case bubbles.KeySelect:
			event = KeyPressMsg{Code: KeyEnter}
		case bubbles.KeyBack:
			event = KeyPressMsg{Code: KeyEscape}
		case bubbles.KeyQuit:
			event = KeyPressMsg{Code: 'q'}
		}
	}
	created := a.callCreated()
	next, cmd := a.model.Update(event)
	if next == nil {
		return nil, nil
	}
	a.model = next
	return a, a.command(cmd, created)
}
func (a *adapter) View() *bubbles.Buffer {
	if m, ok := a.model.(CellViewer); ok {
		return m.CellView()
	}
	if a.buffer == nil {
		a.buffer = bubbles.NewBuffer(a.width, a.height)
	}
	a.buffer.Clear(bubbles.Normal)
	a.buffer.Text(0, 0, a.model.View().Content, bubbles.Normal)
	return a.buffer
}
