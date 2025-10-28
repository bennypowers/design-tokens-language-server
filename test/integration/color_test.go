package integration_test

import (
	"testing"

	"bennypowers.dev/dtls/lsp/methods/textDocument"
	documentcolor "bennypowers.dev/dtls/lsp/methods/textDocument/documentColor"
	"bennypowers.dev/dtls/lsp/types"
	"bennypowers.dev/dtls/test/integration/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// TestDocumentColorBasic tests basic document color functionality
func TestDocumentColorBasic(t *testing.T) {
	server := testutil.NewTestServer(t)
	testutil.LoadBasicTokens(t, server)
	testutil.OpenCSSFixture(t, server, "file:///test.css", "colors-basic.css")

	// Request document colors
	req := types.NewRequestContext(server, nil)
	colors, err := documentcolor.DocumentColor(req, &protocol.DocumentColorParams{
		TextDocument: protocol.TextDocumentIdentifier{
			URI: "file:///test.css",
		},
	})

	require.NoError(t, err)
	require.NotNil(t, colors)

	// Should have 2 colors (primary and secondary)
	assert.Len(t, colors, 2)

	// Check first color (--color-primary: #0000ff - blue)
	if len(colors) >= 1 {
		assert.Equal(t, float32(0.0), colors[0].Color.Red)
		assert.Equal(t, float32(0.0), colors[0].Color.Green)
		assert.Equal(t, float32(1.0), colors[0].Color.Blue)
		assert.Equal(t, float32(1.0), colors[0].Color.Alpha)
	}

	// Check second color (--color-secondary: #00ff00 - green)
	if len(colors) >= 2 {
		assert.Equal(t, float32(0.0), colors[1].Color.Red)
		assert.Equal(t, float32(1.0), colors[1].Color.Green)
		assert.Equal(t, float32(0.0), colors[1].Color.Blue)
		assert.Equal(t, float32(1.0), colors[1].Color.Alpha)
	}
}

// TestDocumentColorMixed tests that only color tokens show colors
func TestDocumentColorMixed(t *testing.T) {
	server := testutil.NewTestServer(t)
	testutil.LoadBasicTokens(t, server)
	testutil.OpenCSSFixture(t, server, "file:///test.css", "colors-mixed.css")

	// Request document colors
	req := types.NewRequestContext(server, nil)
	colors, err := documentcolor.DocumentColor(req, &protocol.DocumentColorParams{
		TextDocument: protocol.TextDocumentIdentifier{
			URI: "file:///test.css",
		},
	})

	require.NoError(t, err)
	require.NotNil(t, colors)

	// Should have only 2 colors (not the spacing dimension)
	assert.Len(t, colors, 2)

	// Verify they are both color type
	for _, colorInfo := range colors {
		// Colors should have valid RGB values
		assert.GreaterOrEqual(t, colorInfo.Color.Red, float32(0.0))
		assert.LessOrEqual(t, colorInfo.Color.Red, float32(1.0))
		assert.GreaterOrEqual(t, colorInfo.Color.Green, float32(0.0))
		assert.LessOrEqual(t, colorInfo.Color.Green, float32(1.0))
		assert.GreaterOrEqual(t, colorInfo.Color.Blue, float32(0.0))
		assert.LessOrEqual(t, colorInfo.Color.Blue, float32(1.0))
	}
}

// TestDocumentColorEmpty tests document with no color tokens
func TestDocumentColorEmpty(t *testing.T) {
	server := testutil.NewTestServer(t)
	testutil.LoadBasicTokens(t, server)
	testutil.OpenCSSFixture(t, server, "file:///test.css", "no-var-call.css")

	// Request document colors
	req := types.NewRequestContext(server, nil)
	colors, err := documentcolor.DocumentColor(req, &protocol.DocumentColorParams{
		TextDocument: protocol.TextDocumentIdentifier{
			URI: "file:///test.css",
		},
	})

	require.NoError(t, err)

	// Should have no colors
	if colors != nil {
		assert.Len(t, colors, 0)
	}
}

// TestColorPresentation tests that ColorPresentation returns matching token names
func TestColorPresentation(t *testing.T) {
	server := testutil.NewTestServer(t)
	testutil.LoadBasicTokens(t, server)

	// Request color presentations for blue (#0000ff from basic tokens)
	req := types.NewRequestContext(server, nil)
	presentations, err := documentcolor.ColorPresentation(req, &protocol.ColorPresentationParams{
		TextDocument: protocol.TextDocumentIdentifier{
			URI: "file:///test.css",
		},
		Color: protocol.Color{
			Red:   0.0,
			Green: 0.0,
			Blue:  1.0,
			Alpha: 1.0,
		},
		Range: protocol.Range{
			Start: protocol.Position{Line: 0, Character: 0},
			End:   protocol.Position{Line: 0, Character: 10},
		},
	})

	require.NoError(t, err)
	require.NotNil(t, presentations)

	// Should return token names that match this color
	labels := make([]string, len(presentations))
	for i, p := range presentations {
		labels[i] = p.Label
	}

	// Basic tokens should have color-primary = #0000ff (blue)
	assert.Contains(t, labels, "color-primary")
}

// TestColorPresentationWithAlpha tests color presentation with alpha channel
func TestColorPresentationWithAlpha(t *testing.T) {
	server := testutil.NewTestServer(t)

	// Load tokens with alpha channel
	tokens := []byte(`{
		"color-overlay": {
			"$type": "color",
			"$value": "rgba(255, 0, 0, 0.5)"
		},
		"color-solid": {
			"$type": "color",
			"$value": "#ff0000"
		}
	}`)
	err := server.LoadTokensFromJSON(tokens, "")
	require.NoError(t, err)

	// Request color presentations for semi-transparent red
	req := types.NewRequestContext(server, nil)
	presentations, err := documentcolor.ColorPresentation(req, &protocol.ColorPresentationParams{
		TextDocument: protocol.TextDocumentIdentifier{
			URI: "file:///test.css",
		},
		Color: protocol.Color{
			Red:   1.0,
			Green: 0.0,
			Blue:  0.0,
			Alpha: 0.5,
		},
		Range: protocol.Range{
			Start: protocol.Position{Line: 0, Character: 0},
			End:   protocol.Position{Line: 0, Character: 10},
		},
	})

	require.NoError(t, err)
	require.NotNil(t, presentations)

	// Should return only tokens with matching alpha
	labels := make([]string, len(presentations))
	for i, p := range presentations {
		labels[i] = p.Label
	}

	// Should match the semi-transparent token, not the solid one
	assert.Contains(t, labels, "color-overlay")
	assert.NotContains(t, labels, "color-solid") // Different alpha
}

// TestDocumentColorNonCSSFile tests that color returns nil for non-CSS files
func TestDocumentColorNonCSSFile(t *testing.T) {
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

	// Request colors
	req = types.NewRequestContext(server, nil)
	colors, err := documentcolor.DocumentColor(req, &protocol.DocumentColorParams{
		TextDocument: protocol.TextDocumentIdentifier{
			URI: "file:///test.json",
		},
	})

	require.NoError(t, err)
	assert.Nil(t, colors)
}

// TestDocumentColorVariables tests colors from CSS variable declarations
func TestDocumentColorVariables(t *testing.T) {
	server := testutil.NewTestServer(t)
	testutil.LoadBasicTokens(t, server)

	// Open CSS file with variable declarations
	content := `:root {
    --color-primary: #0000ff;
    --color-secondary: #00ff00;
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

	// Request colors
	req = types.NewRequestContext(server, nil)
	colors, err := documentcolor.DocumentColor(req, &protocol.DocumentColorParams{
		TextDocument: protocol.TextDocumentIdentifier{
			URI: "file:///test.css",
		},
	})

	require.NoError(t, err)
	require.NotNil(t, colors)
	// Should have colors from variable declarations
	assert.GreaterOrEqual(t, len(colors), 1)
}

// TestDocumentColorInvalidColorValue tests handling of invalid color values
func TestDocumentColorInvalidColorValue(t *testing.T) {
	server := testutil.NewTestServer(t)

	// Load token with invalid color value
	tokens := []byte(`{
		"color-invalid": {
			"$type": "color",
			"$value": "not-a-color"
		}
	}`)
	err := server.LoadTokensFromJSON(tokens, "")
	require.NoError(t, err)

	// Open CSS file using the invalid color
	content := `.button {
    color: var(--color-invalid);
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

	// Request colors
	req = types.NewRequestContext(server, nil)
	colors, err := documentcolor.DocumentColor(req, &protocol.DocumentColorParams{
		TextDocument: protocol.TextDocumentIdentifier{
			URI: "file:///test.css",
		},
	})

	require.NoError(t, err)
	// Should skip invalid colors
	if colors != nil {
		assert.Len(t, colors, 0)
	}
}
