# Makefile for Design Tokens Language Server (Go implementation)
# Uses go-release-workflows for cross-compilation

SHELL := /bin/bash
BINARY_NAME := design-tokens-language-server
DIST_DIR := dist/bin

# Shared Windows cross-compilation image (from go-release-workflows)
SHARED_WINDOWS_CC_IMAGE := dtls-shared-windows-cc

# Extract version from goals if present (e.g., "make release v0.1.1" or "make release patch")
VERSION ?= $(filter v% patch minor major,$(MAKECMDGOALS))

# Go build flags with version injection
GO_BUILD_FLAGS := -ldflags="$(shell ./scripts/ldflags.sh) -s -w"

.PHONY: all build build-all test test-coverage patch-coverage show-coverage lint install clean \
        linux-x64 linux-arm64 darwin-x64 darwin-arm64 win32-x64 win32-arm64 \
        build-shared-windows-image release patch minor major

all: build

## Clean build artifacts
clean:
	rm -rf dist/ coverage/ coverage.out

## Build native binary
build:
	@mkdir -p $(DIST_DIR)
	go build $(GO_BUILD_FLAGS) -o $(DIST_DIR)/$(BINARY_NAME) ./cmd/design-tokens-language-server
	@echo "Built native binary: $(DIST_DIR)/$(BINARY_NAME)"

## Install to ~/.local/bin (for local development)
install: build
	@mkdir -p ~/.local/bin/
	cp $(DIST_DIR)/$(BINARY_NAME) ~/.local/bin/
	@echo "Installed to ~/.local/bin/$(BINARY_NAME)"

## Run tests
test:
	go test -v ./...

## Run linter and format check
lint:
	@echo "=== Running golangci-lint ==="
	@if ! command -v golangci-lint &> /dev/null; then \
		echo "golangci-lint not found. Installing..."; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
	fi
	@golangci-lint run

## Run tests with coverage (Go 1.20+ includes cross-process coverage for integration tests)
test-coverage:
	@echo "=== Running Unit Tests with Coverage ==="
	@rm -rf coverage/unit coverage/integration
	@mkdir -p coverage/unit
	@go test -cover ./... -args -test.gocoverdir="$$(pwd)/coverage/unit" 2>&1 | grep -v "no test files"
	@echo ""
	@echo "=== Running Integration Tests with Subprocess Coverage ==="
	@mkdir -p coverage/integration
	@go test -cover -coverpkg=./... ./test/integration -args -test.gocoverdir="$$(pwd)/coverage/integration"
	@echo ""
	@echo "=== Merging Coverage Files ==="
	@go tool covdata textfmt -i=./coverage/unit,./coverage/integration -o=coverage.out
	@echo ""
	@echo "=== Coverage Report ==="
	@echo ""
	@echo "Unit Test Coverage:"
	@go tool covdata percent -i=./coverage/unit
	@echo ""
	@echo "Integration Test Coverage:"
	@go tool covdata percent -i=./coverage/integration
	@echo ""
	@echo "Merged Coverage:"
	@go tool cover -func=coverage.out | tail -1
	@echo ""
	@echo "Merged coverage saved to coverage.out for codecov upload"

## Show patch coverage (lines added in this branch vs main)
patch-coverage: coverage.out
	@./scripts/patch-coverage.sh

## Show coverage in browser
show-coverage: test-coverage
	@go tool cover -html=coverage.out

## Build all platform binaries (requires cross-compilation toolchains)
build-all: linux-x64 linux-arm64 darwin-x64 darwin-arm64 win32-x64 win32-arm64
	@echo "All binaries built successfully!"
	@ls -lh $(DIST_DIR)/

## Linux x86_64 (CGO cross-compilation)
linux-x64:
	@mkdir -p $(DIST_DIR)
	CGO_ENABLED=1 GOOS=linux GOARCH=amd64 \
		go build $(GO_BUILD_FLAGS) \
		-o $(DIST_DIR)/$(BINARY_NAME)-linux-x64 \
		./cmd/design-tokens-language-server

## Linux ARM64 (CGO cross-compilation - requires gcc-aarch64-linux-gnu)
linux-arm64:
	@mkdir -p $(DIST_DIR)
	CGO_ENABLED=1 GOOS=linux GOARCH=arm64 \
		CC=aarch64-linux-gnu-gcc \
		go build $(GO_BUILD_FLAGS) \
		-o $(DIST_DIR)/$(BINARY_NAME)-linux-arm64 \
		./cmd/design-tokens-language-server

## macOS x86_64 (requires macOS host)
## Explicit -arch flags ensure correct architecture when cross-compiling on macOS
darwin-x64:
	@mkdir -p $(DIST_DIR)
	CGO_ENABLED=1 GOOS=darwin GOARCH=amd64 \
		CC="clang -arch x86_64" \
		CGO_CFLAGS="-arch x86_64" CGO_LDFLAGS="-arch x86_64" \
		go build $(GO_BUILD_FLAGS) \
		-o $(DIST_DIR)/$(BINARY_NAME)-darwin-x64 \
		./cmd/design-tokens-language-server

## macOS ARM64 (Apple Silicon - requires macOS host)
## Explicit -arch flags ensure correct architecture when cross-compiling on macOS
darwin-arm64:
	@mkdir -p $(DIST_DIR)
	CGO_ENABLED=1 GOOS=darwin GOARCH=arm64 \
		CC="clang -arch arm64" \
		CGO_CFLAGS="-arch arm64" CGO_LDFLAGS="-arch arm64" \
		go build $(GO_BUILD_FLAGS) \
		-o $(DIST_DIR)/$(BINARY_NAME)-darwin-arm64 \
		./cmd/design-tokens-language-server

## Build the shared Windows cross-compilation image (uses go-release-workflows Containerfile)
build-shared-windows-image:
	@if ! podman image exists $(SHARED_WINDOWS_CC_IMAGE); then \
		echo "Building shared Windows cross-compilation image..."; \
		curl -fsSL https://raw.githubusercontent.com/bennypowers/go-release-workflows/main/.github/actions/setup-windows-build/Containerfile \
			| podman build -t $(SHARED_WINDOWS_CC_IMAGE) -f - .; \
	else \
		echo "Image $(SHARED_WINDOWS_CC_IMAGE) already exists, skipping build."; \
	fi

## Windows x86_64 (requires Podman)
win32-x64: build-shared-windows-image
	@mkdir -p $(DIST_DIR)
	podman run --rm \
		-v $(PWD):/src:Z \
		-w /src \
		-e GOOS=windows \
		-e GOARCH=amd64 \
		-e CGO_ENABLED=1 \
		-e CC=x86_64-w64-mingw32-gcc \
		-e CXX=x86_64-w64-mingw32-g++ \
		$(SHARED_WINDOWS_CC_IMAGE) \
		go build $(GO_BUILD_FLAGS) \
			-o $(DIST_DIR)/$(BINARY_NAME)-win32-x64.exe \
			./cmd/design-tokens-language-server

## Windows ARM64 (requires Podman)
win32-arm64: build-shared-windows-image
	@mkdir -p $(DIST_DIR)
	podman run --rm \
		-v $(PWD):/src:Z \
		-w /src \
		-e GOOS=windows \
		-e GOARCH=arm64 \
		-e CGO_ENABLED=1 \
		-e CC=aarch64-w64-mingw32-gcc \
		-e CXX=aarch64-w64-mingw32-g++ \
		$(SHARED_WINDOWS_CC_IMAGE) \
		go build $(GO_BUILD_FLAGS) \
			-o $(DIST_DIR)/$(BINARY_NAME)-win32-arm64.exe \
			./cmd/design-tokens-language-server

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

## Make version targets (v*) and bump types no-ops for "make release" syntax
v%:
	@:

patch minor major:
	@:

## Release (creates version commit, pushes it, then uses gh to tag and create release)
release:
	@if [ -z "$(VERSION)" ]; then \
		echo "Error: VERSION or bump type is required"; \
		echo "Usage: make release <version|patch|minor|major>"; \
		echo "  make release v0.1.1   - Release explicit version"; \
		echo "  make release patch    - Bump patch version (0.0.x)"; \
		echo "  make release minor    - Bump minor version (0.x.0)"; \
		echo "  make release major    - Bump major version (x.0.0)"; \
		exit 1; \
	fi
	@./scripts/release.sh $(VERSION)

## Help
help:
	@echo "Design Tokens Language Server - Makefile"
	@echo ""
	@echo "Common targets:"
	@echo "  make build              Build native binary"
	@echo "  make build-all          Build all platform binaries (Linux, macOS, Windows)"
	@echo "  make test               Run tests"
	@echo "  make test-coverage      Run tests with coverage"
	@echo "  make lint               Run golangci-lint"
	@echo "  make install            Install to ~/.local/bin"
	@echo "  make clean              Clean build artifacts"
	@echo ""
	@echo "Release:"
	@echo "  make release v0.1.1     Create release with explicit version"
	@echo "  make release patch      Bump patch version (0.0.x) and release"
	@echo "  make release minor      Bump minor version (0.x.0) and release"
	@echo "  make release major      Bump major version (x.0.0) and release"
	@echo ""
	@echo "Platform-specific builds:"
	@echo "  make linux-x64          Build for Linux x86_64"
	@echo "  make linux-arm64        Build for Linux ARM64 (requires gcc-aarch64-linux-gnu)"
	@echo "  make darwin-x64         Build for macOS x86_64 (requires macOS host)"
	@echo "  make darwin-arm64       Build for macOS ARM64 (requires macOS host)"
	@echo "  make win32-x64          Build for Windows x64 (requires Podman)"
	@echo "  make win32-arm64        Build for Windows ARM64 (requires Podman)"
	@echo ""
	@echo "VSCode extension:"
	@echo "  make vscode-build       Build VSCode extension for local testing"
	@echo "  make vscode-package     Package VSCode extension with all binaries"
	@echo ""
	@echo "Requirements for cross-compilation:"
	@echo "  - Linux ARM64: gcc-aarch64-linux-gnu"
	@echo "  - macOS: macOS host with Xcode"
	@echo "  - Windows: Podman (uses shared go-release-workflows image)"
