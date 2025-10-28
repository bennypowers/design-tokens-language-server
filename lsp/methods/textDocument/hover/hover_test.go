package hover

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

// TestIsPositionInRange tests the isPositionInRange function with half-open range semantics [start, end)
func TestIsPositionInRange(t *testing.T) {
	tests := []struct {
		name     string
		pos      protocol.Position
		r        css.Range
		expected bool
	}{
		{
			name: "position at start boundary - included",
			pos:  protocol.Position{Line: 0, Character: 5},
			r: css.Range{
				Start: css.Position{Line: 0, Character: 5},
				End:   css.Position{Line: 0, Character: 10},
			},
			expected: true,
		},
		{
			name: "position at end boundary - excluded",
			pos:  protocol.Position{Line: 0, Character: 10},
			r: css.Range{
				Start: css.Position{Line: 0, Character: 5},
				End:   css.Position{Line: 0, Character: 10},
			},
			expected: false,
		},
		{
			name: "position inside range",
			pos:  protocol.Position{Line: 0, Character: 7},
			r: css.Range{
				Start: css.Position{Line: 0, Character: 5},
				End:   css.Position{Line: 0, Character: 10},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isPositionInRange(tt.pos, tt.r)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHover_CSSVariableReference(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)

	// Add a token
	_ = ctx.TokenManager().Add(&tokens.Token{
		Name:        "color.primary",
		Value:       "#ff0000",
		Type:        "color",
		Description: "Primary brand color",
		FilePath:    "tokens.json",
	})

	// Open a CSS document with var() call
	uri := "file:///test.css"
	cssContent := `.button { color: var(--color-primary); }`
	_ = ctx.DocumentManager().DidOpen(uri, "css", 1, cssContent)

	// Hover over --color-primary in var() call
	hover, err := Hover(req, &protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     protocol.Position{Line: 0, Character: 24},
		},
	})

	require.NoError(t, err)
	require.NotNil(t, hover)

	// Assert hover content
	content, ok := hover.Contents.(protocol.MarkupContent)
	require.True(t, ok, "Contents should be MarkupContent")
	assert.Contains(t, content.Value, "--color-primary")
	assert.Contains(t, content.Value, "#ff0000")
	assert.Contains(t, content.Value, "color")
	assert.Contains(t, content.Value, "Primary brand color")
	assert.Contains(t, content.Value, "tokens.json")

	// Assert Range is present for var() calls
	require.NotNil(t, hover.Range, "Range should be present for var() call")
}

func TestHover_DeprecatedToken(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)

	// Add deprecated token
	_ = ctx.TokenManager().Add(&tokens.Token{
		Name:               "color.old-primary",
		Value:              "#cc0000",
		Type:               "color",
		Deprecated:         true,
		DeprecationMessage: "Use color.primary instead",
	})

	uri := "file:///test.css"
	cssContent := `.button { color: var(--color-old-primary); }`
	_ = ctx.DocumentManager().DidOpen(uri, "css", 1, cssContent)

	hover, err := Hover(req, &protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     protocol.Position{Line: 0, Character: 28},
		},
	})

	require.NoError(t, err)
	require.NotNil(t, hover)

	// Assert deprecation warning
	content, ok := hover.Contents.(protocol.MarkupContent)
	require.True(t, ok)
	assert.Contains(t, content.Value, "DEPRECATED")
	assert.Contains(t, content.Value, "Use color.primary instead")
}

func TestHover_UnknownToken(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)

	uri := "file:///test.css"
	cssContent := `.button { color: var(--unknown-token); }`
	_ = ctx.DocumentManager().DidOpen(uri, "css", 1, cssContent)

	hover, err := Hover(req, &protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     protocol.Position{Line: 0, Character: 28},
		},
	})

	require.NoError(t, err)
	require.NotNil(t, hover, "Should show 'unknown token' message for var() calls with unknown tokens")

	// Assert unknown token message
	content, ok := hover.Contents.(protocol.MarkupContent)
	require.True(t, ok)
	assert.Contains(t, content.Value, "Unknown token")
	assert.Contains(t, content.Value, "--unknown-token")

	// Assert Range is present for unknown token (consistency with known tokens)
	require.NotNil(t, hover.Range, "Range should be present for unknown token var() call")
}

func TestHover_VarCallWithFallback(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)

	_ = ctx.TokenManager().Add(&tokens.Token{
		Name:  "spacing.large",
		Value: "2rem",
		Type:  "dimension",
	})

	uri := "file:///test.css"
	cssContent := `.card { padding: var(--spacing-large, 1rem); }`
	_ = ctx.DocumentManager().DidOpen(uri, "css", 1, cssContent)

	// Hover over the token name in var() call with fallback
	hover, err := Hover(req, &protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     protocol.Position{Line: 0, Character: 28},
		},
	})

	require.NoError(t, err)
	require.NotNil(t, hover)

	content, ok := hover.Contents.(protocol.MarkupContent)
	require.True(t, ok)
	assert.Contains(t, content.Value, "--spacing-large")
	assert.Contains(t, content.Value, "2rem")
}

func TestHover_NestedVarCalls(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)

	_ = ctx.TokenManager().Add(&tokens.Token{
		Name:  "color.primary",
		Value: "#ff0000",
		Type:  "color",
	})

	uri := "file:///test.css"
	// Nested var() - hover should work on the inner one
	cssContent := `.element { background: linear-gradient(var(--color-primary), white); }`
	_ = ctx.DocumentManager().DidOpen(uri, "css", 1, cssContent)

	hover, err := Hover(req, &protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     protocol.Position{Line: 0, Character: 47},
		},
	})

	require.NoError(t, err)
	require.NotNil(t, hover)

	content, ok := hover.Contents.(protocol.MarkupContent)
	require.True(t, ok)
	assert.Contains(t, content.Value, "--color-primary")
}

func TestHover_VarCallOutsideCursorRange(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)

	_ = ctx.TokenManager().Add(&tokens.Token{
		Name:  "color.primary",
		Value: "#ff0000",
	})

	uri := "file:///test.css"
	cssContent := `.button { color: var(--color-primary); }`
	_ = ctx.DocumentManager().DidOpen(uri, "css", 1, cssContent)

	// Hover on "color:" property, not in var() range
	hover, err := Hover(req, &protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     protocol.Position{Line: 0, Character: 12},
		},
	})

	require.NoError(t, err)
	assert.Nil(t, hover, "Should not show hover outside var() range")
}

func TestHover_VariableDeclaration(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)

	// Add a token
	_ = ctx.TokenManager().Add(&tokens.Token{
		Name:        "color.primary",
		Value:       "#ff0000",
		Type:        "color",
		Description: "Primary brand color",
	})

	uri := "file:///test.css"
	cssContent := `:root { --color-primary: #ff0000; }`
	_ = ctx.DocumentManager().DidOpen(uri, "css", 1, cssContent)

	// Hover over variable declaration (on the property name)
	hover, err := Hover(req, &protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     protocol.Position{Line: 0, Character: 10},
		},
	})

	require.NoError(t, err)
	require.NotNil(t, hover)

	// Assert hover content for declaration
	content, ok := hover.Contents.(protocol.MarkupContent)
	require.True(t, ok)
	assert.Contains(t, content.Value, "--color-primary")
	assert.Contains(t, content.Value, "#ff0000")
	assert.Contains(t, content.Value, "Primary brand color")

	// Assert Range is present and covers only the property name
	require.NotNil(t, hover.Range, "Range should be present for known token declaration")
	assert.Equal(t, uint32(0), hover.Range.Start.Line)
	assert.Equal(t, uint32(8), hover.Range.Start.Character) // Start of --color-primary (first dash)
	assert.Equal(t, uint32(0), hover.Range.End.Line)
	assert.Equal(t, uint32(23), hover.Range.End.Character) // End of --color-primary (just before colon)
}

func TestHover_VariableDeclaration_UnknownToken(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)

	uri := "file:///test.css"
	// --local-var is not a known design token, just a local CSS custom property
	cssContent := `:root { --local-var: blue; }`
	_ = ctx.DocumentManager().DidOpen(uri, "css", 1, cssContent)

	// Hover over unknown variable declaration
	hover, err := Hover(req, &protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     protocol.Position{Line: 0, Character: 10},
		},
	})

	require.NoError(t, err)
	assert.Nil(t, hover, "Should not show hover for unknown token declaration (local CSS var)")
}

func TestHover_VariableDeclaration_OnValue(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)

	// Add a token
	_ = ctx.TokenManager().Add(&tokens.Token{
		Name:  "color.primary",
		Value: "#ff0000",
		Type:  "color",
	})

	uri := "file:///test.css"
	cssContent := `:root { --color-primary: #ff0000; }`
	_ = ctx.DocumentManager().DidOpen(uri, "css", 1, cssContent)

	// Hover on the value side (RHS) - should not trigger hover
	// Character 25 is on "#ff0000"
	hover, err := Hover(req, &protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     protocol.Position{Line: 0, Character: 25},
		},
	})

	require.NoError(t, err)
	assert.Nil(t, hover, "Should not show hover when cursor is on value side (RHS)")
}

func TestHover_VariableDeclaration_Boundaries(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)

	_ = ctx.TokenManager().Add(&tokens.Token{
		Name:  "color.primary",
		Value: "#ff0000",
	})

	uri := "file:///test.css"
	cssContent := `:root { --color-primary: #ff0000; }`
	_ = ctx.DocumentManager().DidOpen(uri, "css", 1, cssContent)

	tests := []struct {
		name      string
		character uint32
		expectHit bool
	}{
		{"before property name (space)", 7, false},
		{"at start of property name (first dash)", 8, true},
		{"middle of property name", 15, true},
		{"near end of property name", 22, true},
		{"at end boundary (colon) - excluded", 23, false},
		{"after property name (space after colon)", 24, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hover, err := Hover(req, &protocol.HoverParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
					Position:     protocol.Position{Line: 0, Character: tt.character},
				},
			})

			require.NoError(t, err)
			if tt.expectHit {
				assert.NotNil(t, hover, "Expected hover at character %d", tt.character)
			} else {
				assert.Nil(t, hover, "Expected no hover at character %d", tt.character)
			}
		})
	}
}

func TestHover_VariableDeclaration_WithPrefix(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)

	// Add a token with prefix
	_ = ctx.TokenManager().Add(&tokens.Token{
		Name:        "color.primary",
		Value:       "#0000ff",
		Type:        "color",
		Description: "Blue color",
		Prefix:      "ds",
	})

	uri := "file:///test.css"
	// Token with prefix: --ds-color-primary
	cssContent := `:root { --ds-color-primary: #0000ff; }`
	_ = ctx.DocumentManager().DidOpen(uri, "css", 1, cssContent)

	hover, err := Hover(req, &protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     protocol.Position{Line: 0, Character: 12},
		},
	})

	require.NoError(t, err)
	require.NotNil(t, hover)

	content, ok := hover.Contents.(protocol.MarkupContent)
	require.True(t, ok)
	assert.Contains(t, content.Value, "--ds-color-primary")
	assert.Contains(t, content.Value, "#0000ff")
	assert.Contains(t, content.Value, "Blue color")
}

func TestHover_VariableDeclaration_MultipleInSameBlock(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)

	_ = ctx.TokenManager().Add(&tokens.Token{
		Name:  "color.primary",
		Value: "#ff0000",
		Type:  "color",
	})
	_ = ctx.TokenManager().Add(&tokens.Token{
		Name:  "color.secondary",
		Value: "#00ff00",
		Type:  "color",
	})

	uri := "file:///test.css"
	cssContent := `:root {
  --color-primary: #ff0000;
  --color-secondary: #00ff00;
}`
	_ = ctx.DocumentManager().DidOpen(uri, "css", 1, cssContent)

	// Test first declaration
	hover1, err := Hover(req, &protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     protocol.Position{Line: 1, Character: 5},
		},
	})
	require.NoError(t, err)
	require.NotNil(t, hover1)
	content1, ok := hover1.Contents.(protocol.MarkupContent)
	require.True(t, ok)
	assert.Contains(t, content1.Value, "--color-primary")

	// Test second declaration
	hover2, err := Hover(req, &protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     protocol.Position{Line: 2, Character: 5},
		},
	})
	require.NoError(t, err)
	require.NotNil(t, hover2)
	content2, ok := hover2.Contents.(protocol.MarkupContent)
	require.True(t, ok)
	assert.Contains(t, content2.Value, "--color-secondary")
}

func TestHover_InvalidPosition(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)

	_ = ctx.TokenManager().Add(&tokens.Token{
		Name:  "color.primary",
		Value: "#ff0000",
	})

	uri := "file:///test.css"
	cssContent := `.button { color: var(--color-primary); }`
	_ = ctx.DocumentManager().DidOpen(uri, "css", 1, cssContent)

	// Hover outside var() call
	hover, err := Hover(req, &protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     protocol.Position{Line: 0, Character: 5},
		},
	})

	require.NoError(t, err)
	assert.Nil(t, hover)
}

func TestHover_NonCSSDocument(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)

	uri := "file:///test.json"
	jsonContent := `{"color": {"$value": "#ff0000"}}`
	_ = ctx.DocumentManager().DidOpen(uri, "json", 1, jsonContent)

	hover, err := Hover(req, &protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     protocol.Position{Line: 0, Character: 10},
		},
	})

	require.NoError(t, err)
	assert.Nil(t, hover)
}

func TestHover_DocumentNotFound(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)

	hover, err := Hover(req, &protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: "file:///nonexistent.css"},
			Position:     protocol.Position{Line: 0, Character: 10},
		},
	})

	require.NoError(t, err)
	assert.Nil(t, hover)
}

// TestHover_NestedVarInFallback tests hovering over nested var() calls in fallback position
// This is the RHDS pattern: var(--local, var(--design-token, fallback))
func TestHover_NestedVarInFallback(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)

	// Add design tokens (not the local variables)
	_ = ctx.TokenManager().Add(&tokens.Token{
		Name:        "color-text-primary",
		Value:       "#000000",
		Type:        "color",
		Description: "Primary text color",
	})
	_ = ctx.TokenManager().Add(&tokens.Token{
		Name:        "color-surface-lightest",
		Value:       "#ffffff",
		Type:        "color",
		Description: "Lightest surface color",
	})

	uri := "file:///test.css"
	// RHDS pattern: local variable with design token fallback
	// The outer var(--_local, ...) has a nested var(--design-token, fallback)
	cssContent := `.card {
  color: var(--_local-color, var(--color-text-primary, #000000));
  background: var(--_card-background, var(--color-surface-lightest, #ffffff));
}`
	_ = ctx.DocumentManager().DidOpen(uri, "css", 1, cssContent)

	t.Run("hover over inner token in nested fallback", func(t *testing.T) {
		// Hover over --color-text-primary (the inner/nested var)
		// Line 1, character 40 is approximately over --color-text-primary
		hover, err := Hover(req, &protocol.HoverParams{
			TextDocumentPositionParams: protocol.TextDocumentPositionParams{
				TextDocument: protocol.TextDocumentIdentifier{URI: uri},
				Position:     protocol.Position{Line: 1, Character: 40},
			},
		})

		require.NoError(t, err)
		require.NotNil(t, hover, "Should find hover for inner token")

		content, ok := hover.Contents.(protocol.MarkupContent)
		require.True(t, ok)

		// Should show info for the INNER token, not the outer --_local-color
		assert.Contains(t, content.Value, "--color-text-primary", "Should show inner token name")
		assert.Contains(t, content.Value, "#000000", "Should show inner token value")
		assert.Contains(t, content.Value, "Primary text color", "Should show inner token description")
		assert.NotContains(t, content.Value, "Unknown token", "Should not report as unknown")
		assert.NotContains(t, content.Value, "--_local-color", "Should not show outer local variable")
	})

	t.Run("hover over outer local variable", func(t *testing.T) {
		// Hover over --_local-color (the outer var, which is a local variable)
		// Line 1, character 18 is approximately over --_local-color
		hover, err := Hover(req, &protocol.HoverParams{
			TextDocumentPositionParams: protocol.TextDocumentPositionParams{
				TextDocument: protocol.TextDocumentIdentifier{URI: uri},
				Position:     protocol.Position{Line: 1, Character: 18},
			},
		})

		require.NoError(t, err)
		// May be nil or show unknown token (local variables aren't in token manager)
		// This is acceptable behavior - we're just testing it doesn't crash
		// and doesn't incorrectly show the inner token
		if hover != nil {
			content, ok := hover.Contents.(protocol.MarkupContent)
			if ok {
				// If it shows content, it should be about --_local-color, not --color-text-primary
				assert.NotContains(t, content.Value, "--color-text-primary", "Should not show inner token")
				assert.NotContains(t, content.Value, "Primary text color", "Should not show inner token description")
			}
		}
	})

	t.Run("hover over second nested var in same document", func(t *testing.T) {
		// Hover over --color-surface-lightest (line 2)
		// Line 2, character 50 is approximately over --color-surface-lightest
		hover, err := Hover(req, &protocol.HoverParams{
			TextDocumentPositionParams: protocol.TextDocumentPositionParams{
				TextDocument: protocol.TextDocumentIdentifier{URI: uri},
				Position:     protocol.Position{Line: 2, Character: 50},
			},
		})

		require.NoError(t, err)
		require.NotNil(t, hover, "Should find hover for second inner token")

		content, ok := hover.Contents.(protocol.MarkupContent)
		require.True(t, ok)

		assert.Contains(t, content.Value, "--color-surface-lightest", "Should show correct token name")
		assert.Contains(t, content.Value, "#ffffff", "Should show correct token value")
		assert.Contains(t, content.Value, "Lightest surface color", "Should show correct token description")
		assert.NotContains(t, content.Value, "--_card-background", "Should not show outer local variable")
	})
}
