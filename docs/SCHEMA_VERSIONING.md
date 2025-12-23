# Schema Versioning Guide

This guide explains how the Design Tokens Language Server supports multiple DTCG schema versions and how to add support for new versions.

## Architecture Overview

The language server uses a registry-based architecture to support multiple schema versions simultaneously:

1. **Schema Detection** (`internal/schema/detector.go`): Automatically detects the schema version from `$schema` field, configuration, or duck typing
2. **Schema Handlers** (`internal/schema/handler.go`): Version-specific logic for parsing, validation, and formatting
3. **Schema Registry** (`internal/schema/registry.go`): Manages registered handlers and provides lookup by version

## Supported Schema Versions

Currently supported:
- **Editor's Draft** (`https://www.designtokens.org/schemas/draft.json`)
- **2025.10 Stable** (`https://www.designtokens.org/schemas/2025.10.json`)

## Adding a New Schema Version

Follow these steps to add support for a new DTCG schema version:

### 1. Define the Schema Version

Add a new constant to `internal/schema/version.go`:

```go
const (
    Unknown    SchemaVersion = "unknown"
    Draft      SchemaVersion = "draft"
    V2025_10   SchemaVersion = "2025.10"
    V2026_01   SchemaVersion = "2026.01"  // NEW VERSION
)

// Map schema URLs to versions
var schemaURLs = map[string]SchemaVersion{
    "https://www.designtokens.org/schemas/draft.json":   Draft,
    "https://www.designtokens.org/schemas/2025.10.json": V2025_10,
    "https://www.designtokens.org/schemas/2026.01.json": V2026_01,  // NEW URL
}
```

### 2. Implement the SchemaHandler Interface

Create a new handler type in `internal/schema/registry.go`:

```go
// V2026_01SchemaHandler implements SchemaHandler for the 2026.01 schema
type V2026_01SchemaHandler struct{}

func (h *V2026_01SchemaHandler) Version() SchemaVersion {
    return V2026_01
}

func (h *V2026_01SchemaHandler) ValidateTokenNode(node *yaml.Node) error {
    // Add version-specific validation logic
    // Check for required fields, validate structure, etc.
    return nil
}

func (h *V2026_01SchemaHandler) FormatColorForCSS(colorValue interface{}) string {
    // Implement color formatting for this version
    // Handle new color formats introduced in this version
    return ""
}

func (h *V2026_01SchemaHandler) SupportsFeature(feature string) bool {
    // Declare which features this version supports
    switch feature {
    case "curly-brace-references", "json-pointer", "extends", "root":
        return true
    case "resolution-order":  // Example new feature
        return true
    default:
        return false
    }
}
```

### 3. Register the Handler

Add the handler to the registry initialization in `NewRegistry()`:

```go
func NewRegistry() *Registry {
    r := &Registry{
        handlers: make(map[SchemaVersion]SchemaHandler),
    }

    // Register built-in handlers
    r.Register(&DraftSchemaHandler{})
    r.Register(&V2025_10SchemaHandler{})
    r.Register(&V2026_01SchemaHandler{})  // NEW HANDLER

    return r
}
```

### 4. Update Schema Detection

Add duck typing heuristics to `internal/schema/detector.go` if the new version introduces unique features:

```go
func detectByFeatures(content []byte) (SchemaVersion, error) {
    // Check for version-specific reserved fields
    if hasField(content, "newFeature2026") {
        return V2026_01, nil
    }

    // Existing detection logic...
}
```

### 5. Update Parser Logic

If the new version introduces breaking changes to token structure, update the parser in `internal/parser/json/parser.go`:

```go
func (p *Parser) extractTokensWithSchemaVersion(...) error {
    // Route to version-specific logic
    switch version {
    case schema.Draft:
        // Draft-specific parsing
    case schema.V2025_10:
        // 2025.10-specific parsing
    case schema.V2026_01:  // NEW VERSION
        // 2026.01-specific parsing
    }
}
```

### 6. Add Test Fixtures

Create test fixtures in `test/fixtures/schema/2026.01/`:

```
test/fixtures/schema/2026.01/
├── basic-tokens.json
├── new-features.json
└── migration-example.json
```

Example fixture:
```json
{
  "$schema": "https://www.designtokens.org/schemas/2026.01.json",
  "color": {
    "primary": {
      "$value": { ... },
      "$type": "color"
    }
  }
}
```

### 7. Write Tests

Add tests in `internal/schema/registry_test.go`:

```go
func TestV2026_01SchemaHandler_SupportsFeature(t *testing.T) {
    handler := &schema.V2026_01SchemaHandler{}

    assert.True(t, handler.SupportsFeature("resolution-order"))
    assert.True(t, handler.SupportsFeature("json-pointer"))
    // Test all features
}

func TestV2026_01SchemaHandler_FormatColorForCSS(t *testing.T) {
    handler := &schema.V2026_01SchemaHandler{}

    // Test color formatting for this version
}
```

### 8. Update Documentation

- Update README.md with new supported version
- Document breaking changes and migration path
- Add examples for new features

## Feature Flags

The `SupportsFeature()` method allows conditional feature activation:

```go
handler, _ := registry.Get(schemaVersion)
if handler.SupportsFeature("json-pointer") {
    // Enable JSON Pointer references
}
```

Common features to check:
- `curly-brace-references`: `{token.path}` syntax
- `json-pointer`: `$ref` fields with JSON Pointers
- `extends`: `$extends` group inheritance
- `root`: `$root` reserved token name
- `resolution-order`: Context-based token resolution

## Migration Guide for Users

### Migrating from Draft to 2025.10

1. **Add `$schema` field** to token files:
   ```json
   {
     "$schema": "https://www.designtokens.org/schemas/2025.10.json",
     ...
   }
   ```

2. **Update color values** from strings to structured objects:
   ```json
   // Before (Draft)
   "primary": {
     "$type": "color",
     "$value": "#FF6B35"
   }

   // After (2025.10)
   "primary": {
     "$type": "color",
     "$value": {
       "colorSpace": "srgb",
       "components": [1.0, 0.42, 0.21],
       "alpha": 1.0,
       "hex": "#FF6B35"
     }
   }
   ```

3. **Replace groupMarkers** with `$root`:
   ```json
   // Before (Draft with groupMarkers)
   "color": {
     "_": { "$type": "color", "$value": "#DD0000" },
     "light": { ... }
   }

   // After (2025.10)
   "color": {
     "$root": {
       "$type": "color",
       "$value": { "colorSpace": "srgb", ... }
     },
     "light": { ... }
   }
   ```

4. **Adopt new reference syntax** (optional):
   ```json
   // Both still work in 2025.10
   "alias1": {
     "$value": "{color.primary}"  // Curly braces
   },
   "alias2": {
     "$ref": "#/color/primary"     // JSON Pointer (2025.10+)
   }
   ```

### Multi-Schema Workspaces

The language server supports loading multiple token files with different schemas simultaneously:

```json
// package.json or LSP configuration
{
  "tokenFiles": [
    {
      "path": "legacy/tokens.json",
      "schemaVersion": "draft"
    },
    {
      "path": "new/design.tokens.json"
      // Auto-detects 2025.10 from $schema field
    }
  ]
}
```

Cross-schema references are supported with automatic value normalization.

## Best Practices

1. **Always specify `$schema`**: Don't rely on duck typing for production files
2. **Use official schema URLs**: Ensures correct version detection
3. **Test migrations**: Use test fixtures to verify schema upgrades
4. **Document breaking changes**: Clearly communicate changes to users
5. **Maintain backwards compatibility**: Support older schemas as long as feasible
6. **Implement graceful degradation**: Handle unknown features without crashing

## Troubleshooting

### Schema Detection Failed

Error: `ErrSchemaDetectionFailed`

**Solution**: Add explicit `$schema` field to token file:
```json
{
  "$schema": "https://www.designtokens.org/schemas/2025.10.json",
  ...
}
```

### Mixed Schema Features

Error: `ErrMixedSchemaFeatures`

**Solution**: Files must use features from a single schema version. Update the file to be consistent:
- Either use draft features only (string colors, groupMarkers)
- Or use 2025.10 features only (structured colors, $root, $extends)

### No Handler Registered

Error: `no handler registered for schema version: X`

**Solution**: The schema version is not yet supported. Either:
- Update to a supported version
- Implement a custom handler (see "Adding a New Schema Version" above)

## Further Reading

- [DTCG 2025.10 Specification](https://www.designtokens.org/tr/2025.10/)
- [DTCG Editor's Draft](https://www.designtokens.org/TR/drafts/format/)
- [Internal Schema Architecture](../internal/schema/README.md)
