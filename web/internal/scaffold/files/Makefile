GOTOOLCHAIN ?= go1.26.8
export GOTOOLCHAIN
SITE := .
MAIN ?= ./cmd/demo
HARNESS_DIR := $(shell go list -m -f '{{.Dir}}' nimbleterminal.dev/tinybubbles/web)
include $(HARNESS_DIR)/application.mk
