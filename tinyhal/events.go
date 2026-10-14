package tinyhal

import "image/color"

// EventKind identifies a high-level visual or bus event emitted by HAL or a device model.
type EventKind string

const (
	EventPin         EventKind = "pin"
	EventLED         EventKind = "led"
	EventButton      EventKind = "button"
	EventFramebuffer EventKind = "framebuffer"
	EventUART        EventKind = "uart"
)

// Event describes a high-level visual or peripheral update for a host or UI layer.
type Event struct {
	Kind       EventKind  `json:"kind"`
	ID         string     `json:"id,omitempty"`
	Bus        uint8      `json:"bus,omitempty"`
	Pin        Pin        `json:"pin,omitempty"`
	High       bool       `json:"high,omitempty"`
	Brightness uint8      `json:"brightness,omitempty"`
	Color      color.RGBA `json:"color,omitempty"`
	Pressed    bool       `json:"pressed,omitempty"`
	Width      int16      `json:"width,omitempty"`
	Height     int16      `json:"height,omitempty"`
	Pixels     []byte     `json:"-"`
	Data       []byte     `json:"data,omitempty"`
}

// InputKind identifies a high-level host input event injected into HAL.
type InputKind string

const (
	InputButton InputKind = "button"
	InputPin    InputKind = "pin"
	InputADC    InputKind = "adc"
	InputUART   InputKind = "uart"
)

// InputEvent describes an external stimulus from the host or browser UI.
type InputEvent struct {
	Kind    InputKind `json:"kind"`
	ID      string    `json:"id,omitempty"`
	Pin     Pin       `json:"pin,omitempty"`
	Bus     uint8     `json:"bus,omitempty"`
	Pressed bool      `json:"pressed,omitempty"`
	High    bool      `json:"high,omitempty"`
	Value   uint16    `json:"value,omitempty"`
	Data    []byte    `json:"data,omitempty"`
}

// OnEvent registers a callback invoked synchronously whenever HAL emits an Event.
func (h *HAL) OnEvent(fn func(Event)) {
	h.mu.Lock()
	h.onEvent = fn
	h.mu.Unlock()
}

// Emit queues ev in the bounded event buffer and notifies the OnEvent callback if set.
func (h *HAL) Emit(ev Event) {
	h.mu.Lock()
	if h.maxEvents > 0 {
		if len(h.events) >= h.maxEvents {
			copy(h.events, h.events[1:])
			h.events[len(h.events)-1] = ev
		} else {
			h.events = append(h.events, ev)
		}
	}
	cb := h.onEvent
	h.mu.Unlock()

	if cb != nil {
		cb(ev)
	}
}

// DrainEvents returns and clears all buffered high-level events.
func (h *HAL) DrainEvents() []Event {
	h.mu.Lock()
	if len(h.events) == 0 {
		h.mu.Unlock()
		return nil
	}
	out := make([]Event, len(h.events))
	copy(out, h.events)
	h.events = h.events[:0]
	h.mu.Unlock()
	return out
}

// InjectInput applies a high-level host input event to the simulation.
func (h *HAL) InjectInput(in InputEvent) error {
	switch in.Kind {
	case InputButton:
		h.mu.Lock()
		btn, ok := h.buttons[in.ID]
		if !ok && in.Pin != NoPin {
			for _, candidate := range h.buttons {
				if candidate.pin == in.Pin {
					btn = candidate
					ok = true
					break
				}
			}
		}
		h.mu.Unlock()
		if !ok {
			return ErrUnknownID
		}
		btn.SetPressed(in.Pressed)
		return nil
	case InputPin:
		h.SetPinInput(in.Pin, in.High)
		return nil
	case InputADC:
		h.SetADC(in.Pin, in.Value)
		return nil
	case InputUART:
		h.InjectUARTRX(in.Bus, in.Data)
		return nil
	default:
		return ErrUnknownID
	}
}
