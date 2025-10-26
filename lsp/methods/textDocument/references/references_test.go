package references

import (
	"testing"

	"bennypowers.dev/dtls/internal/parser/css"
	"bennypowers.dev/dtls/internal/tokens"
	"bennypowers.dev/dtls/lsp/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func TestReferences_FindAllReferences(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}

	// Add a token
	ctx.TokenManager().Add(&tokens.Token{
		Name:          "color.primary",
		Value:         "#ff0000",
		Type:          "color",
		DefinitionURI: "file:///workspace/tokens.json",
		Path:          []string{"color", "primary"},
	})

	// Open multiple CSS documents with references
	uri1 := "file:///test1.css"
	cssContent1 := `.button { color: var(--color-primary); }`
	ctx.DocumentManager().DidOpen(uri1, "css", 1, cssContent1)

	uri2 := "file:///test2.css"
	cssContent2 := `.link { background: var(--color-primary); }`
	ctx.DocumentManager().DidOpen(uri2, "css", 1, cssContent2)

	// Request references from first document
	result, err := References(ctx, glspCtx, &protocol.ReferenceParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri1},
			Position:     protocol.Position{Line: 0, Character: 24}, // Inside var()
		},
		Context: protocol.ReferenceContext{
			IncludeDeclaration: false,
		},
	})

	require.NoError(t, err)
	require.NotNil(t, result)

	// Should find references in both documents
	assert.GreaterOrEqual(t, len(result), 2)

	// Check that locations are correct
	foundInDoc1 := false
	foundInDoc2 := false
	for _, loc := range result {
		if loc.URI == uri1 {
			foundInDoc1 = true
		}
		if loc.URI == uri2 {
			foundInDoc2 = true
		}
	}
	assert.True(t, foundInDoc1, "Should find reference in test1.css")
	assert.True(t, foundInDoc2, "Should find reference in test2.css")
}

func TestReferences_WithIncludeDeclaration(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}

	// Add a token with definition URI
	ctx.TokenManager().Add(&tokens.Token{
		Name:          "color.primary",
		Value:         "#ff0000",
		DefinitionURI: "file:///workspace/tokens.json",
		Path:          []string{"color", "primary"},
	})

	uri := "file:///test.css"
	cssContent := `.button { color: var(--color-primary); }`
	ctx.DocumentManager().DidOpen(uri, "css", 1, cssContent)

	result, err := References(ctx, glspCtx, &protocol.ReferenceParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     protocol.Position{Line: 0, Character: 24},
		},
		Context: protocol.ReferenceContext{
			IncludeDeclaration: true,
		},
	})

	require.NoError(t, err)
	require.NotNil(t, result)

	// Should include the declaration location
	foundDeclaration := false
	for _, loc := range result {
		if loc.URI == "file:///workspace/tokens.json" {
			foundDeclaration = true
			break
		}
	}
	assert.True(t, foundDeclaration, "Should include declaration when IncludeDeclaration is true")
}

func TestReferences_UnknownToken(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}

	uri := "file:///test.css"
	cssContent := `.button { color: var(--unknown-token); }`
	ctx.DocumentManager().DidOpen(uri, "css", 1, cssContent)

	result, err := References(ctx, glspCtx, &protocol.ReferenceParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     protocol.Position{Line: 0, Character: 24},
		},
		Context: protocol.ReferenceContext{
			IncludeDeclaration: false,
		},
	})

	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestReferences_OutsideVarCall(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}

	ctx.TokenManager().Add(&tokens.Token{
		Name:  "color.primary",
		Value: "#ff0000",
	})

	uri := "file:///test.css"
	cssContent := `.button { color: var(--color-primary); }`
	ctx.DocumentManager().DidOpen(uri, "css", 1, cssContent)

	// Position outside var() call
	result, err := References(ctx, glspCtx, &protocol.ReferenceParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     protocol.Position{Line: 0, Character: 5}, // Inside ".button"
		},
		Context: protocol.ReferenceContext{
			IncludeDeclaration: false,
		},
	})

	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestReferences_NonCSSDocument(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}

	uri := "file:///test.json"
	jsonContent := `{"color": {"$value": "#ff0000"}}`
	ctx.DocumentManager().DidOpen(uri, "json", 1, jsonContent)

	result, err := References(ctx, glspCtx, &protocol.ReferenceParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     protocol.Position{Line: 0, Character: 10},
		},
		Context: protocol.ReferenceContext{
			IncludeDeclaration: false,
		},
	})

	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestReferences_DocumentNotFound(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}

	result, err := References(ctx, glspCtx, &protocol.ReferenceParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: "file:///nonexistent.css"},
			Position:     protocol.Position{Line: 0, Character: 10},
		},
		Context: protocol.ReferenceContext{
			IncludeDeclaration: false,
		},
	})

	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestReferences_OnlyNonCSSDocuments(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}

	// Add a token
	ctx.TokenManager().Add(&tokens.Token{
		Name:  "color.primary",
		Value: "#ff0000",
	})

	// Open only JSON documents (no CSS documents)
	uri := "file:///test.json"
	jsonContent := `{"color": {"$value": "#ff0000"}}`
	ctx.DocumentManager().DidOpen(uri, "json", 1, jsonContent)

	uri2 := "file:///test.css"
	cssContent := `.button { color: var(--color-primary); }`
	ctx.DocumentManager().DidOpen(uri2, "css", 1, cssContent)

	result, err := References(ctx, glspCtx, &protocol.ReferenceParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri2},
			Position:     protocol.Position{Line: 0, Character: 24},
		},
		Context: protocol.ReferenceContext{
			IncludeDeclaration: false,
		},
	})

	require.NoError(t, err)
	require.NotNil(t, result)

	// Should only include the CSS document reference
	assert.Len(t, result, 1)
	assert.Equal(t, uri2, result[0].URI)
}

func TestIsPositionInVarCall(t *testing.T) {
	tests := []struct {
		name     string
		pos      protocol.Position
		varCall  *css.VarCall
		expected bool
	}{
		{
			name: "position at start boundary - included",
			pos:  protocol.Position{Line: 0, Character: 10},
			varCall: &css.VarCall{
				TokenName: "color-primary",
				Range: css.Range{
					Start: css.Position{Line: 0, Character: 10},
					End:   css.Position{Line: 0, Character: 30},
				},
			},
			expected: true,
		},
		{
			name: "position at end boundary - excluded",
			pos:  protocol.Position{Line: 0, Character: 30},
			varCall: &css.VarCall{
				TokenName: "color-primary",
				Range: css.Range{
					Start: css.Position{Line: 0, Character: 10},
					End:   css.Position{Line: 0, Character: 30},
				},
			},
			expected: false,
		},
		{
			name: "position inside var call",
			pos:  protocol.Position{Line: 0, Character: 20},
			varCall: &css.VarCall{
				TokenName: "color-primary",
				Range: css.Range{
					Start: css.Position{Line: 0, Character: 10},
					End:   css.Position{Line: 0, Character: 30},
				},
			},
			expected: true,
		},
		{
			name: "position before var call",
			pos:  protocol.Position{Line: 0, Character: 5},
			varCall: &css.VarCall{
				TokenName: "color-primary",
				Range: css.Range{
					Start: css.Position{Line: 0, Character: 10},
					End:   css.Position{Line: 0, Character: 30},
				},
			},
			expected: false,
		},
		{
			name: "position after var call",
			pos:  protocol.Position{Line: 0, Character: 35},
			varCall: &css.VarCall{
				TokenName: "color-primary",
				Range: css.Range{
					Start: css.Position{Line: 0, Character: 10},
					End:   css.Position{Line: 0, Character: 30},
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isPositionInVarCall(tt.pos, tt.varCall)
			assert.Equal(t, tt.expected, result)
		})
	}
}
