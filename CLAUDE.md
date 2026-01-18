## Go

Use go 1.25+ features. Use `go vet` and the `gopls` LSP plugin - check all code actions, and apply all suggested refactors (e.g. use `any` instead of `interface{}`)

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
