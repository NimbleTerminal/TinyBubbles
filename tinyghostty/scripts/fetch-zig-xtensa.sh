#!/bin/sh
# Download the Xtensa-capable Zig that builds the ESP32-S3 archive.
#
#   scripts/fetch-zig-xtensa.sh [dir]   # default .tools/zig-xtensa
#
# Upstream Zig 0.17 has no esp32s3 CPU. kassane/zig-espressif-bootstrap builds
# Zig 0.17.0 on Espressif's LLVM fork; this pins one release and checks its
# sha256. The fork is maintained outside Zig and Espressif.
set -eu

tag=0.17.0-xtensa
here=$(cd "$(dirname "$0")/.." && pwd)
dir=${1:-$here/.tools/zig-xtensa}

case $(uname -s)-$(uname -m) in
Darwin-arm64) host=aarch64-macos sum=ad389106a954daba4527f6d176beab17048b1024f264b37adc21003fb5e31d65 ;;
Linux-aarch64 | Linux-arm64) host=aarch64-linux-musl sum=caf295687d16d1cf939d709faeafc56c499f6735394baf42f187ea3ea1db7c6e ;;
Linux-x86_64) host=x86_64-linux-musl sum=aeb04ec3b091c040dd6201f923c3f779ae13c06c9556b3d8a571f489bbe23031 ;;
*) echo "no $tag build for $(uname -s) $(uname -m); build kassane/zig-espressif-bootstrap and set ZIG_XTENSA" >&2; exit 2 ;;
esac
name=zig-relsafe-$host-baseline

if [ -x "$dir/zig" ] && [ "$("$dir/zig" version)" = "$tag" ]; then
	echo "$dir/zig"
	exit 0
fi
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT
echo "fetching $name ($tag)" >&2
curl -fsSL -o "$tmp/zig.tar.xz" "https://github.com/kassane/zig-espressif-bootstrap/releases/download/$tag/$name.tar.xz"
if command -v sha256sum >/dev/null; then
	echo "$sum  $tmp/zig.tar.xz" | sha256sum -c - >&2
else
	echo "$sum  $tmp/zig.tar.xz" | shasum -a 256 -c - >&2
fi
tar -xf "$tmp/zig.tar.xz" -C "$tmp"
rm -rf "$dir"
mkdir -p "$(dirname "$dir")"
mv "$tmp/$name" "$dir"
echo "$dir/zig"
