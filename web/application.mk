GOTOOLCHAIN ?= go1.26.8
export GOTOOLCHAIN
OUT ?= build
MAIN ?= ./cmd/demo
PORT ?= 8080
TINYGO ?= tinygo
.PHONY: assets build build-tinygo test serve
assets:
	go tool tinybubbles-web --assets --force $(OUT)
	@if [ -n "$(SITE)" ]; then cp "$(SITE)/index.html" "$(SITE)/style.css" "$(SITE)/main.js" "$(SITE)/keys.js" "$(OUT)/"; fi
build: assets
	GOOS=js GOARCH=wasm go build -trimpath -o $(OUT)/app.wasm $(MAIN)
build-tinygo: assets
	GOROOT="$$(go env GOROOT)" $(TINYGO) build -target=wasm -stack-size=128KB -o $(OUT)/app.wasm $(MAIN)
	cp "$$($(TINYGO) env TINYGOROOT)/targets/wasm_exec.js" $(OUT)/wasm_exec.js
	python3 $(OUT)/notices.py $(OUT) "$$($(TINYGO) env TINYGOROOT)"
test:
	go mod tidy -diff
	go mod verify
	go test $(TEST_FLAGS) ./...
	go vet ./...
	GOOS=js GOARCH=wasm go vet $(MAIN)
serve: build
	python3 -m http.server $(PORT) --bind 127.0.0.1 --directory $(OUT)
