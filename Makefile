SHELL := /bin/bash

GENERATOR := ./cmd/whoami
OUT_DIR := ./build
PORT ?= 8000

.PHONY: build run test format clean

## build: run the generator, writing the static site into ./build
build: format
	CGO_ENABLED=0 go vet ./...
	CGO_ENABLED=0 go run $(GENERATOR)

## run: build the site, then serve ./build locally for preview
run: build
	@echo "serving $(OUT_DIR) on http://localhost:$(PORT)"
	cd $(OUT_DIR) && python3 -m http.server $(PORT)

## test: format, vet and run the test suite
test: format
	CGO_ENABLED=0 go vet ./...
	CGO_ENABLED=0 go test ./...

## format: gofmt the tree
format:
	go fmt ./...

## clean: remove the generated site and tidy modules
clean:
	rm -rf $(OUT_DIR)
	go mod tidy
