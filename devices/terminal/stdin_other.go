//go:build !wasip1

package main

import (
	"io"
	"os"
)

var stdinMode = "os.Stdin"

// stdin returns the terminal input. Bubble Tea puts the tty in raw mode itself
// on native targets.
func stdin() io.Reader { return os.Stdin }
