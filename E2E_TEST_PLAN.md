# Integration Test Plan for LSP Server

## Overview

With LSP methods now isolated into dedicated packages, **most business logic can be unit tested** with ServerContext mocks. Integration tests should focus on:

1. **Server lifecycle** - Full initialize → initialized → shutdown flow
2. **Document lifecycle** - Open → change → diagnostics → close flow
3. **Workspace management** - Configuration, file watching, token reloading

## Current Coverage Status

### ✅ Well Unit-Tested (80%+ coverage)
- **Internal packages**: 81-99% coverage (business logic)
- **Lifecycle handlers**: 94.6% coverage (new unit tests)
- **Semantic tokens**: 89.4% coverage
- **Code actions**: 71.7% coverage

### 🎯 Unit Test Coverage Targets
- **Feature handlers** (hover, completion, definition, references, documentColor): 70%+ via unit tests with mocks
  - Currently: 15-25% (only smoke tests)
  - Approach: Mock ServerContext like lifecycle tests

### 🔧 Integration Test Targets
- **Server lifecycle**: Full flow testing
- **Document lifecycle**: Open/change/close with diagnostics
- **Workspace management**: Config, file watching, token loading

---

## Integration Test Structure

```
test/
├── integration/
│   ├── server_lifecycle_test.go    # Initialize → initialized → shutdown
│   ├── document_lifecycle_test.go  # Document open → change → diagnostics → close
│   ├── workspace_test.go           # Config, file watchers, token loading
│   └── testutil/
│       ├── server.go               # Test server setup
│       └── fixtures.go             # Token/CSS file loaders
└── fixtures/
    ├── tokens/
    │   ├── basic.json
    │   ├── colors.json
    │   └── with-references.json
    └── css/
        ├── simple.css
        └── fallbacks.css
```

---

## Integration Test Scenarios

### 1. Server Lifecycle (`test/integration/server_lifecycle_test.go`)

**Goal**: Test full LSP server initialization flow

**Scenarios**:
- ✅ Initialize with RootURI → sets workspace root correctly
- ✅ Initialize with RootPath → converts to URI correctly
- ✅ Initialized notification → loads tokens from config
- ✅ Initialized notification → registers file watchers
- ✅ Initialized error handling → continues on token load failure
- ✅ Shutdown → cleans up CSS parser pool
- ✅ Full flow: initialize → initialized → work → shutdown

**Example**:
```go
func TestServerLifecycle_FullFlow(t *testing.T) {
    server := NewTestServer(t)
    glspCtx := &glsp.Context{}

    // 1. Initialize
    rootURI := "file:///workspace"
    initResult, err := server.Initialize(glspCtx, &protocol.InitializeParams{
        RootURI: &rootURI,
    })
    require.NoError(t, err)
    require.NotNil(t, initResult)

    // Verify capabilities
    assert.Contains(t, initResult.Capabilities, "hoverProvider")

    // 2. Initialized
    err = server.Initialized(glspCtx, &protocol.InitializedParams{})
    require.NoError(t, err)

    // 3. Do some work (open document, get diagnostics)
    // ...

    // 4. Shutdown
    err = server.Shutdown(glspCtx)
    require.NoError(t, err)
}
```

### 2. Document Lifecycle (`test/integration/document_lifecycle_test.go`)

**Goal**: Test document management and diagnostic publishing

**Scenarios**:
- ✅ DidOpen → document tracked, diagnostics published
- ✅ DidChange (full document) → content updated, diagnostics re-published
- ✅ DidChange (incremental) → content patched correctly
- ✅ DidClose → document removed from tracking
- ✅ Multiple documents → managed independently
- ✅ Document with unknown token reference → publishes `unknown-reference` diagnostic
- ✅ Document with incorrect fallback → publishes `incorrect-fallback` diagnostic

**Example**:
```go
func TestDocumentLifecycle_DiagnosticsFlow(t *testing.T) {
    server := setupServerWithTokens(t)
    glspCtx := &glsp.Context{}

    // Track published diagnostics
    diagnosticsPublished := make(map[string][]protocol.Diagnostic)
    server.onPublishDiagnostics = func(uri string, diags []protocol.Diagnostic) {
        diagnosticsPublished[uri] = diags
    }

    // 1. Open document with error
    uri := "file:///test.css"
    err := server.DidOpen(glspCtx, &protocol.DidOpenTextDocumentParams{
        TextDocument: protocol.TextDocumentItem{
            URI:        uri,
            LanguageID: "css",
            Version:    1,
            Text:       ".btn { color: var(--unknown-token); }",
        },
    })
    require.NoError(t, err)

    // Verify diagnostics were published
    assert.Len(t, diagnosticsPublished[uri], 1)
    assert.Equal(t, "unknown-reference", diagnosticsPublished[uri][0].Code)

    // 2. Fix the error via DidChange
    err = server.DidChange(glspCtx, &protocol.DidChangeTextDocumentParams{
        TextDocument: protocol.VersionedTextDocumentIdentifier{
            TextDocumentIdentifier: protocol.TextDocumentIdentifier{URI: uri},
            Version:                2,
        },
        ContentChanges: []interface{}{
            protocol.TextDocumentContentChangeEvent{
                Text: ".btn { color: var(--color-primary); }",
            },
        },
    })
    require.NoError(t, err)

    // Verify diagnostics cleared
    assert.Empty(t, diagnosticsPublished[uri])

    // 3. Close document
    err = server.DidClose(glspCtx, &protocol.DidCloseTextDocumentParams{
        TextDocument: protocol.TextDocumentIdentifier{URI: uri},
    })
    require.NoError(t, err)

    // Verify document removed
    assert.Nil(t, server.Document(uri))
}
```

### 3. Workspace Management (`test/integration/workspace_test.go`)

**Goal**: Test configuration, file watching, and token loading

**Scenarios**:
- ✅ Load tokens from config (tokensFiles glob patterns)
- ✅ Token prefix configuration → tokens accessible with prefix
- ✅ Group markers configuration → filters tokens correctly
- ✅ File watcher: token file changed → reloads tokens and republishes diagnostics
- ✅ File watcher: token file deleted → removes tokens and republishes diagnostics
- ✅ File watcher: non-token file changed → ignored
- ✅ Multiple token files → all loaded and accessible

**Example**:
```go
func TestWorkspace_FileWatcherReloadsTokens(t *testing.T) {
    server := setupServerWithConfig(t, WorkspaceConfig{
        TokensFiles: []string{"tokens/*.json"},
    })

    // Load initial tokens
    initialCount := server.TokenCount()
    assert.Equal(t, 5, initialCount)

    // Open a document using a token
    uri := "file:///test.css"
    server.DidOpen(ctx, &protocol.DidOpenTextDocumentParams{
        TextDocument: protocol.TextDocumentItem{
            URI:  uri,
            Text: ".btn { color: var(--color-primary); }",
        },
    })

    // No diagnostics (token exists)
    diagnostics := getDiagnostics(server, uri)
    assert.Empty(t, diagnostics)

    // Simulate token file change (remove --color-primary)
    updateTokenFile(t, "tokens/colors.json", removeToken("color.primary"))

    server.DidChangeWatchedFiles(ctx, &protocol.DidChangeWatchedFilesParams{
        Changes: []protocol.FileEvent{
            {URI: "file:///workspace/tokens/colors.json", Type: protocol.FileChangeTypeChanged},
        },
    })

    // Verify tokens reloaded
    assert.Equal(t, 4, server.TokenCount())

    // Verify diagnostics republished for affected documents
    diagnostics = getDiagnostics(server, uri)
    assert.Len(t, diagnostics, 1)
    assert.Equal(t, "unknown-reference", diagnostics[0].Code)
}
```

---

## Unit Test Strategy for Feature Handlers

**Pattern**: Same as lifecycle tests - mock ServerContext

### Hover (`lsp/methods/textDocument/hover/hover_test.go`)
- Mock ServerContext with pre-loaded tokens
- Test hover over CSS variable → returns token info
- Test hover over deprecated token → includes deprecation warning
- Test hover over composite token → formats correctly
- Test hover at invalid position → returns null

### Completion (`lsp/methods/textDocument/completion/completion_test.go`)
- Mock ServerContext with tokens
- Test completion inside var() → returns all tokens
- Test completion with prefix filter → filters correctly
- Test CompletionResolve → adds documentation
- Test completion outside var() → returns empty

### Definition (`lsp/methods/textDocument/definition/definition_test.go`)
- Mock ServerContext with tokens
- Test definition from CSS var() → returns token location
- Test definition for aliased token → returns source location
- Test definition for unknown token → returns empty

### References (`lsp/methods/textDocument/references/references_test.go`)
- Mock ServerContext with multiple documents
- Test find references for token → returns all usages
- Test find references for unused token → returns empty

### Code Actions (`lsp/methods/textDocument/codeAction/codeAction_test.go`)
- Already at 71.7% - add a few more edge cases

### Document Color (`lsp/methods/textDocument/documentColor/documentColor_test.go`)
- Mock ServerContext with color tokens
- Test extract colors from var() → returns ColorInformation
- Test color presentation → converts formats

---

## Implementation Plan

### Phase 1: Integration Tests (High Priority)
- [ ] `test/integration/testutil/` - Server setup helpers
- [ ] `test/integration/server_lifecycle_test.go` - 7 scenarios
- [ ] `test/integration/document_lifecycle_test.go` - 7 scenarios
- [ ] `test/integration/workspace_test.go` - 7 scenarios

### Phase 2: Feature Handler Unit Tests (Medium Priority)
- [ ] `lsp/methods/textDocument/hover/hover_test.go` - 5-7 scenarios
- [ ] `lsp/methods/textDocument/completion/completion_test.go` - 5-7 scenarios
- [ ] `lsp/methods/textDocument/definition/definition_test.go` - 4-5 scenarios
- [ ] `lsp/methods/textDocument/references/references_test.go` - 3-4 scenarios
- [ ] `lsp/methods/textDocument/documentColor/documentColor_test.go` - 3-4 scenarios

### Success Criteria
- ✅ Integration tests cover server/document/workspace lifecycle
- ✅ Feature handlers reach 70%+ coverage via unit tests
- ✅ Overall LSP package coverage > 60%
- ✅ No regressions in existing tests

### Estimated Effort
- **Integration tests**: 1-2 days (21 scenarios, reusable test utilities)
- **Unit tests**: 2-3 days (25-30 scenarios across 5 handlers)
- **Total**: 3-5 days

---

## Why This Approach?

### Unit Tests for Feature Handlers
- ✅ **Fast**: No server setup overhead
- ✅ **Focused**: Test handler logic in isolation
- ✅ **Easy to maintain**: Mock dependencies are explicit
- ✅ **High coverage**: Can test edge cases easily

### Integration Tests for Lifecycle
- ✅ **Realistic**: Test actual server flow
- ✅ **Catches integration bugs**: Config parsing, file watching, etc.
- ✅ **Documents behavior**: Shows how features work together

### What We're NOT Testing (Already Tested)
- ❌ Token parsing - tested in `internal/parser/*` (80-92%)
- ❌ Token resolution - tested in `internal/tokens` (91.8%)
- ❌ Document management - tested in `internal/documents` (96.5%)
- ❌ CSS variable extraction - tested in `internal/parser/css` (80.2%)

We're testing the **integration points** and **LSP protocol handling**, not the business logic.
