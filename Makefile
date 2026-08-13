SHELL := /bin/bash

GENERATOR := ./cmd/whoami
OUT_DIR := ./build
PORT ?= 8000

GOLANGCI_VERSION := v2.12.2
# Revision the adoption gate compares against: only code newer than this is
# required to be clean while the standing findings are worked off.
LINT_BASE ?= origin/main

.PHONY: build run test lint lint-new format clean

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

## lint: golangci-lint over the whole tree, then the text-level review checks
lint: format
	@command -v golangci-lint >/dev/null 2>&1 || { \
		echo "golangci-lint not found — install $(GOLANGCI_VERSION): https://golangci-lint.run/docs/welcome/install/local/"; \
		exit 1; \
	}
	CGO_ENABLED=0 golangci-lint run ./...
	@scripts/lint-checks.sh

## lint-new: lint only code changed since $(LINT_BASE); this is the mergeable gate
lint-new: format
	@command -v golangci-lint >/dev/null 2>&1 || { \
		echo "golangci-lint not found — install $(GOLANGCI_VERSION): https://golangci-lint.run/docs/welcome/install/local/"; \
		exit 1; \
	}
	CGO_ENABLED=0 golangci-lint run --new-from-rev=$(LINT_BASE) ./...
	@scripts/lint-checks.sh

## format: gofmt the tree
format:
	go fmt ./...

## clean: remove the generated site and tidy modules
clean:
	rm -rf $(OUT_DIR)
	go mod tidy
