//go:build !tinygo

package vt

// #cgo CFLAGS: -I${SRCDIR}/../lib/include -DGHOSTTY_STATIC
import "C"
