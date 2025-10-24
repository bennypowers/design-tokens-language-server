package integration_test

import (
	"testing"

	"github.com/bennypowers/design-tokens-language-server/test/integration/testutil"
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
	hover, err := server.Hover(&protocol.HoverParams{
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
	server := testutil.NewTestServer(t)
	// Don't load any tokens
	testutil.OpenCSSFixture(t, server, "file:///test.css", "unknown-token.css")

	// Request hover - see fixture for position
	hover, err := server.Hover(&protocol.HoverParams{
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
	require.NotNil(t, hover)

	content, ok := hover.Contents.(protocol.MarkupContent)
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
	hover, err := server.Hover(&protocol.HoverParams{
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
	require.NotNil(t, hover)

	content, ok := hover.Contents.(protocol.MarkupContent)
	require.True(t, ok)

	assert.Contains(t, content.Value, "--my-ds-color-primary")
	assert.Contains(t, content.Value, "#ff0000")
}

// TestHoverOutsideVarCall tests that hover returns nil outside var() calls
func TestHoverOutsideVarCall(t *testing.T) {
	server := testutil.NewTestServer(t)
	testutil.OpenCSSFixture(t, server, "file:///test.css", "no-var-call.css")

	// Request hover - see fixture for position
	hover, err := server.Hover(&protocol.HoverParams{
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
	assert.Nil(t, hover, "Should return nil when not hovering over var() call")
}
