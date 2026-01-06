package common

import "regexp"

// Shared regex patterns for token references across LSP methods

// CurlyBraceReferenceRegexp matches curly brace token references: {token.reference.path}
var CurlyBraceReferenceRegexp = regexp.MustCompile(`\{([^}]+)\}`)

// JSONPointerReferenceRegexp matches JSON Pointer references: "$ref": "#/path/to/token"
var JSONPointerReferenceRegexp = regexp.MustCompile(`"\$ref"\s*:\s*"(#[^"]+)"`)

// RootKeywordRegexp matches $root keyword in token definitions
var RootKeywordRegexp = regexp.MustCompile(`"\$root"\s*:`)

// SchemaFieldRegexp matches the $schema field with its value
// Captures the schema URL to avoid false positives from schema versions appearing elsewhere
var SchemaFieldRegexp = regexp.MustCompile(`"\$schema"\s*:\s*"([^"]+)"`)
