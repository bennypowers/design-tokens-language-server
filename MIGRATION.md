# Infrastructure Migration: Deno/TypeScript → Go

This document describes the infrastructure migration from the Deno/TypeScript implementation to the Go implementation.

## Summary

The Design Tokens Language Server has been fully rewritten in Go, achieving 100% feature parity with the TypeScript/Deno version. This migration brings several benefits:

- **Better Performance**: Native compilation with tree-sitter integration
- **Simpler Distribution**: Single binary per platform (no runtime dependencies)
- **Standard Tooling**: Go's built-in testing, benchmarking, and toolchain
- **CGO for tree-sitter**: Robust CSS parsing using battle-tested tree-sitter library

## Build System Changes

### Before (Deno)
```bash
deno task build        # Bundle with esbuild + deno compile
deno task test         # Run tests
deno task vscode:build # Build VSCode extension
```

### After (Go + Make)
```bash
make build             # Build native binary
make test              # Run tests
make build-all         # Build all platforms
make vscode-package    # Build VSCode extension
```

## Cross-Compilation Strategy

### CGO Requirement

The Go implementation uses `github.com/tree-sitter/go-tree-sitter` for CSS parsing, which requires CGO (C bindings). This affects cross-compilation:

| Platform | Build Method | Requirements |
|----------|--------------|--------------|
| Linux x64 | Native or cross-compile | `CGO_ENABLED=1` |
| Linux ARM64 | Cross-compile | `gcc-aarch64-linux-gnu` |
| macOS x64/ARM64 | macOS host | Native build on macOS |
| Windows x64 | Podman container | MinGW-w64 in Fedora container |

### Makefile Targets

- `make linux-x64` - Linux x86_64
- `make linux-arm64` - Linux ARM64 (requires gcc-aarch64-linux-gnu)
- `make darwin-x64` - macOS Intel (requires macOS host)
- `make darwin-arm64` - macOS Apple Silicon (requires macOS host)
- `make windows-x64` - Windows x64 (uses Podman + Containerfile)

## CI/CD Changes

### Test Workflow (`.github/workflows/test.yaml`)
- **Before**: `deno task test` with Deno runtime
- **After**: `make test-coverage` with Go 1.25.3
- **Coverage**: Changed from `coverage/lcov.info` to `coverage.out`

### Build Workflow (`.github/workflows/build.yaml`)
- **Before**: Single job with `deno compile` for all platforms
- **After**: Multi-job strategy:
  1. `test` - Run tests with coverage
  2. `build-linux-windows` - Build Linux/Windows on Ubuntu (with Podman)
  3. `build-macos` - Build macOS binaries on macOS runner
  4. `build-vscode` - Package VSCode extension with all binaries
  5. `release` - Create GitHub release with artifacts

### Release Workflow (`.github/workflows/release.yaml`)
- **Before**: `deno task vscode:publish` bundles and publishes
- **After**:
  - Downloads binaries from GitHub release
  - Packages VSCode extension with downloaded binaries
  - Publishes to VSCode Marketplace
- **Zed Extension**: No changes (downloads from GitHub releases)

## VSCode Extension Changes

### build.js
- **Removed**: Copying of `dist/` directory from Deno build
- **Removed**: WASM file handling
- **Kept**: esbuild bundling of extension client TypeScript code
- **Added**: Binary verification with helpful warnings

### package.json
- **Scripts simplified**:
  ```json
  {
    "vscode:prepublish": "node build.js --production",
    "build": "node build.js && npx @vscode/vsce package",
    "publish": "npx @vscode/vsce publish"
  }
  ```
- **No changes** to extension configuration (schema, activation events, etc.)

## Version Management

### Before: `scripts/version.ts` (Deno)
```bash
deno task version 0.0.30
```

### After: `scripts/version.sh` (Bash)
```bash
./scripts/version.sh 0.0.30
```

Updates:
- `extensions/vscode/package.json` version
- `extensions/zed/extension.toml` version
- Creates git tag `v0.0.30` (with confirmation)

## File Changes

### New Files
- `Makefile` - Build system
- `Containerfile` - Windows cross-compilation (Podman)
- `scripts/version.sh` - Version management
- `MIGRATION.md` - This document

### Modified Files
- `.github/workflows/test.yaml` - Use Go instead of Deno
- `.github/workflows/build.yaml` - Multi-stage build with cross-compilation
- `.github/workflows/release.yaml` - Download binaries from release
- `extensions/vscode/build.js` - Remove Deno bundling logic
- `extensions/vscode/package.json` - Simplify scripts

### Files to Remove (Future)
Once the migration is fully tested and deployed:
- `src/` - All TypeScript source files
- `scripts/bundle.ts` - Deno bundling script
- `scripts/version.ts` - Deno version script
- `deno.json` - Deno configuration
- `deno.lock` - Deno lockfile
- `import-map-bundle.json` - Deno import map
- Root `package.json` - npm dependencies for Deno build

## Binary Naming Convention

Both Deno and Go implementations use the same naming:
- `design-tokens-language-server-x86_64-unknown-linux-gnu`
- `design-tokens-language-server-aarch64-unknown-linux-gnu`
- `design-tokens-language-server-x86_64-apple-darwin`
- `design-tokens-language-server-aarch64-apple-darwin`
- `design-tokens-language-server-x86_64-pc-windows-msvc.exe`

This ensures compatibility with existing Zed and VSCode extension code.

## Testing the Migration

### Local Development
```bash
# 1. Build native binary
make build

# 2. Run tests
make test

# 3. Test VSCode extension packaging
make vscode-package
cd extensions/vscode && npm run build
```

### CI Testing
1. Create feature branch
2. Push tag: `git tag v0.0.30-test && git push origin v0.0.30-test`
3. Verify build workflow completes
4. Check GitHub release has all binaries

## Breaking Changes

**None** - This is purely an infrastructure migration. End users will not notice any difference in functionality or behavior.

## Performance Comparison

(To be added after deployment)

## Troubleshooting

### Windows Build Fails
```bash
# Rebuild Podman container
make rebuild-windows-cc-image
make windows-x64
```

### macOS Binaries Missing in CI
- Ensure `build-macos` job ran on `macos-latest` runner
- Check macOS runner availability in GitHub Actions

### Cross-Compilation Errors
```bash
# Install required toolchains
sudo apt-get install gcc-aarch64-linux-gnu  # Linux ARM64
sudo apt-get install podman                  # Windows via container
```

## Migration Checklist

- [x] Create Makefile with all targets
- [x] Create Containerfile for Windows builds
- [x] Update CI workflows (test, build, release)
- [x] Update VSCode extension build process
- [x] Create version management script
- [x] Test local builds
- [ ] Test full CI/CD pipeline with tag
- [ ] Verify VSCode Marketplace publishing
- [ ] Verify Zed extension publishing
- [ ] Remove TypeScript/Deno source files
- [ ] Update documentation
