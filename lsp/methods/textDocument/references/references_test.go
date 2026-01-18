package references

import (
	"testing"

	"bennypowers.dev/dtls/internal/tokens"
	"bennypowers.dev/dtls/lsp/testutil"
	"bennypowers.dev/dtls/lsp/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// TestReferences_CSSFile_ReturnsTokenDefinition tests that references from CSS returns the token definition
func TestReferences_CSSFile_ReturnsTokenDefinition(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)

	// Add token with definition location
	_ = ctx.TokenManager().Add(&tokens.Token{
		Name:          "color-primary",
		Path:          []string{"color", "primary"},
		DefinitionURI: "file:///tokens.json",
		Line:          2,
		Character:     4,
	})

	uri := "file:///test.css"
	cssContent := `.button { color: var(--color-primary); }`
	_ = ctx.DocumentManager().DidOpen(uri, "css", 1, cssContent)

	result, err := References(req, &protocol.ReferenceParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     protocol.Position{Line: 0, Character: 24},
		},
		Context: protocol.ReferenceContext{IncludeDeclaration: false},
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result, 1)
	assert.Equal(t, "file:///tokens.json", string(result[0].URI))
	assert.Equal(t, uint32(2), result[0].Range.Start.Line)
	assert.Equal(t, uint32(4), result[0].Range.Start.Character)
}

// TestReferences_CSSFile_UnknownToken tests that references returns nil when token is not found
func TestReferences_CSSFile_UnknownToken(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)

	uri := "file:///test.css"
	cssContent := `.button { color: var(--unknown-token); }`
	_ = ctx.DocumentManager().DidOpen(uri, "css", 1, cssContent)

	result, err := References(req, &protocol.ReferenceParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     protocol.Position{Line: 0, Character: 24},
		},
		Context: protocol.ReferenceContext{IncludeDeclaration: false},
	})

	require.NoError(t, err)
	assert.Nil(t, result)
}

// TestReferences_CSSFile_OutsideVarCall tests that references returns nil when cursor is not on var()
func TestReferences_CSSFile_OutsideVarCall(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)

	// Add a token
	_ = ctx.TokenManager().Add(&tokens.Token{
		Name:          "color-primary",
		Path:          []string{"color", "primary"},
		DefinitionURI: "file:///tokens.json",
		Line:          2,
		Character:     4,
	})

	uri := "file:///test.css"
	cssContent := `.button { color: var(--color-primary); }`
	_ = ctx.DocumentManager().DidOpen(uri, "css", 1, cssContent)

	// Cursor at position 0 (on the dot of .button)
	result, err := References(req, &protocol.ReferenceParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     protocol.Position{Line: 0, Character: 0},
		},
		Context: protocol.ReferenceContext{IncludeDeclaration: false},
	})

	require.NoError(t, err)
	assert.Nil(t, result)
}

// TestReferences_CSSFile_TokenWithoutDefinitionURI tests that references returns nil when token has no DefinitionURI
func TestReferences_CSSFile_TokenWithoutDefinitionURI(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)

	// Add token without definition location
	_ = ctx.TokenManager().Add(&tokens.Token{
		Name: "color-primary",
		Path: []string{"color", "primary"},
		// No DefinitionURI set
	})

	uri := "file:///test.css"
	cssContent := `.button { color: var(--color-primary); }`
	_ = ctx.DocumentManager().DidOpen(uri, "css", 1, cssContent)

	result, err := References(req, &protocol.ReferenceParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     protocol.Position{Line: 0, Character: 24},
		},
		Context: protocol.ReferenceContext{IncludeDeclaration: false},
	})

	require.NoError(t, err)
	assert.Nil(t, result)
}

// TestReferences_CSSFile_PositionOnDifferentLine tests cursor on a different line than the var() call
func TestReferences_CSSFile_PositionOnDifferentLine(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)

	// Add a token
	_ = ctx.TokenManager().Add(&tokens.Token{
		Name:          "color-primary",
		Path:          []string{"color", "primary"},
		DefinitionURI: "file:///tokens.json",
		Line:          2,
		Character:     4,
	})

	uri := "file:///test.css"
	// Multi-line CSS - var() is on line 1
	cssContent := `.button {
  color: var(--color-primary);
}`
	_ = ctx.DocumentManager().DidOpen(uri, "css", 1, cssContent)

	// Cursor on line 0, which is before the var() call on line 1
	result, err := References(req, &protocol.ReferenceParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     protocol.Position{Line: 0, Character: 5},
		},
		Context: protocol.ReferenceContext{IncludeDeclaration: false},
	})

	require.NoError(t, err)
	assert.Nil(t, result)
}

// TestReferences_CSSFile_PositionAtEndOfVarCall tests cursor at the closing paren of var()
func TestReferences_CSSFile_PositionAtEndOfVarCall(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)

	// Add a token
	_ = ctx.TokenManager().Add(&tokens.Token{
		Name:          "color-primary",
		Path:          []string{"color", "primary"},
		DefinitionURI: "file:///tokens.json",
		Line:          2,
		Character:     4,
	})

	uri := "file:///test.css"
	// .button { color: var(--color-primary); }
	// Position 37 is just after the closing paren ')' (on the semicolon)
	cssContent := `.button { color: var(--color-primary); }`
	_ = ctx.DocumentManager().DidOpen(uri, "css", 1, cssContent)

	// Cursor at position 37 (past the var() call), which is >= end character
	result, err := References(req, &protocol.ReferenceParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     protocol.Position{Line: 0, Character: 37},
		},
		Context: protocol.ReferenceContext{IncludeDeclaration: false},
	})

	require.NoError(t, err)
	assert.Nil(t, result)
}

// TestReferences_JSONFile_FindsReferencesInCSS tests finding CSS var() references from JSON token file
func TestReferences_JSONFile_FindsReferencesInCSS(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)

	// Add a token with extension data
	token := &tokens.Token{
		Name:          "color-primary",
		Value:         "#ff0000",
		Type:          "color",
		Path:          []string{"color", "primary"},
		Reference:     "{color.primary}",
		DefinitionURI: "file:///tokens.json",
	}
	_ = ctx.TokenManager().Add(token)

	// Open JSON token file
	jsonURI := "file:///tokens.json"
	jsonContent := `{
  "color": {
    "primary": {
      "$type": "color",
      "$value": "#ff0000"
    }
  }
}`
	_ = ctx.DocumentManager().DidOpen(jsonURI, "json", 1, jsonContent)

	// Open CSS files with var() calls
	cssURI1 := "file:///styles1.css"
	cssContent1 := `.button { color: var(--color-primary); }`
	_ = ctx.DocumentManager().DidOpen(cssURI1, "css", 1, cssContent1)

	cssURI2 := "file:///styles2.css"
	cssContent2 := `.link { background: var(--color-primary, red); }`
	_ = ctx.DocumentManager().DidOpen(cssURI2, "css", 1, cssContent2)

	// Request references from the JSON token file (cursor on token)
	result, err := References(req, &protocol.ReferenceParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: jsonURI},
			Position:     protocol.Position{Line: 2, Character: 6}, // On "primary" key
		},
		Context: protocol.ReferenceContext{
			IncludeDeclaration: false,
		},
	})

	require.NoError(t, err)
	require.NotNil(t, result)

	// Should find references in both CSS files
	assert.GreaterOrEqual(t, len(result), 2)

	foundInCSS1 := false
	foundInCSS2 := false
	for _, loc := range result {
		if loc.URI == cssURI1 {
			foundInCSS1 = true
		}
		if loc.URI == cssURI2 {
			foundInCSS2 = true
		}
	}
	assert.True(t, foundInCSS1, "Should find var() reference in styles1.css")
	assert.True(t, foundInCSS2, "Should find var() reference in styles2.css")
}

// TestReferences_JSONFile_FindsReferencesInJSON tests finding token references in other JSON files
func TestReferences_JSONFile_FindsReferencesInJSON(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)

	// Add tokens
	primaryToken := &tokens.Token{
		Name:          "color-primary",
		Value:         "#ff0000",
		Type:          "color",
		Path:          []string{"color", "primary"},
		Reference:     "{color.primary}",
		DefinitionURI: "file:///tokens.json",
	}
	_ = ctx.TokenManager().Add(primaryToken)

	// Open JSON token file with token definition
	jsonURI := "file:///tokens.json"
	jsonContent := `{
  "color": {
    "primary": {
      "$type": "color",
      "$value": "#ff0000"
    },
    "brand": {
      "$type": "color",
      "$value": "{color.primary}"
    }
  }
}`
	_ = ctx.DocumentManager().DidOpen(jsonURI, "json", 1, jsonContent)

	// Request references from the JSON file
	result, err := References(req, &protocol.ReferenceParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: jsonURI},
			Position:     protocol.Position{Line: 2, Character: 6}, // On "primary"
		},
		Context: protocol.ReferenceContext{
			IncludeDeclaration: false,
		},
	})

	require.NoError(t, err)
	require.NotNil(t, result)

	// Should find reference in the same JSON file where brand references primary
	foundReference := false
	for _, loc := range result {
		if loc.URI == jsonURI && loc.Range.Start.Line == 8 {
			foundReference = true
		}
	}
	assert.True(t, foundReference, "Should find {color.primary} reference in brand token")
}

// TestReferences_WithIncludeDeclaration tests including the token definition
func TestReferences_WithIncludeDeclaration(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)

	token := &tokens.Token{
		Name:          "color-primary",
		Value:         "#ff0000",
		Type:          "color",
		Path:          []string{"color", "primary"},
		Reference:     "{color.primary}",
		DefinitionURI: "file:///tokens.json",
	}
	_ = ctx.TokenManager().Add(token)

	jsonURI := "file:///tokens.json"
	jsonContent := `{
  "color": {
    "primary": {
      "$type": "color",
      "$value": "#ff0000"
    }
  }
}`
	_ = ctx.DocumentManager().DidOpen(jsonURI, "json", 1, jsonContent)

	cssURI := "file:///styles.css"
	cssContent := `.button { color: var(--color-primary); }`
	_ = ctx.DocumentManager().DidOpen(cssURI, "css", 1, cssContent)

	result, err := References(req, &protocol.ReferenceParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: jsonURI},
			Position:     protocol.Position{Line: 2, Character: 6},
		},
		Context: protocol.ReferenceContext{
			IncludeDeclaration: true,
		},
	})

	require.NoError(t, err)
	require.NotNil(t, result)

	// Should include declaration
	foundDeclaration := false
	for _, loc := range result {
		if loc.URI == jsonURI && loc.Range.Start.Line == 2 {
			foundDeclaration = true
		}
	}
	assert.True(t, foundDeclaration, "Should include declaration when IncludeDeclaration is true")
}

// TestReferences_UnknownToken tests when cursor is not on a token
func TestReferences_UnknownToken(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)

	jsonURI := "file:///tokens.json"
	jsonContent := `{
  "color": {
    "primary": {
      "$type": "color",
      "$value": "#ff0000"
    }
  }
}`
	_ = ctx.DocumentManager().DidOpen(jsonURI, "json", 1, jsonContent)

	// Position not on a token
	result, err := References(req, &protocol.ReferenceParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: jsonURI},
			Position:     protocol.Position{Line: 0, Character: 0}, // On opening brace
		},
		Context: protocol.ReferenceContext{
			IncludeDeclaration: false,
		},
	})

	require.NoError(t, err)
	assert.Nil(t, result)
}

// TestReferences_DocumentNotFound tests when document doesn't exist
func TestReferences_DocumentNotFound(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)

	result, err := References(req, &protocol.ReferenceParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: "file:///nonexistent.json"},
			Position:     protocol.Position{Line: 0, Character: 10},
		},
		Context: protocol.ReferenceContext{
			IncludeDeclaration: false,
		},
	})

	require.NoError(t, err)
	assert.Nil(t, result)
}

// TestReferences_YAMLFile tests references from YAML token files
func TestReferences_YAMLFile(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)

	token := &tokens.Token{
		Name:          "color-primary",
		Value:         "#ff0000",
		Type:          "color",
		Path:          []string{"color", "primary"},
		Reference:     "{color.primary}",
		DefinitionURI: "file:///tokens.yaml",
	}
	_ = ctx.TokenManager().Add(token)

	yamlURI := "file:///tokens.yaml"
	yamlContent := `color:
  primary:
    $type: color
    $value: "#ff0000"
`
	_ = ctx.DocumentManager().DidOpen(yamlURI, "yaml", 1, yamlContent)

	cssURI := "file:///styles.css"
	cssContent := `.button { color: var(--color-primary); }`
	_ = ctx.DocumentManager().DidOpen(cssURI, "css", 1, cssContent)

	result, err := References(req, &protocol.ReferenceParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: yamlURI},
			Position:     protocol.Position{Line: 1, Character: 3}, // On "primary"
		},
		Context: protocol.ReferenceContext{
			IncludeDeclaration: false,
		},
	})

	require.NoError(t, err)
	require.NotNil(t, result)

	// Should find var() reference in CSS
	foundInCSS := false
	for _, loc := range result {
		if loc.URI == cssURI {
			foundInCSS = true
		}
	}
	assert.True(t, foundInCSS, "Should find var() reference in CSS from YAML token file")
}
