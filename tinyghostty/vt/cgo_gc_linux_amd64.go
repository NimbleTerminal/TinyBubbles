//go:build !tinygo && linux && amd64

package vt

// #cgo LDFLAGS: -L${SRCDIR}/../lib/linux_amd64 -lghostty-vt
import "C"
