GOTOOLCHAIN ?= go1.26.8
export GOTOOLCHAIN
TINYGO ?= tinygo
OUT := ../build/web

.PHONY: build build-tinygo assets test serve
build: assets
	GOOS=js GOARCH=wasm go build -trimpath -o $(OUT)/app.wasm ./cmd/demo

assets:
	mkdir -p $(OUT)
	cp site/index.html site/style.css site/main.js site/keys.js site/demos.js $(OUT)/
	go tool booba-assets $(OUT)
	python3 notices.py $(OUT)

build-tinygo: assets
	GOROOT="$$(go env GOROOT)" $(TINYGO) build -target=wasm -stack-size=128KB -o $(OUT)/app.wasm ./cmd/demo
	cp "$$($(TINYGO) env TINYGOROOT)/targets/wasm_exec.js" $(OUT)/wasm_exec.js
	python3 notices.py $(OUT) "$$($(TINYGO) env TINYGOROOT)"

test:
	go mod tidy -diff
	go mod verify
	go test -race ./...
	go vet ./...
	cd site && node --test keys.test.mjs

serve:
	python3 -m http.server 8080 --bind 127.0.0.1 --directory $(OUT)
