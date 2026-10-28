// Package demo runs a small in-process shell inside a tinyghostty terminal for
// the browser page, which has no PTY or real shell. The shell edits a line,
// keeps history, and prints VT output that exercises colors, styles, cursor
// movement, and wide characters.
package demo

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"time"
	"unicode/utf8"
)

const prompt = "\x1b[1;32mtiny\x1b[0m:\x1b[34m~\x1b[0m$ "

// Shell reads keys the terminal widget encodes and writes VT output back.
type Shell struct {
	out     io.Writer
	size    func() (cols, rows int)
	in      []byte
	line    []rune
	history []string
	browse  int // history index while browsing with arrows
	started time.Time
}

// NewShell writes to out; size reports the terminal grid for wrapping output.
func NewShell(out io.Writer, size func() (int, int)) *Shell {
	return &Shell{out: out, size: size, started: time.Now()}
}

// Write queues input. Process handles it after the widget finishes encoding,
// so shell output never re-enters the widget mid-write.
func (s *Shell) Write(p []byte) (int, error) {
	s.in = append(s.in, p...)
	return len(p), nil
}

// Start prints the banner and first prompt.
func (s *Shell) Start() {
	s.print("\x1b[1mtinyghostty\x1b[0m: Ghostty VT in TinyGo WASM.\r\n")
	s.print("Type \x1b[33mhelp\x1b[0m. Nothing leaves your browser.\r\n\r\n")
	s.print(prompt)
}

// Type queues text as if typed, for action buttons.
func (s *Shell) Type(text string) { s.in = append(s.in, text...) }

// Process consumes queued input. Incomplete escape sequences and UTF-8 stay
// queued for the next call.
func (s *Shell) Process() {
	for len(s.in) > 0 {
		b := s.in[0]
		if b == 0x1b {
			n, final := escape(s.in)
			if n == 0 {
				return
			}
			s.in = s.in[n:]
			switch final {
			case 'A':
				s.recall(-1)
			case 'B':
				s.recall(1)
			}
			continue
		}
		if b >= 0x80 && !utf8.FullRune(s.in) {
			return
		}
		r, n := utf8.DecodeRune(s.in)
		s.in = s.in[n:]
		switch r {
		case '\r', '\n':
			s.print("\r\n")
			s.run(string(s.line))
			s.line = s.line[:0]
			s.print(prompt)
		case 0x7f, 0x08:
			if len(s.line) > 0 {
				s.line = s.line[:len(s.line)-1]
				s.print("\b \b")
			}
		case 0x03: // Ctrl+C
			s.line = s.line[:0]
			s.print("^C\r\n" + prompt)
		case 0x0c: // Ctrl+L
			s.print("\x1b[H\x1b[2J" + prompt + string(s.line))
		case 0x15: // Ctrl+U
			s.erase()
		case '\t':
			s.complete()
		default:
			if r >= 0x20 && r != utf8.RuneError {
				s.line = append(s.line, r)
				s.print(string(r))
			}
		}
	}
}

// escape returns the length and final byte of the CSI or SS3 sequence at the
// start of p, or 0 when it is incomplete. A lone ESC is consumed.
func escape(p []byte) (int, byte) {
	if len(p) < 2 {
		return 0, 0
	}
	switch p[1] {
	case 'O':
		if len(p) < 3 {
			return 0, 0
		}
		return 3, p[2]
	case '[':
		for i := 2; i < len(p); i++ {
			if p[i] >= 0x40 && p[i] <= 0x7e {
				return i + 1, p[i]
			}
		}
		return 0, 0
	}
	return 1, 0
}

func (s *Shell) print(text string) { io.WriteString(s.out, text) }

func (s *Shell) erase() {
	for range s.line {
		s.print("\b \b")
	}
	s.line = s.line[:0]
}

func (s *Shell) recall(step int) {
	if len(s.history) == 0 {
		return
	}
	s.browse = min(max(s.browse+step, 0), len(s.history))
	s.erase()
	if s.browse < len(s.history) {
		s.line = append(s.line, []rune(s.history[s.browse])...)
		s.print(string(s.line))
	}
}

func (s *Shell) complete() {
	word := string(s.line)
	var match []string
	for _, name := range names() {
		if strings.HasPrefix(name, word) {
			match = append(match, name)
		}
	}
	switch {
	case len(match) == 1:
		rest := match[0][len(word):] + " "
		s.line = append(s.line, []rune(rest)...)
		s.print(rest)
	case len(match) > 1 && word != "":
		s.print("\r\n" + strings.Join(match, "  ") + "\r\n" + prompt + word)
	}
}

func (s *Shell) run(line string) {
	fields := strings.Fields(line)
	if len(fields) == 0 {
		return
	}
	s.history = append(s.history, line)
	s.browse = len(s.history)
	cmd, ok := commands[fields[0]]
	if !ok {
		s.print("sh: " + fields[0] + ": command not found\r\n")
		return
	}
	cmd.run(s, fields[1:])
}

type command struct {
	help string
	run  func(*Shell, []string)
}

var commands map[string]command

func init() {
	commands = map[string]command{
		"help":    {"list commands", (*Shell).help},
		"clear":   {"clear the screen", func(s *Shell, _ []string) { s.print("\x1b[H\x1b[2J") }},
		"colors":  {"ANSI, 256-color, and true color", (*Shell).colors},
		"styles":  {"bold, faint, reverse, hidden", (*Shell).styles},
		"box":     {"draw with cursor movement", (*Shell).box},
		"ls":      {"list files", (*Shell).ls},
		"cat":     {"print a file", (*Shell).cat},
		"echo":    {"print arguments", func(s *Shell, a []string) { s.print(strings.Join(a, " ") + "\r\n") }},
		"wide":    {"wide characters", func(s *Shell, _ []string) { s.print("ascii | 界 | wide cells\r\n") }},
		"size":    {"terminal size", (*Shell).sizeCmd},
		"uptime":  {"time since load", (*Shell).uptime},
		"history": {"previous commands", (*Shell).historyCmd},
		"about":   {"what this is", (*Shell).about},
	}
}

func names() []string {
	out := make([]string, 0, len(commands))
	for name := range commands {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

func (s *Shell) help(_ []string) {
	for _, name := range names() {
		s.print(fmt.Sprintf("\x1b[33m%-8s\x1b[0m %s\r\n", name, commands[name].help))
	}
}

func (s *Shell) colors(_ []string) {
	for i, name := range []string{"black", "red", "green", "yellow", "blue", "magenta", "cyan", "white"} {
		s.print(fmt.Sprintf("\x1b[3%dm%s\x1b[0m ", i, name))
		if i == 3 {
			s.print("\r\n")
		}
	}
	// Swatches use colored glyphs: theme styles have black backgrounds, so
	// colored backgrounds would snap to black.
	cols, _ := s.size()
	n := max(8, min(32, cols-6))
	s.print("\r\n256: ")
	for i := 0; i < n; i++ {
		s.print(fmt.Sprintf("\x1b[38;5;%dm#", 16+i*216/n))
	}
	s.print("\x1b[0m\r\nrgb: ")
	for i := 0; i < n; i++ {
		s.print(fmt.Sprintf("\x1b[38;2;%d;%d;%dm#", 255*i/n, 200-150*i/n, 255-255*i/n))
	}
	s.print("\x1b[0m")
	s.print("\r\n\x1b[2mColors snap to eight theme styles.\x1b[0m\r\n")
}

func (s *Shell) styles(_ []string) {
	s.print("\x1b[1mbold\x1b[0m \x1b[2mfaint\x1b[0m \x1b[7mreverse\x1b[0m [\x1b[8mhidden\x1b[0m] \x1b[4munderline\x1b[0m\r\n")
	s.print("\x1b[2mUnderline has no cell style, so it draws plain.\x1b[0m\r\n")
}

func (s *Shell) box(_ []string) {
	cols, _ := s.size()
	w := max(6, min(24, cols-2))
	s.print("+" + strings.Repeat("-", w-2) + "+\r\n")
	for i := 0; i < 3; i++ {
		s.print("|" + strings.Repeat(" ", w-2) + "|\r\n")
	}
	s.print("+" + strings.Repeat("-", w-2) + "+")
	// Return to the box's middle row with relative moves, then label it.
	s.print(fmt.Sprintf("\x1b[2A\r\x1b[%dC\x1b[7m CUP \x1b[0m", w/2-3))
	s.print("\x1b[2B\r\n")
}

var files = map[string]string{
	"README.md": "tinyghostty embeds Ghostty's VT parser in\r\nTinyBubbles. Each cell keeps one rune and\r\none of eight theme styles.\r\n",
	"hello.go":  "\x1b[34mpackage\x1b[0m main\r\n\r\n\x1b[34mfunc\x1b[0m main() {\r\n\tprintln(\x1b[32m\"hello\"\x1b[0m)\r\n}\r\n",
	"notes.txt": "Buttons below the display run commands.\r\nCtrl+C, Ctrl+L, Ctrl+U, Tab, and arrows work.\r\n",
}

func (s *Shell) ls(_ []string) {
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	s.print("\x1b[1;34mbin\x1b[0m  ")
	for _, name := range names {
		color := "0"
		if strings.HasSuffix(name, ".go") {
			color = "32"
		}
		s.print("\x1b[" + color + "m" + name + "\x1b[0m  ")
	}
	s.print("\r\n")
}

func (s *Shell) cat(args []string) {
	if len(args) == 0 {
		s.print("usage: cat FILE (try ls)\r\n")
		return
	}
	for _, name := range args {
		text, ok := files[name]
		if !ok {
			s.print("cat: " + name + ": no such file\r\n")
			continue
		}
		s.print(text)
	}
}

func (s *Shell) sizeCmd(_ []string) {
	cols, rows := s.size()
	s.print(fmt.Sprintf("%d×%d cells\r\n", cols, rows))
}

func (s *Shell) uptime(_ []string) {
	s.print("up " + time.Since(s.started).Round(time.Second).String() + "\r\n")
}

func (s *Shell) historyCmd(_ []string) {
	for i, line := range s.history {
		s.print(fmt.Sprintf("%3d  %s\r\n", i+1, line))
	}
}

func (s *Shell) about(_ []string) {
	s.print("libghostty-vt parses this output inside a\r\nTinyGo WASM binary. tinyghostty copies its\r\nscreen into a TinyBubbles cell buffer, which\r\nthe display renderer draws on the canvas.\r\n")
}
