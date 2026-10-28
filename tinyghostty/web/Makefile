# The page links libghostty-vt through cgo, which Go's js/wasm lacks, so only
# TinyGo builds it. Run make -C .. lib first.
GOTOOLCHAIN ?= go1.26.8
export GOTOOLCHAIN
TINYGO ?= tinygo
OUT ?= ../../build/tinyghostty-web
MAIN := ./cmd/demo
PORT ?= 8080

.PHONY: assets build test serve
assets:
	go tool tinybubbles-web --assets --force $(OUT)
build: assets
	GOROOT="$$(go env GOROOT)" $(TINYGO) build -target=wasm -gc=boehm -stack-size=128KB -o $(OUT)/app.wasm $(MAIN)
	cp "$$($(TINYGO) env TINYGOROOT)/targets/wasm_exec.js" $(OUT)/wasm_exec.js
	python3 $(OUT)/notices.py $(OUT) "$$($(TINYGO) env TINYGOROOT)"
	@# libghostty-vt is C, not a Go module, so add the code compiled into it.
	@for f in ../lib/licenses/*.txt; do \
		printf '\n%s\nlibghostty-vt bundled: %s\n%s\n' "$$(printf '=%.0s' $$(seq 72))" "$$(basename $$f .txt)" "$$(printf '=%.0s' $$(seq 72))"; \
		cat "$$f"; \
	done >> $(OUT)/THIRD_PARTY_NOTICES.txt
test:
	go mod tidy -diff
	go mod verify
	go test ./...
	go vet ./...
serve: build
	python3 -m http.server $(PORT) --bind 127.0.0.1 --directory $(OUT)
