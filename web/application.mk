GOTOOLCHAIN ?= go1.26.8
export GOTOOLCHAIN
OUT ?= build
MAIN ?= ./cmd/demo
PORT ?= 8080
TINYGO ?= tinygo
# Optional: build tags for TinyGo, and license files for code that is not a Go
# module (such as a linked C library) to append to the notices.
TINYGO_TAGS ?=
EXTRA_NOTICES ?=
.PHONY: assets build build-tinygo test serve serve-tinygo
assets:
	go tool tinybubbles-web --assets --force $(OUT)
	@if [ -n "$(SITE)" ]; then cp "$(SITE)/index.html" "$(SITE)/style.css" "$(OUT)/"; fi
build: assets
	GOOS=js GOARCH=wasm go build -trimpath -o $(OUT)/app.wasm $(MAIN)
build-tinygo: assets
	GOROOT="$$(go env GOROOT)" $(TINYGO) build -target=wasm -gc=boehm -stack-size=128KB $(if $(TINYGO_TAGS),-tags=$(TINYGO_TAGS)) -o $(OUT)/app.wasm $(MAIN)
	cp "$$($(TINYGO) env TINYGOROOT)/targets/wasm_exec.js" $(OUT)/wasm_exec.js
	python3 $(OUT)/notices.py $(OUT) "$$($(TINYGO) env TINYGOROOT)"
	@for f in $(EXTRA_NOTICES); do \
		printf '\n%s\nBundled native code: %s\n%s\n' "$$(printf '=%.0s' $$(seq 72))" "$$(basename $$f .txt)" "$$(printf '=%.0s' $$(seq 72))"; \
		cat "$$f"; \
	done >> $(OUT)/THIRD_PARTY_NOTICES.txt
test:
	go mod tidy -diff
	go mod verify
	go test $(TEST_FLAGS) ./...
	go vet ./...
	GOOS=js GOARCH=wasm go vet $(MAIN)
serve: build
	python3 -m http.server $(PORT) --bind 127.0.0.1 --directory $(OUT)
serve-tinygo: build-tinygo
	python3 -m http.server $(PORT) --bind 127.0.0.1 --directory $(OUT)
