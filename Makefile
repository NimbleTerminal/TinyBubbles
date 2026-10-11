TINYGO ?= tinygo

.PHONY: ci check check-format check-modules check-docs test firmware snapshot hello hello-tinytea hello-preview
check: check-format check-modules check-docs test
	go build ./...

# Keep preparation and builds sequential, including when invoked with make -j.
ci:
	$(MAKE) check
	$(MAKE) hello-preview snapshot
	$(MAKE) firmware
	$(MAKE) -C bubbletea prepare
	$(MAKE) -C bubbletea test
	$(MAKE) -C bubbletea firmware TINYGO="$(TINYGO)"

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

firmware:
	mkdir -p build
	$(TINYGO) build -target=pico -size=short -o build/pico-oled.uf2 ./devices/pico-oled
	$(TINYGO) build -target=pico2 -size=short -o build/pico2-oled.uf2 ./devices/pico-oled
	$(TINYGO) build -target=clue -size=short -o build/clue.uf2 ./devices/clue
	$(TINYGO) build -target=xiao-esp32s3 -size=short -o build/esp32s3-check.elf ./internal/compilecheck
	$(TINYGO) build -target=xiao-esp32s3 -size=short -o build/tinytea-check.elf ./internal/tinyteacheck
	$(MAKE) hello-tinytea

hello:
	$(MAKE) -C bubbletea firmware TINYGO="$(TINYGO)"

hello-tinytea:
	mkdir -p build
	$(TINYGO) build -target=./devices/feather-reverse-tft/target.json -size=short -o build/feather-reverse-tft.bin ./devices/feather-reverse-tft

hello-preview:
	go run ./examples/hello-preview

snapshot:
	go run ./examples/snapshot
