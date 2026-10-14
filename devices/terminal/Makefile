GOTOOLCHAIN ?= go1.26.8
export GOTOOLCHAIN
TINYGO ?= tinygo
OUT ?= ../../build/terminal

.PHONY: build build-wasip1 build-tinygo check smoke
build:
	mkdir -p $(OUT)
	go build -o $(OUT)/terminal .
build-wasip1:
	mkdir -p $(OUT)
	GOOS=wasip1 GOARCH=wasm go build -o $(OUT)/terminal-go.wasm .
build-tinygo:
	mkdir -p $(OUT)
	GOROOT="$$(go env GOROOT)" $(TINYGO) build -target=wasip1 -gc=boehm -o $(OUT)/terminal-tinygo.wasm .
check:
	go mod tidy -diff
	go mod verify
	go test .
	go vet .
	GOOS=wasip1 GOARCH=wasm go vet .
	$(MAKE) build build-wasip1
# Drives each build through a pseudo-terminal; needs wasmtime for the WASI builds.
smoke: build build-wasip1 build-tinygo
	python3 -I smoke.py $(OUT)/terminal
	python3 -I smoke.py $(OUT)/terminal-go.wasm
	python3 -I smoke.py $(OUT)/terminal-tinygo.wasm -cols 120 -rows 34
	python3 -I smoke.py $(OUT)/terminal --kitty
	python3 -I smoke.py $(OUT)/terminal-go.wasm --kitty
	python3 -I smoke.py $(OUT)/terminal-tinygo.wasm --kitty
