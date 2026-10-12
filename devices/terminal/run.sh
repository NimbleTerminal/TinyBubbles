#!/bin/sh
# Run the hello demo in this terminal: run.sh [native|wasip1|tinygo] [-glyph]
# Build first with make build, build-wasip1, or build-tinygo in this directory.
# Keys: Enter=Pop  Right=Color  Esc=Pause  k=toggle Kitty/half-blocks  q or Ctrl-C=Quit
set -e
here="$(cd "$(dirname "$0")" && pwd)"
out="${OUT:-$here/../../build/terminal}"
which="native"; mode=""
for arg in "$@"; do case "$arg" in native|wasip1|tinygo) which="$arg";; -glyph) mode="-glyph";; esac; done
# stty asks the tty driver; tput fails when TERM (for example xterm-ghostty) has no local terminfo.
set -- $(stty size </dev/tty 2>/dev/null || echo "0 0"); lines=$1; cols=$2
[ "${COLUMNS:-0}" -gt 0 ] && cols=$COLUMNS
[ "${LINES:-0}" -gt 0 ] && lines=$LINES
[ "$cols" -le 0 ] && { cols=80; lines=24; }
# One column per pixel and two pixel rows per text row needs 240x68 cells. Half-block
# mode builds a 150 KB styled string per frame, which TinyGo cannot do at 10 Hz at
# full size; Kitty mode sends a small bitmap instead, so it keeps full size everywhere.
if [ "$cols" -ge 240 ] && [ "$lines" -ge 70 ] && { [ "$which" != tinygo ] || [ -z "$mode" ]; }; then c=240; r=68; else c=120; r=34; fi
[ "$cols" -lt 120 ] && { echo "terminal is $cols columns; need at least 120" >&2; exit 1; }
case "$which" in
  native) exec "$out/terminal" -cols "$c" -rows "$r" $mode ;;
  wasip1) wasm="$out/terminal-go.wasm" ;;
  tinygo) wasm="$out/terminal-tinygo.wasm" ;;
esac
saved=$(stty -g)
trap 'stty "$saved"' EXIT INT TERM
# Raw input only. Under WASI Bubble Tea cannot see the tty, assumes a cooked output
# side, and emits bare line feeds that the driver must turn into CR+LF, so keep opost.
stty -icanon -echo -isig -icrnl -ixon -ixoff
wasmtime run -S inherit-env=y "$wasm" -cols "$c" -rows "$r" $mode
