//go:build !tinygo && darwin && arm64

package vt

// #cgo LDFLAGS: -L${SRCDIR}/../lib/darwin_arm64 -lghostty-vt
import "C"
