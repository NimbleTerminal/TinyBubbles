//go:build tinygo && baremetal

package vt

// #cgo LDFLAGS: -L../lib/baremetal -lghostty-vt
import "C"
