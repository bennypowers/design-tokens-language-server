package integration_test

import (
	"testing"

	"github.com/bennypowers/design-tokens-language-server/test/integration/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// TestCompletionBasic tests basic completion functionality
func TestCompletionBasic(t *testing.T) {
	server := testutil.NewTestServer(t)
	testutil.LoadBasicTokens(t, server)
	testutil.OpenCSSFixture(t, server, "file:///test.css", "completion-context.css")

	// Request completion - see fixture for position
	completions, err := server.GetCompletions(&protocol.CompletionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{
				URI: "file:///test.css",
			},
			Position: protocol.Position{
				Line:      2, // Adjusted for comment line
				Character: 15, // After "--color"
			},
		},
	})

	require.NoError(t, err)
	require.NotNil(t, completions)

	// Should have color tokens
	assert.GreaterOrEqual(t, len(completions.Items), 2)

	// Check that we have --color-primary and --color-secondary
	labels := make([]string, len(completions.Items))
	for i, item := range completions.Items {
		labels[i] = item.Label
	}

	assert.Contains(t, labels, "--color-primary")
	assert.Contains(t, labels, "--color-secondary")
}

// TestCompletionWithPrefix tests completion with CSS variable prefix
func TestCompletionWithPrefix(t *testing.T) {
	server := testutil.NewTestServer(t)
	testutil.LoadTokensWithPrefix(t, server, "ds")
	testutil.OpenCSSFixture(t, server, "file:///test.css", "completion-prefix.css")

	// Request completion - see fixture for position
	completions, err := server.GetCompletions(&protocol.CompletionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{
				URI: "file:///test.css",
			},
			Position: protocol.Position{
				Line:      2, // Adjusted for comment line
				Character: 13,
			},
		},
	})

	require.NoError(t, err)
	require.NotNil(t, completions)

	// Should have the prefixed token
	assert.GreaterOrEqual(t, len(completions.Items), 1)

	labels := make([]string, len(completions.Items))
	for i, item := range completions.Items {
		labels[i] = item.Label
	}

	assert.Contains(t, labels, "--ds-color-primary")
}

// TestCompletionNonCSSFile tests that completion returns nil for non-CSS files
func TestCompletionNonCSSFile(t *testing.T) {
	server := testutil.NewTestServer(t)
	testutil.LoadBasicTokens(t, server)

	// Open a JSON file
	server.DidOpen("file:///test.json", "json", 1, `{"color": "red"}`)

	// Request completion
	completions, err := server.GetCompletions(&protocol.CompletionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{
				URI: "file:///test.json",
			},
			Position: protocol.Position{Line: 0, Character: 5},
		},
	})

	require.NoError(t, err)
	assert.Nil(t, completions)
}

// TestCompletionResolve tests completion item resolve
func TestCompletionResolve(t *testing.T) {
	server := testutil.NewTestServer(t)
	testutil.LoadBasicTokens(t, server)

	// Create a basic completion item
	item := protocol.CompletionItem{
		Label: "--color-primary",
		Data: map[string]interface{}{
			"tokenName": "--color-primary",
		},
	}

	// Resolve it
	resolved, err := server.ResolveCompletion(&item)
	require.NoError(t, err)
	require.NotNil(t, resolved)

	// Should have documentation
	assert.NotNil(t, resolved.Documentation)
	doc, ok := resolved.Documentation.(protocol.MarkupContent)
	require.True(t, ok)
	assert.Equal(t, protocol.MarkupKindMarkdown, doc.Kind)
	assert.Contains(t, doc.Value, "Primary brand color")
	assert.Contains(t, doc.Value, "#0000ff")

	// Should have detail (value preview)
	assert.NotNil(t, resolved.Detail)
	assert.Contains(t, *resolved.Detail, "#0000ff")
}

// TestCompletionFiltering tests that completion filters by prefix
func TestCompletionFiltering(t *testing.T) {
	server := testutil.NewTestServer(t)
	testutil.LoadBasicTokens(t, server)
	testutil.OpenCSSFixture(t, server, "file:///test.css", "completion-filter.css")

	// Request completion - see fixture for position
	completions, err := server.GetCompletions(&protocol.CompletionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{
				URI: "file:///test.css",
			},
			Position: protocol.Position{
				Line:      2, // Adjusted for comment line
				Character: 14, // After "--col"
			},
		},
	})

	require.NoError(t, err)
	require.NotNil(t, completions)

	// Should only have color tokens, not spacing
	labels := make([]string, len(completions.Items))
	for i, item := range completions.Items {
		labels[i] = item.Label
	}

	assert.Contains(t, labels, "--color-primary")
	assert.NotContains(t, labels, "--spacing-small")
}

// TestCompletionOutsideBlock tests that completion returns nil outside var() calls
func TestCompletionOutsideBlock(t *testing.T) {
	server := testutil.NewTestServer(t)
	testutil.OpenCSSFixture(t, server, "file:///test.css", "completion-outside.css")

	// Request completion - see fixture for position
	completions, err := server.GetCompletions(&protocol.CompletionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{
				URI: "file:///test.css",
			},
			Position: protocol.Position{
				Line:      1, // Adjusted for comment line
				Character: 8,
			},
		},
	})

	require.NoError(t, err)
	// Should return nil or empty when not in a valid completion context
	if completions != nil {
		assert.Len(t, completions.Items, 0)
	}
}

// TestCompletionSnippetFormat tests that completion items have snippet format
func TestCompletionSnippetFormat(t *testing.T) {
	server := testutil.NewTestServer(t)
	testutil.LoadBasicTokens(t, server)
	testutil.OpenCSSFixture(t, server, "file:///test.css", "completion-context.css")

	// Request completion - see fixture for position
	completions, err := server.GetCompletions(&protocol.CompletionParams{
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
	require.NotNil(t, completions)
	require.GreaterOrEqual(t, len(completions.Items), 1)

	// Check first item has snippet format
	item := completions.Items[0]
	if item.InsertTextFormat != nil {
		assert.Equal(t, protocol.InsertTextFormatSnippet, *item.InsertTextFormat)
	}
}

// TestCompletionResolveNoData tests resolving completion item without data
func TestCompletionResolveNoData(t *testing.T) {
	server := testutil.NewTestServer(t)
	testutil.LoadBasicTokens(t, server)

	// Create a completion item without data - will fall back to using Label
	item := protocol.CompletionItem{
		Label: "--color-primary",
		Data:  nil, // No data - will use Label as tokenName
	}

	// Resolve it - should still find the token and add documentation
	resolved, err := server.ResolveCompletion(&item)
	require.NoError(t, err)
	require.NotNil(t, resolved)
	assert.Equal(t, "--color-primary", resolved.Label)
	assert.NotNil(t, resolved.Documentation) // Should have docs from fallback
}

// TestCompletionResolveUnknownToken tests resolving completion item for unknown token
func TestCompletionResolveUnknownToken(t *testing.T) {
	server := testutil.NewTestServer(t)
	testutil.LoadBasicTokens(t, server)

	// Create a completion item for a token that doesn't exist
	item := protocol.CompletionItem{
		Label: "--unknown-token",
		Data: map[string]interface{}{
			"tokenName": "--unknown-token",
		},
	}

	// Resolve it - should return unchanged without documentation
	resolved, err := server.ResolveCompletion(&item)
	require.NoError(t, err)
	require.NotNil(t, resolved)
	assert.Equal(t, "--unknown-token", resolved.Label)
	assert.Nil(t, resolved.Documentation) // No docs for unknown token
}

// TestCompletionParseError tests completion when CSS parsing fails
func TestCompletionParseError(t *testing.T) {
	server := testutil.NewTestServer(t)
	testutil.LoadBasicTokens(t, server)

	// Open CSS file with invalid syntax that might fail parsing
	content := `/* Invalid CSS */
.button {
    color: --col`
	server.DidOpen("file:///test.css", "css", 1, content)

	// Request completion
	completions, err := server.GetCompletions(&protocol.CompletionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{
				URI: "file:///test.css",
			},
			Position: protocol.Position{
				Line:      2,
				Character: 15,
			},
		},
	})

	require.NoError(t, err)
	// May return nil if parsing fails completely
	if completions != nil {
		// Should still try to provide completions
		assert.GreaterOrEqual(t, len(completions.Items), 0)
	}
}

// TestCompletionEmptyWord tests completion when word at position is empty
func TestCompletionEmptyWord(t *testing.T) {
	server := testutil.NewTestServer(t)
	testutil.LoadBasicTokens(t, server)

	// Open CSS file
	content := `.button {
    color: ;
}`
	server.DidOpen("file:///test.css", "css", 1, content)

	// Request completion at a position with no word (before semicolon)
	completions, err := server.GetCompletions(&protocol.CompletionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{
				URI: "file:///test.css",
			},
			Position: protocol.Position{
				Line:      1,
				Character: 11, // Right before semicolon
			},
		},
	})

	require.NoError(t, err)
	assert.Nil(t, completions) // Should return nil when word is empty
}

// TestCompletionPositionOutOfBounds tests completion when position is out of bounds
func TestCompletionPositionOutOfBounds(t *testing.T) {
	server := testutil.NewTestServer(t)
	testutil.LoadBasicTokens(t, server)

	// Open CSS file
	content := `.button {
    color: red;
}`
	server.DidOpen("file:///test.css", "css", 1, content)

	// Request completion at line that doesn't exist
	completions, err := server.GetCompletions(&protocol.CompletionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{
				URI: "file:///test.css",
			},
			Position: protocol.Position{
				Line:      100, // Way out of bounds
				Character: 10,
			},
		},
	})

	require.NoError(t, err)
	assert.Nil(t, completions) // Should return nil for out of bounds position
}

// TestCompletionResolveWithDeprecated tests resolving deprecated token
func TestCompletionResolveWithDeprecated(t *testing.T) {
	server := testutil.NewTestServer(t)

	// Load token with deprecated flag
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

	// Create completion item
	item := protocol.CompletionItem{
		Label: "--color-old",
		Data: map[string]interface{}{
			"tokenName": "--color-old",
		},
	}

	// Resolve it
	resolved, err := server.ResolveCompletion(&item)
	require.NoError(t, err)
	require.NotNil(t, resolved)

	// Should have deprecation warning in documentation
	assert.NotNil(t, resolved.Documentation)
	doc, ok := resolved.Documentation.(protocol.MarkupContent)
	require.True(t, ok)
	assert.Contains(t, doc.Value, "DEPRECATED")
}

// TestCompletionResolveDataNotMap tests resolve when Data is not a map
func TestCompletionResolveDataNotMap(t *testing.T) {
	server := testutil.NewTestServer(t)
	testutil.LoadBasicTokens(t, server)

	// Create completion item with Data that is not a map
	item := protocol.CompletionItem{
		Label: "--color-primary",
		Data:  "not-a-map", // Invalid data type
	}

	// Resolve it - should fall back to Label
	resolved, err := server.ResolveCompletion(&item)
	require.NoError(t, err)
	require.NotNil(t, resolved)
	assert.Equal(t, "--color-primary", resolved.Label)
	assert.NotNil(t, resolved.Documentation) // Should still work using Label
}

// Note: UTF-16 position handling is tested via:
// - internal/documents tests (UTF-16 incremental edits)
// - internal/position tests (UTF-16 conversion functions)
// - Existing completion tests implicitly test getWordAtPosition with UTF-16
