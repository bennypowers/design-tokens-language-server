# Containerfile for cross-compiling design-tokens-language-server to Windows
# Based on CEM's proven Windows cross-compilation setup
#
# Usage:
#   podman build -t dtls-windows-cc-image .
#   podman run --rm -v $(PWD):/app:Z -w /app -e GOARCH=amd64 dtls-windows-cc-image

FROM fedora:latest

# Install Go and MinGW cross-compilation toolchains
# Note: Fedora 42 uses mingw64-gcc-c++ instead of mingw64-g++
RUN dnf install -y \
    golang \
    mingw64-gcc \
    mingw64-gcc-c++ \
    mingw32-gcc \
    mingw32-gcc-c++ \
    git \
    && dnf clean all

# Default environment for Windows cross-compilation
ENV CGO_ENABLED=1
ENV GOOS=windows

# Set default compiler for x86_64 Windows
ENV CC=x86_64-w64-mingw32-gcc
ENV CXX=x86_64-w64-mingw32-g++

# Build script that will be executed by default
# Allows overriding GOARCH via environment variable
CMD ["sh", "-c", "go build -o dist/bin/design-tokens-language-server-${GOARCH:-amd64}-pc-windows-msvc.exe ./cmd/design-tokens-lsp"]
