//go:build tinygo && esp32c3

package vt

// #cgo LDFLAGS: -L../lib/esp32c3 -lghostty-vt
import "C"
