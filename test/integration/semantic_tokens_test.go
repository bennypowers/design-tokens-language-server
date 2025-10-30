package integration_test

import (
	"testing"

	semantictokens "bennypowers.dev/dtls/lsp/methods/textDocument/semanticTokens"
	"bennypowers.dev/dtls/lsp/types"
	"bennypowers.dev/dtls/test/integration/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// TestSemanticTokens_JSONWithReferences tests semantic tokens for JSON file with token references
func TestSemanticTokens_JSONWithReferences(t *testing.T) {
	server := testutil.NewTestServer(t)

	// Open the fixture file
	testutil.OpenTokenFixture(t, server, "file:///tokens.json", "semantic-tokens/tokens.json")

	// Load the tokens from this file into the token manager so references can be resolved
	tokensContent := testutil.LoadTokenFixture(t, "semantic-tokens/tokens.json")
	err := server.LoadTokensFromJSON(tokensContent, "")
	require.NoError(t, err)

	req := types.NewRequestContext(server, nil)
	result, err := semantictokens.SemanticTokensFull(req, &protocol.SemanticTokensParams{
		TextDocument: protocol.TextDocumentIdentifier{
			URI: "file:///tokens.json",
		},
	})

	require.NoError(t, err)
	require.NotNil(t, result, "Should return semantic tokens for JSON file")
	assert.NotEmpty(t, result.Data, "Should have semantic token data")

	// Data should be in delta encoding format: [deltaLine, deltaStart, length, tokenType, tokenModifiers]
	// Each token takes 5 values
	assert.Equal(t, 0, len(result.Data)%5, "Token data should be groups of 5 values")
}

// TestSemanticTokens_YAMLWithReferences tests semantic tokens for YAML file with token references
func TestSemanticTokens_YAMLWithReferences(t *testing.T) {
	server := testutil.NewTestServer(t)
	testutil.LoadBasicTokens(t, server)

	// Open YAML fixture with token references
	testutil.OpenTokenFixture(t, server,"file:///tokens.yaml", "semantic-tokens/yaml-with-refs.yaml")

	req := types.NewRequestContext(server, nil)
	result, err := semantictokens.SemanticTokensFull(req, &protocol.SemanticTokensParams{
		TextDocument: protocol.TextDocumentIdentifier{
			URI: "file:///tokens.yaml",
		},
	})

	require.NoError(t, err)
	require.NotNil(t, result, "Should return semantic tokens for YAML file")
	assert.NotEmpty(t, result.Data, "Should have semantic token data for YAML")

	// Should have at least 2 tokens (two references: {color.primary} and {spacing.small})
	assert.GreaterOrEqual(t, len(result.Data), 10, "Should have at least 2 tokens (10 values)")
}

// TestSemanticTokens_EmptyDocument tests semantic tokens for empty token file
func TestSemanticTokens_EmptyDocument(t *testing.T) {
	server := testutil.NewTestServer(t)

	// Open empty JSON fixture
	testutil.OpenTokenFixture(t, server,"file:///empty.json", "semantic-tokens/empty-tokens.json")

	req := types.NewRequestContext(server, nil)
	result, err := semantictokens.SemanticTokensFull(req, &protocol.SemanticTokensParams{
		TextDocument: protocol.TextDocumentIdentifier{
			URI: "file:///empty.json",
		},
	})

	require.NoError(t, err)
	require.NotNil(t, result, "Should return result even for empty file")
	assert.Empty(t, result.Data, "Should have no semantic tokens for empty file")
}

// TestSemanticTokens_NonTokenFile tests semantic tokens for CSS file (non-token file)
func TestSemanticTokens_NonTokenFile(t *testing.T) {
	server := testutil.NewTestServer(t)
	testutil.LoadBasicTokens(t, server)

	// Open CSS fixture
	testutil.OpenCSSFixture(t, server, "file:///test.css", "basic-var-calls.css")

	req := types.NewRequestContext(server, nil)
	result, err := semantictokens.SemanticTokensFull(req, &protocol.SemanticTokensParams{
		TextDocument: protocol.TextDocumentIdentifier{
			URI: "file:///test.css",
		},
	})

	require.NoError(t, err)
	// CSS files should return nil (not supported for semantic tokens)
	assert.Nil(t, result, "Should return nil for non-token files (CSS)")
}

// TestSemanticTokens_MalformedReferences tests semantic tokens with malformed token references
func TestSemanticTokens_MalformedReferences(t *testing.T) {
	server := testutil.NewTestServer(t)
	testutil.LoadBasicTokens(t, server)

	// Open fixture with malformed references
	testutil.OpenTokenFixture(t, server,"file:///malformed.json", "semantic-tokens/malformed-reference.json")

	req := types.NewRequestContext(server, nil)
	result, err := semantictokens.SemanticTokensFull(req, &protocol.SemanticTokensParams{
		TextDocument: protocol.TextDocumentIdentifier{
			URI: "file:///malformed.json",
		},
	})

	require.NoError(t, err)
	require.NotNil(t, result, "Should handle malformed references gracefully")

	// Malformed references should be skipped, but the file should still be processed
	// The result might be empty or have partial data depending on what's valid
	assert.Equal(t, 0, len(result.Data)%5, "Token data should still be valid format")
}

// TestSemanticTokens_UnicodeHandling tests semantic tokens with unicode characters
func TestSemanticTokens_UnicodeHandling(t *testing.T) {
	server := testutil.NewTestServer(t)

	// Load tokens first so references can resolve
	testutil.OpenTokenFixture(t, server,"file:///unicode-base.json", "semantic-tokens/unicode-refs.json")
	tokensContent := testutil.LoadTokenFixture(t, "semantic-tokens/unicode-refs.json")
	err := server.LoadTokensFromJSON(tokensContent, "")
	require.NoError(t, err)

	req := types.NewRequestContext(server, nil)
	result, err := semantictokens.SemanticTokensFull(req, &protocol.SemanticTokensParams{
		TextDocument: protocol.TextDocumentIdentifier{
			URI: "file:///unicode-base.json",
		},
	})

	require.NoError(t, err)
	require.NotNil(t, result, "Should handle unicode characters")
	assert.NotEmpty(t, result.Data, "Should extract tokens from unicode file")

	// Should have valid delta encoding even with multi-byte characters
	assert.Equal(t, 0, len(result.Data)%5, "Token data should be valid format with unicode")
}

// TestSemanticTokensRange_EdgeCases tests range requests with edge cases
func TestSemanticTokensRange_EdgeCases(t *testing.T) {
	server := testutil.NewTestServer(t)

	// Open fixture with multiple token references
	testutil.OpenTokenFixture(t, server, "file:///range-test.json", "semantic-tokens/tokens.json")

	// Load the tokens from this file into the token manager so references can be resolved
	tokensContent := testutil.LoadTokenFixture(t, "semantic-tokens/tokens.json")
	err := server.LoadTokensFromJSON(tokensContent, "")
	require.NoError(t, err)

	req := types.NewRequestContext(server, nil)

	t.Run("range before all tokens", func(t *testing.T) {
		// Request tokens for range before any actual tokens exist
		result, err := semantictokens.SemanticTokensRange(req, &protocol.SemanticTokensRangeParams{
			TextDocument: protocol.TextDocumentIdentifier{
				URI: "file:///range-test.json",
			},
			Range: protocol.Range{
				Start: protocol.Position{Line: 0, Character: 0},
				End:   protocol.Position{Line: 0, Character: 10},
			},
		})

		require.NoError(t, err)
		require.NotNil(t, result)
		// Should have no tokens in this range
		assert.Empty(t, result.Data, "Should have no tokens before first token")
	})

	t.Run("range after all tokens", func(t *testing.T) {
		// Request tokens for range after all tokens
		result, err := semantictokens.SemanticTokensRange(req, &protocol.SemanticTokensRangeParams{
			TextDocument: protocol.TextDocumentIdentifier{
				URI: "file:///range-test.json",
			},
			Range: protocol.Range{
				Start: protocol.Position{Line: 9999, Character: 0},
				End:   protocol.Position{Line: 9999, Character: 100},
			},
		})

		require.NoError(t, err)
		require.NotNil(t, result)
		// Should have no tokens in this range
		assert.Empty(t, result.Data, "Should have no tokens after last token")
	})

	t.Run("range covering entire document", func(t *testing.T) {
		// Request tokens for entire document range
		result, err := semantictokens.SemanticTokensRange(req, &protocol.SemanticTokensRangeParams{
			TextDocument: protocol.TextDocumentIdentifier{
				URI: "file:///range-test.json",
			},
			Range: protocol.Range{
				Start: protocol.Position{Line: 0, Character: 0},
				End:   protocol.Position{Line: 9999, Character: 9999},
			},
		})

		require.NoError(t, err)
		require.NotNil(t, result)
		// Should have all tokens
		assert.NotEmpty(t, result.Data, "Should have tokens for entire document")
		assert.Equal(t, 0, len(result.Data)%5, "Should be valid delta encoding")
	})
}

// TestSemanticTokensRange_PartialOverlap tests range with partial token overlap
func TestSemanticTokensRange_PartialOverlap(t *testing.T) {
	server := testutil.NewTestServer(t)

	// Load basic tokens first so references can be resolved
	testutil.LoadBasicTokens(t, server)

	// Use range-test fixture which has token references
	testutil.OpenTokenFixture(t, server, "file:///partial.json", "semantic-tokens/range-test.json")

	req := types.NewRequestContext(server, nil)
	result, err := semantictokens.SemanticTokensRange(req, &protocol.SemanticTokensRangeParams{
		TextDocument: protocol.TextDocumentIdentifier{
			URI: "file:///partial.json",
		},
		Range: protocol.Range{
			Start: protocol.Position{Line: 1, Character: 0},
			End:   protocol.Position{Line: 3, Character: 50},
		},
	})

	require.NoError(t, err)
	require.NotNil(t, result)

	// Should have some tokens that fall within this range
	// The exact count depends on the fixture content
	assert.Equal(t, 0, len(result.Data)%5, "Should be valid delta encoding")
}
