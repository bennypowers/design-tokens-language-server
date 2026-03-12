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

func TestHover(t *testing.T) {
	tests := []struct {
		name      string
		tokens    []*tokens.Token
		uri       string
		lang      string
		content   string
		line      uint32
		char      uint32
		golden    string // empty = expect nil hover
		wantRange *protocol.Range
	}{
		// CSS var() references
		{
			name: "CSS var() reference",
			tokens: []*tokens.Token{{
				Name: "color.primary", Value: "#ff0000", Type: "color",
				Description: "Primary brand color", FilePath: "tokens.json",
			}},
			uri: "file:///test.css", lang: "css",
			content: `.button { color: var(--color-primary); }`,
			line: 0, char: 24,
			golden: "testdata/golden/color-primary-described-filed.md",
		},
		{
			name: "deprecated token",
			tokens: []*tokens.Token{{
				Name: "color.old-primary", Value: "#cc0000", Type: "color",
				Deprecated: true, DeprecationMessage: "Use color.primary instead",
			}},
			uri: "file:///test.css", lang: "css",
			content: `.button { color: var(--color-old-primary); }`,
			line: 0, char: 28,
			golden: "testdata/golden/color-old-primary-deprecated.md",
		},
		{
			name:    "unknown token in var()",
			uri:     "file:///test.css", lang: "css",
			content: `.button { color: var(--unknown-token); }`,
			line: 0, char: 28,
			golden: "testdata/golden/unknown-token.md",
		},
		{
			name: "var() with fallback",
			tokens: []*tokens.Token{{
				Name: "spacing.large", Value: "2rem", Type: "dimension",
			}},
			uri: "file:///test.css", lang: "css",
			content: `.card { padding: var(--spacing-large, 1rem); }`,
			line: 0, char: 28,
			golden: "testdata/golden/spacing-large-typed.md",
		},
		{
			name: "nested var() in linear-gradient",
			tokens: []*tokens.Token{{
				Name: "color.primary", Value: "#ff0000", Type: "color",
			}},
			uri: "file:///test.css", lang: "css",
			content: `.element { background: linear-gradient(var(--color-primary), white); }`,
			line: 0, char: 47,
			golden: "testdata/golden/color-primary-typed.md",
		},
		{
			name: "cursor outside var() range",
			tokens: []*tokens.Token{{
				Name: "color.primary", Value: "#ff0000",
			}},
			uri: "file:///test.css", lang: "css",
			content: `.button { color: var(--color-primary); }`,
			line: 0, char: 12,
		},
		{
			name: "cursor on property value (not declaration name)",
			tokens: []*tokens.Token{{
				Name: "color.primary", Value: "#ff0000", Type: "color",
			}},
			uri: "file:///test.css", lang: "css",
			content: `:root { --color-primary: #ff0000; }`,
			line: 0, char: 25,
		},
		{
			name:    "cursor outside var() range (invalid position)",
			tokens:  []*tokens.Token{{Name: "color.primary", Value: "#ff0000"}},
			uri:     "file:///test.css", lang: "css",
			content: `.button { color: var(--color-primary); }`,
			line: 0, char: 5,
		},

		// CSS variable declarations
		{
			name: "variable declaration",
			tokens: []*tokens.Token{{
				Name: "color.primary", Value: "#ff0000", Type: "color",
				Description: "Primary brand color",
			}},
			uri: "file:///test.css", lang: "css",
			content: `:root { --color-primary: #ff0000; }`,
			line: 0, char: 10,
			golden: "testdata/golden/color-primary-described.md",
			wantRange: &protocol.Range{
				Start: protocol.Position{Line: 0, Character: 8},
				End:   protocol.Position{Line: 0, Character: 23},
			},
		},
		{
			name:    "unknown variable declaration (local CSS var)",
			uri:     "file:///test.css", lang: "css",
			content: `:root { --local-var: blue; }`,
			line: 0, char: 10,
		},
		{
			name: "variable declaration with prefix",
			tokens: []*tokens.Token{{
				Name: "color.primary", Value: "#0000ff", Type: "color",
				Description: "Blue color", Prefix: "ds",
			}},
			uri: "file:///test.css", lang: "css",
			content: `:root { --ds-color-primary: #0000ff; }`,
			line: 0, char: 12,
			golden: "testdata/golden/ds-color-primary-described.md",
		},

		// HTML style tags
		{
			name: "HTML style tag",
			tokens: []*tokens.Token{{
				Name: "color.primary", Value: "#ff0000", Type: "color",
			}},
			uri: "file:///test.html", lang: "html",
			content: `<style>.button { color: var(--color-primary); }</style>`,
			line: 0, char: 30,
			golden: "testdata/golden/color-primary-typed.md",
		},

		// JS/TS CSS templates
		{
			name: "JS css template literal",
			tokens: []*tokens.Token{{
				Name: "spacing.small", Value: "8px", Type: "dimension",
			}},
			uri: "file:///test.js", lang: "javascript",
			content: "const s = css`\n  .card { padding: var(--spacing-small); }\n`;",
			line: 1, char: 30,
			golden: "testdata/golden/spacing-small-typed.md",
		},
		{
			name: "TSX css template literal",
			tokens: []*tokens.Token{{
				Name: "spacing.small", Value: "8px", Type: "dimension",
			}},
			uri: "file:///test.tsx", lang: "typescriptreact",
			content: "const s = css`\n  .card { padding: var(--spacing-small); }\n`;",
			line: 1, char: 30,
			golden: "testdata/golden/spacing-small-typed.md",
		},

		// JSON/YAML token references
		{
			name: "curly brace reference in JSON",
			tokens: []*tokens.Token{{
				Name: "color-primary", Value: "#ff0000", Type: "color",
				Description: "Primary brand color", FilePath: "tokens.json",
			}},
			uri: "file:///tokens.json", lang: "json",
			content: "{\n  \"color\": {\n    \"secondary\": {\n      \"$value\": \"{color.primary}\"\n    }\n  }\n}",
			line: 3, char: 20,
			golden: "testdata/golden/color-primary-described-filed.md",
		},
		{
			name: "curly brace reference in YAML",
			tokens: []*tokens.Token{{
				Name: "color-accent-base", Value: "#0066cc", Type: "color",
				Description: "Base accent color",
			}},
			uri: "file:///tokens.yaml", lang: "yaml",
			content: "color:\n  button:\n    background:\n      $value: \"{color.accent.base}\"",
			line: 3, char: 20,
			golden: "testdata/golden/color-accent-base-described.md",
		},
		{
			name: "JSON pointer reference ($ref)",
			tokens: []*tokens.Token{{
				Name: "spacing-large", Value: "2rem", Type: "dimension",
				Description: "Large spacing unit",
			}},
			uri: "file:///tokens.json", lang: "json",
			content: "{\n  \"padding\": {\n    \"card\": {\n      \"$ref\": \"#/spacing/large\"\n    }\n  }\n}",
			line: 3, char: 20,
			golden: "testdata/golden/spacing-large-described.md",
		},
		{
			name:    "unknown token reference in JSON",
			uri:     "file:///tokens.json", lang: "json",
			content: "{\n  \"color\": {\n    \"alias\": {\n      \"$value\": \"{unknown.token}\"\n    }\n  }\n}",
			line: 3, char: 20,
			golden: "testdata/golden/unknown-token-ref.md",
		},
		{
			name:    "no reference at cursor position",
			uri:     "file:///tokens.json", lang: "json",
			content: "{\n  \"color\": {\n    \"primary\": {\n      \"$value\": \"#ff0000\"\n    }\n  }\n}",
			line: 3, char: 10,
		},
		{
			name: "deprecated token reference in YAML",
			tokens: []*tokens.Token{{
				Name: "color-old-primary", Value: "#cc0000", Type: "color",
				Deprecated: true, DeprecationMessage: "Use color.primary instead",
			}},
			uri: "file:///tokens.yaml", lang: "yaml",
			content: "color:\n  alias:\n    $value: \"{color.old.primary}\"",
			line: 2, char: 18,
			golden: "testdata/golden/color-old-primary-deprecated.md",
		},

		// Edge cases
		{
			name:    "non-CSS document without references",
			uri:     "file:///test.json", lang: "json",
			content: `{"color": {"$value": "#ff0000"}}`,
			line: 0, char: 10,
		},
		{
			name: "document not found",
			uri:  "file:///nonexistent.css",
			line: 0, char: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := testutil.NewMockServerContext()
			req := types.NewRequestContext(ctx, &glsp.Context{})

			for _, tok := range tt.tokens {
				_ = ctx.TokenManager().Add(tok)
			}
			if tt.content != "" {
				_ = ctx.DocumentManager().DidOpen(tt.uri, tt.lang, 1, tt.content)
			}

			hover, err := Hover(req, &protocol.HoverParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: tt.uri},
					Position:     protocol.Position{Line: tt.line, Character: tt.char},
				},
			})

			require.NoError(t, err)

			if tt.golden == "" {
				assert.Nil(t, hover)
				return
			}

			assertHoverContent(t, hover, tt.golden)

			if tt.wantRange != nil {
				require.NotNil(t, hover.Range)
				assert.Equal(t, *tt.wantRange, *hover.Range)
			}
		})
	}
}

func TestHover_VariableDeclaration_Boundaries(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	req := types.NewRequestContext(ctx, &glsp.Context{})

	_ = ctx.TokenManager().Add(&tokens.Token{
		Name: "color.primary", Value: "#ff0000",
	})

	uri := "file:///test.css"
	_ = ctx.DocumentManager().DidOpen(uri, "css", 1, `:root { --color-primary: #ff0000; }`)

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

func TestHover_VariableDeclaration_MultipleInSameBlock(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	req := types.NewRequestContext(ctx, &glsp.Context{})

	_ = ctx.TokenManager().Add(&tokens.Token{
		Name: "color.primary", Value: "#ff0000", Type: "color",
	})
	_ = ctx.TokenManager().Add(&tokens.Token{
		Name: "color.secondary", Value: "#00ff00", Type: "color",
	})

	uri := "file:///test.css"
	_ = ctx.DocumentManager().DidOpen(uri, "css", 1, ":root {\n  --color-primary: #ff0000;\n  --color-secondary: #00ff00;\n}")

	tests := []struct {
		name   string
		line   uint32
		char   uint32
		golden string
	}{
		{"first declaration", 1, 5, "testdata/golden/color-primary-typed.md"},
		{"second declaration", 2, 5, "testdata/golden/color-secondary-typed.md"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hover, err := Hover(req, &protocol.HoverParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
					Position:     protocol.Position{Line: tt.line, Character: tt.char},
				},
			})
			require.NoError(t, err)
			assertHoverContent(t, hover, tt.golden)
		})
	}
}

// TestHover_NestedVarInFallback tests hovering over nested var() calls in fallback position.
// This is the RHDS pattern: var(--local, var(--design-token, fallback))
func TestHover_NestedVarInFallback(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	req := types.NewRequestContext(ctx, &glsp.Context{})

	_ = ctx.TokenManager().Add(&tokens.Token{
		Name: "color-text-primary", Value: "#000000", Type: "color",
		Description: "Primary text color",
	})
	_ = ctx.TokenManager().Add(&tokens.Token{
		Name: "color-surface-lightest", Value: "#ffffff", Type: "color",
		Description: "Lightest surface color",
	})

	uri := "file:///test.css"
	_ = ctx.DocumentManager().DidOpen(uri, "css", 1,
		".card {\n  color: var(--_local-color, var(--color-text-primary, #000000));\n  background: var(--_card-background, var(--color-surface-lightest, #ffffff));\n}")

	tests := []struct {
		name   string
		line   uint32
		char   uint32
		golden string
	}{
		{"inner token in nested fallback", 1, 40, "testdata/golden/color-text-primary-described.md"},
		{"second nested var in same document", 2, 50, "testdata/golden/color-surface-lightest-described.md"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hover, err := Hover(req, &protocol.HoverParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
					Position:     protocol.Position{Line: tt.line, Character: tt.char},
				},
			})
			require.NoError(t, err)
			assertHoverContent(t, hover, tt.golden)
		})
	}

	t.Run("outer local variable does not show inner token", func(t *testing.T) {
		hover, err := Hover(req, &protocol.HoverParams{
			TextDocumentPositionParams: protocol.TextDocumentPositionParams{
				TextDocument: protocol.TextDocumentIdentifier{URI: uri},
				Position:     protocol.Position{Line: 1, Character: 18},
			},
		})

		require.NoError(t, err)
		if hover != nil {
			content, ok := hover.Contents.(protocol.MarkupContent)
			if ok {
				assert.NotContains(t, content.Value, "--color-text-primary")
			}
		}
	})
}

func TestHover_ContentFormat(t *testing.T) {
	tests := []struct {
		name       string
		format     *protocol.MarkupKind // nil = don't set (test default)
		wantKind   protocol.MarkupKind
		golden     string // empty = don't check content
	}{
		{
			name:     "markdown when client prefers it",
			format:   ptr(protocol.MarkupKindMarkdown),
			wantKind: protocol.MarkupKindMarkdown,
			golden:   "testdata/golden/color-primary-described.md",
		},
		{
			name:     "plaintext when client only supports plaintext",
			format:   ptr(protocol.MarkupKindPlainText),
			wantKind: protocol.MarkupKindPlainText,
			golden:   "testdata/golden/color-primary-described.txt",
		},
		{
			name:     "defaults to markdown when no preference",
			wantKind: protocol.MarkupKindMarkdown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := testutil.NewMockServerContext()
			if tt.format != nil {
				ctx.SetPreferredHoverFormat(*tt.format)
			}
			req := types.NewRequestContext(ctx, &glsp.Context{})

			tok := &tokens.Token{
				Name: "color.primary", Value: "#ff0000", Type: "color",
				Description: "Primary brand color",
			}
			if tt.golden == "" {
				// Bare token for default-format test
				tok = &tokens.Token{Name: "color.primary", Value: "#ff0000"}
			}
			_ = ctx.TokenManager().Add(tok)

			uri := "file:///test.css"
			_ = ctx.DocumentManager().DidOpen(uri, "css", 1, `.button { color: var(--color-primary); }`)

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
			assert.Equal(t, tt.wantKind, content.Kind)

			if tt.golden != "" {
				assertHoverContent(t, hover, tt.golden)
			}
		})
	}
}

func ptr[T any](v T) *T { return &v }

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
		{"srgb color", "color-primary", tokens2025, "testdata/golden/color-primary.md", protocol.MarkupKindMarkdown},
		{"display-p3 color", "color-accent", tokens2025, "testdata/golden/color-accent.md", protocol.MarkupKindMarkdown},
		{"color with hex field", "color-brand", tokens2025, "testdata/golden/color-brand.md", protocol.MarkupKindMarkdown},
		{"color with none component", "color-achromatic", tokens2025, "testdata/golden/color-achromatic.md", protocol.MarkupKindMarkdown},
		{"color without alpha", "color-no-alpha", tokens2025, "testdata/golden/color-no-alpha.md", protocol.MarkupKindMarkdown},
		{"string color (draft schema)", "color-simple", tokensDraft, "testdata/golden/color-simple.md", protocol.MarkupKindMarkdown},
		{"non-color token", "spacing-large", tokens2025, "testdata/golden/spacing-large.md", protocol.MarkupKindMarkdown},
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
