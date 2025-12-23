package definition

import (
	"regexp"
	"strings"

	"bennypowers.dev/dtls/internal/documents"
	"bennypowers.dev/dtls/internal/position"
	"bennypowers.dev/dtls/lsp/types"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// Token reference patterns
var curlyBraceReferenceRegexp = regexp.MustCompile(`\{([^}]+)\}`)
var jsonPointerReferenceRegexp = regexp.MustCompile(`"\$ref"\s*:\s*"(#[^"]+)"`)

// findReferenceAtPosition finds a token reference at the given position in a JSON/YAML file
// Returns the token name if found, empty string otherwise
func findReferenceAtPosition(content string, pos protocol.Position) string {
	lines := strings.Split(content, "\n")
	if int(pos.Line) >= len(lines) {
		return ""
	}

	line := lines[pos.Line]

	// Check for curly brace references
	if tokenName := findCurlyBraceReferenceAtPosition(line, pos); tokenName != "" {
		return tokenName
	}

	// Check for JSON Pointer references
	if tokenName := findJSONPointerReferenceAtPosition(line, pos); tokenName != "" {
		return tokenName
	}

	return ""
}

// findCurlyBraceReferenceAtPosition finds a curly brace reference at the position
func findCurlyBraceReferenceAtPosition(line string, pos protocol.Position) string {
	matches := curlyBraceReferenceRegexp.FindAllStringSubmatchIndex(line, -1)
	if matches == nil {
		return ""
	}

	for _, match := range matches {
		// match[0], match[1] - full match including braces
		// match[2], match[3] - captured reference without braces

		// Convert byte offsets to UTF-16 positions
		matchStartUTF16 := position.ByteOffsetToUTF16(line, match[0])
		matchEndUTF16 := position.ByteOffsetToUTF16(line, match[1])

		// Check if cursor is within this match
		if pos.Character >= uint32(matchStartUTF16) && pos.Character <= uint32(matchEndUTF16) {
			// Extract the reference (e.g., "color.primary")
			reference := line[match[2]:match[3]]
			// Convert to token name (e.g., "color-primary")
			return strings.ReplaceAll(reference, ".", "-")
		}
	}

	return ""
}

// findJSONPointerReferenceAtPosition finds a JSON Pointer reference at the position
func findJSONPointerReferenceAtPosition(line string, pos protocol.Position) string {
	matches := jsonPointerReferenceRegexp.FindAllStringSubmatchIndex(line, -1)
	if matches == nil {
		return ""
	}

	for _, match := range matches {
		// match[0], match[1] - full match
		// match[2], match[3] - JSON Pointer path (e.g., "#/color/primary")

		// Convert byte offsets to UTF-16 positions
		matchStartUTF16 := position.ByteOffsetToUTF16(line, match[0])
		matchEndUTF16 := position.ByteOffsetToUTF16(line, match[1])

		// Check if cursor is within this match
		if pos.Character >= uint32(matchStartUTF16) && pos.Character <= uint32(matchEndUTF16) {
			// Extract the JSON Pointer path (e.g., "#/color/primary")
			pointerPath := line[match[2]:match[3]]
			// Convert to token name: remove "#/" prefix and replace "/" with "-"
			// e.g., "#/color/primary" -> "color-primary"
			tokenName := strings.TrimPrefix(pointerPath, "#/")
			return strings.ReplaceAll(tokenName, "/", "-")
		}
	}

	return ""
}

// DefinitionForTokenFile handles go-to-definition for references within token files
func DefinitionForTokenFile(req *types.RequestContext, doc *documents.Document, position protocol.Position) (any, error) {
	// Find reference at the cursor position
	tokenName := findReferenceAtPosition(doc.Content(), position)
	if tokenName == "" {
		return nil, nil
	}

	// Look up the token
	token := req.Server.Token(tokenName)
	if token == nil {
		return nil, nil
	}

	// Return the definition location
	if token.DefinitionURI != "" && len(token.Path) > 0 {
		location := protocol.Location{
			URI: token.DefinitionURI,
			Range: protocol.Range{
				Start: protocol.Position{Line: token.Line, Character: token.Character},
				End:   protocol.Position{Line: token.Line, Character: token.Character},
			},
		}

		return []protocol.Location{location}, nil
	}

	return nil, nil
}
