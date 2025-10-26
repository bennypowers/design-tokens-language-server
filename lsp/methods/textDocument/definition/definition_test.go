package definition

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

func TestDefinition_CSSVariableReference(t *testing.T) {
	ctx := newMockServerContext()
	glspCtx := &glsp.Context{}

	// Add a token with definition URI
	ctx.tokens.Add(&tokens.Token{
		Name:          "color.primary",
		Value:         "#ff0000",
		Type:          "color",
		DefinitionURI: "file:///workspace/tokens.json",
		Path:          []string{"color", "primary"},
	})

	uri := "file:///test.css"
	cssContent := `.button { color: var(--color-primary); }`
	ctx.docs.DidOpen(uri, "css", 1, cssContent)

	result, err := Definition(ctx, glspCtx, &protocol.DefinitionParams{
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
	ctx := newMockServerContext()
	glspCtx := &glsp.Context{}

	uri := "file:///test.css"
	cssContent := `.button { color: var(--unknown-token); }`
	ctx.docs.DidOpen(uri, "css", 1, cssContent)

	result, err := Definition(ctx, glspCtx, &protocol.DefinitionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     protocol.Position{Line: 0, Character: 24},
		},
	})

	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestDefinition_TokenWithoutDefinitionURI(t *testing.T) {
	ctx := newMockServerContext()
	glspCtx := &glsp.Context{}

	// Add token without DefinitionURI
	ctx.tokens.Add(&tokens.Token{
		Name:  "color.primary",
		Value: "#ff0000",
	})

	uri := "file:///test.css"
	cssContent := `.button { color: var(--color-primary); }`
	ctx.docs.DidOpen(uri, "css", 1, cssContent)

	result, err := Definition(ctx, glspCtx, &protocol.DefinitionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     protocol.Position{Line: 0, Character: 24},
		},
	})

	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestDefinition_OutsideVarCall(t *testing.T) {
	ctx := newMockServerContext()
	glspCtx := &glsp.Context{}

	ctx.tokens.Add(&tokens.Token{
		Name:          "color.primary",
		Value:         "#ff0000",
		DefinitionURI: "file:///workspace/tokens.json",
		Path:          []string{"color", "primary"},
	})

	uri := "file:///test.css"
	cssContent := `.button { color: var(--color-primary); }`
	ctx.docs.DidOpen(uri, "css", 1, cssContent)

	// Position outside the var() call
	result, err := Definition(ctx, glspCtx, &protocol.DefinitionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     protocol.Position{Line: 0, Character: 5}, // Inside ".button"
		},
	})

	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestDefinition_NonCSSDocument(t *testing.T) {
	ctx := newMockServerContext()
	glspCtx := &glsp.Context{}

	uri := "file:///test.json"
	jsonContent := `{"color": {"$value": "#ff0000"}}`
	ctx.docs.DidOpen(uri, "json", 1, jsonContent)

	result, err := Definition(ctx, glspCtx, &protocol.DefinitionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     protocol.Position{Line: 0, Character: 10},
		},
	})

	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestDefinition_DocumentNotFound(t *testing.T) {
	ctx := newMockServerContext()
	glspCtx := &glsp.Context{}

	result, err := Definition(ctx, glspCtx, &protocol.DefinitionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: "file:///nonexistent.css"},
			Position:     protocol.Position{Line: 0, Character: 10},
		},
	})

	require.NoError(t, err)
	assert.Nil(t, result)
}

// TestIsPositionInVarCall tests the isPositionInVarCall function with half-open range semantics [start, end)
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
