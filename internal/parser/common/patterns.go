package common

import "regexp"

// Shared regex patterns for token references across LSP methods
// These patterns support both JSON (quoted field names) and YAML (unquoted field names)

// CurlyBraceReferenceRegexp matches curly brace token references: {token.reference.path}
var CurlyBraceReferenceRegexp = regexp.MustCompile(`\{([^}]+)\}`)

// JSONPointerReferenceRegexp matches JSON Pointer references in both JSON and YAML:
// JSON: "$ref": "#/path/to/token"
// YAML: $ref: "#/path/to/token" or $ref: '#/path/to/token'
var JSONPointerReferenceRegexp = regexp.MustCompile(`"?\$ref"?\s*:\s*["']?(#[^"'\s]+)["']?`)

// RootKeywordRegexp matches $root keyword in token definitions (JSON and YAML):
// JSON: "$root": { ... }
// YAML: $root: { ... }
var RootKeywordRegexp = regexp.MustCompile(`"?\$root"?\s*:`)

// SchemaFieldRegexp matches the $schema field with its value in JSON and YAML
// Anchored to line start to match only top-level $schema declarations
// Captures the schema URL to avoid false positives from schema versions appearing elsewhere
// JSON: "$schema": "https://..."
// YAML: $schema: "https://..." or $schema: 'https://...'
var SchemaFieldRegexp = regexp.MustCompile(`(?m)^\s*"?\$schema"?\s*:\s*["']([^"']+)["']`)
