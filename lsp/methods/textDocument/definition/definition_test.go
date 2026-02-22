package definition

import (
	"testing"

	"bennypowers.dev/dtls/internal/parser/css"
	"bennypowers.dev/dtls/internal/tokens"
	"bennypowers.dev/dtls/lsp/testutil"
	"bennypowers.dev/dtls/lsp/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func TestDefinition_CSSVariableReference(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)

	// Add a token with definition URI
	_ = ctx.TokenManager().Add(&tokens.Token{
		Name:          "color.primary",
		Value:         "#ff0000",
		Type:          "color",
		DefinitionURI: "file:///workspace/tokens.json",
		Path:          []string{"color", "primary"},
	})

	uri := "file:///test.css"
	cssContent := `.button { color: var(--color-primary); }`
	_ = ctx.DocumentManager().DidOpen(uri, "css", 1, cssContent)

	result, err := Definition(req, &protocol.DefinitionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     protocol.Position{Line: 0, Character: 24}, // Inside var()
		},
	})

	require.NoError(t, err)
	require.NotNil(t, result)

	locations, ok := result.([]protocol.Location)
	require.True(t, ok)
	require.Len(t, locations, 1)

	assert.Equal(t, "file:///workspace/tokens.json", locations[0].URI)
}

func TestDefinition_UnknownToken(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)

	uri := "file:///test.css"
	cssContent := `.button { color: var(--unknown-token); }`
	_ = ctx.DocumentManager().DidOpen(uri, "css", 1, cssContent)

	result, err := Definition(req, &protocol.DefinitionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     protocol.Position{Line: 0, Character: 24},
		},
	})

	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestDefinition_TokenWithoutDefinitionURI(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)

	// Add token without DefinitionURI
	_ = ctx.TokenManager().Add(&tokens.Token{
		Name:  "color.primary",
		Value: "#ff0000",
	})

	uri := "file:///test.css"
	cssContent := `.button { color: var(--color-primary); }`
	_ = ctx.DocumentManager().DidOpen(uri, "css", 1, cssContent)

	result, err := Definition(req, &protocol.DefinitionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     protocol.Position{Line: 0, Character: 24},
		},
	})

	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestDefinition_OutsideVarCall(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)

	_ = ctx.TokenManager().Add(&tokens.Token{
		Name:          "color.primary",
		Value:         "#ff0000",
		DefinitionURI: "file:///workspace/tokens.json",
		Path:          []string{"color", "primary"},
	})

	uri := "file:///test.css"
	cssContent := `.button { color: var(--color-primary); }`
	_ = ctx.DocumentManager().DidOpen(uri, "css", 1, cssContent)

	// Position outside the var() call
	result, err := Definition(req, &protocol.DefinitionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     protocol.Position{Line: 0, Character: 5}, // Inside ".button"
		},
	})

	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestDefinition_NonCSSDocument(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)

	uri := "file:///test.json"
	jsonContent := `{"color": {"$value": "#ff0000"}}`
	_ = ctx.DocumentManager().DidOpen(uri, "json", 1, jsonContent)

	result, err := Definition(req, &protocol.DefinitionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     protocol.Position{Line: 0, Character: 10},
		},
	})

	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestDefinition_DocumentNotFound(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)

	result, err := Definition(req, &protocol.DefinitionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: "file:///nonexistent.css"},
			Position:     protocol.Position{Line: 0, Character: 10},
		},
	})

	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestDefinition_PreciseTokenPosition(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)

	// Add a token with definition URI and precise position
	// Simulates a token at line 2, character 4 in the source file
	_ = ctx.TokenManager().Add(&tokens.Token{
		Name:          "color.primary",
		Value:         "#ff0000",
		Type:          "color",
		DefinitionURI: "file:///workspace/tokens.json",
		Path:          []string{"color", "primary"},
		Line:          2, // Token is on line 2
		Character:     4, // Token starts at character 4
	})

	uri := "file:///test.css"
	cssContent := `.button { color: var(--color-primary); }`
	_ = ctx.DocumentManager().DidOpen(uri, "css", 1, cssContent)

	result, err := Definition(req, &protocol.DefinitionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     protocol.Position{Line: 0, Character: 24},
		},
	})

	require.NoError(t, err)
	require.NotNil(t, result)

	locations, ok := result.([]protocol.Location)
	require.True(t, ok)
	require.Len(t, locations, 1)

	// Verify the location points to the precise position
	assert.Equal(t, "file:///workspace/tokens.json", locations[0].URI)
	assert.Equal(t, uint32(2), locations[0].Range.Start.Line, "Should jump to line 2")
	assert.Equal(t, uint32(4), locations[0].Range.Start.Character, "Should jump to character 4")
}

// TestIsPositionInVarCall tests the isPositionInVarCall function with half-open range semantics [start, end)
func TestIsPositionInVarCall(t *testing.T) {
	tests := []struct {
		varCall  *css.VarCall
		name     string
		pos      protocol.Position
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
			expected: true, // Start is inclusive
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
			expected: false, // End is exclusive in half-open range [start, end)
		},
		{
			name: "position before var call",
			pos:  protocol.Position{Line: 0, Character: 9},
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
			pos:  protocol.Position{Line: 0, Character: 31},
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isPositionInVarCall(tt.pos, tt.varCall)
			assert.Equal(t, tt.expected, result, "isPositionInVarCall(%+v, %+v)", tt.pos, tt.varCall)
		})
	}
}

func TestDefinition_LinkSupport(t *testing.T) {
	t.Run("returns LocationLink when client supports linkSupport", func(t *testing.T) {
		ctx := testutil.NewMockServerContext()
		ctx.SetSupportsDefinitionLinks(true)
		glspCtx := &glsp.Context{}
		req := types.NewRequestContext(ctx, glspCtx)

		_ = ctx.TokenManager().Add(&tokens.Token{
			Name:          "color.primary",
			Value:         "#ff0000",
			DefinitionURI: "file:///tokens.json",
			Line:          5,
			Character:     2,
			Path:          []string{"color", "primary"},
		})

		uri := "file:///test.css"
		cssContent := `.button { color: var(--color-primary); }`
		_ = ctx.DocumentManager().DidOpen(uri, "css", 1, cssContent)

		result, err := Definition(req, &protocol.DefinitionParams{
			TextDocumentPositionParams: protocol.TextDocumentPositionParams{
				TextDocument: protocol.TextDocumentIdentifier{URI: uri},
				Position:     protocol.Position{Line: 0, Character: 24},
			},
		})

		require.NoError(t, err)
		require.NotNil(t, result)

		// Should return LocationLink array
		links, ok := result.([]protocol.LocationLink)
		require.True(t, ok, "Result should be []protocol.LocationLink")
		require.Len(t, links, 1)

		link := links[0]
		assert.NotNil(t, link.OriginSelectionRange, "Should have OriginSelectionRange")
		assert.Equal(t, "file:///tokens.json", string(link.TargetURI))
		assert.Equal(t, protocol.Position{Line: 5, Character: 2}, link.TargetRange.Start)
	})

	t.Run("returns Location when client does not support linkSupport", func(t *testing.T) {
		ctx := testutil.NewMockServerContext()
		ctx.SetSupportsDefinitionLinks(false)
		glspCtx := &glsp.Context{}
		req := types.NewRequestContext(ctx, glspCtx)

		_ = ctx.TokenManager().Add(&tokens.Token{
			Name:          "color.primary",
			Value:         "#ff0000",
			DefinitionURI: "file:///tokens.json",
			Line:          5,
			Character:     2,
			Path:          []string{"color", "primary"},
		})

		uri := "file:///test.css"
		cssContent := `.button { color: var(--color-primary); }`
		_ = ctx.DocumentManager().DidOpen(uri, "css", 1, cssContent)

		result, err := Definition(req, &protocol.DefinitionParams{
			TextDocumentPositionParams: protocol.TextDocumentPositionParams{
				TextDocument: protocol.TextDocumentIdentifier{URI: uri},
				Position:     protocol.Position{Line: 0, Character: 24},
			},
		})

		require.NoError(t, err)
		require.NotNil(t, result)

		// Should return Location array (not LocationLink)
		locations, ok := result.([]protocol.Location)
		require.True(t, ok, "Result should be []protocol.Location")
		require.Len(t, locations, 1)

		loc := locations[0]
		assert.Equal(t, "file:///tokens.json", loc.URI)
	})

	t.Run("defaults to Location when capability is unknown", func(t *testing.T) {
		ctx := testutil.NewMockServerContext()
		// Don't set link support - test default behavior
		glspCtx := &glsp.Context{}
		req := types.NewRequestContext(ctx, glspCtx)

		_ = ctx.TokenManager().Add(&tokens.Token{
			Name:          "color.primary",
			Value:         "#ff0000",
			DefinitionURI: "file:///tokens.json",
			Line:          5,
			Character:     2,
			Path:          []string{"color", "primary"},
		})

		uri := "file:///test.css"
		cssContent := `.button { color: var(--color-primary); }`
		_ = ctx.DocumentManager().DidOpen(uri, "css", 1, cssContent)

		result, err := Definition(req, &protocol.DefinitionParams{
			TextDocumentPositionParams: protocol.TextDocumentPositionParams{
				TextDocument: protocol.TextDocumentIdentifier{URI: uri},
				Position:     protocol.Position{Line: 0, Character: 24},
			},
		})

		require.NoError(t, err)
		require.NotNil(t, result)

		// Should return Location array (default)
		locations, ok := result.([]protocol.Location)
		require.True(t, ok, "Result should be []protocol.Location")
		require.Len(t, locations, 1)
	})
}

func TestDefinition_HTMLDocument(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)

	_ = ctx.TokenManager().Add(&tokens.Token{
		Name:          "color.primary",
		Value:         "#ff0000",
		DefinitionURI: "file:///tokens.json",
		Path:          []string{"color", "primary"},
		Line:          5,
		Character:     4,
	})

	uri := "file:///test.html"
	content := `<style>.btn { color: var(--color-primary); }</style>`
	_ = ctx.DocumentManager().DidOpen(uri, "html", 1, content)

	result, err := Definition(req, &protocol.DefinitionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     protocol.Position{Line: 0, Character: 30},
		},
	})
	require.NoError(t, err)
	require.NotNil(t, result)

	locations, ok := result.([]protocol.Location)
	require.True(t, ok)
	require.Len(t, locations, 1)
	assert.Equal(t, "file:///tokens.json", locations[0].URI)
}

func TestDefinition_HTMLNoCSS(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)

	uri := "file:///test.html"
	content := `<p>Hello</p>`
	_ = ctx.DocumentManager().DidOpen(uri, "html", 1, content)

	result, err := Definition(req, &protocol.DefinitionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     protocol.Position{Line: 0, Character: 5},
		},
	})
	require.NoError(t, err)
	assert.Nil(t, result)
}
