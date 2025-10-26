package codeaction

import (
	"strings"
	"testing"

	"github.com/bennypowers/design-tokens-language-server/internal/documents"
	"github.com/bennypowers/design-tokens-language-server/internal/tokens"
	"github.com/bennypowers/design-tokens-language-server/lsp/types"
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

func TestCodeAction_IncorrectFallback(t *testing.T) {
	ctx := newMockServerContext()
	glspCtx := &glsp.Context{}

	// Add a color token
	ctx.tokens.Add(&tokens.Token{
		Name:  "color.primary",
		Value: "#0000ff",
		Type:  "color",
	})

	uri := "file:///test.css"
	// Incorrect fallback: token is #0000ff but fallback is #ff0000
	cssContent := `.button { color: var(--color-primary, #ff0000); }`
	ctx.docs.DidOpen(uri, "css", 1, cssContent)

	result, err := CodeAction(ctx, glspCtx, &protocol.CodeActionParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: uri},
		Range: protocol.Range{
			Start: protocol.Position{Line: 0, Character: 17},
			End:   protocol.Position{Line: 0, Character: 45},
		},
		Context: protocol.CodeActionContext{
			Diagnostics: []protocol.Diagnostic{},
		},
	})

	require.NoError(t, err)
	actions, ok := result.([]protocol.CodeAction)
	require.True(t, ok)
	require.NotEmpty(t, actions)

	// Should have a fix fallback action
	foundFix := false
	for _, action := range actions {
		if strings.HasPrefix(action.Title, "Fix fallback value") {
			foundFix = true
			break
		}
	}
	assert.True(t, foundFix, "Should have a fix fallback action")
}

func TestCodeAction_AddFallback(t *testing.T) {
	ctx := newMockServerContext()
	glspCtx := &glsp.Context{}

	// Add a color token
	ctx.tokens.Add(&tokens.Token{
		Name:  "color.primary",
		Value: "#0000ff",
		Type:  "color",
	})

	uri := "file:///test.css"
	// No fallback provided
	cssContent := `.button { color: var(--color-primary); }`
	ctx.docs.DidOpen(uri, "css", 1, cssContent)

	result, err := CodeAction(ctx, glspCtx, &protocol.CodeActionParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: uri},
		Range: protocol.Range{
			Start: protocol.Position{Line: 0, Character: 17},
			End:   protocol.Position{Line: 0, Character: 36},
		},
		Context: protocol.CodeActionContext{
			Diagnostics: []protocol.Diagnostic{},
		},
	})

	require.NoError(t, err)
	actions, ok := result.([]protocol.CodeAction)
	require.True(t, ok)
	require.NotEmpty(t, actions)

	// Should suggest adding a fallback
	foundAdd := false
	for _, action := range actions {
		if strings.HasPrefix(action.Title, "Add fallback value") {
			foundAdd = true
			break
		}
	}
	assert.True(t, foundAdd, "Should suggest adding fallback")
}

func TestCodeAction_NonCSSDocument(t *testing.T) {
	ctx := newMockServerContext()
	glspCtx := &glsp.Context{}

	uri := "file:///test.json"
	jsonContent := `{"color": {"$value": "#ff0000"}}`
	ctx.docs.DidOpen(uri, "json", 1, jsonContent)

	result, err := CodeAction(ctx, glspCtx, &protocol.CodeActionParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: uri},
		Range: protocol.Range{
			Start: protocol.Position{Line: 0, Character: 0},
			End:   protocol.Position{Line: 0, Character: 10},
		},
		Context: protocol.CodeActionContext{
			Diagnostics: []protocol.Diagnostic{},
		},
	})

	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestCodeAction_DocumentNotFound(t *testing.T) {
	ctx := newMockServerContext()
	glspCtx := &glsp.Context{}

	result, err := CodeAction(ctx, glspCtx, &protocol.CodeActionParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: "file:///nonexistent.css"},
		Range: protocol.Range{
			Start: protocol.Position{Line: 0, Character: 0},
			End:   protocol.Position{Line: 0, Character: 10},
		},
		Context: protocol.CodeActionContext{
			Diagnostics: []protocol.Diagnostic{},
		},
	})

	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestCodeAction_OutsideRange(t *testing.T) {
	ctx := newMockServerContext()
	glspCtx := &glsp.Context{}

	ctx.tokens.Add(&tokens.Token{
		Name:  "color.primary",
		Value: "#0000ff",
		Type:  "color",
	})

	uri := "file:///test.css"
	cssContent := `.button { color: var(--color-primary, #ff0000); }`
	ctx.docs.DidOpen(uri, "css", 1, cssContent)

	// Request range that doesn't intersect with var()
	result, err := CodeAction(ctx, glspCtx, &protocol.CodeActionParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: uri},
		Range: protocol.Range{
			Start: protocol.Position{Line: 0, Character: 0},
			End:   protocol.Position{Line: 0, Character: 7}, // Before var()
		},
		Context: protocol.CodeActionContext{
			Diagnostics: []protocol.Diagnostic{},
		},
	})

	require.NoError(t, err)
	actions, ok := result.([]protocol.CodeAction)
	require.True(t, ok)
	assert.Empty(t, actions) // No actions for range outside var()
}

func TestCodeActionResolve_ReturnsActionUnchanged(t *testing.T) {
	ctx := newMockServerContext()
	glspCtx := &glsp.Context{}

	action := &protocol.CodeAction{
		Title: "Test action",
		Kind:  ptrCodeActionKind(protocol.CodeActionKindQuickFix),
	}

	resolved, err := CodeActionResolve(ctx, glspCtx, action)

	require.NoError(t, err)
	assert.Equal(t, action, resolved) // Should return same action
}

func TestIsCSSValueSemanticallyEquivalent(t *testing.T) {
	tests := []struct {
		name     string
		a        string
		b        string
		expected bool
	}{
		{
			name:     "exact match",
			a:        "#ff0000",
			b:        "#ff0000",
			expected: true,
		},
		{
			name:     "case insensitive",
			a:        "#FF0000",
			b:        "#ff0000",
			expected: true,
		},
		{
			name:     "whitespace differences",
			a:        "rgb( 255, 0, 0 )",
			b:        "rgb(255,0,0)",
			expected: true,
		},
		{
			name:     "different values",
			a:        "#ff0000",
			b:        "#0000ff",
			expected: false,
		},
		{
			name:     "tab and newline normalization",
			a:        "rgba(\n\t255,\n\t0,\n\t0,\n\t1\n)",
			b:        "rgba(255,0,0,1)",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isCSSValueSemanticallyEquivalent(tt.a, tt.b)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func ptrCodeActionKind(k protocol.CodeActionKind) *protocol.CodeActionKind {
	return &k
}
