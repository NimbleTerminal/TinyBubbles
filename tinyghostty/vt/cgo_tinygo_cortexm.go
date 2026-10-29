//go:build tinygo && cortexm

// Build lib/cortexm for your CPU; make lib-cortexm targets the Cortex-M4F.

package vt

// #cgo LDFLAGS: -L../lib/cortexm -lghostty-vt
import "C"
