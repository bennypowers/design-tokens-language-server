# Add `$extends` resolution (DTCG 2025.10)

## Context

asimonim's `resolver.ResolveGroupExtensions()` resolves `$extends`
relationships in DTCG 2025.10 token files. It creates copies of inherited
tokens with updated paths and names. Child tokens override inherited tokens
with the same terminal name.

This function is available since asimonim v0.0.3 but dtls doesn't call it yet.

## Implementation

### Call `ResolveGroupExtensions` after parsing

**File:** `lsp/tokens.go`

After each `parser.Parse()` call, add `resolver.ResolveGroupExtensions()`.
It's a no-op for Draft schema tokens and only activates for v2025.10 files
that use `$extends`.

Three call sites to update:

1. `loadTokenFileInternal` (~line 103, after `parser.Parse`):
   ```go
   parsedTokens, err = resolver.ResolveGroupExtensions(parsedTokens, data)
   if err != nil {
       return fmt.Errorf("failed to resolve $extends in %s: %w", filePath, err)
   }
   ```

2. `LoadTokensFromJSON` (~line 157, after `parser.Parse`)

3. `LoadTokensFromDocumentContent` (~line 204, after `parser.Parse`)

If `parseAndAddTokens` helper exists (from the network fallback PR), add it
there instead -- single call site.

Import: `"bennypowers.dev/asimonim/resolver"` (already used in
`configuration.go`).

### Tests

Add test fixtures in `lsp/testdata/extends/` with v2025.10 token files:

- Basic `$extends` inheritance
- Override behavior (child token overrides inherited token)
- Chained `$extends` (A extends B extends C)
- Circular `$extends` detection (expect error)

Verify inherited tokens appear with correct paths, names, and values.
Verify `DefinitionURI` and `FilePath` are set correctly on inherited tokens.
