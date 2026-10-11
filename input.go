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
