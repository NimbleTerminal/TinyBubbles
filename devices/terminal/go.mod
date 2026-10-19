module nimbleterminal.dev/tinybubbles/devices/terminal

go 1.26.8

require (
	charm.land/bubbles/v2 v2.2.1
	charm.land/bubbletea/v2 v2.1.0
	github.com/NimbleMarkets/go-booba v0.10.0
	github.com/charmbracelet/colorprofile v0.4.3
	nimbleterminal.dev/ntpicture v0.1.0
	nimbleterminal.dev/tinybubbles v0.2.0
	nimbleterminal.dev/tinybubbles/bubbletea v0.2.0
)

require (
	charm.land/lipgloss/v2 v2.0.6 // indirect
	github.com/charmbracelet/ultraviolet v0.0.0-20261008173134-6b8d4baf91b4 // indirect
	github.com/charmbracelet/x/ansi v0.11.9 // indirect
	github.com/charmbracelet/x/term v0.2.2 // indirect
	github.com/charmbracelet/x/termios v0.1.1 // indirect
	github.com/charmbracelet/x/windows v0.2.2 // indirect
	github.com/clipperhouse/displaywidth v0.11.0 // indirect
	github.com/clipperhouse/uax29/v2 v2.7.0 // indirect
	github.com/google/shlex v0.0.0-20191202100458-e7afc7fbc510 // indirect
	github.com/lucasb-eyer/go-colorful v1.4.1 // indirect
	github.com/mattn/go-runewidth v0.0.31 // indirect
	github.com/muesli/cancelreader v0.2.2 // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	github.com/xo/terminfo v1.2.0 // indirect
	golang.org/x/image v0.47.0 // indirect
	golang.org/x/sync v0.24.0 // indirect
	golang.org/x/sys v0.49.0 // indirect
	tinygo.org/x/drivers v0.36.0 // indirect
	tinygo.org/x/tinyfont v0.7.0 // indirect
)

replace nimbleterminal.dev/tinybubbles => ../..

replace nimbleterminal.dev/tinybubbles/bubbletea => ../../bubbletea

// Upstream v2.1 has no tty or signal files for wasip1. The fork adds stubs;
// web/go.mod pins the same commit.
replace charm.land/bubbletea/v2 => github.com/neomantra/bubbletea/v2 v2.0.0-20261009201241-c397c32a5074
