# tinyghostty needs static libghostty-vt archives in lib/; run make lib first.
# It is not part of the root make ci because it needs Zig and a Ghostty checkout.
GOTOOLCHAIN ?= go1.26.0
export GOTOOLCHAIN
TINYGO ?= tinygo
ZIG ?= zig
GHOSTTY_SRC ?= ../../ghostty
OUT ?= ../build/tinyghostty

.PHONY: lib lib-clue check test-tinygo preview clue
lib:
	GHOSTTY_SRC="$(GHOSTTY_SRC)" ZIG="$(ZIG)" scripts/build-libghostty-vt.sh host wasm
lib-clue:
	GHOSTTY_SRC="$(GHOSTTY_SRC)" ZIG="$(ZIG)" scripts/build-libghostty-vt.sh clue
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
# Link check only. TinyGo's stack analysis cannot parse Zig's symbols, so the
# scheduler is disabled; Bubble Tea firmware needs the bubbletea vendor patches.
clue:
	mkdir -p $(OUT)
	GOROOT="$$(go env GOROOT)" $(TINYGO) build -target=clue -scheduler=none -size=short -o $(OUT)/smoke-clue.elf ./internal/smoke
