//go:build js && wasm

package web

import "syscall/js"

func exportVCD(filename, vcd string) {
	global := js.Global()
	doc := global.Get("document")
	blobCtor := global.Get("Blob")
	urlObj := global.Get("URL")
	if doc.IsUndefined() || blobCtor.IsUndefined() || urlObj.IsUndefined() {
		return
	}
	parts := global.Get("Array").New(vcd)
	opts := global.Get("Object").New()
	opts.Set("type", "text/plain;charset=utf-8")
	blob := blobCtor.New(parts, opts)
	href := urlObj.Call("createObjectURL", blob)
	link := doc.Call("createElement", "a")
	link.Set("href", href)
	link.Set("download", filename)
	link.Call("click")
	urlObj.Call("revokeObjectURL", href)
}
