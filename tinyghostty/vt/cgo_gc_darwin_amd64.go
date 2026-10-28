//go:build !tinygo && darwin && amd64

package vt

// #cgo LDFLAGS: -L${SRCDIR}/../lib/darwin_amd64 -lghostty-vt
import "C"
