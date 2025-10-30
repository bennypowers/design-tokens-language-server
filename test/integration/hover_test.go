package integration_test

import (
	"testing"

	"bennypowers.dev/dtls/lsp/methods/textDocument"
	"bennypowers.dev/dtls/lsp/methods/textDocument/hover"
	"bennypowers.dev/dtls/lsp/types"
	"bennypowers.dev/dtls/test/integration/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// TestHoverOnVarCall tests hover on a var() function call
func TestHoverOnVarCall(t *testing.T) {
	server := testutil.NewTestServer(t)
	testutil.LoadBasicTokens(t, server)
	testutil.OpenCSSFixture(t, server, "file:///test.css", "basic-var-calls.css")

	// Request hover - see fixture for position
	req := types.NewRequestContext(server, nil)
	result, err := hover.Hover(req, &protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{
				URI: "file:///test.css",
			},
			Position: protocol.Position{
				Line:      2, // Adjusted for comment line
				Character: 18,
			},
		},
	})

	require.NoError(t, err)
	require.NotNil(t, result, "Hover should return content")

	// Verify hover content
	content, ok := result.Contents.(protocol.MarkupContent)
	require.True(t, ok, "Contents should be MarkupContent")

	assert.Equal(t, protocol.MarkupKindMarkdown, content.Kind)
	assert.Contains(t, content.Value, "--color-primary")
	assert.Contains(t, content.Value, "#0000ff")
	assert.Contains(t, content.Value, "Primary brand color")
	assert.Contains(t, content.Value, "color")
}

// TestHoverOnUnknownToken tests hover on undefined token
func TestHoverOnUnknownToken(t *testing.T) {
	server := testutil.NewTestServer(t)
	// Don't load any tokens
	testutil.OpenCSSFixture(t, server, "file:///test.css", "unknown-token.css")

	// Request hover - see fixture for position
	req := types.NewRequestContext(server, nil)
	result, err := hover.Hover(req, &protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{
				URI: "file:///test.css",
			},
			Position: protocol.Position{
				Line:      2, // Adjusted for comment line
				Character: 15,
			},
		},
	})

	require.NoError(t, err)
	require.NotNil(t, result)

	content, ok := result.Contents.(protocol.MarkupContent)
	require.True(t, ok)

	assert.Contains(t, content.Value, "Unknown token")
	assert.Contains(t, content.Value, "--unknown-token")
}

// TestHoverWithPrefix tests hover with CSS variable prefix
func TestHoverWithPrefix(t *testing.T) {
	server := testutil.NewTestServer(t)
	testutil.LoadTokensWithPrefix(t, server, "my-ds")
	testutil.OpenCSSFixture(t, server, "file:///test.css", "prefixed-var-call.css")

	// Request hover - see fixture for position
	req := types.NewRequestContext(server, nil)
	result, err := hover.Hover(req, &protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{
				URI: "file:///test.css",
			},
			Position: protocol.Position{
				Line:      2, // Adjusted for comment line
				Character: 18,
			},
		},
	})

	require.NoError(t, err)
	require.NotNil(t, result)

	content, ok := result.Contents.(protocol.MarkupContent)
	require.True(t, ok)

	assert.Contains(t, content.Value, "--my-ds-color-primary")
	assert.Contains(t, content.Value, "#ff0000")
}

// TestHoverOutsideVarCall tests that hover returns nil outside var() calls
func TestHoverOutsideVarCall(t *testing.T) {
	server := testutil.NewTestServer(t)
	testutil.OpenCSSFixture(t, server, "file:///test.css", "no-var-call.css")

	// Request hover - see fixture for position
	req := types.NewRequestContext(server, nil)
	result, err := hover.Hover(req, &protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{
				URI: "file:///test.css",
			},
			Position: protocol.Position{
				Line:      2, // Adjusted for comment line
				Character: 10,
			},
		},
	})

	require.NoError(t, err)
	assert.Nil(t, result, "Should return nil when not hovering over var() call")
}

// TestHoverNonCSSFile tests that hover returns nil for non-CSS files
func TestHoverNonCSSFile(t *testing.T) {
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

	// Request hover
	req = types.NewRequestContext(server, nil)
	result, err := hover.Hover(req, &protocol.HoverParams{
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

// TestHoverOnVariableDeclaration tests hover on CSS variable declaration
func TestHoverOnVariableDeclaration(t *testing.T) {
	server := testutil.NewTestServer(t)
	testutil.LoadBasicTokens(t, server)

	// Open CSS file with variable declaration
	content := `:root {
    --color-primary: #0000ff;
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

	// Request hover on the variable name
	req = types.NewRequestContext(server, nil)
	result, err := hover.Hover(req, &protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{
				URI: "file:///test.css",
			},
			Position: protocol.Position{
				Line:      1,
				Character: 8, // On "--color-primary"
			},
		},
	})

	require.NoError(t, err)
	require.NotNil(t, result)

	content_hover, ok := result.Contents.(protocol.MarkupContent)
	require.True(t, ok)
	assert.Contains(t, content_hover.Value, "--color-primary")
	assert.Contains(t, content_hover.Value, "#0000ff")
}

// TestHoverOnVariableDeclarationUnknown tests hover on unknown variable declaration
func TestHoverOnVariableDeclarationUnknown(t *testing.T) {
	server := testutil.NewTestServer(t)
	testutil.LoadBasicTokens(t, server)

	// Open CSS file with unknown variable declaration
	content := `:root {
    --unknown-var: #123456;
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

	// Request hover on the unknown variable name
	req = types.NewRequestContext(server, nil)
	result, err := hover.Hover(req, &protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{
				URI: "file:///test.css",
			},
			Position: protocol.Position{
				Line:      1,
				Character: 8, // On "--unknown-var"
			},
		},
	})

	require.NoError(t, err)
	assert.Nil(t, result) // Should return nil for unknown variable declaration
}

// TestHoverWithDeprecated tests hover on deprecated token
func TestHoverWithDeprecated(t *testing.T) {
	server := testutil.NewTestServer(t)

	// Load deprecated token
	tokens := []byte(`{
		"color-old": {
			"$type": "color",
			"$value": "#ff0000",
			"$description": "Old color",
			"$deprecated": true
		}
	}`)
	err := server.LoadTokensFromJSON(tokens, "")
	require.NoError(t, err)

	// Open CSS file using deprecated token
	content := `.button {
    color: var(--color-old);
}`
	req := types.NewRequestContext(server, nil)
	err = textDocument.DidOpen(req, &protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{
			URI:        "file:///test.css",
			LanguageID: "css",
			Version:    1,
			Text:       content,
		},
	})
	require.NoError(t, err)

	// Request hover
	req = types.NewRequestContext(server, nil)
	result, err := hover.Hover(req, &protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{
				URI: "file:///test.css",
			},
			Position: protocol.Position{
				Line:      1,
				Character: 18,
			},
		},
	})

	require.NoError(t, err)
	require.NotNil(t, result)

	content_hover, ok := result.Contents.(protocol.MarkupContent)
	require.True(t, ok)
	assert.Contains(t, content_hover.Value, "DEPRECATED")
}
