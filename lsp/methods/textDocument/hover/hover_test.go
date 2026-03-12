package hover

import (
	"flag"
	"os"
	"testing"

	asimonimParser "bennypowers.dev/asimonim/parser"
	"bennypowers.dev/dtls/internal/parser/css"
	"bennypowers.dev/dtls/internal/tokens"
	"bennypowers.dev/dtls/lsp/testutil"
	"bennypowers.dev/dtls/lsp/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

var update = flag.Bool("update", false, "update golden files")

// assertHoverContent extracts MarkupContent from hover and asserts it matches the golden file.
func assertHoverContent(t *testing.T, hover *protocol.Hover, goldenPath string) {
	t.Helper()
	require.NotNil(t, hover)
	content, ok := hover.Contents.(protocol.MarkupContent)
	require.True(t, ok, "Contents should be MarkupContent")

	if *update {
		err := os.WriteFile(goldenPath, []byte(content.Value), 0o644)
		require.NoError(t, err)
		return
	}

	golden, err := os.ReadFile(goldenPath)
	require.NoError(t, err, "Failed to load golden file: %s", goldenPath)
	assert.Equal(t, string(golden), content.Value)
}

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

	_ = ctx.TokenManager().Add(&tokens.Token{
		Name:        "color.primary",
		Value:       "#ff0000",
		Type:        "color",
		Description: "Primary brand color",
		FilePath:    "tokens.json",
	})

	uri := "file:///test.css"
	cssContent := `.button { color: var(--color-primary); }`
	_ = ctx.DocumentManager().DidOpen(uri, "css", 1, cssContent)

	hover, err := Hover(req, &protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     protocol.Position{Line: 0, Character: 24},
		},
	})

	require.NoError(t, err)
	assertHoverContent(t, hover, "testdata/golden/color-primary-described-filed.md")
	require.NotNil(t, hover.Range, "Range should be present for var() call")
}

func TestHover_DeprecatedToken(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)

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
	assertHoverContent(t, hover, "testdata/golden/color-old-primary-deprecated.md")
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
	assertHoverContent(t, hover, "testdata/golden/unknown-token.md")
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

	hover, err := Hover(req, &protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     protocol.Position{Line: 0, Character: 28},
		},
	})

	require.NoError(t, err)
	assertHoverContent(t, hover, "testdata/golden/spacing-large-typed.md")
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
	cssContent := `.element { background: linear-gradient(var(--color-primary), white); }`
	_ = ctx.DocumentManager().DidOpen(uri, "css", 1, cssContent)

	hover, err := Hover(req, &protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     protocol.Position{Line: 0, Character: 47},
		},
	})

	require.NoError(t, err)
	assertHoverContent(t, hover, "testdata/golden/color-primary-typed.md")
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

	_ = ctx.TokenManager().Add(&tokens.Token{
		Name:        "color.primary",
		Value:       "#ff0000",
		Type:        "color",
		Description: "Primary brand color",
	})

	uri := "file:///test.css"
	cssContent := `:root { --color-primary: #ff0000; }`
	_ = ctx.DocumentManager().DidOpen(uri, "css", 1, cssContent)

	hover, err := Hover(req, &protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     protocol.Position{Line: 0, Character: 10},
		},
	})

	require.NoError(t, err)
	assertHoverContent(t, hover, "testdata/golden/color-primary-described.md")

	// Assert Range is present and covers only the property name
	require.NotNil(t, hover.Range, "Range should be present for known token declaration")
	assert.Equal(t, uint32(0), hover.Range.Start.Line)
	assert.Equal(t, uint32(8), hover.Range.Start.Character)
	assert.Equal(t, uint32(0), hover.Range.End.Line)
	assert.Equal(t, uint32(23), hover.Range.End.Character)
}

func TestHover_VariableDeclaration_UnknownToken(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)

	uri := "file:///test.css"
	cssContent := `:root { --local-var: blue; }`
	_ = ctx.DocumentManager().DidOpen(uri, "css", 1, cssContent)

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

	_ = ctx.TokenManager().Add(&tokens.Token{
		Name:  "color.primary",
		Value: "#ff0000",
		Type:  "color",
	})

	uri := "file:///test.css"
	cssContent := `:root { --color-primary: #ff0000; }`
	_ = ctx.DocumentManager().DidOpen(uri, "css", 1, cssContent)

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

	_ = ctx.TokenManager().Add(&tokens.Token{
		Name:        "color.primary",
		Value:       "#0000ff",
		Type:        "color",
		Description: "Blue color",
		Prefix:      "ds",
	})

	uri := "file:///test.css"
	cssContent := `:root { --ds-color-primary: #0000ff; }`
	_ = ctx.DocumentManager().DidOpen(uri, "css", 1, cssContent)

	hover, err := Hover(req, &protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     protocol.Position{Line: 0, Character: 12},
		},
	})

	require.NoError(t, err)
	assertHoverContent(t, hover, "testdata/golden/ds-color-primary-described.md")
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

	t.Run("first declaration", func(t *testing.T) {
		hover, err := Hover(req, &protocol.HoverParams{
			TextDocumentPositionParams: protocol.TextDocumentPositionParams{
				TextDocument: protocol.TextDocumentIdentifier{URI: uri},
				Position:     protocol.Position{Line: 1, Character: 5},
			},
		})
		require.NoError(t, err)
		assertHoverContent(t, hover, "testdata/golden/color-primary-typed.md")
	})

	t.Run("second declaration", func(t *testing.T) {
		hover, err := Hover(req, &protocol.HoverParams{
			TextDocumentPositionParams: protocol.TextDocumentPositionParams{
				TextDocument: protocol.TextDocumentIdentifier{URI: uri},
				Position:     protocol.Position{Line: 2, Character: 5},
			},
		})
		require.NoError(t, err)
		assertHoverContent(t, hover, "testdata/golden/color-secondary-typed.md")
	})
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
	cssContent := `.card {
  color: var(--_local-color, var(--color-text-primary, #000000));
  background: var(--_card-background, var(--color-surface-lightest, #ffffff));
}`
	_ = ctx.DocumentManager().DidOpen(uri, "css", 1, cssContent)

	t.Run("hover over inner token in nested fallback", func(t *testing.T) {
		hover, err := Hover(req, &protocol.HoverParams{
			TextDocumentPositionParams: protocol.TextDocumentPositionParams{
				TextDocument: protocol.TextDocumentIdentifier{URI: uri},
				Position:     protocol.Position{Line: 1, Character: 40},
			},
		})

		require.NoError(t, err)
		assertHoverContent(t, hover, "testdata/golden/color-text-primary-described.md")
	})

	t.Run("hover over outer local variable", func(t *testing.T) {
		hover, err := Hover(req, &protocol.HoverParams{
			TextDocumentPositionParams: protocol.TextDocumentPositionParams{
				TextDocument: protocol.TextDocumentIdentifier{URI: uri},
				Position:     protocol.Position{Line: 1, Character: 18},
			},
		})

		require.NoError(t, err)
		// Local variables aren't in token manager — may be nil or show unknown token
		if hover != nil {
			content, ok := hover.Contents.(protocol.MarkupContent)
			if ok {
				assert.NotContains(t, content.Value, "--color-text-primary", "Should not show inner token")
			}
		}
	})

	t.Run("hover over second nested var in same document", func(t *testing.T) {
		hover, err := Hover(req, &protocol.HoverParams{
			TextDocumentPositionParams: protocol.TextDocumentPositionParams{
				TextDocument: protocol.TextDocumentIdentifier{URI: uri},
				Position:     protocol.Position{Line: 2, Character: 50},
			},
		})

		require.NoError(t, err)
		assertHoverContent(t, hover, "testdata/golden/color-surface-lightest-described.md")
	})
}

func TestHover_ContentFormat(t *testing.T) {
	t.Run("returns markdown when client prefers it", func(t *testing.T) {
		ctx := testutil.NewMockServerContext()
		ctx.SetPreferredHoverFormat(protocol.MarkupKindMarkdown)
		glspCtx := &glsp.Context{}
		req := types.NewRequestContext(ctx, glspCtx)

		_ = ctx.TokenManager().Add(&tokens.Token{
			Name:        "color.primary",
			Value:       "#ff0000",
			Type:        "color",
			Description: "Primary brand color",
		})

		uri := "file:///test.css"
		cssContent := `.button { color: var(--color-primary); }`
		_ = ctx.DocumentManager().DidOpen(uri, "css", 1, cssContent)

		hover, err := Hover(req, &protocol.HoverParams{
			TextDocumentPositionParams: protocol.TextDocumentPositionParams{
				TextDocument: protocol.TextDocumentIdentifier{URI: uri},
				Position:     protocol.Position{Line: 0, Character: 24},
			},
		})

		require.NoError(t, err)
		require.NotNil(t, hover)
		content, ok := hover.Contents.(protocol.MarkupContent)
		require.True(t, ok)
		assert.Equal(t, protocol.MarkupKindMarkdown, content.Kind)
		assertHoverContent(t, hover, "testdata/golden/color-primary-described.md")
	})

	t.Run("returns plaintext when client only supports plaintext", func(t *testing.T) {
		ctx := testutil.NewMockServerContext()
		ctx.SetPreferredHoverFormat(protocol.MarkupKindPlainText)
		glspCtx := &glsp.Context{}
		req := types.NewRequestContext(ctx, glspCtx)

		_ = ctx.TokenManager().Add(&tokens.Token{
			Name:        "color.primary",
			Value:       "#ff0000",
			Type:        "color",
			Description: "Primary brand color",
		})

		uri := "file:///test.css"
		cssContent := `.button { color: var(--color-primary); }`
		_ = ctx.DocumentManager().DidOpen(uri, "css", 1, cssContent)

		hover, err := Hover(req, &protocol.HoverParams{
			TextDocumentPositionParams: protocol.TextDocumentPositionParams{
				TextDocument: protocol.TextDocumentIdentifier{URI: uri},
				Position:     protocol.Position{Line: 0, Character: 24},
			},
		})

		require.NoError(t, err)
		require.NotNil(t, hover)
		content, ok := hover.Contents.(protocol.MarkupContent)
		require.True(t, ok)
		assert.Equal(t, protocol.MarkupKindPlainText, content.Kind)
		assertHoverContent(t, hover, "testdata/golden/color-primary-described.txt")
	})

	t.Run("defaults to markdown when no preference", func(t *testing.T) {
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

		hover, err := Hover(req, &protocol.HoverParams{
			TextDocumentPositionParams: protocol.TextDocumentPositionParams{
				TextDocument: protocol.TextDocumentIdentifier{URI: uri},
				Position:     protocol.Position{Line: 0, Character: 24},
			},
		})

		require.NoError(t, err)
		require.NotNil(t, hover)
		content, ok := hover.Contents.(protocol.MarkupContent)
		require.True(t, ok)
		assert.Equal(t, protocol.MarkupKindMarkdown, content.Kind)
	})
}

// ============================================================================
// HTML/JS Hover Tests
// ============================================================================

func TestHover_HTMLStyleTag(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)

	_ = ctx.TokenManager().Add(&tokens.Token{
		Name:  "color.primary",
		Value: "#ff0000",
		Type:  "color",
	})

	uri := "file:///test.html"
	content := `<style>.button { color: var(--color-primary); }</style>`
	_ = ctx.DocumentManager().DidOpen(uri, "html", 1, content)

	hover, err := Hover(req, &protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     protocol.Position{Line: 0, Character: 30},
		},
	})

	require.NoError(t, err)
	assertHoverContent(t, hover, "testdata/golden/color-primary-typed.md")
	require.NotNil(t, hover.Range, "Range should be present for var() call in HTML")
	assert.Equal(t, uint32(0), hover.Range.Start.Line)
}

func TestHover_JSCSSTemplate(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)

	_ = ctx.TokenManager().Add(&tokens.Token{
		Name:  "spacing.small",
		Value: "8px",
		Type:  "dimension",
	})

	uri := "file:///test.js"
	content := "const s = css`\n  .card { padding: var(--spacing-small); }\n`;"
	_ = ctx.DocumentManager().DidOpen(uri, "javascript", 1, content)

	hover, err := Hover(req, &protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     protocol.Position{Line: 1, Character: 30},
		},
	})

	require.NoError(t, err)
	assertHoverContent(t, hover, "testdata/golden/spacing-small-typed.md")
	require.NotNil(t, hover.Range, "Range should be present for var() call in JS template")
	assert.Equal(t, uint32(1), hover.Range.Start.Line)
}

func TestHover_TSXCSSTemplate(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)

	_ = ctx.TokenManager().Add(&tokens.Token{
		Name:  "spacing.small",
		Value: "8px",
		Type:  "dimension",
	})

	uri := "file:///test.tsx"
	content := "const s = css`\n  .card { padding: var(--spacing-small); }\n`;"
	_ = ctx.DocumentManager().DidOpen(uri, "typescriptreact", 1, content)

	hover, err := Hover(req, &protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     protocol.Position{Line: 1, Character: 30},
		},
	})

	require.NoError(t, err)
	assertHoverContent(t, hover, "testdata/golden/spacing-small-typed.md")
	require.NotNil(t, hover.Range, "Range should be present for var() call in TSX template")
	assert.Equal(t, uint32(1), hover.Range.Start.Line)
}

// ============================================================================
// JSON/YAML Token Reference Hover Tests
// ============================================================================

func TestHover_CurlyBraceReference_JSON(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)

	_ = ctx.TokenManager().Add(&tokens.Token{
		Name:        "color-primary",
		Value:       "#ff0000",
		Type:        "color",
		Description: "Primary brand color",
		FilePath:    "tokens.json",
	})

	uri := "file:///tokens.json"
	jsonContent := `{
  "color": {
    "secondary": {
      "$value": "{color.primary}"
    }
  }
}`
	_ = ctx.DocumentManager().DidOpen(uri, "json", 1, jsonContent)

	hover, err := Hover(req, &protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     protocol.Position{Line: 3, Character: 20},
		},
	})

	require.NoError(t, err)
	assertHoverContent(t, hover, "testdata/golden/color-primary-described-filed.md")
	require.NotNil(t, hover.Range, "Range should be present for token reference")
}

func TestHover_CurlyBraceReference_YAML(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)

	_ = ctx.TokenManager().Add(&tokens.Token{
		Name:        "color-accent-base",
		Value:       "#0066cc",
		Type:        "color",
		Description: "Base accent color",
	})

	uri := "file:///tokens.yaml"
	yamlContent := `color:
  button:
    background:
      $value: "{color.accent.base}"`
	_ = ctx.DocumentManager().DidOpen(uri, "yaml", 1, yamlContent)

	hover, err := Hover(req, &protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     protocol.Position{Line: 3, Character: 20},
		},
	})

	require.NoError(t, err)
	assertHoverContent(t, hover, "testdata/golden/color-accent-base-described.md")
}

func TestHover_JSONPointerReference(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)

	_ = ctx.TokenManager().Add(&tokens.Token{
		Name:        "spacing-large",
		Value:       "2rem",
		Type:        "dimension",
		Description: "Large spacing unit",
	})

	uri := "file:///tokens.json"
	jsonContent := `{
  "padding": {
    "card": {
      "$ref": "#/spacing/large"
    }
  }
}`
	_ = ctx.DocumentManager().DidOpen(uri, "json", 1, jsonContent)

	hover, err := Hover(req, &protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     protocol.Position{Line: 3, Character: 20},
		},
	})

	require.NoError(t, err)
	assertHoverContent(t, hover, "testdata/golden/spacing-large-described.md")
}

func TestHover_TokenReference_UnknownToken(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)

	uri := "file:///tokens.json"
	jsonContent := `{
  "color": {
    "alias": {
      "$value": "{unknown.token}"
    }
  }
}`
	_ = ctx.DocumentManager().DidOpen(uri, "json", 1, jsonContent)

	hover, err := Hover(req, &protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     protocol.Position{Line: 3, Character: 20},
		},
	})

	require.NoError(t, err)
	assertHoverContent(t, hover, "testdata/golden/unknown-token-ref.md")
}

func TestHover_TokenReference_NoReferenceAtPosition(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)

	uri := "file:///tokens.json"
	jsonContent := `{
  "color": {
    "primary": {
      "$value": "#ff0000"
    }
  }
}`
	_ = ctx.DocumentManager().DidOpen(uri, "json", 1, jsonContent)

	hover, err := Hover(req, &protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     protocol.Position{Line: 3, Character: 10},
		},
	})

	require.NoError(t, err)
	assert.Nil(t, hover, "Should not show hover when not on a reference")
}

func TestHover_TokenReference_DeprecatedToken(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}
	req := types.NewRequestContext(ctx, glspCtx)

	_ = ctx.TokenManager().Add(&tokens.Token{
		Name:               "color-old-primary",
		Value:              "#cc0000",
		Type:               "color",
		Deprecated:         true,
		DeprecationMessage: "Use color.primary instead",
	})

	uri := "file:///tokens.yaml"
	yamlContent := `color:
  alias:
    $value: "{color.old.primary}"`
	_ = ctx.DocumentManager().DidOpen(uri, "yaml", 1, yamlContent)

	hover, err := Hover(req, &protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     protocol.Position{Line: 2, Character: 18},
		},
	})

	require.NoError(t, err)
	assertHoverContent(t, hover, "testdata/golden/color-old-primary-deprecated.md")
}

// ============================================================================
// Structured Color Value Hover Tests (DTCG 2025.10)
// ============================================================================

func parseTokensFile(t *testing.T, path string) map[string]*tokens.Token {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err)

	p := asimonimParser.NewJSONParser()
	toks, err := p.Parse(data, asimonimParser.Options{SkipPositions: true})
	require.NoError(t, err)

	byName := make(map[string]*tokens.Token, len(toks))
	for _, tok := range toks {
		byName[tok.Name] = tok
	}
	return byName
}

func TestRenderTokenHover_StructuredColor(t *testing.T) {
	tokens2025 := parseTokensFile(t, "testdata/tokens-2025.json")
	tokensDraft := parseTokensFile(t, "testdata/tokens-draft.json")

	tests := []struct {
		name      string
		tokenName string
		tokens    map[string]*tokens.Token
		golden    string
		format    protocol.MarkupKind
	}{
		{
			name:      "srgb color markdown",
			tokenName: "color-primary",
			tokens:    tokens2025,
			golden:    "testdata/golden/color-primary.md",
			format:    protocol.MarkupKindMarkdown,
		},
		{
			name:      "display-p3 color",
			tokenName: "color-accent",
			tokens:    tokens2025,
			golden:    "testdata/golden/color-accent.md",
			format:    protocol.MarkupKindMarkdown,
		},
		{
			name:      "color with hex field",
			tokenName: "color-brand",
			tokens:    tokens2025,
			golden:    "testdata/golden/color-brand.md",
			format:    protocol.MarkupKindMarkdown,
		},
		{
			name:      "color with none component",
			tokenName: "color-achromatic",
			tokens:    tokens2025,
			golden:    "testdata/golden/color-achromatic.md",
			format:    protocol.MarkupKindMarkdown,
		},
		{
			name:      "color without alpha",
			tokenName: "color-no-alpha",
			tokens:    tokens2025,
			golden:    "testdata/golden/color-no-alpha.md",
			format:    protocol.MarkupKindMarkdown,
		},
		{
			name:      "string color (draft schema)",
			tokenName: "color-simple",
			tokens:    tokensDraft,
			golden:    "testdata/golden/color-simple.md",
			format:    protocol.MarkupKindMarkdown,
		},
		{
			name:      "non-color token",
			tokenName: "spacing-large",
			tokens:    tokens2025,
			golden:    "testdata/golden/spacing-large.md",
			format:    protocol.MarkupKindMarkdown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, ok := tt.tokens[tt.tokenName]
			require.True(t, ok, "token %q not found in fixture", tt.tokenName)

			content, err := renderTokenHover(token, tt.format)
			require.NoError(t, err)

			if *update {
				err := os.WriteFile(tt.golden, []byte(content), 0o644)
				require.NoError(t, err)
				return
			}

			golden, err := os.ReadFile(tt.golden)
			require.NoError(t, err)
			assert.Equal(t, string(golden), content)
		})
	}
}
