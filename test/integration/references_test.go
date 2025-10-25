package integration_test

import (
	"testing"

	"github.com/bennypowers/design-tokens-language-server/test/integration/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// TestReferencesOnVarCall tests find-all-references on a var() call
func TestReferencesOnVarCall(t *testing.T) {
	server := testutil.NewTestServer(t)
	testutil.LoadBasicTokens(t, server)
	testutil.OpenCSSFixture(t, server, "file:///test.css", "basic-var-calls.css")

	// Request references - see fixture for position
	locations, err := server.GetReferences(&protocol.ReferenceParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{
				URI: "file:///test.css",
			},
			Position: protocol.Position{
				Line:      2, // Adjusted for comment line
				Character: 18, // Inside first --color-primary
			},
		},
		Context: protocol.ReferenceContext{
			IncludeDeclaration: false,
		},
	})

	require.NoError(t, err)
	require.NotNil(t, locations)

	// Should find the var() call
	assert.GreaterOrEqual(t, len(locations), 1)

	// Check that location is in the same file
	for _, loc := range locations {
		assert.Equal(t, "file:///test.css", loc.URI)
	}
}

// TestReferencesMultipleFiles tests references across multiple files
func TestReferencesMultipleFiles(t *testing.T) {
	server := testutil.NewTestServer(t)
	testutil.LoadBasicTokens(t, server)

	// Open two CSS files
	testutil.OpenCSSFixture(t, server, "file:///test1.css", "references-multi-file-1.css")
	testutil.OpenCSSFixture(t, server, "file:///test2.css", "references-multi-file-2.css")

	// Request references from first file - see fixture for position
	locations, err := server.GetReferences(&protocol.ReferenceParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{
				URI: "file:///test1.css",
			},
			Position: protocol.Position{
				Line:      2, // Adjusted for comment line
				Character: 18,
			},
		},
		Context: protocol.ReferenceContext{
			IncludeDeclaration: false,
		},
	})

	require.NoError(t, err)
	require.NotNil(t, locations)

	// Should find references in both files
	assert.GreaterOrEqual(t, len(locations), 2)

	// Check that we have references from both files
	fileURIs := make(map[string]bool)
	for _, loc := range locations {
		fileURIs[loc.URI] = true
	}

	assert.True(t, fileURIs["file:///test1.css"], "Should have reference in test1.css")
	assert.True(t, fileURIs["file:///test2.css"], "Should have reference in test2.css")
}

// TestReferencesOutsideVarCall tests that references returns nil outside var() calls
func TestReferencesOutsideVarCall(t *testing.T) {
	server := testutil.NewTestServer(t)
	testutil.OpenCSSFixture(t, server, "file:///test.css", "no-var-call.css")

	// Request references - see fixture for position
	locations, err := server.GetReferences(&protocol.ReferenceParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{
				URI: "file:///test.css",
			},
			Position: protocol.Position{
				Line:      2, // Adjusted for comment line
				Character: 10, // On "red"
			},
		},
		Context: protocol.ReferenceContext{
			IncludeDeclaration: false,
		},
	})

	require.NoError(t, err)
	assert.Nil(t, locations)
}

// TestReferencesWithDeclaration tests including the token declaration
func TestReferencesWithDeclaration(t *testing.T) {
	server := testutil.NewTestServer(t)
	testutil.LoadBasicTokens(t, server)
	testutil.OpenCSSFixture(t, server, "file:///test.css", "basic-var-calls.css")

	// Request references with IncludeDeclaration
	locations, err := server.GetReferences(&protocol.ReferenceParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{
				URI: "file:///test.css",
			},
			Position: protocol.Position{
				Line:      2,
				Character: 18,
			},
		},
		Context: protocol.ReferenceContext{
			IncludeDeclaration: true, // Request declaration
		},
	})

	require.NoError(t, err)
	require.NotNil(t, locations)
	assert.GreaterOrEqual(t, len(locations), 1)
}
