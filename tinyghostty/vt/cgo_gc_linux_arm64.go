//go:build !tinygo && linux && arm64

package vt

// #cgo LDFLAGS: -L${SRCDIR}/../lib/linux_arm64 -lghostty-vt
import "C"
