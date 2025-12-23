# Multi-Schema Design Tokens Language Server Support Plan

**Issue:** #59
**Goal:** Support DTCG 2025.10 stable specification alongside existing editor's draft, with multi-schema LSP features

## Executive Summary

This plan outlines the changes necessary to support the DTCG 2025.10 stable specification alongside the existing editor's draft, with the ability to extend support to future schema versions. The primary goal is to enable **simultaneous multi-schema LSP features** where a single project can consume token libraries with different schema versions.

## Key Breaking Changes in 2025.10 Spec

### 1. **Color Token Format (BREAKING CHANGE)**

**Editor's Draft (current):**
```json
{
  "color": {
    "primary": {
      "$value": "#FF6B35",
      "$type": "color"
    }
  }
}
```

**2025.10 Stable:**
```json
{
  "color": {
    "primary": {
      "$value": {
        "colorSpace": "sRGB",
        "components": [1.0, 0.42, 0.21],
        "alpha": 1.0,
        "hex": "#FF6B35"
      },
      "$type": "color"
    }
  }
}
```

**Impact:** Major parser changes needed. Current implementation expects `$value` to be a string for color tokens. The 2025.10 spec uses a structured object with colorSpace and components.

**Supported Color Spaces in 2025.10:**
- sRGB, sRGB linear
- HSL, HWB
- CIELAB, LCH
- OKLAB, OKLCH
- Display P3, A98 RGB, ProPhoto RGB, Rec 2020
- XYZ-D65, XYZ-D50

### 2. **Reference Syntax Addition**

**Editor's Draft:**
- Only curly braces: `{color.brand.primary}`

**2025.10:**
- Curly braces (still supported): `{color.brand.primary}`
- JSON Pointer: `"$ref": "#/color/brand/primary"`

**Impact:** Parser must detect and handle both reference styles. JSON Pointer enables property-level references, not just whole-token aliasing.

### 3. **Group Extensions (NEW FEATURE)**

**2025.10 Only:**
```json
{
  "baseColors": {
    "red": { "$value": "#FF0000", "$type": "color" }
  },
  "themeColors": {
    "$extends": "#/baseColors",
    "blue": { "$value": "#0000FF", "$type": "color" }
  }
}
```

**Impact:** New group resolution logic required with deep merge semantics.

### 4. **Resolver Module (NEW)**

**2025.10 introduces:**
- `resolutionOrder` array for context-based token variation (light/dark modes)
- Sets and modifiers for multi-context tokens
- Stricter circular reference detection
- Formal alias resolution algorithm

**Impact:** Completely new subsystem needed for resolver support.

### 5. **Color Component `none` Keyword**

**2025.10:**
- Color components can be `none` for interpolation purposes
- Example: `"components": [1.0, "none", 0.21]`

**Impact:** Parser must handle mixed numeric/string arrays.

### 6. **Root Tokens in Groups - `$root` (BREAKING CHANGE)**

**Editor's Draft (current - uses groupMarkers):**
```json
{
  "color": {
    "_": {
      "$type": "color",
      "$value": "#DD0000"
    },
    "light": {
      "$type": "color",
      "$value": "#FF0000"
    },
    "dark": {
      "$type": "color",
      "$value": "#AA0000"
    }
  }
}
```

**Current implementation requires configuration:**
```json
{
  "groupMarkers": ["_", "@", "DEFAULT"]
}
```

**2025.10 Stable (standardized `$root`):**
```json
{
  "color": {
    "$root": {
      "$type": "color",
      "$value": {
        "colorSpace": "srgb",
        "components": [0.867, 0, 0]
      }
    },
    "light": { ... },
    "dark": { ... }
  }
}
```

**Impact:**
- `$root` is a **reserved name** in 2025.10, no configuration needed
- groupMarkers are **draft-only** and should NOT be used with 2025.10+ schemas
- Migration path: `color._` → `color.$root`
- Parser must treat `$root` as a special token name in 2025.10+ schemas

**Important:** The resolver module's `default` is different - it's for context-based resolution (light/dark themes), NOT for group-level tokens.

### 7. **File Format Specification**

**2025.10 adds:**
- Recommended file extensions: `.tokens` or `.tokens.json`
- MIME type: `application/design-tokens+json`
- Optional `$schema` field for version declaration

## Current Implementation Analysis

**Current Schema Support:** Editor's Draft (implicit, no version detection)

**Current Color Handling:**
- Location: `lsp/helpers/css/helpers.go`
- Expects string values: hex, rgb(), rgba(), hsl(), hsla(), named colors
- No color space awareness

**Current Group Markers Handling:**
- Location: `internal/parser/json/parser.go` (lines 94-126)
- Configured via `groupMarkers` in LSP settings (default: `["_", "@", "DEFAULT"]`)
- **Draft-only feature** - should NOT be applied to 2025.10+ schemas

**Current Reference Handling:**
- Pattern: `\{([^}]+)\}` (curly braces only)
- Location: `lsp/methods/textDocument/semanticTokens/semanticTokens.go:17`
- Simple string replacement, no JSON Pointer support

**Current Token Structure:**
- Location: `internal/tokens/types.go`
- No `SchemaVersion` field
- `Value` is always string type

**Configuration:**
- Location: `lsp/types/config.go`
- No schema version specification available

**Parsers:**
- JSON: `internal/parser/json/parser.go`
- YAML: `internal/parser/yaml/parser.go`
- No version-aware logic

## Implementation Plan

Implementation must follow TDD, using fixture projects with files on disk, per the best practices in this repo.

### Phase 0: Error Handling Infrastructure

**Goal:** Establish error handling strategy and types for schema validation

**Strategy Decisions:**
- **Fail-fast:** Schema validation is required - files that cannot be validated are rejected
- **Either/or:** Mixed-format files (combining draft + 2025.10 features) are illegal and will error
- **No conflicts:** `$root` and groupMarkers cannot coexist (enforced by schema validation)

**Tasks:**
1. Create `internal/schema/errors.go`:
   - `ErrSchemaDetectionFailed` - Cannot determine schema version
   - `ErrInvalidSchema` - Schema validation failed
   - `ErrMixedSchemaFeatures` - File contains features from multiple schemas
   - `ErrConflictingRootTokens` - Both `$root` and groupMarkers detected
   - `ErrInvalidColorFormat` - Color value doesn't match schema
   - `ErrCircularReference` - Circular alias or extends detected
   - Each error type includes:
     - Schema version context (if known)
     - File path
     - Location in file (line/column if available)
     - Suggested fix

2. Create `internal/schema/validation.go`:
   - `ValidateSchemaConsistency(fileContent, detectedVersion) error`
   - Check for mixed features (2025.10 color objects + draft-only patterns)
   - Verify all features match declared/detected schema
   - Return detailed errors with fix suggestions

3. Error reporting strategy:
   - Use LSP `window/showMessage` for critical schema errors (blocks loading)
   - Use `window/logMessage` for warnings (informational)
   - Include links to relevant spec sections in error messages
   - Provide actionable error messages (what to fix, how to fix it)

4. Test fixtures:
   - `test/fixtures/errors/mixed-schema-features.json` - Should fail validation
   - `test/fixtures/errors/root-and-markers.json` - Should fail validation
   - `test/fixtures/errors/invalid-color-format.json` - Should fail validation

### Phase 1: Schema Detection & Core Types

**Goal:** Detect and track schema version per token file, define core type system

**Tasks:**
1. Create `internal/schema/version.go`:
   - Define `SchemaVersion` enum (`draft`, `v2025_10`, `unknown`)
   - Version string constants and URL mappings
   - Known schema URLs:
     - `https://www.designtokens.org/schemas/draft.json` → `draft`
     - `https://www.designtokens.org/schemas/2025.10.json` → `v2025_10`

2. Create `internal/schema/detector.go`:
   - `DetectVersion(fileContent, configOverride) (SchemaVersion, error)`
   - Returns error if detection fails (fail-fast)
   - Priority order:
     1. `$schema` field in file root (standard JSON Schema approach)
     2. Per-file config override from LSP settings
     3. Global config default from LSP settings
     4. Duck typing (detect reserved fields/structured formats)
     5. Return `ErrSchemaDetectionFailed` if all methods fail
   - Call `ValidateSchemaConsistency` after detection

3. Duck typing heuristics (conservative approach):
   - Only check for **reserved fields** that are unambiguous:
     - `$ref` field (top-level property) → 2025.10
     - `$extends` in groups → 2025.10
     - `resolutionOrder` → 2025.10
   - Check for **structured color values** (most reliable):
     - `$value` objects with `colorSpace` field → 2025.10
   - **Do NOT check** for `$root` (could be a legitimate token name in draft)
   - If ambiguous, **default to draft** (safer for backward compat)
   - Log warning suggesting explicit `$schema` field

4. Create `internal/schema/file.go`:
   - Define `TokenFile` struct for caching:
     ```go
     type TokenFile struct {
         Path          string
         Content       []byte
         SchemaVersion SchemaVersion  // Detected once at load
         Tokens        []*Token
         LastModified  time.Time
     }
     ```
   - Detect version once at file load, cache result
   - Pass cached version to parser (avoid re-detection)

5. Update `internal/tokens/types.go`:
   - Add `SchemaVersion` field to `Token` struct
   - Add resolver state fields:
     ```go
     type Token struct {
         // ... existing fields
         SchemaVersion SchemaVersion
         RawValue      interface{}    // Original $value before resolution
         ResolvedValue interface{}    // After alias/extends resolution
         IsResolved    bool
     }
     ```
   - Define `ColorValue` interface (idiomatic Go):
     ```go
     // ColorValue represents a color token value in any schema format
     type ColorValue interface {
         ToCSS() string
         SchemaVersion() SchemaVersion
         IsValid() bool
     }

     // StringColorValue for draft schema (hex, rgb(), hsl(), named colors)
     type StringColorValue struct {
         Value  string
         Schema SchemaVersion
     }

     // ObjectColorValue for 2025.10 schema (structured format)
     type ObjectColorValue struct {
         ColorSpace string
         Components []interface{}  // Supports numeric or "none" keyword
         Alpha      *float64       // Optional, defaults to 1.0
         Hex        *string        // Optional, for CSS compatibility
         Schema     SchemaVersion
     }
     ```
   - Implement `ColorValue` interface for both types

6. Update `lsp/types/config.go`:
   - Add `DefaultSchemaVersion` to `ServerConfig` (global default)
   - Add `SchemaVersion` to `TokenFileSpec` (per-file override)
   - **Note:** `groupMarkers` config remains for draft schema backward compatibility
   - groupMarkers are ignored for 2025.10+ schemas (use `$root` instead)

### Phase 2: Color & Reference Parsing Utilities

**Goal:** Create schema-agnostic parsing utilities for colors and references

**Tasks:**
1. Create `internal/parser/common/color.go`:
   - `ParseColorValue(value interface{}, version SchemaVersion) (ColorValue, error)`
   - Handle both string and object formats
   - For draft: parse string values (hex, rgb(), hsl(), named colors)
   - For 2025.10: parse structured color objects
   - Validate color format matches schema version (fail-fast)
   - Return `ErrInvalidColorFormat` if mismatch detected

2. Implement `ColorValue` interface methods:
   - `StringColorValue.ToCSS()` - return string value as-is
   - `ObjectColorValue.ToCSS()` - convert to CSS (use hex if available)
   - `IsValid()` for both types - format validation

3. Create `internal/parser/common/references.go`:
   - `ExtractReferences(content string, version SchemaVersion) ([]Reference, error)`
   - Detect `{...}` curly brace references (both schemas)
   - Detect `$ref` JSON Pointer references (2025.10 only)
   - Return reference type and target path
   - Types:
     ```go
     type ReferenceType int
     const (
         CurlyBraceReference ReferenceType = iota
         JSONPointerReference
     )

     type Reference struct {
         Type   ReferenceType
         Path   string
         Line   int
         Column int
     }
     ```

4. Create `internal/parser/common/root.go`:
   - `ParseRootToken(node *yaml.Node, groupPath []string, version SchemaVersion, groupMarkers []string) (*Token, error)`
   - Handle `$root` for 2025.10+ (reserved name, no config needed)
   - Handle groupMarkers for draft (configured names: `_`, `@`, `DEFAULT`)
   - Validate no conflict between `$root` and groupMarkers
   - Generate consistent CSS variable names regardless of schema

5. Create test fixtures:
   - `test/fixtures/color/draft-colors.json` - String color values
   - `test/fixtures/color/2025-colors.json` - Structured color objects
   - `test/fixtures/color/invalid-draft-structured.json` - Should error
   - `test/fixtures/color/invalid-2025-string.json` - Should error
   - `test/fixtures/references/curly-braces.json`
   - `test/fixtures/references/json-pointers.json`
   - `test/fixtures/root/draft-markers.json`
   - `test/fixtures/root/2025-root.json`

### Phase 3: Parser Versioning

**Goal:** Update parsers to use detected schema version and routing logic

**Tasks:**
1. Update `internal/parser/json/parser.go`:
   - Accept `SchemaVersion` parameter (from cached `TokenFile`)
   - Route to version-specific parsing logic based on schema
   - Track version in extracted tokens
   - **CRITICAL:** Only apply groupMarkers logic for draft schema
   - For 2025.10+: Treat `$root` as reserved token name
   - Skip `$schema` field (don't parse as token property)
   - Use `ParseColorValue()` from Phase 2 for color tokens
   - Use `ParseRootToken()` from Phase 2 for group-level tokens

2. Update `internal/parser/yaml/parser.go`:
   - Same changes as JSON parser
   - Schema-aware parsing and routing

3. Handle reserved fields per schema:
   - Draft: `$type`, `$value`, `$description`, `$extensions`
   - 2025.10: All draft fields + `$schema`, `$ref`, `$extends`, `$root`
   - Skip reserved fields during token property parsing
   - Validate reserved fields are used correctly

4. Update token loading flow:
   - `TokenFile` created with detected schema version
   - Version passed to parser (no re-detection needed)
   - Parser uses version for routing and validation

5. Test fixtures:
   - `test/fixtures/parser/draft-tokens.json` - Full draft schema file
   - `test/fixtures/parser/2025-tokens.json` - Full 2025.10 schema file
   - `test/fixtures/parser/draft-with-schema-field.json` - Explicit `$schema`

### Phase 4a: Basic Alias Resolution

**Goal:** Resolve curly brace and JSON Pointer references (whole tokens only)

**Tasks:**
1. Create `internal/resolver/aliases.go`:
   - `ResolveAliases(tokens []*Token, version SchemaVersion) error`
   - Follow curly brace references: `{color.brand.primary}` (both schemas)
   - Follow JSON Pointer references: `$ref: "#/color/brand/primary"` (2025.10 only)
   - **Limitation:** Whole-token references only (not property-level)
   - Detect circular references (return `ErrCircularReference`)
   - Maintain resolution order (depth-first)
   - Update `ResolvedValue` and set `IsResolved = true`

2. Create `internal/resolver/graph.go`:
   - Build dependency graph for tokens
   - Topological sort for resolution order
   - Cycle detection algorithm

3. Test fixtures:
   - `test/fixtures/resolver/simple-alias.json` - Basic references
   - `test/fixtures/resolver/chained-alias.json` - A → B → C
   - `test/fixtures/resolver/circular-alias.json` - Should error
   - `test/fixtures/resolver/json-pointer-alias.json` - 2025.10 `$ref`

### Phase 4b: JSON Pointer Property References (DEFERRED TO POST-MVP)

**Goal:** Support property-level JSON Pointer references

**Note:** This phase is deferred to post-MVP. Initial implementation only supports whole-token references.

**Future tasks:**
- Parse JSON Pointer fragments: `#/color/primary/$value/colorSpace`
- Navigate to specific properties within tokens
- Handle array element references: `#/color/primary/$value/components/0`

### Phase 5: CSS Output Compatibility

**Goal:** Generate valid CSS from both schema formats

**Tasks:**
1. Create `internal/color/convert.go`:
   - `ToHex(colorValue ObjectColorValue) (string, error)`
   - Convert any color space to sRGB hex for compatibility
   - Support all 14 color spaces from 2025.10 spec
   - Handle `none` keyword in components (use 0 for hex conversion)

2. Update `lsp/helpers/css/helpers.go`:
   - `FormatColorForCSS(colorValue ColorValue) string`
   - For draft (StringColorValue): return value as-is
   - For 2025.10 (ObjectColorValue):
     - If `hex` field present, use it
     - Otherwise, convert using CSS Color Module Level 4 `color()` function
     - Example: `color(srgb 1.0 0.42 0.21 / 1.0)`
     - Fallback to hex conversion for older browser support

3. Maintain CSS variable name consistency:
   - Schema version shouldn't affect variable names
   - `color.$root` → `--color` (2025.10)
   - `color._` → `--color` (draft with groupMarkers)
   - `--color-primary` works for both formats

4. Test fixtures:
   - `test/fixtures/css/draft-output.json` - Draft schema CSS generation
   - `test/fixtures/css/2025-output.json` - 2025.10 schema CSS generation
   - `test/fixtures/css/color-spaces.json` - All 14 color spaces

### Phase 6: LSP Features Per Schema

**Goal:** Provide schema-appropriate LSP features

**Tasks:**
1. **Hover** (`lsp/methods/textDocument/hover/`):
   - Show detected schema version
   - Link to appropriate spec version docs:
     - Draft: `https://www.designtokens.org/TR/drafts/format/`
     - 2025.10: `https://www.designtokens.org/tr/2025.10/`
   - Display color values with format info
   - Show color space for 2025.10 tokens
   - Show resolved value if token is an alias

2. **Completion** (`lsp/methods/textDocument/completion/`):
   - Suggest schema-appropriate property names
   - For 2025.10 color objects: suggest `colorSpace`, `components`, `alpha`, `hex`
   - For draft: continue current behavior
   - Suggest reference syntaxes based on schema:
     - Draft: only `{...}` curly braces
     - 2025.10: both `{...}` and `$ref`

3. **Diagnostics**:
   - Show schema validation errors via `window/showMessage` (error level)
   - Log info via `window/logMessage` (info level)
   - Include spec links in error messages
   - Actionable error messages with fix suggestions

4. **Semantic Tokens** (`lsp/methods/textDocument/semanticTokens/`):
   - Update regex to detect `$ref` fields (2025.10)
   - Highlight JSON Pointer paths
   - Differentiate reference types in token highlighting
   - Highlight `$root` as reserved keyword (2025.10)

5. **Go to Definition**:
   - Navigate curly brace references (both schemas)
   - Navigate JSON Pointer references (2025.10)
   - Navigate `$extends` references (deferred to Phase 8)

6. Test fixtures:
   - `test/fixtures/lsp/hover-draft.json`
   - `test/fixtures/lsp/hover-2025.json`
   - `test/fixtures/lsp/completion-draft.json`
   - `test/fixtures/lsp/completion-2025.json`

### Phase 7: Multi-Schema Workspace Support

**Goal:** Handle projects with multiple token libraries using different schemas

**Tasks:**
1. Update `internal/tokens/manager.go`:
   - Track schema version per token source file
   - Support tokens with same name but different schemas
   - Qualify tokens by source file when ambiguous
   - Track multiple `TokenFile` instances

2. Update token loading:
   - Detect version independently for each configured token file
   - Log detected versions via `window/logMessage`
   - Allow mixed-version workspaces
   - Each file maintains its own schema context

3. Cross-schema reference handling:
   - Allow references between different schema versions
   - Normalize values when crossing schema boundaries:
     - 2025.10 color object → CSS string for draft consumer
     - Draft color string → structured object for 2025.10 consumer (if possible)
   - Log warning when reference crosses schemas

4. Test fixtures:
   - `test/fixtures/multi-schema/mixed-workspace/`:
     - `draft-tokens.json` - Draft schema
     - `2025-tokens.json` - 2025.10 schema
     - Both loaded simultaneously
   - `test/fixtures/multi-schema/cross-reference/`:
     - Draft file referencing 2025.10 tokens
     - Verify value normalization

5. Integration tests:
   - Test LSP `initialize` with multi-schema workspace
   - Test `textDocument/didOpen` for each schema type
   - Test hover/completion across schemas
   - Verify no schema cross-contamination

### Phase 8: Group Extensions (2025.10)

**Goal:** Support `$extends` for group inheritance

**Tasks:**
1. Create `internal/resolver/extends.go`:
   - `ResolveGroupExtensions(tokens []*Token) error`
   - Deep merge algorithm for group inheritance
   - Circular reference detection
   - Only applies to 2025.10 schema

2. Deep merge semantics:
   - Child group tokens override parent tokens
   - Nested groups are merged recursively
   - `$extends` can point to any group via JSON Pointer

3. Integration:
   - Call after alias resolution (Phase 4a)
   - Update token tree structure
   - Preserve both pre- and post-extension state

4. Test fixtures:
   - `test/fixtures/extends/simple.json` - Basic group extension
   - `test/fixtures/extends/nested.json` - Nested group extensions
   - `test/fixtures/extends/circular.json` - Should error
   - `test/fixtures/extends/override.json` - Child overrides parent

### Phase 9: Future-Proofing

**Goal:** Easy addition of future schema versions

**Tasks:**
1. Create schema handler interface:
   ```go
   type SchemaHandler interface {
       ParseToken(node *yaml.Node) (*Token, error)
       FormatForCSS(token *Token) string
       ValidateToken(token *Token) []Diagnostic
   }
   ```

2. Create schema registry:
   - `internal/schema/registry.go`
   - Register handlers by version
   - Look up handler during parsing

3. Documentation:
   - Document process for adding new schema versions
   - Provide example handler implementation
   - Version migration guide for users

## POST-MVP Features

### Phase 10: Resolution Order & Context Resolution (2025.10)

**Goal:** Support `resolutionOrder` for context-based token resolution (light/dark modes)

**Note:** This is a complex feature deferred to post-MVP.

**Tasks:**
1. Create `internal/resolver/context.go`:
   - Parse `resolutionOrder` arrays
   - Support sets and modifiers
   - Enable context-based token resolution
   - Resolve tokens differently based on context (e.g., light vs dark mode)

2. Integration with LSP:
   - Allow user to switch resolution context
   - Show different token values based on context
   - Preview tokens in multiple contexts

### Phase 11: Auto-Discovery

**Goal:** Automatically discover `*.tokens.json` files in workspace

**Note:** Orthogonal to multi-schema support, can be implemented separately.

**Tasks:**
1. Watch for `*.tokens.json` and `*.tokens` files:
   - Implement file watcher in LSP server
   - Auto-load discovered files

2. Configuration:
   - Add `autoDiscover` boolean to config (default: false)
   - Add `autoDiscoverPattern` string array (default: `["**/*.tokens.json", "**/*.tokens"]`)
   - Exclude patterns (node_modules, etc.)

3. Workspace folders:
   - Scan all workspace folders on initialization
   - Watch for new files during session

4. Merge with configured files:
   - Configured files take precedence
   - Auto-discovered files use global prefix
   - groupMarkers only applied to draft-schema discovered files
   - Log discovered files via `window/logMessage`

## Testing Requirements (TDD)

### Test Fixtures Required

**Single-Schema Fixtures:**
1. `test/fixtures/schema/draft/` - Editor's draft format tokens
2. `test/fixtures/schema/2025.10/` - Stable 2025.10 format tokens

**Multi-Schema Workspace Fixtures:**
3. `test/fixtures/multi-schema/project-mixed/`:
   - `library-a/tokens.json` - Draft schema with `$schema` field
   - `library-b/design.tokens.json` - 2025.10 schema
   - `package.json` - Config referencing both libraries
   - `app.css` - CSS file using tokens from both libraries

4. `test/fixtures/multi-schema/cross-references/`:
   - `base.tokens.json` - 2025.10 base tokens
   - `theme.tokens.json` - Draft tokens referencing base
   - Test cross-schema reference resolution

**Version Detection Fixtures:**
5. `test/fixtures/detection/explicit-schema.json` - Has `$schema` field
6. `test/fixtures/detection/duck-type-2025.json` - 2025.10 without `$schema`
7. `test/fixtures/detection/duck-type-draft.json` - Draft without `$schema`
8. `test/fixtures/detection/ambiguous.json` - Requires user input

**Color Format Fixtures:**
9. `test/fixtures/color/draft-formats.json` - Hex, RGB, HSL, named
10. `test/fixtures/color/2025-structured.json` - All 14 color spaces
11. `test/fixtures/color/2025-none-keyword.json` - Components with `none`

**Reference Syntax Fixtures:**
12. `test/fixtures/references/curly-braces.json` - `{path.to.token}`
13. `test/fixtures/references/json-pointer.json` - `$ref` fields
14. `test/fixtures/references/mixed.json` - Both syntaxes

**Group Extensions Fixtures:**
15. `test/fixtures/groups/extends.json` - `$extends` with deep merge
16. `test/fixtures/groups/circular.json` - Should error on circular extends

**Root Token Fixtures:**
17. `test/fixtures/root/draft-group-markers.json` - Draft schema with `_`, `@`, `DEFAULT`
18. `test/fixtures/root/2025-root.json` - 2025.10 schema with `$root`
19. `test/fixtures/root/mixed-workspace/` - Draft with markers + 2025.10 with `$root`

**Resolver Fixtures:**
20. `test/fixtures/resolver/resolution-order.json` - Sets and modifiers
21. `test/fixtures/resolver/circular-alias.json` - Should error

### Test Coverage Requirements

**Unit Tests:**
- Schema detection (all methods)
- Color value parsing (both formats)
- Reference extraction (both syntaxes)
- CSS output (both schemas)
- Group extension resolution
- Alias resolution
- Duck typing heuristics
- `$root` vs groupMarkers handling (schema-aware)
- CSS variable generation consistency (both `$root` and groupMarkers produce same vars)

**Integration Tests:**
- Load multi-schema workspace
- LSP features with mixed schemas
- Cross-schema references
- Hover showing correct schema docs
- Completion suggesting schema-appropriate properties

**TDD Approach:**
1. Write failing tests for each fixture project FIRST
2. Implement minimum code to pass tests
3. Refactor with tests green
4. Add edge case tests
5. Ensure no regressions in existing draft support

## Files Requiring Changes

### Core Implementation (New/Modified)

**New Files (by Phase):**

**Phase 0:**
- `internal/schema/errors.go` - Schema error types
- `internal/schema/validation.go` - Schema consistency validation

**Phase 1:**
- `internal/schema/version.go` - Version enum and constants
- `internal/schema/detector.go` - Version detection logic
- `internal/schema/file.go` - TokenFile caching structure

**Phase 2:**
- `internal/parser/common/color.go` - Color parsing utilities
- `internal/parser/common/references.go` - Reference extraction
- `internal/parser/common/root.go` - Root token and group marker handling

**Phase 4a:**
- `internal/resolver/aliases.go` - Alias resolution
- `internal/resolver/graph.go` - Dependency graph for resolution order

**Phase 5:**
- `internal/color/convert.go` - Color space conversion to hex/CSS

**Phase 8:**
- `internal/resolver/extends.go` - Group extension resolution (2025.10)

**Phase 9:**
- `internal/schema/registry.go` - Schema handler registry

**POST-MVP:**
- `internal/resolver/context.go` - Resolution order handling (Phase 10)

**Modified Files (by Phase):**

**Phase 1:**
- `internal/tokens/types.go` - Add SchemaVersion, ColorValue interface, resolver state
- `lsp/types/config.go` - Add DefaultSchemaVersion config

**Phase 3:**
- `internal/parser/json/parser.go` - Version-aware parsing & routing
- `internal/parser/yaml/parser.go` - Version-aware parsing & routing
- `lsp/token_loader.go` - Create TokenFile, detect schema, pass to parser

**Phase 5:**
- `lsp/helpers/css/helpers.go` - Multi-format color conversion

**Phase 6:**
- `lsp/methods/textDocument/hover/*.go` - Schema-aware hover
- `lsp/methods/textDocument/completion/*.go` - Schema-aware completion
- `lsp/methods/textDocument/semanticTokens/*.go` - Detect $ref, highlight JSON Pointers
- `lsp/methods/textDocument/definition/*.go` - JSON Pointer navigation

**Phase 7:**
- `internal/tokens/manager.go` - Track schema per file, multi-schema support

### Test Files (New)

**Unit Tests (by Phase):**

**Phase 0:**
- `internal/schema/errors_test.go`
- `internal/schema/validation_test.go`

**Phase 1:**
- `internal/schema/detector_test.go`
- `internal/schema/version_test.go`
- `internal/schema/file_test.go`

**Phase 2:**
- `internal/parser/common/color_test.go`
- `internal/parser/common/references_test.go`
- `internal/parser/common/root_test.go`

**Phase 3:**
- `internal/parser/json/parser_versioned_test.go`
- `internal/parser/yaml/parser_versioned_test.go`

**Phase 4a:**
- `internal/resolver/aliases_test.go`
- `internal/resolver/graph_test.go`

**Phase 5:**
- `internal/color/convert_test.go`
- `lsp/helpers/css/helpers_test.go` (extended)

**Phase 7:**
- `internal/tokens/manager_multi_schema_test.go`

**Phase 8:**
- `internal/resolver/extends_test.go`

**Integration Tests:**
- `lsp/multi_schema_workspace_test.go` - Multi-schema workspace scenarios
- `lsp/cross_schema_reference_test.go` - Cross-schema references
- `lsp/color_formats_integration_test.go` - End-to-end color parsing and CSS output
- `lsp/lsp_features_schema_test.go` - Hover, completion, semantic tokens per schema

**Test Fixtures:**
- All fixtures created per-phase as listed in phase descriptions above

## Migration Strategy

### For Existing Users

**No Breaking Changes:**
- Editor's draft remains default for files without `$schema`
- Existing token files continue to work
- CSS output remains compatible

**Gradual Adoption:**
- Users can add `$schema` field to opt into 2025.10
- Per-file configuration allows incremental migration
- Mixed-schema projects supported from day one

**Clear Documentation:**
- Migration guide: draft → 2025.10
  - Color format: string → structured object
  - Group markers: `_`/`@`/`DEFAULT` → `$root`
  - Reference syntax: add `$ref` support
- Schema version specification best practices
- Color format conversion examples
- `$root` vs groupMarkers usage guide

### For New Users

**Recommended Defaults:**
- Use 2025.10 for new projects
- Always specify `$schema` field
- Use `.tokens.json` or `.tokens` extension

## Success Criteria

**MVP (Phases 0-9):**
1. ✅ Schema detection works via `$schema` field, config, and duck typing
2. ✅ Fail-fast validation rejects invalid/mixed-schema files
3. ✅ Parse both draft and 2025.10 color formats correctly
4. ✅ Resolve curly brace and JSON Pointer aliases (whole tokens)
5. ✅ Generate valid CSS from both schema formats
6. ✅ Load workspace with both draft and 2025.10 token files simultaneously
7. ✅ Correct hover/completion/highlighting for each schema version
8. ✅ CSS variables work for both color formats
9. ✅ Cross-schema references resolve correctly with value normalization
10. ✅ Support group extensions (`$extends`) for 2025.10 schema
11. ✅ All existing tests pass (no regressions)
12. ✅ TDD fixtures cover all major scenarios

**Post-MVP (Phases 10-11):**
- ✅ Resolution order and context-based token resolution (Phase 10)
- ✅ Auto-discover `*.tokens.json` files in workspace (Phase 11)

## Revised Phase Summary

**Core Phases (MVP):**
- **Phase 0:** Error Handling Infrastructure
- **Phase 1:** Schema Detection & Core Types
- **Phase 2:** Color & Reference Parsing Utilities
- **Phase 3:** Parser Versioning
- **Phase 4a:** Basic Alias Resolution
- **Phase 4b:** JSON Pointer Property References (DEFERRED)
- **Phase 5:** CSS Output Compatibility
- **Phase 6:** LSP Features Per Schema
- **Phase 7:** Multi-Schema Workspace Support
- **Phase 8:** Group Extensions (2025.10)
- **Phase 9:** Future-Proofing

**Post-MVP:**
- **Phase 10:** Resolution Order & Context Resolution
- **Phase 11:** Auto-Discovery

## Key Design Decisions

### `$root` vs groupMarkers

**Decision:** Schema-aware handling
- **2025.10+ schemas:** Use `$root` (reserved name, no configuration)
- **Draft schema:** Use groupMarkers (configured names: `_`, `@`, `DEFAULT`)
- **Rationale:** Maintain backward compatibility while adopting standard

**CSS Variable Output:**
- `color.$root` → `--color` (2025.10)
- `color._` → `--color` (draft)
- Both produce identical CSS variable names for compatibility

**Migration Path:**
1. Add `$schema` field to token files
2. Rename groupMarker tokens to `$root`
3. Remove groupMarkers from config

## Key Design Decisions (Resolved)

### Error Handling Strategy

**Q:** Fail-fast vs. best-effort on schema detection failures?
**A:** **Fail-fast** - Schema validation is required. Files that cannot be validated are rejected.

**Q:** How to handle mixed-format files?
**A:** **Either/or** - Mixed-format files (combining draft + 2025.10 features) are illegal and will error.

**Q:** What to do when `$root` conflicts with groupMarkers?
**A:** **No conflicts** - Because of fail-fast validation, there should never be a combination of `$root` and groupMarkers in a valid file.

### Open Questions (Remaining)

1. Should we support automatic conversion between schemas (e.g., auto-migrate draft → 2025.10)?
   - **Recommendation:** No, keep it explicit. Users should migrate manually.

2. How to handle future 2025.10 features not in draft (show as unsupported in draft mode)?
   - **Recommendation:** Show error if draft file uses 2025.10-only features.

3. CSS Color Module Level 4 `color()` function support - when to use vs hex fallback?
   - **Recommendation:** Provide both in hover/completion, default to hex for compatibility.

4. Should we validate against official JSON Schema definitions if available?
   - **Recommendation:** Yes, if schemas are published and accessible.

## References

- **2025.10 Spec:** https://www.designtokens.org/tr/2025.10/
- **Editor's Draft:** https://www.designtokens.org/TR/drafts/format/
- **Format Module:** https://www.designtokens.org/tr/2025.10/format/
- **Color Module:** https://www.designtokens.org/tr/2025.10/color/
- **Resolver Module:** https://www.designtokens.org/tr/2025.10/resolver/
