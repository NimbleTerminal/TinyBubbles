GOTOOLCHAIN ?= go1.26.8
export GOTOOLCHAIN
OUT ?= ../../build/web
TEST_FLAGS := -race
MAIN := ./cmd/demo
# The Ghostty terminal tab needs libghostty-vt (make -C ../../tinyghostty lib)
# and TinyGo; without the archive the gallery builds without it.
GHOSTTY_LIB := ../../tinyghostty/lib
ifneq ($(wildcard $(GHOSTTY_LIB)/wasm/libghostty-vt.a),)
TINYGO_TAGS := tinyghostty
EXTRA_NOTICES := $(wildcard $(GHOSTTY_LIB)/licenses/*.txt)
endif
include ../application.mk
