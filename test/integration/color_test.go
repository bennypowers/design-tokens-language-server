package integration_test

import (
	"testing"

	"github.com/bennypowers/design-tokens-language-server/test/integration/testutil"
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
	colors, err := server.DocumentColor(&protocol.DocumentColorParams{
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
	colors, err := server.DocumentColor(&protocol.DocumentColorParams{
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
	colors, err := server.DocumentColor(&protocol.DocumentColorParams{
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

// TestColorPresentation tests color presentation formatting
func TestColorPresentation(t *testing.T) {
	server := testutil.NewTestServer(t)

	// Request color presentations for a blue color
	presentations, err := server.ColorPresentation(&protocol.ColorPresentationParams{
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

	// Should have multiple format options (hex, rgb, rgba, hsl)
	assert.GreaterOrEqual(t, len(presentations), 3)

	// Check for hex format
	labels := make([]string, len(presentations))
	for i, p := range presentations {
		labels[i] = p.Label
	}

	assert.Contains(t, labels, "#0000ff")
	assert.Contains(t, labels, "rgb(0, 0, 255)")
}

// TestColorPresentationWithAlpha tests color presentation with alpha channel
func TestColorPresentationWithAlpha(t *testing.T) {
	server := testutil.NewTestServer(t)

	// Request color presentations for a semi-transparent red
	presentations, err := server.ColorPresentation(&protocol.ColorPresentationParams{
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

	// Check for hex with alpha format
	labels := make([]string, len(presentations))
	for i, p := range presentations {
		labels[i] = p.Label
	}

	// Hex with alpha should include alpha channel (0.5 * 255 = 127.5 -> 0x7f)
	assert.Contains(t, labels, "#ff00007f")
	// RGBA should show alpha
	assert.Contains(t, labels, "rgba(255, 0, 0, 0.50)")
}
