package tinybubbles

import "time"

type Key uint8

const (
	KeyUnknown Key = iota
	KeyUp
	KeyDown
	KeyLeft
	KeyRight
	KeySelect
	KeyBack
	KeyQuit
)

// KeyMsg is a press event. KeyQuit is delivered to the model; only QuitMsg stops
// the program, so applications can confirm or ignore a quit button.
type KeyMsg struct{ Key Key }

// Button converts a logical pressed state from any hardware into debounced
// press events. Its zero value has no debounce. No repeat or release is emitted.
type Button struct {
	Key         Key
	Debounce    time.Duration
	raw, stable bool
	changed     time.Time
}

func (b *Button) Update(pressed bool, now time.Time) (KeyMsg, bool) {
	if pressed != b.raw {
		b.raw = pressed
		b.changed = now
	}
	if b.raw != b.stable && now.Sub(b.changed) >= b.Debounce {
		b.stable = b.raw
		if b.stable {
			return KeyMsg{b.Key}, true
		}
	}
	return KeyMsg{}, false
}

// ButtonPoller polls buttons in rotating order, returning at most one press.
// It stops sampling when it emits a press so simultaneous events are preserved
// for later polls. The caller owns Buttons and supplies logical pressed states.
// Like Button, it must be used from a single loop.
type ButtonPoller struct {
	Buttons []Button
	next    int
}

func (p *ButtonPoller) Poll(now time.Time, pressed func(int) bool) Msg {
	for n := 0; n < len(p.Buttons); n++ {
		i := (p.next + n) % len(p.Buttons)
		if msg, ok := p.Buttons[i].Update(pressed(i), now); ok {
			p.next = (i + 1) % len(p.Buttons)
			return msg
		}
	}
	return nil
}
