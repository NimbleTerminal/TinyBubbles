TINYGO ?= tinygo

.PHONY: ci check check-format check-modules check-docs test firmware snapshot hello hello-preview
check: check-format check-modules check-docs test
	go build ./...

# Keep preparation and builds sequential, including when invoked with make -j.
ci:
	$(MAKE) check
	$(MAKE) hello-preview snapshot
	$(MAKE) firmware
	$(MAKE) -C experimental/bubbletea prepare
	$(MAKE) -C experimental/bubbletea test
	$(MAKE) -C experimental/bubbletea firmware TINYGO="$(TINYGO)"

check-format:
	@files="$$(gofmt -l $$(git ls-files '*.go'))"; if [ -n "$$files" ]; then echo "Run gofmt on:"; echo "$$files"; exit 1; fi

check-modules:
	go mod tidy -diff
	go mod verify

check-docs:
	python3 scripts/check_docs.py

test:
	go test -race -cover ./...
	go vet ./...

firmware:
	mkdir -p build
	$(TINYGO) build -target=pico -size=short -o build/pico-oled.uf2 ./devices/pico-oled
	$(TINYGO) build -target=pico2 -size=short -o build/pico2-oled.uf2 ./devices/pico-oled
	$(TINYGO) build -target=clue -size=short -o build/clue.uf2 ./devices/clue
	$(TINYGO) build -target=xiao-esp32s3 -size=short -o build/esp32s3-check.elf ./internal/compilecheck
	$(MAKE) hello

hello:
	mkdir -p build
	$(TINYGO) build -target=./devices/feather-reverse-tft/target.json -size=short -o build/feather-reverse-tft.bin ./devices/feather-reverse-tft

hello-preview:
	go run ./examples/hello-preview

snapshot:
	go run ./examples/snapshot
