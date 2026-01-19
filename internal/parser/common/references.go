package common

import (
	"strings"

	posutil "bennypowers.dev/dtls/internal/position"
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

// ExtractReferences extracts references from a string value.
// The version parameter is reserved for future schema-specific extraction logic.
func ExtractReferences(content string, version schema.SchemaVersion) ([]Reference, error) {
	var refs []Reference

	// Extract curly brace references (supported in both schemas)
	matches := CurlyBraceReferenceRegexp.FindAllStringSubmatch(content, -1)
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

			// Strip "#/" prefix from JSON Pointer
			// Keep slash format for Reference.Path (conversion to dots happens later if needed)
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
// Automatically strips the "#/" prefix if present
// Examples:
//   "#/color/brand/primary" -> "color.brand.primary"
//   "color/brand/primary" -> "color.brand.primary"
func ConvertJSONPointerToTokenPath(jsonPointer string) string {
	// Strip leading "#/" if present
	jsonPointer = strings.TrimPrefix(jsonPointer, "#/")
	return strings.ReplaceAll(jsonPointer, "/", ".")
}

// ConvertTokenPathToJSONPointer converts a token path to a JSON Pointer
// Example: "color.brand.primary" -> "#/color/brand/primary"
func ConvertTokenPathToJSONPointer(tokenPath string) string {
	return "#/" + strings.ReplaceAll(tokenPath, ".", "/")
}

// TokenReferenceWithRange represents a token reference found in content with position info
type TokenReferenceWithRange struct {
	// TokenName is the normalized token name (e.g., "color-primary")
	TokenName string
	// RawReference is the original reference text (e.g., "color.primary" or "#/color/primary")
	RawReference string
	// StartChar is the UTF-16 character offset of the reference start (within the line)
	StartChar uint32
	// EndChar is the UTF-16 character offset of the reference end (within the line)
	EndChar uint32
	// Line is the line number where the reference was found
	Line uint32
	// Type indicates whether this is a curly brace or JSON pointer reference
	Type ReferenceType
}

// NormalizeLineEndings normalizes line endings to LF for consistent processing
func NormalizeLineEndings(content string) string {
	// Replace CRLF with LF, then replace any remaining CR with LF
	content = strings.ReplaceAll(content, "\r\n", "\n")
	content = strings.ReplaceAll(content, "\r", "\n")
	return content
}

// FindReferenceAtPosition finds a token reference at the given position in a JSON/YAML file.
// Returns the TokenReferenceWithRange if found, nil otherwise.
func FindReferenceAtPosition(content string, line uint32, character uint32) *TokenReferenceWithRange {
	// Normalize line endings (CRLF -> LF) to handle Windows files correctly
	content = NormalizeLineEndings(content)

	lines := strings.Split(content, "\n")
	if int(line) >= len(lines) {
		return nil
	}

	lineText := lines[line]

	// Check for curly brace references
	if ref := findCurlyBraceReferenceAtPositionWithRange(lineText, line, character); ref != nil {
		return ref
	}

	// Check for JSON Pointer references
	if ref := findJSONPointerReferenceAtPositionWithRange(lineText, line, character); ref != nil {
		return ref
	}

	return nil
}

// findCurlyBraceReferenceAtPositionWithRange finds a curly brace reference at the position
func findCurlyBraceReferenceAtPositionWithRange(lineText string, line uint32, character uint32) *TokenReferenceWithRange {
	matches := CurlyBraceReferenceRegexp.FindAllStringSubmatchIndex(lineText, -1)
	if matches == nil {
		return nil
	}

	for _, match := range matches {
		// match[0], match[1] - full match including braces
		// match[2], match[3] - captured reference without braces

		// Convert byte offsets to UTF-16 positions
		matchStartUTF16 := uint32(posutil.ByteOffsetToUTF16(lineText, match[0]))
		matchEndUTF16 := uint32(posutil.ByteOffsetToUTF16(lineText, match[1]))

		// Check if cursor is within this match
		if character >= matchStartUTF16 && character <= matchEndUTF16 {
			// Extract the reference (e.g., "color.primary")
			rawReference := lineText[match[2]:match[3]]
			// Convert to token name (e.g., "color-primary")
			tokenName := strings.ReplaceAll(rawReference, ".", "-")

			return &TokenReferenceWithRange{
				TokenName:    tokenName,
				RawReference: rawReference,
				StartChar:    matchStartUTF16,
				EndChar:      matchEndUTF16,
				Line:         line,
				Type:         CurlyBraceReference,
			}
		}
	}

	return nil
}

// findJSONPointerReferenceAtPositionWithRange finds a JSON Pointer reference at the position
func findJSONPointerReferenceAtPositionWithRange(lineText string, line uint32, character uint32) *TokenReferenceWithRange {
	matches := JSONPointerReferenceRegexp.FindAllStringSubmatchIndex(lineText, -1)
	if matches == nil {
		return nil
	}

	for _, match := range matches {
		// match[0], match[1] - full match
		// match[2], match[3] - JSON Pointer path (e.g., "#/color/primary")

		// Convert byte offsets to UTF-16 positions
		matchStartUTF16 := uint32(posutil.ByteOffsetToUTF16(lineText, match[0]))
		matchEndUTF16 := uint32(posutil.ByteOffsetToUTF16(lineText, match[1]))

		// Check if cursor is within this match
		if character >= matchStartUTF16 && character <= matchEndUTF16 {
			// Extract the JSON Pointer path (e.g., "#/color/primary")
			pointerPath := lineText[match[2]:match[3]]
			// Convert to token name: remove "#/" prefix and replace "/" with "-"
			// e.g., "#/color/primary" -> "color-primary"
			tokenName := strings.TrimPrefix(pointerPath, "#/")
			tokenName = strings.ReplaceAll(tokenName, "/", "-")

			return &TokenReferenceWithRange{
				TokenName:    tokenName,
				RawReference: pointerPath,
				StartChar:    matchStartUTF16,
				EndChar:      matchEndUTF16,
				Line:         line,
				Type:         JSONPointerReference,
			}
		}
	}

	return nil
}
