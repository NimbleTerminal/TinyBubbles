package tinybubbles

import (
	"errors"
	"fmt"
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

func TestDisplayFailurePolicy(t *testing.T) {
	for _, initial := range []bool{true, false} {
		for _, rectangle := range []bool{true, false} {
			t.Run(fmt.Sprintf("initial=%v/rectangle=%v", initial, rectangle), func(t *testing.T) {
				d := &fillDisplay{testDisplay: newDisplay(12, 16)}
				m := &testModel{}
				polls := 0
				p, err := NewProgram(m, Config{Display: d, Input: func() Msg { polls++; return "input" }})
				if err != nil {
					t.Fatal(err)
				}
				if !initial {
					if err := p.Start(); err != nil {
						t.Fatal(err)
					}
					m.view.Set(0, 0, Cell{Rune: 'X'})
				}
				want := errors.New("temporary bus failure")
				if rectangle {
					d.fillErr = want
				} else {
					d.err = want
				}
				err = p.Run()
				var displayErr *DisplayError
				if !errors.As(err, &displayErr) || !errors.Is(err, want) {
					t.Fatalf("Run must fail fast with a display error: %v", err)
				}
				if p.Stopped() {
					t.Fatal("display failure discarded the running model")
				}
				d.fillErr, d.err = nil, nil
				beforePolls, beforeFlushes, beforeWrites := polls, d.flushes, d.writes
				if err := p.Step(); err != nil {
					t.Fatal(err)
				}
				if polls != beforePolls+1 || d.flushes != beforeFlushes+1 || d.writes <= beforeWrites {
					t.Fatal("Step did not poll and repaint after recovery")
				}
				if m.initCalls != 1 {
					t.Fatal("recovery reran Init")
				}
				if err := p.Send(Quit()); err != nil || !p.Stopped() {
					t.Fatal("quit lost after recovery", err)
				}
			})
		}
	}
}
