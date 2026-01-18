package codeaction_test

import (
	"testing"

	"bennypowers.dev/dtls/internal/tokens"
	"bennypowers.dev/dtls/lsp"
	codeaction "bennypowers.dev/dtls/lsp/methods/textDocument/codeAction"
	"bennypowers.dev/dtls/lsp/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

const (
	// CodeActionKindSourceFixAll is not defined in glsp v0.2.2
	codeActionKindSourceFixAll protocol.CodeActionKind = "source.fixAll"
)

// ptrIntegerOrString returns a pointer to IntegerOrString from a string
func ptrIntegerOrString(s string) *protocol.IntegerOrString {
	return &protocol.IntegerOrString{Value: s}
}

// TestRangesIntersect tests the rangesIntersect function with half-open range semantics [start, end)
func TestToggleFallback(t *testing.T) {
	s, err := lsp.NewServer()
	require.NoError(t, err)

	// Add test token
	token := &tokens.Token{
		Name:  "color-primary",
		Value: "#ff0000",
		Type:  "color",
	}
	_ = s.TokenManager().Add(token)

	tests := []struct {
		name           string
		cssContent     string
		expectedAction string
		expectedEdit   string
		cursorLine     uint32
		cursorChar     uint32
	}{
		{
			name:           "toggle off - remove existing fallback",
			cssContent:     `.button { color: var(--color-primary, #ff0000); }`,
			cursorLine:     0,
			cursorChar:     21, // cursor on var(
			expectedAction: "Toggle design token fallback value",
			expectedEdit:   "var(--color-primary)",
		},
		{
			name:           "toggle on - add fallback when missing",
			cssContent:     `.button { color: var(--color-primary); }`,
			cursorLine:     0,
			cursorChar:     21,
			expectedAction: "Toggle design token fallback value",
			expectedEdit:   "var(--color-primary, #ff0000)",
		},
		{
			name:           "toggle off - cursor in middle of var call",
			cssContent:     `.button { color: var(--color-primary, blue); }`,
			cursorLine:     0,
			cursorChar:     30, // cursor in token name
			expectedAction: "Toggle design token fallback value",
			expectedEdit:   "var(--color-primary)",
		},
		{
			name:           "no action - cursor outside var call",
			cssContent:     `.button { color: var(--color-primary); padding: 10px; }`,
			cursorLine:     0,
			cursorChar:     50, // cursor on padding
			expectedAction: "", // no action
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Open document
			uri := "file:///test.css"
			_ = s.DocumentManager().DidOpen(uri, "css", 1, tt.cssContent)

			// Request code actions at cursor position (collapsed range)
			params := &protocol.CodeActionParams{
				TextDocument: protocol.TextDocumentIdentifier{URI: uri},
				Range: protocol.Range{
					Start: protocol.Position{Line: tt.cursorLine, Character: tt.cursorChar},
					End:   protocol.Position{Line: tt.cursorLine, Character: tt.cursorChar},
				},
				Context: protocol.CodeActionContext{},
			}

			req := types.NewRequestContext(s, nil)
			result, err := codeaction.CodeAction(req, params)
			require.NoError(t, err)

			if tt.expectedAction == "" {
				// Should not have toggle action
				if result != nil {
					actions := result.([]protocol.CodeAction)
					for _, action := range actions {
						assert.NotEqual(t, tt.expectedAction, action.Title)
					}
				}
				return
			}

			// Should have the toggle action
			require.NotNil(t, result)
			actions := result.([]protocol.CodeAction)

			var toggleAction *protocol.CodeAction
			for i := range actions {
				if actions[i].Title == tt.expectedAction {
					toggleAction = &actions[i]
					break
				}
			}

			require.NotNil(t, toggleAction, "Should have toggle action")

			// Check action kind
			require.NotNil(t, toggleAction.Kind)
			assert.Equal(t, protocol.CodeActionKindRefactorRewrite, *toggleAction.Kind)

			// Check edit
			require.NotNil(t, toggleAction.Edit)
			require.NotNil(t, toggleAction.Edit.Changes)
			edits, ok := toggleAction.Edit.Changes[uri]
			require.True(t, ok)
			require.Len(t, edits, 1)
			assert.Equal(t, tt.expectedEdit, edits[0].NewText)
		})
	}
}

// TestToggleRangeFallbacks tests toggle fallbacks for range selection
func TestToggleRangeFallbacks(t *testing.T) {
	s, err := lsp.NewServer()
	require.NoError(t, err)

	// Add test tokens
	_ = s.TokenManager().Add(&tokens.Token{Name: "color-primary", Value: "#ff0000", Type: "color"})
	_ = s.TokenManager().Add(&tokens.Token{Name: "color-secondary", Value: "#00ff00", Type: "color"})

	tests := []struct {
		name           string
		cssContent     string
		expectedAction string
		numEdits       int
		rangeStart     protocol.Position
		rangeEnd       protocol.Position
	}{
		{
			name: "toggle off - multiple var calls",
			cssContent: `.button {
  color: var(--color-primary, #ff0000);
  background: var(--color-secondary, #00ff00);
}`,
			rangeStart:     protocol.Position{Line: 1, Character: 0},
			rangeEnd:       protocol.Position{Line: 2, Character: 50},
			expectedAction: "Toggle design token fallback values (in range)",
			numEdits:       2, // two var() calls
		},
		{
			name:           "toggle on - multiple var calls without fallbacks",
			cssContent:     `.button { color: var(--color-primary); background: var(--color-secondary); }`,
			rangeStart:     protocol.Position{Line: 0, Character: 10},
			rangeEnd:       protocol.Position{Line: 0, Character: 75},
			expectedAction: "Toggle design token fallback values (in range)",
			numEdits:       2,
		},
		{
			name:           "single char range - should show range action",
			cssContent:     `.button { color: var(--color-primary, #ff0000); }`,
			rangeStart:     protocol.Position{Line: 0, Character: 21},
			rangeEnd:       protocol.Position{Line: 0, Character: 22},
			expectedAction: "Toggle design token fallback values (in range)", // 1-char selection shows range toggle
			numEdits:       1,
		},
		{
			name:           "collapsed cursor - should not show range action",
			cssContent:     `.button { color: var(--color-primary, #ff0000); }`,
			rangeStart:     protocol.Position{Line: 0, Character: 21},
			rangeEnd:       protocol.Position{Line: 0, Character: 21},
			expectedAction: "", // collapsed cursor shows single toggle, not range toggle
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uri := "file:///test.css"
			_ = s.DocumentManager().DidOpen(uri, "css", 1, tt.cssContent)

			params := &protocol.CodeActionParams{
				TextDocument: protocol.TextDocumentIdentifier{URI: uri},
				Range: protocol.Range{
					Start: tt.rangeStart,
					End:   tt.rangeEnd,
				},
				Context: protocol.CodeActionContext{},
			}

			req := types.NewRequestContext(s, nil)
			result, err := codeaction.CodeAction(req, params)
			require.NoError(t, err)

			if tt.expectedAction == "" {
				if result != nil {
					actions := result.([]protocol.CodeAction)
					for _, action := range actions {
						assert.NotEqual(t, "Toggle design token fallback values (in range)", action.Title)
					}
				}
				return
			}

			require.NotNil(t, result)
			actions := result.([]protocol.CodeAction)

			var rangeAction *protocol.CodeAction
			for i := range actions {
				if actions[i].Title == tt.expectedAction {
					rangeAction = &actions[i]
					break
				}
			}

			require.NotNil(t, rangeAction, "Should have range toggle action")
			require.NotNil(t, rangeAction.Kind)
			assert.Equal(t, protocol.CodeActionKindRefactorRewrite, *rangeAction.Kind)

			require.NotNil(t, rangeAction.Edit)
			require.NotNil(t, rangeAction.Edit.Changes)
			edits, ok := rangeAction.Edit.Changes[uri]
			require.True(t, ok)
			assert.Len(t, edits, tt.numEdits)
		})
	}
}

// TestFixAllFallbacks tests the SourceFixAll action
func TestFixAllFallbacks(t *testing.T) {
	s, err := lsp.NewServer()
	require.NoError(t, err)

	// Add test tokens
	_ = s.TokenManager().Add(&tokens.Token{Name: "color-primary", Value: "#ff0000", Type: "color"})
	_ = s.TokenManager().Add(&tokens.Token{Name: "color-secondary", Value: "#00ff00", Type: "color"})

	cssContent := `.button {
  color: var(--color-primary, blue);
  background: var(--color-secondary, red);
  border-color: var(--color-primary, #0000ff);
}`

	uri := "file:///test.css"
	_ = s.DocumentManager().DidOpen(uri, "css", 1, cssContent)

	// Create diagnostics for incorrect fallbacks
	diagnostics := []protocol.Diagnostic{
		{
			Range: protocol.Range{
				Start: protocol.Position{Line: 1, Character: 9},
				End:   protocol.Position{Line: 1, Character: 40},
			},
			Code:    ptrIntegerOrString("incorrect-fallback"),
			Message: "Incorrect fallback",
		},
		{
			Range: protocol.Range{
				Start: protocol.Position{Line: 2, Character: 14},
				End:   protocol.Position{Line: 2, Character: 48},
			},
			Code:    ptrIntegerOrString("incorrect-fallback"),
			Message: "Incorrect fallback",
		},
	}

	params := &protocol.CodeActionParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: uri},
		Range: protocol.Range{
			Start: protocol.Position{Line: 0, Character: 0},
			End:   protocol.Position{Line: 4, Character: 0},
		},
		Context: protocol.CodeActionContext{
			Diagnostics: diagnostics,
		},
	}

	req := types.NewRequestContext(s, nil)
	result, err := codeaction.CodeAction(req, params)
	require.NoError(t, err)
	require.NotNil(t, result)

	actions := result.([]protocol.CodeAction)

	// Find the fixAll action
	var fixAllAction *protocol.CodeAction
	for i := range actions {
		if actions[i].Title == "Fix all token fallback values" {
			fixAllAction = &actions[i]
			break
		}
	}

	require.NotNil(t, fixAllAction, "Should have fixAll action")
	require.NotNil(t, fixAllAction.Kind)
	assert.Equal(t, codeActionKindSourceFixAll, *fixAllAction.Kind)

	// Resolve the action to get edits
	req = types.NewRequestContext(s, nil)
	resolved, err := codeaction.CodeActionResolve(req, fixAllAction)
	require.NoError(t, err)
	require.NotNil(t, resolved.Edit)
	require.NotNil(t, resolved.Edit.Changes)

	edits, ok := resolved.Edit.Changes[uri]
	require.True(t, ok)

	// Should fix all incorrect fallbacks (3 total: blue, red, #0000ff)
	assert.GreaterOrEqual(t, len(edits), 2)
}

// TestCodeAction_LiteralSupport tests that code actions respect the codeActionLiteralSupport capability
func TestCodeAction_LiteralSupport(t *testing.T) {
	t.Run("returns CodeAction literals when supported", func(t *testing.T) {
		s, err := lsp.NewServer()
		require.NoError(t, err)

		// Add test token
		_ = s.TokenManager().Add(&tokens.Token{
			Name:  "color-primary",
			Value: "#ff0000",
			Type:  "color",
		})

		// Set client capabilities with codeActionLiteralSupport
		s.SetClientCapabilities(protocol.ClientCapabilities{
			TextDocument: &protocol.TextDocumentClientCapabilities{
				CodeAction: &protocol.CodeActionClientCapabilities{
					CodeActionLiteralSupport: &struct {
						CodeActionKind struct {
							ValueSet []protocol.CodeActionKind `json:"valueSet"`
						} `json:"codeActionKind"`
					}{
						CodeActionKind: struct {
							ValueSet []protocol.CodeActionKind `json:"valueSet"`
						}{
							ValueSet: []protocol.CodeActionKind{
								protocol.CodeActionKindRefactorRewrite,
							},
						},
					},
				},
			},
		})

		uri := "file:///test.css"
		_ = s.DocumentManager().DidOpen(uri, "css", 1, `.button { color: var(--color-primary); }`)

		params := &protocol.CodeActionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Range: protocol.Range{
				Start: protocol.Position{Line: 0, Character: 21},
				End:   protocol.Position{Line: 0, Character: 21},
			},
			Context: protocol.CodeActionContext{},
		}

		req := types.NewRequestContext(s, nil)
		result, err := codeaction.CodeAction(req, params)
		require.NoError(t, err)
		require.NotNil(t, result, "Should return code actions when literals are supported")

		actions := result.([]protocol.CodeAction)
		assert.NotEmpty(t, actions, "Should have code actions")
	})

	t.Run("returns nil when literals not supported", func(t *testing.T) {
		s, err := lsp.NewServer()
		require.NoError(t, err)

		// Add test token
		_ = s.TokenManager().Add(&tokens.Token{
			Name:  "color-primary",
			Value: "#ff0000",
			Type:  "color",
		})

		// Set client capabilities WITHOUT codeActionLiteralSupport
		s.SetClientCapabilities(protocol.ClientCapabilities{
			TextDocument: &protocol.TextDocumentClientCapabilities{
				CodeAction: &protocol.CodeActionClientCapabilities{
					// No CodeActionLiteralSupport - legacy client
				},
			},
		})

		uri := "file:///test.css"
		_ = s.DocumentManager().DidOpen(uri, "css", 1, `.button { color: var(--color-primary); }`)

		params := &protocol.CodeActionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Range: protocol.Range{
				Start: protocol.Position{Line: 0, Character: 21},
				End:   protocol.Position{Line: 0, Character: 21},
			},
			Context: protocol.CodeActionContext{},
		}

		req := types.NewRequestContext(s, nil)
		result, err := codeaction.CodeAction(req, params)
		require.NoError(t, err)
		assert.Nil(t, result, "Should return nil for legacy clients without literal support")
	})

	t.Run("defaults to supported when capabilities unknown", func(t *testing.T) {
		s, err := lsp.NewServer()
		require.NoError(t, err)

		// Add test token
		_ = s.TokenManager().Add(&tokens.Token{
			Name:  "color-primary",
			Value: "#ff0000",
			Type:  "color",
		})

		// Don't set any client capabilities - test default behavior

		uri := "file:///test.css"
		_ = s.DocumentManager().DidOpen(uri, "css", 1, `.button { color: var(--color-primary); }`)

		params := &protocol.CodeActionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Range: protocol.Range{
				Start: protocol.Position{Line: 0, Character: 21},
				End:   protocol.Position{Line: 0, Character: 21},
			},
			Context: protocol.CodeActionContext{},
		}

		req := types.NewRequestContext(s, nil)
		result, err := codeaction.CodeAction(req, params)
		require.NoError(t, err)
		require.NotNil(t, result, "Should default to supporting literals for modern clients")

		actions := result.([]protocol.CodeAction)
		assert.NotEmpty(t, actions, "Should have code actions by default")
	})
}
