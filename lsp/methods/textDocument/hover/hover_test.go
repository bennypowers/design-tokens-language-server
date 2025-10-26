package hover

import (
	"github.com/bennypowers/design-tokens-language-server/lsp/types"
	"testing"

	"github.com/bennypowers/design-tokens-language-server/internal/documents"
	"github.com/bennypowers/design-tokens-language-server/internal/parser/css"
	"github.com/bennypowers/design-tokens-language-server/internal/tokens"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// mockServerContext implements types.ServerContext for testing
type mockServerContext struct {
	docs   *documents.Manager
	tokens *tokens.Manager
}

func (m *mockServerContext) Document(uri string) *documents.Document {
	return m.docs.Get(uri)
}

func (m *mockServerContext) DocumentManager() *documents.Manager {
	return m.docs
}

func (m *mockServerContext) AllDocuments() []*documents.Document {
	return m.docs.GetAll()
}

func (m *mockServerContext) Token(name string) *tokens.Token {
	return m.tokens.Get(name)
}

func (m *mockServerContext) TokenManager() *tokens.Manager {
	return m.tokens
}

func (m *mockServerContext) TokenCount() int {
	return m.tokens.Count()
}

func (m *mockServerContext) RootURI() string {
	return "file:///workspace"
}

func (m *mockServerContext) RootPath() string {
	return "/workspace"
}

func (m *mockServerContext) SetRootURI(uri string) {}

func (m *mockServerContext) SetRootPath(path string) {}

func (m *mockServerContext) LoadTokensFromConfig() error {
	return nil
}

func (m *mockServerContext) RegisterFileWatchers(ctx *glsp.Context) error {
	return nil
}

func (m *mockServerContext) GLSPContext() *glsp.Context {
	return nil
}

func (m *mockServerContext) SetGLSPContext(ctx *glsp.Context) {}



func (m *mockServerContext) GetConfig() types.ServerConfig {
	return types.DefaultConfig()
}

func (m *mockServerContext) SetConfig(config types.ServerConfig) {}

func (m *mockServerContext) IsTokenFile(path string) bool {
	return false
}

func (m *mockServerContext) PublishDiagnostics(context *glsp.Context, uri string) error {
	return nil
}

func newMockServerContext() *mockServerContext {
	return &mockServerContext{
		docs:   documents.NewManager(),
		tokens: tokens.NewManager(),
	}
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
	ctx := newMockServerContext()
	glspCtx := &glsp.Context{}

	// Add a token
	ctx.tokens.Add(&tokens.Token{
		Name:        "color.primary",
		Value:       "#ff0000",
		Type:        "color",
		Description: "Primary brand color",
		FilePath:    "tokens.json",
	})

	// Open a CSS document with var() call
	uri := "file:///test.css"
	cssContent := `.button { color: var(--color-primary); }`
	ctx.docs.DidOpen(uri, "css", 1, cssContent)

	// Hover over --color-primary
	hover, err := Hover(ctx, glspCtx, &protocol.HoverParams{
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
}

func TestHover_DeprecatedToken(t *testing.T) {
	ctx := newMockServerContext()
	glspCtx := &glsp.Context{}

	// Add deprecated token
	ctx.tokens.Add(&tokens.Token{
		Name:               "color.old-primary",
		Value:              "#cc0000",
		Type:               "color",
		Deprecated:         true,
		DeprecationMessage: "Use color.primary instead",
	})

	uri := "file:///test.css"
	cssContent := `.button { color: var(--color-old-primary); }`
	ctx.docs.DidOpen(uri, "css", 1, cssContent)

	hover, err := Hover(ctx, glspCtx, &protocol.HoverParams{
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
	ctx := newMockServerContext()
	glspCtx := &glsp.Context{}

	uri := "file:///test.css"
	cssContent := `.button { color: var(--unknown-token); }`
	ctx.docs.DidOpen(uri, "css", 1, cssContent)

	hover, err := Hover(ctx, glspCtx, &protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     protocol.Position{Line: 0, Character: 28},
		},
	})

	require.NoError(t, err)
	require.NotNil(t, hover)

	// Assert unknown token message
	content, ok := hover.Contents.(protocol.MarkupContent)
	require.True(t, ok)
	assert.Contains(t, content.Value, "Unknown token")
	assert.Contains(t, content.Value, "--unknown-token")
}

func TestHover_VariableDeclaration(t *testing.T) {
	ctx := newMockServerContext()
	glspCtx := &glsp.Context{}

	// Add a token
	ctx.tokens.Add(&tokens.Token{
		Name:        "color.primary",
		Value:       "#ff0000",
		Type:        "color",
		Description: "Primary brand color",
	})

	uri := "file:///test.css"
	cssContent := `:root { --color-primary: #ff0000; }`
	ctx.docs.DidOpen(uri, "css", 1, cssContent)

	// Hover over variable declaration
	hover, err := Hover(ctx, glspCtx, &protocol.HoverParams{
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
}

func TestHover_InvalidPosition(t *testing.T) {
	ctx := newMockServerContext()
	glspCtx := &glsp.Context{}

	ctx.tokens.Add(&tokens.Token{
		Name:  "color.primary",
		Value: "#ff0000",
	})

	uri := "file:///test.css"
	cssContent := `.button { color: var(--color-primary); }`
	ctx.docs.DidOpen(uri, "css", 1, cssContent)

	// Hover outside var() call
	hover, err := Hover(ctx, glspCtx, &protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     protocol.Position{Line: 0, Character: 5},
		},
	})

	require.NoError(t, err)
	assert.Nil(t, hover)
}

func TestHover_NonCSSDocument(t *testing.T) {
	ctx := newMockServerContext()
	glspCtx := &glsp.Context{}

	uri := "file:///test.json"
	jsonContent := `{"color": {"$value": "#ff0000"}}`
	ctx.docs.DidOpen(uri, "json", 1, jsonContent)

	hover, err := Hover(ctx, glspCtx, &protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     protocol.Position{Line: 0, Character: 10},
		},
	})

	require.NoError(t, err)
	assert.Nil(t, hover)
}

func TestHover_DocumentNotFound(t *testing.T) {
	ctx := newMockServerContext()
	glspCtx := &glsp.Context{}

	hover, err := Hover(ctx, glspCtx, &protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: "file:///nonexistent.css"},
			Position:     protocol.Position{Line: 0, Character: 10},
		},
	})

	require.NoError(t, err)
	assert.Nil(t, hover)
}
