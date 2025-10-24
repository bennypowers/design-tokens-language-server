package integration_test

import (
	"testing"

	"github.com/bennypowers/design-tokens-language-server/internal/lsp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// TestHoverOnVarCall tests hover on a var() function call
func TestHoverOnVarCall(t *testing.T) {
	// Create server
	server, err := lsp.NewServer()
	require.NoError(t, err)

	// Load test tokens
	tokenJSON := []byte(`{
		"color": {
			"primary": {
				"$value": "#0000ff",
				"$type": "color",
				"$description": "Primary brand color"
			}
		}
	}`)

	err = server.LoadTokensFromJSON(tokenJSON, "")
	require.NoError(t, err)

	// Open a CSS document
	cssContent := `.button {
  color: var(--color-primary);
}`

	err = server.DidOpen("file:///test.css", "css", 1, cssContent)
	require.NoError(t, err)

	// Request hover on the var(--color-primary)
	// Line 1, character 15 should be inside "color-primary"
	hover, err := server.Hover(&protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{
				URI: "file:///test.css",
			},
			Position: protocol.Position{
				Line:      1,
				Character: 15,
			},
		},
	})

	require.NoError(t, err)
	require.NotNil(t, hover, "Hover should return content")

	// Verify hover content
	content, ok := hover.Contents.(protocol.MarkupContent)
	require.True(t, ok, "Contents should be MarkupContent")

	assert.Equal(t, protocol.MarkupKindMarkdown, content.Kind)
	assert.Contains(t, content.Value, "--color-primary")
	assert.Contains(t, content.Value, "#0000ff")
	assert.Contains(t, content.Value, "Primary brand color")
	assert.Contains(t, content.Value, "color")
}

// TestHoverOnUnknownToken tests hover on undefined token
func TestHoverOnUnknownToken(t *testing.T) {
	server, err := lsp.NewServer()
	require.NoError(t, err)

	// Don't load any tokens

	cssContent := `.button {
  color: var(--unknown-token);
}`

	err = server.DidOpen("file:///test.css", "css", 1, cssContent)
	require.NoError(t, err)

	hover, err := server.Hover(&protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{
				URI: "file:///test.css",
			},
			Position: protocol.Position{
				Line:      1,
				Character: 15,
			},
		},
	})

	require.NoError(t, err)
	require.NotNil(t, hover)

	content, ok := hover.Contents.(protocol.MarkupContent)
	require.True(t, ok)

	assert.Contains(t, content.Value, "Unknown token")
	assert.Contains(t, content.Value, "--unknown-token")
}

// TestHoverWithPrefix tests hover with CSS variable prefix
func TestHoverWithPrefix(t *testing.T) {
	server, err := lsp.NewServer()
	require.NoError(t, err)

	// Load tokens with prefix
	tokenJSON := []byte(`{
		"color": {
			"primary": {
				"$value": "#ff0000",
				"$type": "color"
			}
		}
	}`)

	err = server.LoadTokensFromJSON(tokenJSON, "my-ds")
	require.NoError(t, err)

	cssContent := `.button {
  color: var(--my-ds-color-primary);
}`

	err = server.DidOpen("file:///test.css", "css", 1, cssContent)
	require.NoError(t, err)

	hover, err := server.Hover(&protocol.HoverParams{
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
	require.NotNil(t, hover)

	content, ok := hover.Contents.(protocol.MarkupContent)
	require.True(t, ok)

	assert.Contains(t, content.Value, "--my-ds-color-primary")
	assert.Contains(t, content.Value, "#ff0000")
}

// TestHoverOutsideVarCall tests that hover returns nil outside var() calls
func TestHoverOutsideVarCall(t *testing.T) {
	server, err := lsp.NewServer()
	require.NoError(t, err)

	cssContent := `.button {
  color: red;
}`

	err = server.DidOpen("file:///test.css", "css", 1, cssContent)
	require.NoError(t, err)

	// Hover on "red" (not a var call)
	hover, err := server.Hover(&protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{
				URI: "file:///test.css",
			},
			Position: protocol.Position{
				Line:      1,
				Character: 10,
			},
		},
	})

	require.NoError(t, err)
	assert.Nil(t, hover, "Should return nil when not hovering over var() call")
}
