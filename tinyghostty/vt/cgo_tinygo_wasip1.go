//go:build tinygo && wasip1

package vt

// #cgo LDFLAGS: -L../lib/wasip1_wasm -lghostty-vt
import "C"
