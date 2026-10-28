#!/bin/sh
# Build static libghostty-vt archives for tinyghostty/vt.
#
#   GHOSTTY_SRC=../ghostty ZIG=zig scripts/build-libghostty-vt.sh [target...]
#
# Targets are "host", "wasm", "clue", or "<zig-triple>:<lib dir>[:<zig cpu>]".
# host installs into lib/<GOOS>_<GOARCH>; wasm builds wasm32-freestanding into
# lib/wasm; clue builds Cortex-M4F into lib/baremetal.
# Headers are copied to lib/include and the licenses of code compiled into the
# archives (Ghostty, uucode, and Wuffs when fetched) to lib/licenses. SIMD is
# disabled so the archives need no libc++.
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

# Zig fetches dependencies into the checkout's zig-pkg or the global cache.
licenses() {
	mkdir -p "$here/lib/licenses"
	cp "$src/LICENSE" "$here/lib/licenses/ghostty.txt"
	for dir in "$src/zig-pkg" "${ZIG_GLOBAL_CACHE_DIR:-$HOME/.cache/zig}/p"; do
		for f in "$dir"/uucode-*/LICENSE.md; do
			[ -f "$f" ] && cp "$f" "$here/lib/licenses/uucode.txt"
		done
		for d in "$dir"/*/; do
			if [ -f "$d/LICENSE-MIT" ] && ls "$d"release/c/wuffs-v*.c >/dev/null 2>&1; then
				cat "$d/LICENSE" "$d/LICENSE-MIT" > "$here/lib/licenses/wuffs.txt"
			fi
		done
	done
	# Zig fetches Wuffs lazily, only for targets that compile it (not wasm).
	[ -f "$here/lib/licenses/uucode.txt" ] || echo "warning: uucode license not found" >&2
}

for t in "$@"; do
	case $t in
	host) build "" "$(go env GOOS)_$(go env GOARCH)" ;;
	wasm) build wasm32-freestanding wasm ;;
	clue) build thumb-freestanding-eabihf baremetal cortex_m4 ;;
	*:*:*) rest=${t#*:}; build "${t%%:*}" "${rest%%:*}" "${rest#*:}" ;;
	*:*) build "${t%%:*}" "${t#*:}" ;;
	*) echo "unknown target $t" >&2; exit 2 ;;
	esac
done
licenses
