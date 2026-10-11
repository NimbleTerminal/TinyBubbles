TINYGO ?= tinygo
GOTOOLCHAIN ?= go1.26.0
export GOTOOLCHAIN

.PHONY: prepare test firmware
prepare:
	go mod vendor
	python3 prepare.py

test:
	go test -race ./...

firmware: prepare
	mkdir -p ../build
	GOROOT="$$(go env GOROOT)" $(TINYGO) build -target=../devices/feather-reverse-tft/target.json -size=short -o ../build/bubbletea-feather.bin ./devices/feather
