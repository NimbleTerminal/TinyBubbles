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

// KeyMsg is a press or repeat event. KeyQuit is delivered to the model; only
// QuitMsg stops the program, so applications can confirm or ignore a quit button.
type KeyMsg struct{ Key Key }

// KeyReleaseMsg is a release event emitted by Button.Poll and ButtonPoller.Poll
// when Button.EmitRelease is enabled.
type KeyReleaseMsg struct{ Key Key }

// Button converts a logical pressed state from any hardware into debounced
// press, optional hold-to-repeat, optional long-press, and optional release events.
// Its zero value has no debounce, repeat, long-press, or release events.
type Button struct {
	Key            Key
	LongKey        Key           // Optional key emitted once held for LongPress; defaults to Key when LongPress > 0.
	Debounce       time.Duration // Minimum stable time before a press or release edge is recognized.
	RepeatDelay    time.Duration // Optional hold time before auto-repeat begins (0 disables repeat).
	RepeatInterval time.Duration // Optional interval between repeats (defaults to RepeatDelay when <= 0).
	LongPress      time.Duration // Optional hold time to trigger LongKey and defer short Key to release.
	EmitRelease    bool          // When true, Poll and ButtonPoller.Poll emit KeyReleaseMsg on debounced release.

	raw, stable    bool
	longFired      bool
	activeKey      Key
	pendingRelease Key
	changed        time.Time
	pressedAt      time.Time
	nextRepeat     time.Time
}

// Update advances the debounced button state and reports a press, long-press, or repeat KeyMsg.
func (b *Button) Update(pressed bool, now time.Time) (KeyMsg, bool) {
	if pressed != b.raw {
		b.raw = pressed
		b.changed = now
	}
	if b.raw != b.stable && now.Sub(b.changed) >= b.Debounce {
		b.stable = b.raw
		if b.stable {
			b.pressedAt = now
			b.longFired = false
			b.activeKey = b.Key
			b.pendingRelease = KeyUnknown
			if b.LongPress > 0 {
				return KeyMsg{}, false
			}
			if b.RepeatDelay > 0 {
				b.nextRepeat = now.Add(b.RepeatDelay)
			}
			return KeyMsg{b.Key}, true
		}
		// Debounced release edge.
		if b.LongPress > 0 && !b.longFired {
			if b.EmitRelease {
				b.pendingRelease = b.Key
			}
			return KeyMsg{b.Key}, true
		}
		if b.EmitRelease {
			relKey := b.activeKey
			if relKey == KeyUnknown {
				relKey = b.Key
			}
			b.pendingRelease = relKey
		}
		return KeyMsg{}, false
	}

	if !b.stable {
		return KeyMsg{}, false
	}

	// Held state: check long-press threshold first.
	if b.LongPress > 0 && !b.longFired {
		if now.Sub(b.pressedAt) >= b.LongPress {
			b.longFired = true
			key := b.LongKey
			if key == KeyUnknown {
				key = b.Key
			}
			b.activeKey = key
			if b.RepeatDelay > 0 {
				b.nextRepeat = now.Add(b.RepeatDelay)
			}
			return KeyMsg{key}, true
		}
		return KeyMsg{}, false
	}

	// Held state: check auto-repeat schedule.
	if b.RepeatDelay > 0 && !now.Before(b.nextRepeat) {
		interval := b.RepeatInterval
		if interval <= 0 {
			interval = b.RepeatDelay
		}
		for !now.Before(b.nextRepeat) {
			b.nextRepeat = b.nextRepeat.Add(interval)
		}
		key := b.activeKey
		if key == KeyUnknown {
			key = b.Key
		}
		return KeyMsg{key}, true
	}

	return KeyMsg{}, false
}

// Poll advances the button state and returns a KeyMsg, a KeyReleaseMsg (when
// EmitRelease is enabled), or nil.
func (b *Button) Poll(pressed bool, now time.Time) Msg {
	if b.pendingRelease != KeyUnknown {
		key := b.pendingRelease
		b.pendingRelease = KeyUnknown
		return KeyReleaseMsg{Key: key}
	}
	if msg, ok := b.Update(pressed, now); ok {
		return msg
	}
	if b.pendingRelease != KeyUnknown {
		key := b.pendingRelease
		b.pendingRelease = KeyUnknown
		return KeyReleaseMsg{Key: key}
	}
	return nil
}

// ButtonPoller polls buttons in rotating order, returning at most one event.
// It stops sampling when it emits an event so simultaneous presses, repeats, or
// releases are preserved for later polls. The caller owns Buttons and supplies
// logical pressed states. Like Button, it must be used from a single loop.
type ButtonPoller struct {
	Buttons []Button
	next    int
}

func (p *ButtonPoller) Poll(now time.Time, pressed func(int) bool) Msg {
	for n := 0; n < len(p.Buttons); n++ {
		i := (p.next + n) % len(p.Buttons)
		if msg := p.Buttons[i].Poll(pressed(i), now); msg != nil {
			p.next = (i + 1) % len(p.Buttons)
			return msg
		}
	}
	return nil
}
