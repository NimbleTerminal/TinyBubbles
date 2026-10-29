# tinyghostty needs static libghostty-vt archives in lib/; run make lib first.
# It is not part of the root make ci because it needs Zig and a Ghostty checkout.
GOTOOLCHAIN ?= go1.26.0
export GOTOOLCHAIN
TINYGO ?= tinygo
ZIG ?= zig
GHOSTTY_SRC ?= ../../ghostty
# Xtensa Zig fork for the ESP32-S3; make fetches a pinned release on first use.
ZIG_XTENSA ?= $(CURDIR)/.tools/zig-xtensa/zig
OUT ?= ../build/tinyghostty
TINYGO_BUILD = GOROOT="$$(go env GOROOT)" $(TINYGO) build -scheduler=none -size=short
LIBGHOSTTY = GHOSTTY_SRC="$(GHOSTTY_SRC)" ZIG="$(ZIG)" ZIG_XTENSA="$(ZIG_XTENSA)" scripts/build-libghostty-vt.sh

.PHONY: lib lib-wasm lib-cortexm lib-clue lib-esp32c3 lib-esp32s3 zig-xtensa check test-tinygo preview shell web clue esp32c3 esp32s3
lib:
	$(LIBGHOSTTY) host wasm
# Browser and WASI archive only; CI uses this for the gallery's TinyGhostty tab.
lib-wasm:
	$(LIBGHOSTTY) wasm
# Cortex-M4F (CLUE). Other Cortex-M CPUs: see README.
lib-cortexm lib-clue:
	$(LIBGHOSTTY) cortexm
# ESP32-C3 (RISC-V) with upstream Zig.
lib-esp32c3:
	$(LIBGHOSTTY) esp32c3
# ESP32-S3 (Xtensa) with the Zig fork.
lib-esp32s3: $(ZIG_XTENSA)
	$(LIBGHOSTTY) esp32s3
zig-xtensa $(CURDIR)/.tools/zig-xtensa/zig:
	scripts/fetch-zig-xtensa.sh
check:
	go mod tidy -diff
	go mod verify
	go test ./...
	go vet ./...
# Runs under wasmtime.
test-tinygo:
	GOROOT="$$(go env GOROOT)" $(TINYGO) test -target=wasip1 . ./vt ./render
	mkdir -p $(OUT)
	GOROOT="$$(go env GOROOT)" $(TINYGO) build -target=wasip1 -o $(OUT)/smoke.wasm ./internal/smoke
	wasmtime $(OUT)/smoke.wasm
preview:
	mkdir -p $(OUT)
	go run ./examples/preview -out $(OUT)/tinyghostty.png
# Interactive: your shell on an emulated display. Needs a 240×69 terminal for
# the default Feather preset; pass ARGS="-display oled" for 128×33.
shell:
	mkdir -p $(OUT)
	go build -o $(OUT)/shell ./examples/shell
	$(OUT)/shell $(ARGS)
# Browser page with an in-process shell, served on port 8080.
web:
	$(MAKE) -C web serve
# Link checks only, after the matching lib-* target. TinyGo's stack analysis cannot parse Zig's symbols, so the
# scheduler is disabled; Bubble Tea firmware needs the bubbletea vendor patches.
clue esp32c3 esp32s3:
	mkdir -p $(OUT)
	$(TINYGO_BUILD) -target=$(if $(filter clue,$@),clue,$@-generic) -o $(OUT)/smoke-$@.elf ./internal/smoke
