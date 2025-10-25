package lsp

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/bennypowers/design-tokens-language-server/internal/documents"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// Token reference pattern: {token.reference.path}
var tokenReferenceRegexp = regexp.MustCompile(`\{([^}]+)\}`)

// SemanticTokenIntermediate represents an intermediate token before delta encoding
type SemanticTokenIntermediate struct {
	Line           int
	StartChar      int
	Length         int
	TokenType      int // Index into token types array
	TokenModifiers int
}

// handleSemanticTokensFull handles the textDocument/semanticTokens/full request
func (s *Server) handleSemanticTokensFull(context *glsp.Context, params *protocol.SemanticTokensParams) (*protocol.SemanticTokens, error) {
	uri := params.TextDocument.URI
	fmt.Fprintf(os.Stderr, "[DTLS] Semantic tokens requested for: %s\n", uri)

	doc := s.documents.Get(uri)
	if doc == nil {
		return nil, fmt.Errorf("document not found: %s", uri)
	}

	// Only provide semantic tokens for JSON and YAML token files
	languageID := doc.LanguageID()
	if languageID != "json" && languageID != "yaml" {
		return nil, nil
	}

	intermediateTokens := s.getSemanticTokensForDocument(doc)

	// Convert intermediate tokens to delta-encoded format
	data := make([]uint32, 0, len(intermediateTokens)*5)
	prevLine := 0
	prevStartChar := 0

	for _, token := range intermediateTokens {
		deltaLine := token.Line - prevLine
		deltaStart := token.StartChar
		if deltaLine == 0 {
			deltaStart = token.StartChar - prevStartChar
		}

		data = append(data,
			uint32(deltaLine),
			uint32(deltaStart),
			uint32(token.Length),
			uint32(token.TokenType),
			uint32(token.TokenModifiers),
		)

		prevLine = token.Line
		prevStartChar = token.StartChar
	}

	return &protocol.SemanticTokens{
		Data: data,
	}, nil
}

// byteOffsetToUTF16 converts a byte offset within a string to UTF-16 code units
func byteOffsetToUTF16(s string, byteOffset int) int {
	if byteOffset > len(s) {
		byteOffset = len(s)
	}

	utf16Count := 0
	for _, r := range []rune(s[:byteOffset]) {
		if r <= 0xFFFF {
			utf16Count++
		} else {
			utf16Count += 2 // Surrogate pair
		}
	}
	return utf16Count
}

// stringLengthUTF16 returns the length of a string in UTF-16 code units
func stringLengthUTF16(s string) int {
	utf16Count := 0
	for _, r := range []rune(s) {
		if r <= 0xFFFF {
			utf16Count++
		} else {
			utf16Count += 2 // Surrogate pair
		}
	}
	return utf16Count
}

// getSemanticTokensForDocument extracts semantic tokens from a document
// Positions and lengths are in UTF-16 code units (LSP default encoding)
func (s *Server) getSemanticTokensForDocument(doc *documents.Document) []SemanticTokenIntermediate {
	content := doc.Content()
	tokens := []SemanticTokenIntermediate{}

	// Split content into lines
	lines := strings.Split(content, "\n")

	for lineNum, line := range lines {
		// Find all token references in this line
		matches := tokenReferenceRegexp.FindAllStringSubmatchIndex(line, -1)
		if matches == nil {
			continue
		}

		for _, match := range matches {
			// match[2] and match[3] are the start and end of the first capture group (the reference)
			referenceStart := match[2]
			referenceEnd := match[3]
			reference := line[referenceStart:referenceEnd]

			// Check if this reference exists in our token manager
			if s.tokens.Get(reference) == nil {
				continue
			}

			// Split reference into parts (e.g., "color.brand.primary" -> ["color", "brand", "primary"])
			parts := strings.Split(reference, ".")

			// Calculate the starting position of the reference within the line
			// The reference starts at match[2] (after the opening {)
			// Convert byte offset to UTF-16 code units
			partStartChar := byteOffsetToUTF16(line, referenceStart)

			for i, part := range parts {
				tokenType := 1 // property (default)
				if i == 0 {
					tokenType = 0 // class (for first part)
				}

				tokens = append(tokens, SemanticTokenIntermediate{
					Line:           lineNum,
					StartChar:      partStartChar,
					Length:         stringLengthUTF16(part),
					TokenType:      tokenType,
					TokenModifiers: 0,
				})

				// Move to the next part (add UTF-16 length of part + 1 for the dot)
				partStartChar += stringLengthUTF16(part) + 1
			}
		}
	}

	return tokens
}

// handleSemanticTokensDelta handles the textDocument/semanticTokens/full/delta request
func (s *Server) handleSemanticTokensDelta(context *glsp.Context, params *protocol.SemanticTokensDeltaParams) (*protocol.SemanticTokensDelta, error) {
	// Not implemented yet
	return nil, nil
}

// handleSemanticTokensRange handles the textDocument/semanticTokens/range request
func (s *Server) handleSemanticTokensRange(context *glsp.Context, params *protocol.SemanticTokensRangeParams) (*protocol.SemanticTokens, error) {
	// Not implemented yet
	return nil, nil
}
