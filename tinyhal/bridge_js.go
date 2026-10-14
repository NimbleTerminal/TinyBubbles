//go:build js && wasm

package tinyhal

import (
	"encoding/json"
	"syscall/js"
)

// BindJS connects a HAL instance to the minimal browser JavaScript host interface:
//   - Registers window.tinyhalInjectInput(jsonString) to inject InputEvent payloads.
//   - Registers window.tinyhalSetButton(id, pressed) as a convenience helper for buttons.
//   - Forwards emitted HAL Events to window.tinyhalOnEvent(eventObj, pixelsUint8Array) if defined.
func BindJS(h *HAL) {
	if h == nil {
		h = Default()
	}
	global := js.Global()

	injectFn := js.FuncOf(func(_ js.Value, args []js.Value) any {
		if len(args) < 1 {
			return false
		}
		var in InputEvent
		if err := json.Unmarshal([]byte(args[0].String()), &in); err != nil {
			return false
		}
		return h.InjectInput(in) == nil
	})
	global.Set("tinyhalInjectInput", injectFn)

	buttonFn := js.FuncOf(func(_ js.Value, args []js.Value) any {
		if len(args) < 2 {
			return false
		}
		err := h.InjectInput(InputEvent{
			Kind:    InputButton,
			ID:      args[0].String(),
			Pressed: args[1].Bool(),
		})
		return err == nil
	})
	global.Set("tinyhalSetButton", buttonFn)

	h.OnEvent(func(ev Event) {
		handler := global.Get("tinyhalOnEvent")
		if handler.IsUndefined() || handler.IsNull() {
			return
		}
		payload, err := json.Marshal(ev)
		if err != nil {
			return
		}
		var pixelsJS any
		if len(ev.Pixels) > 0 {
			arr := global.Get("Uint8Array").New(len(ev.Pixels))
			js.CopyBytesToJS(arr, ev.Pixels)
			pixelsJS = arr
		}
		handler.Invoke(string(payload), pixelsJS)
	})
}
