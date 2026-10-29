//go:build tinygo && esp32s3

// Upstream Zig cannot target the ESP32-S3; make lib-esp32s3 uses an Xtensa fork.

package vt

// #cgo CFLAGS: -DTGVT_REAL_MALLOC
// #cgo LDFLAGS: -L../lib/esp32s3 -lghostty-vt
import "C"
