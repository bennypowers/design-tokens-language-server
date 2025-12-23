package semantictokens

import (
	"fmt"
	"math"
	"os"

	"bennypowers.dev/dtls/internal/documents"
	"bennypowers.dev/dtls/lsp/types"
	protocol "github.com/tliron/glsp/protocol_3_16"
)


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
func SemanticTokensFull(req *types.RequestContext, params *protocol.SemanticTokensParams) (*protocol.SemanticTokens, error) {
	uri := params.TextDocument.URI
	fmt.Fprintf(os.Stderr, "[DTLS] Semantic tokens requested for: %s\n", uri)

	doc := req.Server.Document(uri)
	if doc == nil {
		return nil, fmt.Errorf("document not found: %s", uri)
	}

	// Only provide semantic tokens for JSON and YAML token files
	languageID := doc.LanguageID()
	if languageID != "json" && languageID != "yaml" {
		return nil, nil
	}

	intermediateTokens := GetSemanticTokensForDocument(req.Server, doc)

	// Encode tokens using delta encoding
	data, err := encodeSemanticTokens(intermediateTokens)
	if err != nil {
		return nil, fmt.Errorf("failed to encode semantic tokens: %w", err)
	}

	return &protocol.SemanticTokens{
		Data: data,
	}, nil
}

// appendValidatedInt validates that value fits in uint32 range and appends it to data
func appendValidatedInt(data []uint32, value int, fieldName string, tokenIndex int) ([]uint32, error) {
	if value < 0 {
		return nil, fmt.Errorf("token %d: %s %d is negative", tokenIndex, fieldName, value)
	}
	if value > math.MaxUint32 {
		return nil, fmt.Errorf("token %d: %s %d exceeds uint32 limit", tokenIndex, fieldName, value)
	}
	return append(data, uint32(value)), nil //nolint:gosec // validated above
}

// encodeSemanticTokens converts intermediate tokens to delta-encoded format (LSP spec).
// Tokens must be sorted by line and character position for delta encoding to work correctly.
// Returns error if tokens are unsorted or values exceed uint32 limits.
func encodeSemanticTokens(intermediateTokens []SemanticTokenIntermediate) ([]uint32, error) {
	data := make([]uint32, 0, len(intermediateTokens)*5)
	prevLine := 0
	prevStartChar := 0

	for i, token := range intermediateTokens {
		deltaLine := token.Line - prevLine
		deltaStart := token.StartChar
		if deltaLine == 0 {
			deltaStart = token.StartChar - prevStartChar
		}

		var err error
		// Append deltaLine
		data, err = appendValidatedInt(data, deltaLine, "deltaLine", i)
		if err != nil {
			return nil, err
		}

		// Append deltaStart
		data, err = appendValidatedInt(data, deltaStart, "deltaStart", i)
		if err != nil {
			return nil, err
		}

		// Append length
		data, err = appendValidatedInt(data, token.Length, "length", i)
		if err != nil {
			return nil, err
		}

		// Append tokenType
		data, err = appendValidatedInt(data, token.TokenType, "tokenType", i)
		if err != nil {
			return nil, err
		}

		// Append tokenModifiers
		data, err = appendValidatedInt(data, token.TokenModifiers, "tokenModifiers", i)
		if err != nil {
			return nil, err
		}

		prevLine = token.Line
		prevStartChar = token.StartChar
	}

	return data, nil
}

// GetSemanticTokensForDocument extracts semantic tokens from a document
// Positions and lengths are in UTF-16 code units (LSP default encoding)
func GetSemanticTokensForDocument(ctx types.ServerContext, doc *documents.Document) []SemanticTokenIntermediate {
	// Use schema-aware extraction
	return GetSemanticTokensForDocumentSchemaAware(ctx, doc)
}

// handleSemanticTokensRange handles the textDocument/semanticTokens/range request

// SemanticTokensRange handles the textDocument/semanticTokens/range request
func SemanticTokensRange(req *types.RequestContext, params *protocol.SemanticTokensRangeParams) (*protocol.SemanticTokens, error) {
	// Get the document
	doc := req.Server.Document(params.TextDocument.URI)
	if doc == nil {
		return nil, fmt.Errorf("document not found: %s", params.TextDocument.URI)
	}

	// Get all semantic tokens for the document
	intermediateTokens := GetSemanticTokensForDocument(req.Server, doc)

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
	encodedData, err := encodeSemanticTokens(filteredTokens)
	if err != nil {
		return nil, fmt.Errorf("failed to encode semantic tokens: %w", err)
	}

	return &protocol.SemanticTokens{
		Data: encodedData,
	}, nil
}
