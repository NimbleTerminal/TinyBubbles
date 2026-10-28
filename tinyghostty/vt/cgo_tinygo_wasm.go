//go:build tinygo && wasm

// The wasm32-freestanding archive serves both wasip1 and js.

package vt

// #cgo LDFLAGS: -L../lib/wasm -lghostty-vt
import "C"
