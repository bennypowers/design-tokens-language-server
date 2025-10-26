# Design Tokens Language Server - Go Rewrite Plan

## 📋 Executive Summary

Migrating from TypeScript/Deno to Go to leverage better performance, simpler deployment (single binary), and the proven LSP infrastructure from the CEM project. This plan follows a Test-Driven Development approach with comprehensive performance monitoring.

---

## 🏗️ Architecture Design

### Core Components

```
design-tokens-language-server/
├── cmd/
│   └── design-tokens-language-server/
│       └── main.go                    # Entry point
├── internal/
│   ├── server/
│   │   ├── server.go                  # Main LSP server (stdio communication)
│   │   └── capabilities.go            # Server capabilities definition
│   ├── lsp/
│   │   ├── lsp.go                     # LSP request processor
│   │   ├── context.go                 # DTLS context and types
│   │   ├── methods/
│   │   │   ├── initialize.go          # Server initialization
│   │   │   ├── textDocument/
│   │   │   │   ├── hover.go
│   │   │   │   ├── completion.go
│   │   │   │   ├── definition.go
│   │   │   │   ├── references.go
│   │   │   │   ├── codeAction.go
│   │   │   │   ├── documentColor.go
│   │   │   │   ├── colorPresentation.go
│   │   │   │   ├── semanticTokens.go
│   │   │   │   └── diagnostic.go
│   │   │   ├── completionItem/
│   │   │   │   └── resolve.go         # Completion item resolution
│   │   │   ├── codeAction/
│   │   │   │   └── resolve.go         # Code action resolution
│   │   │   └── workspace/
│   │   │       └── didChangeConfiguration.go
│   │   ├── helpers/
│   │   │   └── logging.go
│   │   └── types/
│   │       └── protocol.go            # LSP protocol types
│   ├── documents/
│   │   ├── manager.go                 # Document lifecycle management
│   │   ├── document.go                # Document abstraction
│   │   ├── css.go                     # CSS document type
│   │   ├── json.go                    # JSON document type
│   │   ├── yaml.go                    # YAML document type
│   │   └── cache.go                   # Document cache
│   ├── tokens/
│   │   ├── manager.go                 # Token storage and lookup
│   │   ├── resolver.go                # Token resolution & references
│   │   ├── validator.go               # Token validation
│   │   ├── types.go                   # Token data structures
│   │   ├── markdown.go                # Token markdown generation
│   │   └── color.go                   # Color manipulation utilities
│   ├── workspace/
│   │   ├── manager.go                 # Workspace configuration
│   │   ├── config.go                  # Config types (TokenFile, DTLSClientSettings)
│   │   └── scanner.go                 # Token file discovery
│   └── parser/
│       ├── css/
│       │   ├── parser.go              # Tree-sitter CSS parser
│       │   ├── queries.go             # Tree-sitter queries for CSS variables
│       │   └── treesitter.go          # Tree-sitter bindings
│       ├── json/
│       │   └── parser.go              # JSON token parsing (stdlib + jsonc support)
│       └── yaml/
│           └── parser.go              # YAML token parsing
├── pkg/
│   └── protocol/                      # Public protocol definitions (if needed)
├── test/
│   ├── goldens/                       # Golden file tests
│   ├── testdata/                      # Test fixtures (token files, CSS samples)
│   └── integration/                   # Integration tests
├── tools/
│   ├── lsp-bench/                     # LSP benchmarking harness
│   │   └── main.go
│   └── benchmark/
│       ├── run-baseline.sh            # TypeScript baseline benchmarks
│       └── run-go.sh                  # Go implementation benchmarks
└── go.mod
```

### Key Libraries

1. **LSP Framework**: `github.com/tliron/glsp` (proven in CEM project)
2. **Tree-sitter Parsing**:
   - `github.com/tree-sitter/go-tree-sitter` - Go bindings for tree-sitter
   - `github.com/tree-sitter/tree-sitter-css` - CSS grammar
   - Note: The TypeScript version already uses `web-tree-sitter`, so we're maintaining the same parsing approach
3. **Additional Parsing**:
   - `gopkg.in/yaml.v3` - YAML parsing for token files
   - `encoding/json` + custom JSONC support - JSON parsing (stdlib)
4. **Color**: `github.com/lucasb-eyer/go-colorful` - color manipulation (equivalent to tinycolor2)
5. **Testing**: Standard library + `github.com/stretchr/testify`
6. **Logging**: `log/slog` for structured logging

**Go Version**: Go 1.25.3 (installed natively)

### Architecture Principles

1. **Modularity**: Separate concerns (parsing, validation, LSP methods)
2. **Concurrency**: Use goroutines for async operations (diagnostics, file scanning)
3. **Caching**: Cache parsed documents and token graphs
4. **Incremental Updates**: Support incremental document changes (TextDocumentSyncKind.Incremental)
5. **Type Safety**: Leverage Go's type system for robust token handling

### LSP Capabilities Mapping

Based on the TypeScript implementation, the server supports:

| Capability | Method | Resolve Method | Implementation Location |
|------------|--------|----------------|------------------------|
| Text Document Sync | N/A | N/A | `internal/lsp/lsp.go` |
| Hover | `textDocument/hover` | N/A | `internal/lsp/methods/textDocument/hover.go` |
| Completion | `textDocument/completion` | `completionItem/resolve` | `internal/lsp/methods/textDocument/completion.go` + `internal/lsp/methods/completionItem/resolve.go` |
| Definition | `textDocument/definition` | N/A | `internal/lsp/methods/textDocument/definition.go` |
| References | `textDocument/references` | N/A | `internal/lsp/methods/textDocument/references.go` |
| Code Actions | `textDocument/codeAction` | `codeAction/resolve` | `internal/lsp/methods/textDocument/codeAction.go` + `internal/lsp/methods/codeAction/resolve.go` |
| Document Color | `textDocument/documentColor` | `textDocument/colorPresentation` | `internal/lsp/methods/textDocument/documentColor.go` + `internal/lsp/methods/textDocument/colorPresentation.go` |
| Semantic Tokens | `textDocument/semanticTokens/full` | N/A | `internal/lsp/methods/textDocument/semanticTokens.go` |
| Diagnostics | `textDocument/diagnostic` | N/A | `internal/lsp/methods/textDocument/diagnostic.go` |

---

## 🧪 Testing Strategy (TDD Approach)

### Test Levels

#### 1. Unit Tests
- **Coverage Target**: 80%+ for core business logic
- **Focus Areas**:
  - Token parsing (CSS with tree-sitter, JSON, YAML)
  - Token resolution and reference finding
  - Color validation and manipulation
  - Diagnostic generation (incorrect fallback, unknown reference)

**Example Test Structure**:
```go
// internal/tokens/resolver_test.go
func TestResolveTokenReference(t *testing.T) {
    tests := []struct {
        name     string
        token    string
        expected *Token
        wantErr  bool
    }{
        {
            name: "simple reference",
            token: "var(--color-primary)",
            expected: &Token{Name: "color-primary", Value: "#ff0000"},
        },
        {
            name: "nested reference",
            token: "var(--color-primary-hover)",
            expected: &Token{Name: "color-primary-hover", Value: "#cc0000"},
        },
        {
            name: "circular reference",
            token: "var(--color-a)",
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // TDD: Write test first, then implementation
            resolver := NewResolver(tokenGraph)
            got, err := resolver.Resolve(tt.token)

            if tt.wantErr {
                assert.Error(t, err)
                return
            }

            assert.NoError(t, err)
            assert.Equal(t, tt.expected, got)
        })
    }
}
```

#### 2. Tree-sitter Parsing Tests
- Test CSS variable extraction using tree-sitter queries
- Validate correct position information for diagnostics and navigation
- Test incremental parsing for document updates

```go
// internal/parser/css/parser_test.go
func TestExtractCSSVariables(t *testing.T) {
    tests := []struct {
        name     string
        css      string
        expected []CSSVariable
    }{
        {
            name: "simple variable",
            css:  ":root { --color-primary: #ff0000; }",
            expected: []CSSVariable{
                {Name: "--color-primary", Value: "#ff0000", Range: ...},
            },
        },
        {
            name: "variable with var() reference",
            css:  ".button { color: var(--color-primary, blue); }",
            expected: []CSSVariable{
                {
                    Name: "--color-primary",
                    Type: VarReference,
                    Fallback: "blue",
                    Range: ...,
                },
            },
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            parser := NewCSSParser()
            got, err := parser.ExtractVariables(tt.css)
            assert.NoError(t, err)
            assert.Equal(t, tt.expected, got)
        })
    }
}
```

#### 3. Golden File Tests
- Adopt CEM's golden file pattern for LSP responses
- Test each LSP method with real-world token files
- Store expected responses in `test/goldens/`

**Example**:
```go
// test/hover_test.go
func TestHoverGolden(t *testing.T) {
    testCases := []struct {
        name     string
        file     string
        line     int
        char     int
    }{
        {"css-variable-hover", "tokens.css", 5, 10},
        {"token-reference-hover", "components.css", 12, 20},
        {"deprecated-token-hover", "legacy.css", 3, 15},
    }

    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            // Initialize server with test workspace
            server := initTestServer(t)

            // Open document
            uri := fileToURI(filepath.Join("testdata", tc.file))
            content := readFile(t, filepath.Join("testdata", tc.file))
            server.DidOpen(uri, content)

            // Perform hover
            params := lsp.HoverParams{
                TextDocument: lsp.TextDocumentIdentifier{URI: uri},
                Position:     lsp.Position{Line: tc.line, Character: tc.char},
            }
            got := server.Hover(params)

            // Compare with golden file
            goldenPath := filepath.Join("goldens", fmt.Sprintf("%s.json", tc.name))
            compareGolden(t, goldenPath, got)
        })
    }
}
```

#### 4. Integration Tests
- End-to-end LSP server tests
- Test complete workflows (open file → diagnostics → code action → apply)
- Test workspace scanning and multi-file token resolution

```go
// test/integration/workflow_test.go
func TestDiagnosticToCodeActionWorkflow(t *testing.T) {
    server := initTestServer(t)

    // 1. Open document with invalid token fallback
    uri := fileToURI("testdata/invalid-fallback.css")
    content := `:root { color: var(--primary, #ff0000); }` // incorrect fallback
    server.DidOpen(uri, content)

    // 2. Get diagnostics
    diagnostics := server.Diagnostic(uri)
    assert.Len(t, diagnostics, 1)
    assert.Equal(t, "incorrect-fallback", diagnostics[0].Code)

    // 3. Request code actions for the diagnostic
    actions := server.CodeAction(uri, diagnostics[0].Range, diagnostics)
    assert.NotEmpty(t, actions)

    // 4. Resolve and apply the first code action
    resolved := server.ResolveCodeAction(actions[0])
    assert.NotNil(t, resolved.Edit)

    // 5. Apply edit and verify the fix
    applyEdit(server, resolved.Edit)
    newContent := server.GetDocumentContent(uri)
    assert.Contains(t, newContent, "var(--primary, #0000ff)") // corrected fallback
}
```

#### 5. Benchmark Tests
- Performance regression tests
- Compare against TypeScript implementation
- Track memory usage and response times

```go
// internal/tokens/resolver_bench_test.go
func BenchmarkTokenResolution(b *testing.B) {
    // Load large token file with 1000+ tokens
    tokens := loadTestTokens(b, "testdata/large-tokens.json")
    resolver := NewResolver(tokens)

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        resolver.Resolve("--color-primary-500")
    }
}

func BenchmarkCSSParsing(b *testing.B) {
    css := readFile(b, "testdata/large-stylesheet.css")
    parser := NewCSSParser()

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        parser.Parse(css)
    }
}

func BenchmarkHoverResponse(b *testing.B) {
    server := initTestServer(b)
    uri := fileToURI("testdata/tokens.css")

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        server.Hover(lsp.HoverParams{
            TextDocument: lsp.TextDocumentIdentifier{URI: uri},
            Position:     lsp.Position{Line: 10, Character: 15},
        })
    }
}
```

### TDD Workflow

1. **Write test first** (RED phase) - Test fails because feature doesn't exist
2. **Implement minimal code** to pass (GREEN phase) - Make the test pass
3. **Refactor** for clean code (REFACTOR phase) - Improve without breaking tests
4. **Measure performance** with benchmarks - Ensure performance goals are met
5. **Update golden files** when behavior changes intentionally

**Example TDD Cycle for Hover Feature**:
1. Write `TestHoverGolden` with expected output → RED (test fails)
2. Implement basic `Hover()` method → GREEN (test passes)
3. Refactor for cleaner code, add caching → REFACTOR
4. Run `BenchmarkHoverResponse` → measure performance
5. Optimize if needed without breaking tests

---

## 📊 Performance Monitoring

### Before Migration (TypeScript Baseline)

**Metrics to Capture**:
1. **Startup time** - Time from launch to ready
2. **Document parsing time** - Various file sizes (100 lines, 1k lines, 10k lines)
3. **Hover response time** - Average response latency
4. **Completion response time** - Including resolve step
5. **Diagnostic generation time** - Full document analysis
6. **Memory usage** - Idle and under load (10 open documents)
7. **CPU usage** - During intensive operations

**Tool 1: `hyperfine`** (command-line benchmarking)

```bash
# Install hyperfine
brew install hyperfine  # or apt install hyperfine

# Benchmark startup time
hyperfine --warmup 3 --runs 10 \
    'deno run --allow-all src/server/server.ts < /dev/null &'
```

**Tool 2: Custom LSP Benchmark Harness**

Create a Go tool that sends LSP requests and measures response times. This will work for both TypeScript and Go implementations.

```go
// tools/lsp-bench/main.go
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "os"
    "time"
)

type BenchmarkResult struct {
    Operation    string        `json:"operation"`
    AvgLatency   time.Duration `json:"avg_latency"`
    MinLatency   time.Duration `json:"min_latency"`
    MaxLatency   time.Duration `json:"max_latency"`
    MemoryUsage  uint64        `json:"memory_usage_bytes"`
}

func main() {
    serverCmd := os.Args[1] // e.g., "deno run --allow-all src/server/server.ts"

    client := NewLSPClient(serverCmd)
    defer client.Close()

    results := []BenchmarkResult{}

    // Benchmark initialization
    start := time.Now()
    client.Initialize(context.Background(), initParams)
    results = append(results, BenchmarkResult{
        Operation: "initialize",
        AvgLatency: time.Since(start),
    })

    // Open test document
    client.DidOpen(context.Background(), "file:///test/tokens.css", cssContent)

    // Benchmark hover (100 iterations)
    latencies := []time.Duration{}
    for i := 0; i < 100; i++ {
        start := time.Now()
        client.Hover(context.Background(), hoverParams)
        latencies = append(latencies, time.Since(start))
    }
    results = append(results, BenchmarkResult{
        Operation: "hover",
        AvgLatency: avg(latencies),
        MinLatency: min(latencies),
        MaxLatency: max(latencies),
    })

    // Benchmark completion
    // Benchmark diagnostics
    // etc.

    // Output results as JSON
    json.NewEncoder(os.Stdout).Encode(results)
}
```

**Baseline Script**:
```bash
#!/bin/bash
# tools/benchmark/run-baseline.sh

echo "Running TypeScript baseline benchmarks..."

# Build the benchmark tool
cd tools/lsp-bench
go build -o lsp-bench

# Run benchmarks against TypeScript server
./lsp-bench "deno run --allow-all ../../src/server/server.ts" \
    --testdata ../../test/testdata \
    --output ../../baseline-results.json

echo "Baseline results saved to baseline-results.json"
```

### After Migration (Go Implementation)

**Use Same Metrics** for apples-to-apples comparison

**Go Benchmark Script**:
```bash
#!/bin/bash
# tools/benchmark/run-go.sh

echo "Running Go implementation benchmarks..."

# Build the Go LSP server
go build -o design-tokens-language-server ./cmd/design-tokens-language-server

# Run same benchmarks
./tools/lsp-bench/lsp-bench "./design-tokens-language-server" \
    --testdata ./test/testdata \
    --output go-results.json

echo "Go results saved to go-results.json"
```

**Continuous Monitoring**:
- Add benchmark tests to CI/CD
- Track performance over time with `benchstat`
- Set up alerts for regressions > 20%

```bash
# Compare before/after
go test -bench=. -benchmem ./... > new.txt
benchstat baseline.txt new.txt
```

**Expected Improvements** (Goals):
- **Startup time**: < 100ms (vs ~500ms TypeScript)
- **Hover response**: < 10ms (vs ~50ms TypeScript)
- **Completion response**: < 50ms including resolve
- **Memory usage**: < 50MB idle (vs ~150MB TypeScript)
- **Large file parsing** (10k tokens): < 100ms
- **Binary size**: < 20MB (single executable, no runtime needed)

---

## 🔄 Migration Plan

### Phase 1: Foundation (Weeks 1-2)

**Goals**:
- Set up Go project structure
- Implement basic LSP server with `glsp`
- Create document manager with incremental updates
- Integrate tree-sitter for CSS parsing
- Write baseline performance benchmarks

**TDD Tasks**:
1. ✅ Write test for LSP initialization → Implement
2. ✅ Write test for document open/close/change → Implement
3. ✅ Write test for incremental document updates → Implement
4. ✅ Write test for tree-sitter CSS parsing → Implement
5. ✅ Create performance benchmark harness for both TS and Go

**Deliverables**:
- LSP server that can accept connections and manage documents (no features yet)
- Tree-sitter CSS parser extracting variables and var() calls
- Benchmark tool that works with both implementations
- Baseline performance metrics from TypeScript version

### Phase 2: Token Parsing & Management (Weeks 3-4)

**Goals**:
- Implement JSON/YAML token file parsers
- Build token data structures matching the DTCG spec
- Create token manager for storage and lookup
- Implement token graph and reference resolver
- Support token prefixes and group markers

**TDD Tasks**:
1. ✅ Write tests for JSON token parsing (with JSONC support) → Implement
2. ✅ Write tests for YAML token parsing → Implement
3. ✅ Write tests for token graph construction → Implement
4. ✅ Write tests for token reference resolution → Implement
5. ✅ Write tests for circular reference detection → Implement
6. ✅ Write tests for token prefix handling → Implement
7. ✅ Write tests for group markers → Implement

**Deliverables**:
- Token parsing library with 80%+ test coverage
- Token manager that can load and resolve tokens from multiple files
- Performance benchmarks showing parsing speed improvements

### Phase 3: Core LSP Features - Part 1 (Weeks 5-6)

**Priority Order** (based on developer experience impact):

#### Week 5: Diagnostics & Hover
1. **Diagnostics** (most important for developer experience)
   - Invalid token references (`unknown-reference`)
   - Incorrect fallback values (`incorrect-fallback`)
   - Semantic CSS value equivalence
   - Deprecated token warnings

2. **Hover** (documentation and value preview)
   - Token value and type information
   - Token description from metadata
   - Markdown formatting

**TDD Tasks**:
1. ✅ Write golden file tests for diagnostics → Implement → Verify
2. ✅ Write golden file tests for hover → Implement → Verify
3. ✅ Benchmark against TypeScript version

#### Week 6: Navigation Features
3. **Go to Definition**
   - Navigate from var() reference to token definition
   - Support cross-file navigation

4. **Find References**
   - Find all usages of a token
   - Support workspace-wide search

**TDD Tasks**:
1. ✅ Write golden file tests for definition → Implement → Verify
2. ✅ Write golden file tests for references → Implement → Verify
3. ✅ Benchmark against TypeScript version

**Deliverables**:
- Core navigation and diagnostic features working
- Golden file tests for all implemented features
- Performance comparison showing improvements

### Phase 4: Core LSP Features - Part 2 (Week 7)

#### Completion & Code Actions
5. **Completion**
   - Token name completion in var() calls
   - Completion item resolve for detailed information

6. **Code Actions**
   - Fix incorrect fallback values
   - Quick fixes for diagnostics

**TDD Tasks**:
1. ✅ Write golden file tests for completion → Implement → Verify
2. ✅ Write golden file tests for completion resolve → Implement → Verify
3. ✅ Write golden file tests for code actions → Implement → Verify
4. ✅ Write golden file tests for code action resolve → Implement → Verify
5. ✅ Benchmark against TypeScript version

**Deliverables**:
- Completion and code actions working
- Feature parity for core editing features

### Phase 5: Advanced Features (Week 8)

#### Document Color & Semantic Tokens
7. **Document Color**
   - Extract color tokens for color picker
   - Color presentation for format conversion

8. **Semantic Tokens**
   - Syntax highlighting for token variables
   - Semantic token types and modifiers

**TDD Tasks**:
1. ✅ Write golden file tests for document color → Implement → Verify
2. ✅ Write golden file tests for color presentation → Implement → Verify
3. ✅ Write golden file tests for semantic tokens → Implement → Verify

#### Workspace Management
9. **Workspace Configuration**
   - Token file scanning and loading
   - Configuration handling (tokensFiles, prefix, groupMarkers)
   - Multi-workspace support

**TDD Tasks**:
1. ✅ Write tests for workspace scanning → Implement
2. ✅ Write tests for configuration parsing → Implement
3. ✅ Write tests for multi-file token resolution → Implement

**Deliverables**:
- Complete feature parity with TypeScript implementation
- All LSP capabilities working

### Phase 6: Optimization & Polish (Week 9)

**Goals**:
- Performance optimization based on profiling
- Memory optimization and leak detection
- Concurrent diagnostic generation
- Caching improvements

**Tasks**:
1. Profile CPU and memory usage with `pprof`
2. Optimize hot paths identified by profiling
3. Add concurrent processing where safe
4. Implement intelligent caching
5. Fix any performance regressions

**Deliverables**:
- Performance meets or exceeds goals
- No memory leaks
- Profiling reports showing optimizations

### Phase 7: Testing & Validation (Week 10)

**Goals**:
- Comprehensive testing with real-world projects
- Bug fixes and edge cases
- Documentation and migration guide

**Tasks**:
1. Test with large real-world token files (1000+ tokens)
2. Test with complex multi-workspace setups
3. Manual testing with VSCode extension
4. Create test suite for edge cases discovered
5. Write migration guide for users
6. Update documentation

**Deliverables**:
- All tests passing
- Zero critical bugs
- Migration guide complete

### Phase 8: Deployment & Release (Week 11)

**Goals**:
- Package binaries for all platforms
- Update VSCode extension to use Go binary
- Publish new version
- Monitor for issues

**Tasks**:
1. Set up GitHub Actions for multi-platform builds (Linux, macOS, Windows)
2. Create release workflow
3. Update VSCode extension to download/use Go binary
4. Update extension documentation
5. Release v1.0.0 of Go implementation
6. Announce deprecation timeline for TypeScript version

**Deliverables**:
- Multi-platform binaries available
- VSCode extension updated and published
- Migration complete

---

## 🎯 Success Criteria

### Functional Requirements
- ✅ 100% feature parity with TypeScript version
- ✅ All LSP capabilities working correctly
- ✅ All existing test cases ported and passing
- ✅ No regressions in editor experience

### Performance Requirements
- ✅ Startup time: < 100ms (vs ~500ms TypeScript)
- ✅ Hover response: < 10ms (vs ~50ms TypeScript)
- ✅ Memory usage: < 50MB idle (vs ~150MB TypeScript)
- ✅ Large file (10k tokens) parsing: < 100ms
- ✅ Completion response: < 50ms including resolve
- ✅ No performance regressions from TypeScript version

### Quality Requirements
- ✅ Test coverage: > 80%
- ✅ All golden file tests passing
- ✅ Zero critical bugs in production for 2 weeks
- ✅ Performance benchmarks published and meeting goals
- ✅ Clean code passing `golangci-lint`

---

## 🛠️ Development Tools

### Required Tools
- **Go 1.25.3**: Installed natively
- **Tree-sitter CLI**: For testing tree-sitter grammars
- **hyperfine**: Command-line benchmarking tool
- **benchstat**: Go benchmark comparison tool
- **pprof**: CPU and memory profiling
- **golangci-lint**: Comprehensive linting and static analysis
- **gopls**: Go language server (for development)
- **VSCode + Go extension**: IDE setup

### Tree-sitter Setup
```bash
# Install tree-sitter CLI
npm install -g tree-sitter-cli

# Clone tree-sitter-css for testing
git clone https://github.com/tree-sitter/tree-sitter-css.git

# Test queries interactively
tree-sitter query test/queries/highlights.scm
```

### Recommended IDE Setup (VSCode)
```json
// .vscode/settings.json
{
  "go.testFlags": ["-v"],
  "go.coverOnSave": true,
  "go.lintOnSave": "workspace",
  "go.lintTool": "golangci-lint"
}
```

```json
// .vscode/launch.json
{
  "version": "0.2.0",
  "configurations": [
    {
      "name": "Debug LSP Server",
      "type": "go",
      "request": "launch",
      "mode": "debug",
      "program": "${workspaceFolder}/cmd/design-tokens-language-server",
      "env": {
        "DTLS_LOG_LEVEL": "debug"
      }
    },
    {
      "name": "Attach to LSP Server",
      "type": "go",
      "request": "attach",
      "mode": "local",
      "processId": "${command:pickProcess}"
    }
  ]
}
```

---

## 🚨 Risk Mitigation

| Risk | Impact | Mitigation Strategy |
|------|--------|-------------------|
| Tree-sitter Go bindings unstable/incomplete | High | Use official `github.com/tree-sitter/go-tree-sitter` bindings; test early; have fallback plan with regex-based parsing for critical features |
| Performance goals not met | High | Profile early and often; optimize hot paths; use concurrency; benchmark continuously; don't sacrifice correctness for speed |
| Feature gaps discovered late | Medium | Port all tests first to identify features; maintain compatibility matrix; review TypeScript code thoroughly |
| Breaking changes for users | Medium | Semantic versioning; parallel releases during transition; deprecation warnings; comprehensive migration guide |
| Team unfamiliarity with Go | Low | Code reviews; pair programming; reference CEM codebase patterns; Go documentation |
| Memory leaks in long-running server | Medium | Use `pprof` to detect leaks; test with long-running sessions; proper resource cleanup |
| Cross-platform compatibility issues | Low | Test on all platforms (Linux, macOS, Windows); use GitHub Actions for CI; avoid platform-specific code |

---

## 📈 Performance Benchmarking Setup

### Directory Structure
```
tools/
├── lsp-bench/
│   ├── main.go              # LSP benchmark harness
│   ├── client.go            # LSP client implementation
│   ├── operations.go        # Benchmark operations
│   └── results.go           # Result formatting
└── benchmark/
    ├── run-baseline.sh      # TypeScript baseline
    ├── run-go.sh            # Go implementation
    └── compare.sh           # Side-by-side comparison
```

### Benchmark Script (TypeScript Baseline)

```bash
#!/bin/bash
# tools/benchmark/run-baseline.sh
set -e

echo "🔬 Running TypeScript baseline benchmarks..."

# Build benchmark tool if needed
if [ ! -f tools/lsp-bench/lsp-bench ]; then
    echo "Building LSP benchmark tool..."
    cd tools/lsp-bench
    go build -o lsp-bench
    cd ../..
fi

# Ensure TypeScript server is ready
echo "Checking TypeScript server..."
deno check src/server/server.ts

# Run LSP operation benchmarks
echo "Running LSP operation benchmarks..."
./tools/lsp-bench/lsp-bench \
    --server "deno run --allow-all src/server/server.ts" \
    --testdata ./test/testdata \
    --iterations 100 \
    --output baseline-results.json

# Run hyperfine startup benchmark
echo "Running startup benchmark..."
hyperfine --warmup 3 --runs 10 \
    --export-json baseline-startup.json \
    'timeout 1s deno run --allow-all src/server/server.ts < /dev/null' \
    || true

echo "✅ Baseline results saved to baseline-*.json"
```

### Comparison After Go Implementation

```bash
#!/bin/bash
# tools/benchmark/compare.sh

# Run both benchmarks
./tools/benchmark/run-baseline.sh
./tools/benchmark/run-go.sh

# Compare results
echo ""
echo "📊 Performance Comparison:"
echo "=========================="

# Parse and compare JSON results
go run ./tools/benchmark/compare/main.go \
    baseline-results.json \
    go-results.json

# Compare with benchstat if available
if command -v benchstat &> /dev/null; then
    benchstat baseline.txt go.txt
fi
```

---

## 📚 Additional Resources

### Go LSP Development
- [LSP Specification](https://microsoft.github.io/language-server-protocol/)
- [glsp Documentation](https://github.com/tliron/glsp)
- [CEM Project](https://github.com/break-stuff/cem) - Reference implementation

### Tree-sitter Resources
- [Tree-sitter Documentation](https://tree-sitter.github.io/)
- [Tree-sitter CSS Grammar](https://github.com/tree-sitter/tree-sitter-css)
- [Tree-sitter Go Bindings](https://github.com/tree-sitter/go-tree-sitter)

### Design Tokens
- [DTCG Specification](https://design-tokens.github.io/community-group/format/)
- [Style Dictionary](https://amzn.github.io/style-dictionary/)

### Testing & Performance
- [Go Testing Best Practices](https://go.dev/doc/tutorial/add-a-test)
- [pprof Guide](https://go.dev/blog/pprof)
- [hyperfine](https://github.com/sharkdp/hyperfine)

---

## 🎬 Getting Started

### Step 1: Capture Baseline Metrics
```bash
# Run baseline benchmarks on TypeScript implementation
./tools/benchmark/run-baseline.sh

# Review baseline metrics
cat baseline-results.json
```

### Step 2: Set Up Go Project
```bash
# Initialize Go module (using Go 1.25.3)
go mod init bennypowers.dev/dtls

# Install dependencies
go get github.com/tliron/glsp
go get github.com/tree-sitter/go-tree-sitter
go get gopkg.in/yaml.v3
go get github.com/lucasb-eyer/go-colorful
go get github.com/stretchr/testify

# Set up project structure
mkdir -p cmd/design-tokens-language-server
mkdir -p internal/{server,lsp,documents,tokens,workspace,parser}
mkdir -p test/{goldens,testdata,integration}
mkdir -p tools/{lsp-bench,benchmark}
```

### Step 3: Start TDD with Phase 1
```bash
# Create first test
touch internal/server/server_test.go

# Write failing test for LSP initialization
# Implement minimal server to make it pass
# Iterate!
```

---

## 📝 Notes

- **No Regressions**: Performance improvements should never come at the cost of correctness or existing features
- **Tree-sitter First**: Start with tree-sitter from the beginning to match the TypeScript implementation approach
- **Test Everything**: Every feature should have tests before implementation (TDD)
- **Measure Continuously**: Run benchmarks after each phase to catch regressions early
- **Document As You Go**: Update this plan with lessons learned and architecture decisions

---

**Last Updated**: 2025-10-24
**Status**: Planning Phase
**Next Milestone**: Phase 1 - Foundation (Weeks 1-2)
