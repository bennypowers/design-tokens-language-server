package semantictokens

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"bennypowers.dev/dtls/internal/documents"
	"bennypowers.dev/dtls/internal/position"
	"bennypowers.dev/dtls/lsp/types"
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

// SemanticTokensFull handles the textDocument/semanticTokens/full request
func SemanticTokensFull(ctx types.ServerContext, context *glsp.Context, params *protocol.SemanticTokensParams) (*protocol.SemanticTokens, error) {
	uri := params.TextDocument.URI
	fmt.Fprintf(os.Stderr, "[DTLS] Semantic tokens requested for: %s\n", uri)

	doc := ctx.Document(uri)
	if doc == nil {
		return nil, fmt.Errorf("document not found: %s", uri)
	}

	// Only provide semantic tokens for JSON and YAML token files
	languageID := doc.LanguageID()
	if languageID != "json" && languageID != "yaml" {
		return nil, nil
	}

	intermediateTokens := GetSemanticTokensForDocument(ctx, doc)

	// Encode tokens using delta encoding
	data := encodeSemanticTokens(intermediateTokens)

	return &protocol.SemanticTokens{
		Data: data,
	}, nil
}

// encodeSemanticTokens converts intermediate tokens to delta-encoded format (LSP spec)
func encodeSemanticTokens(intermediateTokens []SemanticTokenIntermediate) []uint32 {
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

	return data
}

// GetSemanticTokensForDocument extracts semantic tokens from a document
// Positions and lengths are in UTF-16 code units (LSP default encoding)
func GetSemanticTokensForDocument(ctx types.ServerContext, doc *documents.Document) []SemanticTokenIntermediate {
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
				tokenType := 1 // property (default)
				if i == 0 {
					tokenType = 0 // class (for first part)
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
	}

	return tokens
}

// handleSemanticTokensDelta handles the textDocument/semanticTokens/full/delta request

// SemanticTokensDelta handles the textDocument/semanticTokens/delta request
func SemanticTokensDelta(ctx types.ServerContext, context *glsp.Context, params *protocol.SemanticTokensDeltaParams) (*protocol.SemanticTokensDelta, error) {
	// Get the current document
	doc := ctx.Document(params.TextDocument.URI)
	if doc == nil {
		return nil, fmt.Errorf("document not found: %s", params.TextDocument.URI)
	}

	// Get current semantic tokens
	intermediateTokens := GetSemanticTokensForDocument(ctx, doc)
	newData := encodeSemanticTokens(intermediateTokens)

	// For a full implementation, we would:
	// 1. Store previous results by resultID
	// 2. Compare old and new token arrays
	// 3. Generate minimal edits (SemanticTokensEdit with start, deleteCount, data)
	//
	// For now, we'll implement a simplified version that returns edits
	// showing that new tokens were added at the end.
	//
	// In a real implementation, you would compare params.PreviousResultID's data
	// with newData and generate minimal edits.

	// Simplified implementation: return an edit that appends new tokens
	// This assumes tokens were added at the end (which is common when adding new lines)
	edits := []protocol.SemanticTokensEdit{
		{
			// Start at the beginning and replace all
			// A better implementation would do proper diffing
			Start:       0,
			DeleteCount: 0, // Don't delete anything
			Data:        newData,
		},
	}

	return &protocol.SemanticTokensDelta{
		Edits: edits,
	}, nil
}

// handleSemanticTokensRange handles the textDocument/semanticTokens/range request

// SemanticTokensRange handles the textDocument/semanticTokens/range request
func SemanticTokensRange(ctx types.ServerContext, context *glsp.Context, params *protocol.SemanticTokensRangeParams) (*protocol.SemanticTokens, error) {
	// Get the document
	doc := ctx.Document(params.TextDocument.URI)
	if doc == nil {
		return nil, fmt.Errorf("document not found: %s", params.TextDocument.URI)
	}

	// Get all semantic tokens for the document
	intermediateTokens := GetSemanticTokensForDocument(ctx, doc)

	// Filter tokens to only those within the requested range
	filteredTokens := []SemanticTokenIntermediate{}
	for _, token := range intermediateTokens {
		// Convert protocol.UInteger to int for comparison
		startLine := int(params.Range.Start.Line)
		endLine := int(params.Range.End.Line)
		startChar := int(params.Range.Start.Character)
		endChar := int(params.Range.End.Character)

		// Check if token is within the requested range
		if token.Line >= startLine && token.Line <= endLine {
			// For start line, check if character is >= start character
			if token.Line == startLine && token.StartChar < startChar {
				continue
			}
			// For end line, check if character is < end character (exclusive)
			if token.Line == endLine && token.StartChar >= endChar {
				continue
			}
			filteredTokens = append(filteredTokens, token)
		}
	}

	// Encode filtered tokens
	encodedData := encodeSemanticTokens(filteredTokens)

	return &protocol.SemanticTokens{
		Data: encodedData,
	}, nil
}
