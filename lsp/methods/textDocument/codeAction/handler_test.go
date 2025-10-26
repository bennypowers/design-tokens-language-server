package codeaction

import (
	"testing"

	"bennypowers.dev/dtls/internal/tokens"
	"bennypowers.dev/dtls/lsp/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func TestCodeAction_IncorrectFallback(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}

	// Add a color token
	ctx.TokenManager().Add(&tokens.Token{
		Name:  "color.primary",
		Value: "#0000ff",
		Type:  "color",
	})

	uri := "file:///test.css"
	// Incorrect fallback: token is #0000ff but fallback is #ff0000
	cssContent := `.button { color: var(--color-primary, #ff0000); }`
	ctx.DocumentManager().DidOpen(uri, "css", 1, cssContent)

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

	// Should have a fix fallback action with the correct value
	var fixAction *protocol.CodeAction
	for i := range actions {
		if actions[i].Title == "Fix fallback value to '#0000ff'" {
			fixAction = &actions[i]
			break
		}
	}
	require.NotNil(t, fixAction, "Should have 'Fix fallback value to '#0000ff'' action")

	// Verify it's a quick fix
	assert.NotNil(t, fixAction.Kind)
	assert.Equal(t, protocol.CodeActionKindQuickFix, *fixAction.Kind)

	// Verify the edit contains the correct replacement
	require.NotNil(t, fixAction.Edit)
	require.NotNil(t, fixAction.Edit.Changes)
	edits := fixAction.Edit.Changes[uri]
	require.Len(t, edits, 1)
	assert.Contains(t, edits[0].NewText, "var(--color-primary, #0000ff)")
}

func TestCodeAction_AddFallback(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}

	// Add a color token
	ctx.TokenManager().Add(&tokens.Token{
		Name:  "color.primary",
		Value: "#0000ff",
		Type:  "color",
	})

	uri := "file:///test.css"
	// No fallback provided
	cssContent := `.button { color: var(--color-primary); }`
	ctx.DocumentManager().DidOpen(uri, "css", 1, cssContent)

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

	// Should suggest adding a fallback with the correct value
	var addAction *protocol.CodeAction
	for i := range actions {
		if actions[i].Title == "Add fallback value '#0000ff'" {
			addAction = &actions[i]
			break
		}
	}
	require.NotNil(t, addAction, "Should have 'Add fallback value '#0000ff'' action")

	// Verify the edit contains the correct value
	require.NotNil(t, addAction.Edit)
	require.NotNil(t, addAction.Edit.Changes)
	edits := addAction.Edit.Changes[uri]
	require.Len(t, edits, 1)
	assert.Contains(t, edits[0].NewText, "var(--color-primary, #0000ff)")
}

func TestCodeAction_NonCSSDocument(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}

	uri := "file:///test.json"
	jsonContent := `{"color": {"$value": "#ff0000"}}`
	ctx.DocumentManager().DidOpen(uri, "json", 1, jsonContent)

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
	ctx := testutil.NewMockServerContext()
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
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}

	ctx.TokenManager().Add(&tokens.Token{
		Name:  "color.primary",
		Value: "#0000ff",
		Type:  "color",
	})

	uri := "file:///test.css"
	cssContent := `.button { color: var(--color-primary, #ff0000); }`
	ctx.DocumentManager().DidOpen(uri, "css", 1, cssContent)

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
	ctx := testutil.NewMockServerContext()
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
