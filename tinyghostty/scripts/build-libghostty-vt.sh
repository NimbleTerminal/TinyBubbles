#!/bin/sh
# Build static libghostty-vt archives for tinyghostty/vt.
#
#   GHOSTTY_SRC=../ghostty ZIG=zig scripts/build-libghostty-vt.sh [target...]
#
# Targets are "host", "wasm", "cortexm" (alias "clue"), "esp32c3", "esp32s3",
# or "<zig-triple>:<lib dir>[:<zig cpu>]". Each lib dir matches a vt/cgo_*.go
# file: host installs into lib/<GOOS>_<GOARCH>, wasm builds wasm32-freestanding
# into lib/wasm, cortexm builds Cortex-M4F into lib/cortexm, and esp32c3 builds
# RV32IMC into lib/esp32c3. esp32s3 needs an Xtensa Zig fork (set ZIG_XTENSA
# or run scripts/fetch-zig-xtensa.sh) and builds through scripts/xtensa, which
# works around two of the fork's code generation bugs.
# Headers are copied to lib/include and the licenses of code compiled into the
# archives (Ghostty, uucode, and Wuffs when fetched) to lib/licenses. SIMD is
# disabled so the archives need no libc++.
set -eu

here=$(cd "$(dirname "$0")/.." && pwd)
: "${GHOSTTY_SRC:?set GHOSTTY_SRC to a Ghostty checkout (v1.4.0 or later)}"
ZIG=${ZIG:-zig}
OPTIMIZE=${OPTIMIZE:-ReleaseSmall}
ZIG_XTENSA=${ZIG_XTENSA:-$here/.tools/zig-xtensa/zig}
src=$(cd "$GHOSTTY_SRC" && pwd)
# Zig runs from other directories, so make relative tool paths absolute.
abs() { case $1 in /*) echo "$1" ;; */*) echo "$PWD/$1" ;; *) echo "$1" ;; esac; }
ZIG=$(abs "$ZIG") ZIG_XTENSA=$(abs "$ZIG_XTENSA")
[ $# -gt 0 ] || set -- host wasm

build() { # zig-target lib-dir [cpu [zig [zig build flags...]]]
	target=$1 dir=$2 cpu=${3:-} zig=${4:-$ZIG}
	shift $(($# < 4 ? $# : 4))
	prefix="$here/lib/.build/$dir"
	echo "libghostty-vt: ${target:-native} -> lib/$dir"
	(cd "$src" && "$zig" build -Demit-lib-vt -Dsimd=false -Doptimize="$OPTIMIZE" \
		${target:+-Dtarget="$target"} ${cpu:+-Dcpu="$cpu"} "$@" --prefix "$prefix")
	mkdir -p "$here/lib/$dir"
	cp "$prefix/lib/libghostty-vt.a" "$here/lib/$dir/"
	# Wasm builds install no headers; they are target-independent.
	if [ -d "$prefix/include" ]; then
		rm -rf "$here/lib/include"
		cp -R "$prefix/include" "$here/lib/include"
	elif [ ! -d "$here/lib/include" ]; then
		cp -R "$src/include" "$here/lib/include"
	fi
}

xtensa() { # lib-dir cpu
	wrap="$here/lib/.build/xtensa-$1"
	echo "libghostty-vt: xtensa-freestanding-none ($2) -> lib/$1"
	mkdir -p "$wrap"
	cp "$here/scripts/xtensa/build.zig" "$here/scripts/xtensa/build.zig.zon" "$wrap/"
	ln -sfn "$src" "$wrap/ghostty"
	(cd "$wrap" && "$ZIG_XTENSA" build -Dtarget=xtensa-freestanding-none -Dcpu="$2" \
		-Doptimize="$OPTIMIZE" --prefix "$wrap/out")
	mkdir -p "$here/lib/$1"
	cp "$wrap/out/lib/libghostty-vt-static.a" "$here/lib/$1/libghostty-vt.a"
	[ -d "$here/lib/include" ] || cp -R "$src/include" "$here/lib/include"
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
	cortexm | clue) build thumb-freestanding-eabihf cortexm cortex_m4 ;;
	esp32c3) build riscv32-freestanding-none esp32c3 'generic_rv32+m+c+zmmul-relax' ;;
	esp32s3)
		if ! "$ZIG_XTENSA" targets 2>/dev/null | grep -q '\.esp32s3'; then
			echo "esp32s3: $ZIG_XTENSA is not an Xtensa Zig; run scripts/fetch-zig-xtensa.sh or set ZIG_XTENSA" >&2
			exit 2
		fi
		xtensa esp32s3 esp32s3
		;;
	*:*:*) rest=${t#*:}; build "${t%%:*}" "${rest%%:*}" "${rest#*:}" ;;
	*:*) build "${t%%:*}" "${t#*:}" ;;
	*) echo "unknown target $t" >&2; exit 2 ;;
	esac
done
licenses
