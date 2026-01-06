package common

import (
	"regexp"
	"strings"

	"bennypowers.dev/dtls/internal/schema"
)

// ReferenceType indicates the type of reference
type ReferenceType int

const (
	// CurlyBraceReference is a {token.path} style reference (both schemas)
	CurlyBraceReference ReferenceType = iota

	// JSONPointerReference is a $ref field (2025.10 only)
	JSONPointerReference
)

// Reference represents a reference to another token
type Reference struct {
	Type ReferenceType
	Path string
	// Line and Column are reserved for future position tracking
	// Currently not populated by extraction functions
	Line   int
	Column int
}

// Regex for curly brace references: {path.to.token}
var curlyBracePattern = regexp.MustCompile(`\{([^}]+)\}`)

// ExtractReferences extracts references from a string value.
// The version parameter is reserved for future schema-specific extraction logic.
func ExtractReferences(content string, version schema.SchemaVersion) ([]Reference, error) {
	var refs []Reference

	// Extract curly brace references (supported in both schemas)
	matches := curlyBracePattern.FindAllStringSubmatch(content, -1)
	for _, match := range matches {
		if len(match) > 1 {
			refs = append(refs, Reference{
				Type: CurlyBraceReference,
				Path: match[1], // The captured group (path)
			})
		}
	}

	return refs, nil
}

// ExtractReferencesFromValue extracts references from any value type
// Handles both string interpolation and $ref fields
func ExtractReferencesFromValue(value interface{}, version schema.SchemaVersion) ([]Reference, error) {
	switch v := value.(type) {
	case string:
		// String value - extract curly brace references
		return ExtractReferences(v, version)

	case map[string]interface{}:
		// Object - check for $ref field (2025.10 only)
		if refPath, ok := v["$ref"].(string); ok {
			// $ref is 2025.10+ only
			if version == schema.Draft {
				return nil, schema.NewMixedSchemaFeaturesError("", "draft", []string{"$ref (2025.10+ only)"})
			}

			// Convert JSON Pointer to path
			// Remove leading "#/" if present
			path := strings.TrimPrefix(refPath, "#/")

			return []Reference{
				{
					Type: JSONPointerReference,
					Path: path,
				},
			}, nil
		}

		// No $ref field, return empty
		return nil, nil

	default:
		return nil, nil
	}
}

// ConvertJSONPointerToTokenPath converts a JSON Pointer path to a token path
// Example: "color/brand/primary" -> "color.brand.primary"
func ConvertJSONPointerToTokenPath(jsonPointer string) string {
	return strings.ReplaceAll(jsonPointer, "/", ".")
}

// ConvertTokenPathToJSONPointer converts a token path to a JSON Pointer
// Example: "color.brand.primary" -> "#/color/brand/primary"
func ConvertTokenPathToJSONPointer(tokenPath string) string {
	return "#/" + strings.ReplaceAll(tokenPath, ".", "/")
}
