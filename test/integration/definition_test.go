package integration_test

import (
	"testing"

	"bennypowers.dev/dtls/lsp/methods/textDocument"
	"bennypowers.dev/dtls/lsp/methods/textDocument/definition"
	"bennypowers.dev/dtls/lsp/types"
	"bennypowers.dev/dtls/test/integration/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// TestDefinitionOnVarCall tests go-to-definition on a var() call
func TestDefinitionOnVarCall(t *testing.T) {
	server := testutil.NewTestServer(t)
	testutil.LoadBasicTokens(t, server)
	testutil.OpenCSSFixture(t, server, "file:///test.css", "basic-var-calls.css")

	// Request definition - see fixture for position
	req := types.NewRequestContext(server, nil)
	result, err := definition.Definition(req, &protocol.DefinitionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{
				URI: "file:///test.css",
			},
			Position: protocol.Position{
				Line:      2,  // Adjusted for comment line
				Character: 18, // Inside --color-primary
			},
		},
	})

	require.NoError(t, err)
	// Since LoadTokensFromJSON doesn't set DefinitionURI, this returns nil
	// When loading from a file, this would return the token file location
	assert.Nil(t, result)
}

// TestDefinitionOutsideVarCall tests that definition returns nil outside var() calls
func TestDefinitionOutsideVarCall(t *testing.T) {
	server := testutil.NewTestServer(t)
	testutil.OpenCSSFixture(t, server, "file:///test.css", "no-var-call.css")

	// Request definition - see fixture for position
	req := types.NewRequestContext(server, nil)
	result, err := definition.Definition(req, &protocol.DefinitionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{
				URI: "file:///test.css",
			},
			Position: protocol.Position{
				Line:      2,  // Adjusted for comment line
				Character: 10, // On "red"
			},
		},
	})

	require.NoError(t, err)
	assert.Nil(t, result)
}

// TestDefinitionNonCSSFile tests that definition returns nil for non-CSS files
func TestDefinitionNonCSSFile(t *testing.T) {
	server := testutil.NewTestServer(t)
	testutil.LoadBasicTokens(t, server)

	// Open a JSON file
	req := types.NewRequestContext(server, nil)
	err := textDocument.DidOpen(req, &protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{
			URI:        "file:///test.json",
			LanguageID: "json",
			Version:    1,
			Text:       `{"color": "red"}`,
		},
	})
	require.NoError(t, err)

	// Request definition
	req = types.NewRequestContext(server, nil)
	result, err := definition.Definition(req, &protocol.DefinitionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{
				URI: "file:///test.json",
			},
			Position: protocol.Position{Line: 0, Character: 5},
		},
	})

	require.NoError(t, err)
	assert.Nil(t, result)
}

// TestDefinitionUnknownToken tests definition for an unknown token
func TestDefinitionUnknownToken(t *testing.T) {
	server := testutil.NewTestServer(t)
	testutil.LoadBasicTokens(t, server)

	// Open CSS file with reference to unknown token
	content := `/* Test file */
.button {
    color: var(--unknown-token);
}`
	req := types.NewRequestContext(server, nil)
	err := textDocument.DidOpen(req, &protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{
			URI:        "file:///test.css",
			LanguageID: "css",
			Version:    1,
			Text:       content,
		},
	})
	require.NoError(t, err)

	// Request definition on unknown token
	req = types.NewRequestContext(server, nil)
	result, err := definition.Definition(req, &protocol.DefinitionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{
				URI: "file:///test.css",
			},
			Position: protocol.Position{
				Line:      2,
				Character: 18,
			},
		},
	})

	require.NoError(t, err)
	assert.Nil(t, result)
}
