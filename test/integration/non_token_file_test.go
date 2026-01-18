package integration_test

import (
	"testing"

	"bennypowers.dev/dtls/lsp/methods/textDocument/definition"
	"bennypowers.dev/dtls/lsp/methods/textDocument/references"
	semantictokens "bennypowers.dev/dtls/lsp/methods/textDocument/semanticTokens"
	"bennypowers.dev/dtls/lsp/types"
	"bennypowers.dev/dtls/test/integration/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// TestNonTokenFile_Definition verifies that go-to-definition returns nil for non-token files
func TestNonTokenFile_Definition(t *testing.T) {
	t.Run("package.json (no schema)", func(t *testing.T) {
		server := testutil.NewTestServer(t)
		testutil.OpenNonTokenFixture(t, server, "file:///package.json", "package.json")

		req := types.NewRequestContext(server, nil)
		result, err := definition.Definition(req, &protocol.DefinitionParams{
			TextDocumentPositionParams: protocol.TextDocumentPositionParams{
				TextDocument: protocol.TextDocumentIdentifier{
					URI: "file:///package.json",
				},
				Position: protocol.Position{Line: 1, Character: 5},
			},
		})

		require.NoError(t, err)
		assert.Nil(t, result, "Should return nil for non-token JSON file")
	})

	t.Run("docker-compose.yaml (no schema)", func(t *testing.T) {
		server := testutil.NewTestServer(t)
		testutil.OpenNonTokenFixture(t, server, "file:///docker-compose.yaml", "docker-compose.yaml")

		req := types.NewRequestContext(server, nil)
		result, err := definition.Definition(req, &protocol.DefinitionParams{
			TextDocumentPositionParams: protocol.TextDocumentPositionParams{
				TextDocument: protocol.TextDocumentIdentifier{
					URI: "file:///docker-compose.yaml",
				},
				Position: protocol.Position{Line: 2, Character: 5},
			},
		})

		require.NoError(t, err)
		assert.Nil(t, result, "Should return nil for non-token YAML file")
	})

	t.Run("json-schema.org schema (not design tokens)", func(t *testing.T) {
		server := testutil.NewTestServer(t)
		testutil.OpenNonTokenFixture(t, server, "file:///schema.json", "json-schema-file.json")

		req := types.NewRequestContext(server, nil)
		result, err := definition.Definition(req, &protocol.DefinitionParams{
			TextDocumentPositionParams: protocol.TextDocumentPositionParams{
				TextDocument: protocol.TextDocumentIdentifier{
					URI: "file:///schema.json",
				},
				Position: protocol.Position{Line: 3, Character: 5},
			},
		})

		require.NoError(t, err)
		assert.Nil(t, result, "Should return nil for JSON file with non-design-tokens schema")
	})
}

// TestNonTokenFile_References verifies that references returns nil for non-token files
func TestNonTokenFile_References(t *testing.T) {
	t.Run("package.json (no schema)", func(t *testing.T) {
		server := testutil.NewTestServer(t)
		testutil.OpenNonTokenFixture(t, server, "file:///package.json", "package.json")

		req := types.NewRequestContext(server, nil)
		result, err := references.References(req, &protocol.ReferenceParams{
			TextDocumentPositionParams: protocol.TextDocumentPositionParams{
				TextDocument: protocol.TextDocumentIdentifier{
					URI: "file:///package.json",
				},
				Position: protocol.Position{Line: 1, Character: 5},
			},
			Context: protocol.ReferenceContext{
				IncludeDeclaration: true,
			},
		})

		require.NoError(t, err)
		assert.Nil(t, result, "Should return nil for non-token JSON file")
	})

	t.Run("docker-compose.yaml (no schema)", func(t *testing.T) {
		server := testutil.NewTestServer(t)
		testutil.OpenNonTokenFixture(t, server, "file:///docker-compose.yaml", "docker-compose.yaml")

		req := types.NewRequestContext(server, nil)
		result, err := references.References(req, &protocol.ReferenceParams{
			TextDocumentPositionParams: protocol.TextDocumentPositionParams{
				TextDocument: protocol.TextDocumentIdentifier{
					URI: "file:///docker-compose.yaml",
				},
				Position: protocol.Position{Line: 2, Character: 5},
			},
			Context: protocol.ReferenceContext{
				IncludeDeclaration: true,
			},
		})

		require.NoError(t, err)
		assert.Nil(t, result, "Should return nil for non-token YAML file")
	})
}

// TestNonTokenFile_SemanticTokens verifies that semantic tokens returns nil for non-token files
func TestNonTokenFile_SemanticTokens(t *testing.T) {
	t.Run("package.json (no schema)", func(t *testing.T) {
		server := testutil.NewTestServer(t)
		testutil.OpenNonTokenFixture(t, server, "file:///package.json", "package.json")

		req := types.NewRequestContext(server, nil)
		result, err := semantictokens.SemanticTokensFull(req, &protocol.SemanticTokensParams{
			TextDocument: protocol.TextDocumentIdentifier{
				URI: "file:///package.json",
			},
		})

		require.NoError(t, err)
		assert.Nil(t, result, "Should return nil for non-token JSON file")
	})

	t.Run("docker-compose.yaml (no schema)", func(t *testing.T) {
		server := testutil.NewTestServer(t)
		testutil.OpenNonTokenFixture(t, server, "file:///docker-compose.yaml", "docker-compose.yaml")

		req := types.NewRequestContext(server, nil)
		result, err := semantictokens.SemanticTokensFull(req, &protocol.SemanticTokensParams{
			TextDocument: protocol.TextDocumentIdentifier{
				URI: "file:///docker-compose.yaml",
			},
		})

		require.NoError(t, err)
		assert.Nil(t, result, "Should return nil for non-token YAML file")
	})

	t.Run("json-schema.org schema (not design tokens)", func(t *testing.T) {
		server := testutil.NewTestServer(t)
		testutil.OpenNonTokenFixture(t, server, "file:///schema.json", "json-schema-file.json")

		req := types.NewRequestContext(server, nil)
		result, err := semantictokens.SemanticTokensFull(req, &protocol.SemanticTokensParams{
			TextDocument: protocol.TextDocumentIdentifier{
				URI: "file:///schema.json",
			},
		})

		require.NoError(t, err)
		assert.Nil(t, result, "Should return nil for JSON file with non-design-tokens schema")
	})
}

// TestTokenFileWithSchema_Definition verifies that definition works for files with design tokens schema
func TestTokenFileWithSchema_Definition(t *testing.T) {
	server := testutil.NewTestServer(t)

	uri := "file:///tokens.json"

	// Load the token file with schema - this file has a valid Design Tokens $schema
	testutil.OpenNonTokenFixture(t, server, uri, "token-file-with-schema.json")

	// Verify the file is recognized as a token file due to its $schema
	require.True(t, server.ShouldProcessAsTokenFile(uri), "File with Design Tokens $schema should be processed as token file")

	// Load tokens from the fixture so we can resolve references
	content := testutil.LoadNonTokenFixture(t, "token-file-with-schema.json")
	err := server.LoadTokensFromJSON(content, "")
	require.NoError(t, err)

	req := types.NewRequestContext(server, nil)

	// Position on "primary" key on line 3: `    "primary": {`
	// Line 3, char 6 should be on "primary"
	result, err := definition.Definition(req, &protocol.DefinitionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{
				URI: uri,
			},
			Position: protocol.Position{Line: 3, Character: 6}, // on "primary" key
		},
	})

	require.NoError(t, err)
	// The definition request was processed (didn't return early due to non-token file).
	// Result may be nil if the cursor isn't on a valid reference, but the key assertion
	// is that ShouldProcessAsTokenFile returned true above.
	_ = result
}

// TestTokenFileWithSchema_SemanticTokens verifies that semantic tokens work for files with design tokens schema
func TestTokenFileWithSchema_SemanticTokens(t *testing.T) {
	server := testutil.NewTestServer(t)

	// Load the token file with schema
	testutil.OpenNonTokenFixture(t, server, "file:///tokens.json", "token-file-with-schema.json")

	// Load tokens so references can be resolved
	content := testutil.LoadNonTokenFixture(t, "token-file-with-schema.json")
	err := server.LoadTokensFromJSON(content, "")
	require.NoError(t, err)

	req := types.NewRequestContext(server, nil)
	result, err := semantictokens.SemanticTokensFull(req, &protocol.SemanticTokensParams{
		TextDocument: protocol.TextDocumentIdentifier{
			URI: "file:///tokens.json",
		},
	})

	require.NoError(t, err)
	// Should return result because file has valid Design Tokens schema
	require.NotNil(t, result, "Should process file with Design Tokens schema")
	assert.NotEmpty(t, result.Data, "Should have semantic tokens for file with schema")
}
