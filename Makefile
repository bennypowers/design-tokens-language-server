# Makefile for Design Tokens Language Server (Go implementation)
# Based on CEM's proven CGO cross-compilation setup

SHELL := /bin/bash
WINDOWS_CC_IMAGE := dtls-windows-cc-image
BINARY_NAME := design-tokens-language-server
DIST_DIR := dist/bin

# Go build flags
GO_BUILD_FLAGS := -ldflags="-s -w"

.PHONY: all build build-all test test-coverage install clean windows-x64 windows-arm64 linux-x64 linux-arm64 darwin-x64 darwin-arm64 build-windows-cc-image rebuild-windows-cc-image

all: build

## Clean build artifacts
clean:
	rm -rf dist/ coverage.out

## Build native binary
build:
	@mkdir -p $(DIST_DIR)
	go build $(GO_BUILD_FLAGS) -o $(DIST_DIR)/$(BINARY_NAME) ./cmd/design-tokens-lsp
	@echo "Built native binary: $(DIST_DIR)/$(BINARY_NAME)"

## Install to ~/.local/bin (for local development)
install: build
	@mkdir -p ~/.local/bin/
	cp $(DIST_DIR)/$(BINARY_NAME) ~/.local/bin/
	@echo "Installed to ~/.local/bin/$(BINARY_NAME)"

## Run tests
test:
	go test -v -race ./...

## Run tests with coverage
test-coverage:
	go test -v -coverprofile=coverage.out -covermode=atomic -coverpkg=./internal/...,./cmd/... ./...
	@echo "Coverage report:"
	@go tool cover -func=coverage.out

## Show coverage in browser
show-coverage: test-coverage
	go tool cover -html=coverage.out

## Build all platform binaries (requires cross-compilation toolchains)
build-all: linux-x64 linux-arm64 darwin-x64 darwin-arm64 windows-x64
	@echo "All binaries built successfully!"
	@ls -lh $(DIST_DIR)/

## Linux x86_64 (CGO cross-compilation)
linux-x64:
	@mkdir -p $(DIST_DIR)
	CGO_ENABLED=1 GOOS=linux GOARCH=amd64 \
		go build $(GO_BUILD_FLAGS) \
		-o $(DIST_DIR)/$(BINARY_NAME)-x86_64-unknown-linux-gnu \
		./cmd/design-tokens-lsp
	@echo "Built: $(DIST_DIR)/$(BINARY_NAME)-x86_64-unknown-linux-gnu"

## Linux ARM64 (CGO cross-compilation - requires gcc-aarch64-linux-gnu)
linux-arm64:
	@mkdir -p $(DIST_DIR)
	@if ! command -v aarch64-linux-gnu-gcc &> /dev/null; then \
		echo "Error: aarch64-linux-gnu-gcc not found. Install gcc-aarch64-linux-gnu"; \
		exit 1; \
	fi
	CGO_ENABLED=1 GOOS=linux GOARCH=arm64 \
		CC=aarch64-linux-gnu-gcc \
		go build $(GO_BUILD_FLAGS) \
		-o $(DIST_DIR)/$(BINARY_NAME)-aarch64-unknown-linux-gnu \
		./cmd/design-tokens-lsp
	@echo "Built: $(DIST_DIR)/$(BINARY_NAME)-aarch64-unknown-linux-gnu"

## macOS x86_64 (requires macOS host or osxcross)
darwin-x64:
	@mkdir -p $(DIST_DIR)
	@if [ "$$(uname -s)" != "Darwin" ]; then \
		echo "Warning: Building macOS binaries requires macOS host or osxcross"; \
		echo "Skipping darwin-x64..."; \
	else \
		CGO_ENABLED=1 GOOS=darwin GOARCH=amd64 \
			go build $(GO_BUILD_FLAGS) \
			-o $(DIST_DIR)/$(BINARY_NAME)-x86_64-apple-darwin \
			./cmd/design-tokens-lsp; \
		echo "Built: $(DIST_DIR)/$(BINARY_NAME)-x86_64-apple-darwin"; \
	fi

## macOS ARM64 (Apple Silicon - requires macOS host or osxcross)
darwin-arm64:
	@mkdir -p $(DIST_DIR)
	@if [ "$$(uname -s)" != "Darwin" ]; then \
		echo "Warning: Building macOS binaries requires macOS host or osxcross"; \
		echo "Skipping darwin-arm64..."; \
	else \
		CGO_ENABLED=1 GOOS=darwin GOARCH=arm64 \
			go build $(GO_BUILD_FLAGS) \
			-o $(DIST_DIR)/$(BINARY_NAME)-aarch64-apple-darwin \
			./cmd/design-tokens-lsp; \
		echo "Built: $(DIST_DIR)/$(BINARY_NAME)-aarch64-apple-darwin"; \
	fi

## Build the Podman image for Windows cross-compilation (cached)
build-windows-cc-image:
	@if ! podman image exists $(WINDOWS_CC_IMAGE); then \
		echo "Building Windows cross-compilation image..."; \
		podman build -t $(WINDOWS_CC_IMAGE) . ; \
	else \
		echo "Image $(WINDOWS_CC_IMAGE) already exists, skipping build."; \
		echo "Use 'make rebuild-windows-cc-image' to force rebuild."; \
	fi

## Force rebuild of the Windows cross-compilation image
rebuild-windows-cc-image:
	podman build --no-cache -t $(WINDOWS_CC_IMAGE) .

## Windows x86_64 (requires Podman and Containerfile)
windows-x64: build-windows-cc-image
	@mkdir -p $(DIST_DIR)
	podman run --rm \
		-v $(PWD):/app:Z \
		-w /app \
		-e GOOS=windows \
		-e GOARCH=amd64 \
		-e CGO_ENABLED=1 \
		-e CC=x86_64-w64-mingw32-gcc \
		-e CXX=x86_64-w64-mingw32-g++ \
		$(WINDOWS_CC_IMAGE) \
		go build $(GO_BUILD_FLAGS) \
			-o dist/bin/$(BINARY_NAME)-win-x64.exe \
			./cmd/design-tokens-lsp
	@echo "Built: $(DIST_DIR)/$(BINARY_NAME)-win-x64.exe"

## Windows ARM64 (requires Podman - experimental, MinGW ARM64 support varies)
windows-arm64: build-windows-cc-image
	@mkdir -p $(DIST_DIR)
	@echo "Warning: Windows ARM64 cross-compilation is experimental"
	podman run --rm \
		-v $(PWD):/app:Z \
		-w /app \
		-e GOOS=windows \
		-e GOARCH=arm64 \
		-e CGO_ENABLED=1 \
		$(WINDOWS_CC_IMAGE) \
		go build $(GO_BUILD_FLAGS) \
			-o dist/bin/$(BINARY_NAME)-win-arm64.exe \
			./cmd/design-tokens-lsp || \
		echo "Warning: Windows ARM64 build may require additional MinGW setup"

## VSCode extension targets
vscode-build: build
	@echo "Building VSCode extension..."
	@mkdir -p extensions/vscode/dist/bin
	@# Copy native binary for local testing
	@cp $(DIST_DIR)/$(BINARY_NAME) extensions/vscode/dist/bin/
	@cd extensions/vscode && npm install && node build.js
	@echo "VSCode extension built"

vscode-package: build-all
	@echo "Packaging VSCode extension with all platform binaries..."
	@mkdir -p extensions/vscode/dist/bin
	@# Copy all platform binaries
	@cp $(DIST_DIR)/$(BINARY_NAME)-* extensions/vscode/dist/bin/ 2>/dev/null || true
	@cd extensions/vscode && npm install && npm run build
	@echo "VSCode extension packaged: extensions/vscode/*.vsix"

## Help
help:
	@echo "Design Tokens Language Server - Makefile"
	@echo ""
	@echo "Common targets:"
	@echo "  make build              Build native binary"
	@echo "  make build-all          Build all platform binaries (Linux, macOS, Windows)"
	@echo "  make test               Run tests"
	@echo "  make test-coverage      Run tests with coverage"
	@echo "  make install            Install to ~/.local/bin"
	@echo "  make clean              Clean build artifacts"
	@echo ""
	@echo "Platform-specific builds:"
	@echo "  make linux-x64          Build for Linux x86_64"
	@echo "  make linux-arm64        Build for Linux ARM64 (requires gcc-aarch64-linux-gnu)"
	@echo "  make darwin-x64         Build for macOS x86_64 (requires macOS host)"
	@echo "  make darwin-arm64       Build for macOS ARM64 (requires macOS host)"
	@echo "  make windows-x64        Build for Windows x64 (requires Podman)"
	@echo ""
	@echo "VSCode extension:"
	@echo "  make vscode-build       Build VSCode extension for local testing"
	@echo "  make vscode-package     Package VSCode extension with all binaries"
	@echo ""
	@echo "Requirements for cross-compilation:"
	@echo "  - Linux ARM64: gcc-aarch64-linux-gnu"
	@echo "  - macOS: macOS host or osxcross"
	@echo "  - Windows: Podman (uses Containerfile)"
