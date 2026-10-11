package tinybubbles

import (
	"reflect"
	"testing"
)

type testModel struct {
	events    []Msg
	initCalls int
	view      *Buffer
	next      func(Msg) Cmd
}

func (m *testModel) Init() Cmd { m.initCalls++; return func() Msg { return "init" } }
func (m *testModel) Update(msg Msg) (Model, Cmd) {
	m.events = append(m.events, msg)
	if size, ok := msg.(WindowSizeMsg); ok {
		m.view = NewBuffer(size.Width, size.Height)
	}
	if m.next != nil {
		return m, m.next(msg)
	}
	return m, nil
}
func (m *testModel) View() *Buffer { return m.view }
func newTestProgram(t *testing.T, m Model) *Program {
	t.Helper()
	p, err := NewProgram(m, Config{Display: newDisplay(12, 16)})
	if err != nil {
		t.Fatal(err)
	}
	return p
}
func TestLifecycleAndCustomMessages(t *testing.T) {
	m := &testModel{}
	p := newTestProgram(t, m)
	if err := p.Start(); err != nil {
		t.Fatal(err)
	}
	_ = p.Start()
	if m.initCalls != 1 || len(m.events) != 1 {
		t.Fatal("Start not idempotent")
	}
	if got := m.events[0]; got != (WindowSizeMsg{2, 2, 12, 16}) {
		t.Fatal(got)
	}
	type sensor struct{ Value int }
	if err := p.Send(sensor{42}); err != nil {
		t.Fatal(err)
	}
	if err := p.Step(); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(m.events[1:], []Msg{sensor{42}, "init"}) {
		t.Fatal(m.events)
	}
	if err := p.Send(KeyMsg{KeyQuit}); err != nil {
		t.Fatal(err)
	}
	if p.Stopped() {
		t.Fatal("key bypassed model")
	}
	if err := p.Send(Quit()); err != nil {
		t.Fatal(err)
	}
	n := len(m.events)
	_ = p.Send("ignored")
	_ = p.Step()
	if !p.Stopped() || len(m.events) != n {
		t.Fatal("quit not respected")
	}
}
func TestCommandsAreBoundedAndInputIsNotStarved(t *testing.T) {
	m := &testModel{next: func(msg Msg) Cmd { return func() Msg { return "again" } }}
	p := newTestProgram(t, m)
	calls := 0
	p.input = func() Msg { calls++; return nil }
	for i := 0; i < 100; i++ {
		if err := p.Step(); err != nil {
			t.Fatal(err)
		}
	}
	if calls != 100 || len(m.events) != 101 {
		t.Fatalf("calls=%d events=%d", calls, len(m.events))
	}
	for p.count < len(p.commands) {
		if err := p.Send("more"); err != nil {
			t.Fatal(err)
		}
	}
	if err := p.Send("overflow"); err != ErrCommandQueueFull {
		t.Fatal(err)
	}
}
func TestQuitCommandStopsAndClearsQueue(t *testing.T) {
	m := &testModel{next: func(msg Msg) Cmd {
		if _, ok := msg.(KeyMsg); ok {
			return Quit
		}
		return nil
	}}
	p := newTestProgram(t, m)
	_ = p.Send(KeyMsg{KeyQuit})
	_ = p.Step()
	_ = p.Step()
	if !p.Stopped() || p.count != 0 {
		t.Fatal("quit command failed")
	}
}
func TestProgramValidation(t *testing.T) {
	if _, err := NewProgram(nil, Config{}); err != ErrNilModel {
		t.Fatal(err)
	}
	if _, err := NewProgram(&testModel{}, Config{}); err == nil {
		t.Fatal("nil display accepted")
	}
}
