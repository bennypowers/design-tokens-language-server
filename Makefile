# Makefile for Design Tokens Language Server
# Thin wrapper around bennypowers.dev/asimonim/lsp

SHELL := /bin/bash
BINARY_NAME := design-tokens-language-server
DIST_DIR := dist/bin

.PHONY: all build test lint install clean release help

all: build

## Build native binary
build:
	@mkdir -p $(DIST_DIR)
	go build -ldflags="-s -w" -o $(DIST_DIR)/$(BINARY_NAME) ./cmd/design-tokens-language-server
	@echo "Built: $(DIST_DIR)/$(BINARY_NAME)"

## Install to ~/.local/bin
install: build
	@mkdir -p ~/.local/bin/
	cp $(DIST_DIR)/$(BINARY_NAME) ~/.local/bin/
	@echo "Installed to ~/.local/bin/$(BINARY_NAME)"

## Run tests
test:
	go test -v ./...

## Run linter
lint:
	go vet ./...

## Clean build artifacts
clean:
	rm -rf dist/ $(BINARY_NAME)

## Release: update asimonim dep, tag, and create GitHub release
## Usage: make release ASIMONIM_VERSION=v0.2.0
release:
	@if [ -z "$(ASIMONIM_VERSION)" ]; then \
		echo "Usage: make release ASIMONIM_VERSION=v0.2.0"; \
		exit 1; \
	fi
	go get bennypowers.dev/asimonim@$(ASIMONIM_VERSION)
	go mod tidy
	git add go.mod go.sum
	git commit -m "chore: update asimonim to $(ASIMONIM_VERSION)"
	git push
	gh release create $(ASIMONIM_VERSION) --generate-notes

## Help
help:
	@echo "Design Tokens Language Server (thin wrapper around asimonim)"
	@echo ""
	@echo "Targets:"
	@echo "  make build                              Build native binary"
	@echo "  make install                             Install to ~/.local/bin"
	@echo "  make test                                Run tests"
	@echo "  make lint                                Run linter"
	@echo "  make clean                               Clean build artifacts"
	@echo "  make release ASIMONIM_VERSION=v0.2.0     Sync with asimonim release"
