GOTOOLCHAIN ?= go1.26.8
export GOTOOLCHAIN
TINYGO ?= tinygo
.PHONY: build build-tinygo assets test serve
build build-tinygo assets serve:
	$(MAKE) -C hello $@ TINYGO="$(TINYGO)"
test:
	go mod tidy -diff
	go mod verify
	go test -race ./...
	go vet ./...
	node --test internal/scaffold/keys.test.mjs
	$(MAKE) -C hello test
