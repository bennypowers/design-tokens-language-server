## Go

Use go 1.25+ features. Use `go vet` and the `gopls` LSP plugin - check all code actions, and apply all suggested refactors (e.g. use `any` instead of `interface{}`)

Getter methods should be named `Foo()`, not `GetFoo()`.

When implementing new features or fixing bugs, practice TDD.

## Logging Requirements

**CRITICAL: NEVER pollute stdout - it's used for LSP JSON-RPC communication**

This is a Language Server Protocol (LSP) implementation. LSP uses stdin/stdout for JSON-RPC communication with the client (editor). **Any output to stdout will corrupt the protocol and crash the server.**

### Rules

- **ALWAYS** use the `internal/log` package for all logging
- **NEVER** use `fmt.Println`, `fmt.Printf`, `print`, or `println` - these write to stdout and will break LSP
- **NEVER** use `fmt.Fprintf(os.Stdout, ...)` - stdout is reserved for LSP protocol
- **NEVER** use `log.Print*` from the standard library - it writes to stderr without coordination
- **ALWAYS** use `bennypowers.dev/dtls/internal/log` for all logging needs

### Log Levels

Use appropriate log levels for different message types:

- `log.Debug("message", args...)` - Verbose debugging information (e.g., LSP method calls, internal state)
- `log.Info("message", args...)` - Important operational events (e.g., file loaded, tokens parsed)
- `log.Warn("message", args...)` - Warnings that don't prevent operation (e.g., deprecated features used)
- `log.Error("message", args...)` - Errors that may affect functionality (e.g., parse failures, missing files)

### Examples

```go
// GOOD - Uses internal log package
import "bennypowers.dev/dtls/internal/log"

log.Info("Loading tokens from: %s", filePath)
log.Error("Failed to parse file: %v", err)
log.Debug("Method %s started", methodName)
```

```go
// BAD - Will corrupt LSP protocol
fmt.Println("Loading tokens...")        // BREAKS LSP!
fmt.Printf("Error: %v\n", err)          // BREAKS LSP!
log.Println("Debug message")            // Wrong logger!
fmt.Fprintf(os.Stdout, "msg\n")         // BREAKS LSP!
```

### Testing

When writing tests that capture log output:

```go
import (
	"bytes"
	"bennypowers.dev/dtls/internal/log"
)

func TestSomething(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(nil)

	// Test code that logs...

	output := buf.String()
	assert.Contains(t, output, "expected message")
}
```

When writing tests, always use the fixture/golden patterns with `testdata/` directories per Go idiom.

### testdata Directory Convention

Each package should have its own `testdata/` directory for fixtures and goldens:

```
internal/schema/
├── detector.go
├── detector_test.go
└── testdata/
    ├── detection/          # Fixtures for detector tests
    │   ├── explicit-draft.json
    │   └── explicit-2025.json
    └── golden/             # Expected outputs
        └── expected-output.json
```

- **Fixtures**: Input test data in `testdata/` subdirectories
- **Goldens**: Expected output files in `testdata/golden/`. Tests should support `--update` flag to regenerate golden files when intentional changes occur.
- Always use Makefile targets for running tests or builds, since they export the necessary env vars.

### Loading Test Data

Use `os.ReadFile` with paths relative to the test file:

```go
func TestDetectVersion(t *testing.T) {
    content, err := os.ReadFile("testdata/detection/explicit-draft.json")
    require.NoError(t, err)
    // ...
}
```

### Test Utilities

- `lsp/testutil`: Mock server context for unit tests
  - `NewMockServerContext()` - Create mock server with document/token managers

- `test/integration/testutil`: Integration test helpers
  - `NewTestServer(t)` - Create LSP server for testing

### In-Memory Filesystems

For tests that need filesystem abstraction, use `testing/fstest.MapFS`:

```go
import "testing/fstest"

fs := fstest.MapFS{
    "tokens.json": &fstest.MapFile{
        Data: []byte(`{"color": {"$value": "#fff"}}`),
    },
}
```

### LSP Testing Patterns

- **Fixture Structure**: Each test scenario is a subdirectory in `testdata/` containing:
  - Input files (e.g., `tokens.json`, `input.json`)
  - `expected.json` or `expected-*.json` (optional, for assertions)

- **Multiple Expected Files**: Use `expected-variant.json`, `expected-size.json` pattern for testing multiple positions/cases in one fixture

- **Regression Test Isolation**: Keep regression test fixtures in separate directories (e.g., `testdata-regression/`) to avoid interference with standard test discovery

### Coverage

Each PR must exceed an 80% coverage threshold for line coverage.

Measure coverage using:
```bash
go test -cover ./...
# For detailed HTML report:
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

Coverage is checked automatically in CI. To verify locally before submitting:
```bash
make test-coverage
```

## Git

When commit messages mention AI agents, always use `Assisted-By`, never use `Co-Authored-By`.
