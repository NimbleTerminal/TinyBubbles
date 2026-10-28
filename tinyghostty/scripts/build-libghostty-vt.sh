#!/bin/sh
# Build static libghostty-vt archives for tinyghostty/vt.
#
#   GHOSTTY_SRC=../ghostty ZIG=zig scripts/build-libghostty-vt.sh [target...]
#
# Targets are "host", "wasm", "clue", or "<zig-triple>:<lib dir>[:<zig cpu>]".
# host installs into lib/<GOOS>_<GOARCH>; wasm builds wasm32-freestanding into
# lib/wasip1_wasm; clue builds Cortex-M4F into lib/baremetal.
# Headers are copied to lib/include. SIMD is disabled so archives need no libc++.
set -eu

here=$(cd "$(dirname "$0")/.." && pwd)
: "${GHOSTTY_SRC:?set GHOSTTY_SRC to a Ghostty checkout (v1.4.0 or later)}"
ZIG=${ZIG:-zig}
OPTIMIZE=${OPTIMIZE:-ReleaseSmall}
src=$(cd "$GHOSTTY_SRC" && pwd)
[ $# -gt 0 ] || set -- host wasm

build() { # zig-target lib-dir [cpu]
	prefix="$here/lib/.build/$2"
	echo "libghostty-vt: ${1:-native} -> lib/$2"
	(cd "$src" && "$ZIG" build -Demit-lib-vt -Dsimd=false -Doptimize="$OPTIMIZE" \
		${1:+-Dtarget="$1"} ${3:+-Dcpu="$3"} --prefix "$prefix")
	mkdir -p "$here/lib/$2"
	cp "$prefix/lib/libghostty-vt.a" "$here/lib/$2/"
	# Wasm builds install no headers; they are target-independent.
	if [ -d "$prefix/include" ]; then
		rm -rf "$here/lib/include"
		cp -R "$prefix/include" "$here/lib/include"
	elif [ ! -d "$here/lib/include" ]; then
		cp -R "$src/include" "$here/lib/include"
	fi
}

for t in "$@"; do
	case $t in
	host) build "" "$(go env GOOS)_$(go env GOARCH)" ;;
	wasm) build wasm32-freestanding wasip1_wasm ;;
	clue) build thumb-freestanding-eabihf baremetal cortex_m4 ;;
	*:*:*) rest=${t#*:}; build "${t%%:*}" "${rest%%:*}" "${rest#*:}" ;;
	*:*) build "${t%%:*}" "${t#*:}" ;;
	*) echo "unknown target $t" >&2; exit 2 ;;
	esac
done
