package semantictokens

import (
	"strings"

	"bennypowers.dev/dtls/internal/documents"
	"bennypowers.dev/dtls/internal/parser/common"
	"bennypowers.dev/dtls/internal/position"
	"bennypowers.dev/dtls/internal/schema"
	"bennypowers.dev/dtls/lsp/types"
)

// Token type constants matching LSP SemanticTokensLegend indices
const (
	TokenTypeVariable = 0 // "variable" - first part of reference (class)
	TokenTypeProperty = 1 // "property" - subsequent parts of reference
	TokenTypeKeyword  = 3 // "keyword" - $ref, $root keywords
	TokenTypeString   = 4 // "string" - JSON Pointer paths
)

// detectSchemaVersion detects the schema version from document content
// Uses regex to extract the actual $schema field value to avoid false positives
func detectSchemaVersion(content string) schema.SchemaVersion {
	matches := common.SchemaFieldRegexp.FindStringSubmatch(content)
	if len(matches) < 2 {
		return schema.Draft // Default to draft if no $schema found
	}

	schemaURL := matches[1]
	if strings.Contains(schemaURL, "2025.10") {
		return schema.V2025_10
	}
	if strings.Contains(schemaURL, "draft") {
		return schema.Draft
	}

	return schema.Draft
}

// extractJSONPointerTokens extracts semantic tokens for JSON Pointer references
// Returns tokens for both the $ref keyword and the pointer path
func extractJSONPointerTokens(line string, lineNum int) []SemanticTokenIntermediate {
	tokens := []SemanticTokenIntermediate{}

	matches := common.JSONPointerReferenceRegexp.FindAllStringSubmatchIndex(line, -1)
	if matches == nil {
		return tokens
	}

	for _, match := range matches {
		// match[0], match[1] - full match
		// match[2], match[3] - JSON Pointer path (#/...)

		// Highlight $ref keyword
		refStart := strings.Index(line[match[0]:match[1]], "$ref")
		if refStart != -1 {
			refStartByte := match[0] + refStart
			refStartUTF16 := position.ByteOffsetToUTF16(line, refStartByte)

			tokens = append(tokens, SemanticTokenIntermediate{
				Line:           lineNum,
				StartChar:      refStartUTF16,
				Length:         4, // "$ref"
				TokenType:      TokenTypeKeyword,
				TokenModifiers: 0,
			})
		}

		// Highlight JSON Pointer path
		pointerPath := line[match[2]:match[3]]
		pointerStartUTF16 := position.ByteOffsetToUTF16(line, match[2])

		tokens = append(tokens, SemanticTokenIntermediate{
			Line:           lineNum,
			StartChar:      pointerStartUTF16,
			Length:         position.StringLengthUTF16(pointerPath),
			TokenType:      TokenTypeString,
			TokenModifiers: 0,
		})
	}

	return tokens
}

// extractRootKeywordTokens extracts semantic tokens for $root keywords
func extractRootKeywordTokens(line string, lineNum int) []SemanticTokenIntermediate {
	tokens := []SemanticTokenIntermediate{}

	matches := common.RootKeywordRegexp.FindAllStringIndex(line, -1)
	if matches == nil {
		return tokens
	}

	for _, match := range matches {
		// Find the position of $root within the match
		rootStart := strings.Index(line[match[0]:match[1]], "$root")
		if rootStart != -1 {
			rootStartByte := match[0] + rootStart
			rootStartUTF16 := position.ByteOffsetToUTF16(line, rootStartByte)

			tokens = append(tokens, SemanticTokenIntermediate{
				Line:           lineNum,
				StartChar:      rootStartUTF16,
				Length:         5, // "$root"
				TokenType:      TokenTypeKeyword,
				TokenModifiers: 0,
			})
		}
	}

	return tokens
}

// GetSemanticTokensForDocumentSchemaAware extracts semantic tokens with schema awareness
func GetSemanticTokensForDocumentSchemaAware(ctx types.ServerContext, doc *documents.Document) []SemanticTokenIntermediate {
	content := doc.Content()
	tokens := []SemanticTokenIntermediate{}

	// Detect schema version
	version := detectSchemaVersion(content)

	// Split content into lines
	lines := strings.Split(content, "\n")

	for lineNum, line := range lines {
		// Extract curly brace references (both schemas)
		curlyBraceTokens := extractCurlyBraceReferences(ctx, line, lineNum)
		tokens = append(tokens, curlyBraceTokens...)

		// Extract 2025.10-specific features
		if version == schema.V2025_10 {
			// Extract JSON Pointer references
			jsonPointerTokens := extractJSONPointerTokens(line, lineNum)
			tokens = append(tokens, jsonPointerTokens...)

			// Extract $root keywords
			rootTokens := extractRootKeywordTokens(line, lineNum)
			tokens = append(tokens, rootTokens...)
		}
	}

	return tokens
}

// extractCurlyBraceReferences extracts semantic tokens for curly brace references
// This is the original logic extracted for reuse
func extractCurlyBraceReferences(ctx types.ServerContext, line string, lineNum int) []SemanticTokenIntermediate {
	tokens := []SemanticTokenIntermediate{}

	// Find all token references in this line
	matches := common.CurlyBraceReferenceRegexp.FindAllStringSubmatchIndex(line, -1)
	if matches == nil {
		return tokens
	}

	for _, match := range matches {
		// match[2] and match[3] are the start and end of the first capture group (the reference)
		referenceStart := match[2]
		referenceEnd := match[3]
		reference := line[referenceStart:referenceEnd]

		// Convert dots to dashes for token lookup (design tokens use dots, but we store as dashes)
		tokenName := strings.ReplaceAll(reference, ".", "-")

		// Check if this reference exists in our token manager
		if ctx.Token(tokenName) == nil {
			continue
		}

		// Split reference into parts (e.g., "color.brand.primary" -> ["color", "brand", "primary"])
		parts := strings.Split(reference, ".")

		// Calculate the starting position of the reference within the line
		// The reference starts at match[2] (after the opening {)
		// Convert byte offset to UTF-16 code units
		partStartChar := position.ByteOffsetToUTF16(line, referenceStart)

		for i, part := range parts {
			tokenType := TokenTypeProperty
			if i == 0 {
				tokenType = TokenTypeVariable
			}

			tokens = append(tokens, SemanticTokenIntermediate{
				Line:           lineNum,
				StartChar:      partStartChar,
				Length:         position.StringLengthUTF16(part),
				TokenType:      tokenType,
				TokenModifiers: 0,
			})

			// Move to the next part (add UTF-16 length of part + 1 for the dot)
			partStartChar += position.StringLengthUTF16(part) + 1
		}
	}

	return tokens
}
