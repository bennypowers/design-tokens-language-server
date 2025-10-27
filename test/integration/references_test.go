package integration_test

import (
	"testing"

	"bennypowers.dev/dtls/lsp/methods/textDocument"
	"bennypowers.dev/dtls/lsp/methods/textDocument/references"
	"bennypowers.dev/dtls/test/integration/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// TestReferencesOnVarCall tests find-all-references from a token file
// New behavior: references is called on JSON/YAML files, finds CSS var() references
func TestReferencesOnVarCall(t *testing.T) {
	server := testutil.NewTestServer(t)

	// Open CSS file with var() call
	testutil.OpenCSSFixture(t, server, "file:///test.css", "basic-var-calls.css")

	// Load and open token file
	tokenContent := `{
  "color": {
    "primary": {
      "$type": "color",
      "$value": "#ff0000"
    }
  }
}`
	// Load tokens into token manager
	err := server.LoadTokensFromJSON([]byte(tokenContent), "")
	require.NoError(t, err)

	// Open the token file as a document
	err = textDocument.DidOpen(server, nil, &protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{
			URI:        "file:///tokens.json",
			LanguageID: "json",
			Version:    1,
			Text:       tokenContent,
		},
	})
	require.NoError(t, err)

	// Request references from the token file (cursor on "primary")
	locations, err := references.References(server, nil, &protocol.ReferenceParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{
				URI: "file:///tokens.json",
			},
			Position: protocol.Position{
				Line:      2, // "primary" key
				Character: 6,
			},
		},
		Context: protocol.ReferenceContext{
			IncludeDeclaration: false,
		},
	})

	require.NoError(t, err)
	require.NotNil(t, locations)

	// Should find the var() call in CSS file
	assert.GreaterOrEqual(t, len(locations), 1)

	// Check that location is in the CSS file
	foundInCSS := false
	for _, loc := range locations {
		if loc.URI == "file:///test.css" {
			foundInCSS = true
		}
	}
	assert.True(t, foundInCSS, "Should find reference in CSS file")
}

// TestReferencesMultipleFiles tests references across multiple CSS files from token file
func TestReferencesMultipleFiles(t *testing.T) {
	server := testutil.NewTestServer(t)

	// Open two CSS files with var() calls
	testutil.OpenCSSFixture(t, server, "file:///test1.css", "references-multi-file-1.css")
	testutil.OpenCSSFixture(t, server, "file:///test2.css", "references-multi-file-2.css")

	// Load and open token file
	tokenContent := `{
  "color": {
    "primary": {
      "$type": "color",
      "$value": "#ff0000"
    }
  }
}`
	// Load tokens into token manager
	err := server.LoadTokensFromJSON([]byte(tokenContent), "")
	require.NoError(t, err)

	// Open the token file as a document
	err = textDocument.DidOpen(server, nil, &protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{
			URI:        "file:///tokens.json",
			LanguageID: "json",
			Version:    1,
			Text:       tokenContent,
		},
	})
	require.NoError(t, err)

	// Request references from token file (cursor on "primary")
	locations, err := references.References(server, nil, &protocol.ReferenceParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{
				URI: "file:///tokens.json",
			},
			Position: protocol.Position{
				Line:      2,
				Character: 6,
			},
		},
		Context: protocol.ReferenceContext{
			IncludeDeclaration: false,
		},
	})

	require.NoError(t, err)
	require.NotNil(t, locations)

	// Should find references in both CSS files
	assert.GreaterOrEqual(t, len(locations), 2)

	// Check that we have references from both CSS files
	fileURIs := make(map[string]bool)
	for _, loc := range locations {
		fileURIs[loc.URI] = true
	}

	assert.True(t, fileURIs["file:///test1.css"], "Should have reference in test1.css")
	assert.True(t, fileURIs["file:///test2.css"], "Should have reference in test2.css")
}

// TestReferencesOutsideVarCall tests that references returns nil for CSS files
// New behavior: references always returns nil for CSS files (let css-ls handle it)
func TestReferencesOutsideVarCall(t *testing.T) {
	server := testutil.NewTestServer(t)
	testutil.OpenCSSFixture(t, server, "file:///test.css", "no-var-call.css")

	// Request references on CSS file - should always return nil
	locations, err := references.References(server, nil, &protocol.ReferenceParams{
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
	assert.Nil(t, locations) // CSS files always return nil
}

// TestReferencesWithDeclaration tests including the token declaration
// New behavior: call from token file with IncludeDeclaration
func TestReferencesWithDeclaration(t *testing.T) {
	server := testutil.NewTestServer(t)

	// Open CSS file with var() call
	testutil.OpenCSSFixture(t, server, "file:///test.css", "basic-var-calls.css")

	// Load and open token file
	tokenContent := `{
  "color": {
    "primary": {
      "$type": "color",
      "$value": "#ff0000"
    }
  }
}`
	// Load tokens into token manager
	err := server.LoadTokensFromJSON([]byte(tokenContent), "")
	require.NoError(t, err)

	// Set the DefinitionURI for the token so declaration can be included
	token := server.Token("color-primary")
	require.NotNil(t, token)
	token.DefinitionURI = "file:///tokens.json"

	// Open the token file as a document
	err = textDocument.DidOpen(server, nil, &protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{
			URI:        "file:///tokens.json",
			LanguageID: "json",
			Version:    1,
			Text:       tokenContent,
		},
	})
	require.NoError(t, err)

	// Request references from token file with IncludeDeclaration
	locations, err := references.References(server, nil, &protocol.ReferenceParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{
				URI: "file:///tokens.json",
			},
			Position: protocol.Position{
				Line:      2,
				Character: 6,
			},
		},
		Context: protocol.ReferenceContext{
			IncludeDeclaration: true, // Request declaration
		},
	})

	require.NoError(t, err)
	require.NotNil(t, locations)

	// Should find both references and declaration
	assert.GreaterOrEqual(t, len(locations), 2) // At least: 1 CSS reference + 1 declaration

	// Check that declaration is included
	foundDeclaration := false
	for _, loc := range locations {
		if loc.URI == "file:///tokens.json" && loc.Range.Start.Line == 2 {
			foundDeclaration = true
		}
	}
	assert.True(t, foundDeclaration, "Should include token declaration")
}

// TestReferencesNonCSSFile tests that references works on JSON files
// Should return nil when cursor is not on a token
func TestReferencesNonCSSFile(t *testing.T) {
	server := testutil.NewTestServer(t)

	// Open a JSON file without token structure
	err := textDocument.DidOpen(server, nil, &protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{
			URI:        "file:///test.json",
			LanguageID: "json",
			Version:    1,
			Text:       `{"color": "red"}`,
		},
	})
	require.NoError(t, err)

	// Request references - cursor on "color" which is not a design token
	locations, err := references.References(server, nil, &protocol.ReferenceParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{
				URI: "file:///test.json",
			},
			Position: protocol.Position{Line: 0, Character: 5},
		},
		Context: protocol.ReferenceContext{
			IncludeDeclaration: false,
		},
	})

	require.NoError(t, err)
	assert.Nil(t, locations) // No token at cursor position
}

// TestReferencesUnknownToken tests that CSS files always return nil
// New behavior: CSS files are not processed (let css-ls handle them)
func TestReferencesUnknownToken(t *testing.T) {
	server := testutil.NewTestServer(t)

	// Open CSS file with reference to unknown token
	content := `/* Test file */
.button {
    color: var(--unknown-token);
}`
	err := textDocument.DidOpen(server, nil, &protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{
			URI:        "file:///test.css",
			LanguageID: "css",
			Version:    1,
			Text:       content,
		},
	})
	require.NoError(t, err)

	// Request references on CSS file - always returns nil
	locations, err := references.References(server, nil, &protocol.ReferenceParams{
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
			IncludeDeclaration: false,
		},
	})

	require.NoError(t, err)
	assert.Nil(t, locations) // CSS files always return nil
}

