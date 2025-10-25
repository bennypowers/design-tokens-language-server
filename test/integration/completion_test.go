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

// Note: UTF-16 position handling is tested via:
// - internal/documents tests (UTF-16 incremental edits)
// - internal/position tests (UTF-16 conversion functions)
// - Existing completion tests implicitly test getWordAtPosition with UTF-16
