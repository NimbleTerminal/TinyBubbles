//go:build wasip1

package main

import (
	"io"
	"os"
	"time"
	"unsafe"
)

// WASI preview 1 runtimes block the whole single-threaded Go/TinyGo program
// while fd_read waits on stdin, which starves timers. Poll readiness with a
// zero timeout first and sleep between polls so the scheduler keeps running.

//go:wasmimport wasi_snapshot_preview1 poll_oneoff
//go:noescape
func pollOneoff(in unsafe.Pointer, out unsafe.Pointer, nsubs uint32, nevents unsafe.Pointer) uint32

type subscription struct { // 48 bytes
	userdata  uint64
	tag       uint8 // 0 clock, 1 fd_read, 2 fd_write
	_         [7]byte
	arg0      uint32 // fd for fd_read; clock id for clock
	_         [4]byte
	timeout   uint64
	precision uint64
	flags     uint16
	_         [6]byte
}

type event struct { // 32 bytes
	userdata uint64
	errno    uint16
	typ      uint8
	_        [5]byte
	nbytes   uint64
	flags    uint16 // 1 = hangup
	_        [6]byte
}

var stdinMode = "poll_oneoff"

type polledStdin struct{}

func stdin() io.Reader { return polledStdin{} }

func (polledStdin) Read(p []byte) (int, error) {
	for {
		ready, hangup := stdinReady()
		if hangup {
			return 0, io.EOF
		}
		if ready {
			return os.Stdin.Read(p)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func stdinReady() (ready, hangup bool) {
	subs := [2]subscription{
		{userdata: 1, tag: 1, arg0: 0},
		{userdata: 2, tag: 0, arg0: 1, timeout: 0}, // monotonic clock, zero timeout: pure poll
	}
	var events [2]event
	var n uint32
	if errno := pollOneoff(unsafe.Pointer(&subs[0]), unsafe.Pointer(&events[0]), 2, unsafe.Pointer(&n)); errno != 0 {
		return false, false
	}
	for _, e := range events[:n] {
		if e.userdata == 1 {
			if e.errno != 0 || e.flags&1 != 0 {
				return false, true
			}
			return true, false
		}
	}
	return false, false
}
