package common

import (
	"strings"

	posutil "bennypowers.dev/dtls/internal/position"
)

// ReferenceType indicates the type of reference
type ReferenceType int

const (
	// CurlyBraceReference is a {token.path} style reference (both schemas)
	CurlyBraceReference ReferenceType = iota

	// JSONPointerReference is a $ref field (2025.10 only)
	JSONPointerReference
)

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
func FindReferenceAtPosition(content string, line, character uint32) *TokenReferenceWithRange {
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
func findCurlyBraceReferenceAtPositionWithRange(lineText string, line, character uint32) *TokenReferenceWithRange {
	matches := CurlyBraceReferenceRegexp.FindAllStringSubmatchIndex(lineText, -1)
	if matches == nil {
		return nil
	}

	for _, match := range matches {
		// match[0], match[1] - full match including braces
		// match[2], match[3] - captured reference without braces

		// Convert byte offsets to UTF-16 positions
		matchStartUTF16 := posutil.ByteOffsetToUTF16Uint32(lineText, match[0])
		matchEndUTF16 := posutil.ByteOffsetToUTF16Uint32(lineText, match[1])

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
func findJSONPointerReferenceAtPositionWithRange(lineText string, line, character uint32) *TokenReferenceWithRange {
	matches := JSONPointerReferenceRegexp.FindAllStringSubmatchIndex(lineText, -1)
	if matches == nil {
		return nil
	}

	for _, match := range matches {
		// match[0], match[1] - full match
		// match[2], match[3] - JSON Pointer path (e.g., "#/color/primary")

		// Convert byte offsets to UTF-16 positions
		matchStartUTF16 := posutil.ByteOffsetToUTF16Uint32(lineText, match[0])
		matchEndUTF16 := posutil.ByteOffsetToUTF16Uint32(lineText, match[1])

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
