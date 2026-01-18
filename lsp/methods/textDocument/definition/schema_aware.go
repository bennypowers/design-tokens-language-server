package definition

import (
	"os"
	"strings"

	"bennypowers.dev/dtls/internal/documents"
	"bennypowers.dev/dtls/internal/parser/common"
	posutil "bennypowers.dev/dtls/internal/position"
	"bennypowers.dev/dtls/internal/uriutil"
	"bennypowers.dev/dtls/lsp/types"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// normalizeLineEndings normalizes line endings to LF for consistent processing
func normalizeLineEndings(content string) string {
	// Replace CRLF with LF, then replace any remaining CR with LF
	content = strings.ReplaceAll(content, "\r\n", "\n")
	content = strings.ReplaceAll(content, "\r", "\n")
	return content
}

// findReferenceAtPosition finds a token reference at the given position in a JSON/YAML file
// Returns the token name if found, empty string otherwise
func findReferenceAtPosition(content string, pos protocol.Position) string {
	// Normalize line endings (CRLF -> LF) to handle Windows files correctly
	content = normalizeLineEndings(content)

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
	matches := common.CurlyBraceReferenceRegexp.FindAllStringSubmatchIndex(line, -1)
	if matches == nil {
		return ""
	}

	for _, match := range matches {
		// match[0], match[1] - full match including braces
		// match[2], match[3] - captured reference without braces

		// Convert byte offsets to UTF-16 positions
		matchStartUTF16 := posutil.ByteOffsetToUTF16(line, match[0])
		matchEndUTF16 := posutil.ByteOffsetToUTF16(line, match[1])

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
	matches := common.JSONPointerReferenceRegexp.FindAllStringSubmatchIndex(line, -1)
	if matches == nil {
		return ""
	}

	for _, match := range matches {
		// match[0], match[1] - full match
		// match[2], match[3] - JSON Pointer path (e.g., "#/color/primary")

		// Convert byte offsets to UTF-16 positions
		matchStartUTF16 := posutil.ByteOffsetToUTF16(line, match[0])
		matchEndUTF16 := posutil.ByteOffsetToUTF16(line, match[1])

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

// getLineText retrieves the text of a specific line from a URI.
// First checks the document manager (for open files), then falls back to reading from disk.
func getLineText(req *types.RequestContext, uri string, lineNum uint32) (string, error) {
	// Try to get from document manager first (for open files)
	if doc := req.Server.DocumentManager().Get(uri); doc != nil {
		content := normalizeLineEndings(doc.Content())
		lines := strings.Split(content, "\n")
		if int(lineNum) < len(lines) {
			return lines[lineNum], nil
		}
		return "", nil
	}

	// Fall back to reading from disk
	filePath := uriutil.URIToPath(uri)
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}

	content := normalizeLineEndings(string(data))
	lines := strings.Split(content, "\n")
	if int(lineNum) < len(lines) {
		return lines[lineNum], nil
	}

	return "", nil
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
		// Get the line text where the token is defined
		// token.Character is a byte offset, so we need to convert to UTF-16
		lineText, err := getLineText(req, token.DefinitionURI, token.Line)
		if err != nil || lineText == "" {
			// If we can't get the line text, fall back to zero-width range
			location := protocol.Location{
				URI: token.DefinitionURI,
				Range: protocol.Range{
					Start: protocol.Position{Line: token.Line, Character: 0},
					End:   protocol.Position{Line: token.Line, Character: 0},
				},
			}
			return []protocol.Location{location}, nil
		}

		// Convert byte offset to UTF-16 position
		startCharUTF16 := uint32(posutil.ByteOffsetToUTF16(lineText, int(token.Character)))

		// Calculate end position: start + token name length in UTF-16
		tokenNameLenUTF16 := uint32(posutil.StringLengthUTF16(token.Name))
		endCharUTF16 := startCharUTF16 + tokenNameLenUTF16

		location := protocol.Location{
			URI: token.DefinitionURI,
			Range: protocol.Range{
				Start: protocol.Position{Line: token.Line, Character: startCharUTF16},
				End:   protocol.Position{Line: token.Line, Character: endCharUTF16},
			},
		}

		return []protocol.Location{location}, nil
	}

	return nil, nil
}
