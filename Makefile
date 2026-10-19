TINYGO ?= tinygo

.PHONY: ci check check-format check-modules check-docs test bench bench-tinygo bench-compare tinyhal-wasi firmware snapshot hello settings hello-tinytea hello-preview widgets-preview tinycharts-preview web web-serve web-tinygo
check: check-format check-modules check-docs test
	go build ./...

# Keep preparation and builds sequential, including when invoked with make -j.
ci:
	$(MAKE) check
	$(MAKE) hello-preview snapshot widgets-preview tinycharts-preview
	$(MAKE) firmware TINYGO="$(TINYGO)"
	$(MAKE) tinyhal-wasi TINYGO="$(TINYGO)"
	$(MAKE) -C bubbletea test
	$(MAKE) -C bubbletea firmware TINYGO="$(TINYGO)"
	$(MAKE) -C devices/terminal check
	$(MAKE) -C web test build
	$(MAKE) web-tinygo

web:
	$(MAKE) -C web build

web-tinygo:
	$(MAKE) -C web build-tinygo TINYGO="$(TINYGO)"

web-serve:
	$(MAKE) -C web serve

check-format:
	@files="$$(git ls-files --cached --others --exclude-standard '*.go' | while IFS= read -r file; do if [ -f "$$file" ]; then gofmt -l "$$file"; fi; done)"; if [ -n "$$files" ]; then echo "Run gofmt on:"; echo "$$files"; exit 1; fi

check-modules:
	go mod tidy -diff
	go mod verify

check-docs:
	python3 scripts/check_docs.py

test:
	go test -race -cover ./...
	go vet ./...

bench:
	go test -run=^$$ -bench=. -benchmem . ./tinycharts ./tinyhal
	$(MAKE) bench-compare TINYGO="$(TINYGO)"

bench-tinygo:
	mkdir -p build
	$(TINYGO) build -o build/bench-tinygo-native ./internal/bench
	./build/bench-tinygo-native
	$(TINYGO) build -target=wasip1 -o build/bench-tinygo.wasm ./internal/bench
	wasmtime run build/bench-tinygo.wasm

bench-compare:
	mkdir -p build
	go build -o build/bench-go-native ./internal/bench
	$(TINYGO) build -o build/bench-tinygo-native ./internal/bench
	GOOS=wasip1 GOARCH=wasm go build -o build/bench-go.wasm ./internal/bench
	$(TINYGO) build -target=wasip1 -o build/bench-tinygo.wasm ./internal/bench
	@ls -lh build/bench-go-native build/bench-tinygo-native build/bench-go.wasm build/bench-tinygo.wasm
	./build/bench-go-native
	./build/bench-tinygo-native
	wasmtime run build/bench-go.wasm
	wasmtime run build/bench-tinygo.wasm

tinyhal-wasi:
	mkdir -p build
	$(TINYGO) build -target=wasip1 -o build/tinyhal-demo.wasm ./tinyhal/cmd/tinyhal-demo
	@if command -v wasmtime >/dev/null 2>&1; then \
		wasmtime run build/tinyhal-demo.wasm; \
	else \
		$(TINYGO) run ./tinyhal/cmd/tinyhal-demo; \
	fi

firmware:
	mkdir -p build
	$(TINYGO) build -target=pico -size=short -o build/pico-oled.uf2 ./devices/pico-oled
	$(TINYGO) build -target=pico2 -size=short -o build/pico2-oled.uf2 ./devices/pico-oled
	$(TINYGO) build -target=clue -size=short -o build/clue.uf2 ./devices/clue
	$(TINYGO) build -target=xiao-esp32s3 -size=short -o build/esp32s3-check.elf ./internal/compilecheck
	$(TINYGO) build -target=xiao-esp32s3 -size=short -o build/tinytea-check.elf ./internal/tinyteacheck
	$(TINYGO) build -target=xiao-esp32s3 -size=short -o build/value-check.elf ./internal/valuecheck
	$(TINYGO) build -target=xiao-esp32s3 -size=short -o build/dialog-check.elf ./internal/dialogcheck
	$(TINYGO) build -target=xiao-esp32s3 -size=short -o build/choice-check.elf ./internal/choicecheck
	$(TINYGO) build -target=xiao-esp32s3 -size=short -o build/progress-check.elf ./internal/progresscheck
	$(TINYGO) build -target=xiao-esp32s3 -size=short -o build/status-check.elf ./internal/statuscheck
	$(TINYGO) build -target=xiao-esp32s3 -size=short -o build/viewport-check.elf ./internal/viewportcheck
	$(TINYGO) build -target=xiao-esp32s3 -size=short -o build/layout-check.elf ./internal/layoutcheck
	$(MAKE) hello-tinytea

hello:
	$(MAKE) -C bubbletea firmware TINYGO="$(TINYGO)"

settings:
	$(MAKE) -C bubbletea settings TINYGO="$(TINYGO)"

hello-tinytea:
	mkdir -p build
	$(TINYGO) build -target=./devices/feather-reverse-tft/target.json -size=short -o build/feather-reverse-tft.bin ./devices/feather-reverse-tft

hello-preview:
	go run ./examples/hello-preview

snapshot:
	go run ./examples/snapshot

widgets-preview:
	go run ./examples/widgets-preview
	$(MAKE) -C bubbletea preview

tinycharts-preview:
	go run ./examples/tinycharts-preview
