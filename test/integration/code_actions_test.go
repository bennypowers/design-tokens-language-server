package integration_test

import (
	"testing"

	codeaction "bennypowers.dev/dtls/lsp/methods/textDocument/codeAction"
	"bennypowers.dev/dtls/lsp/methods/textDocument/diagnostic"
	"bennypowers.dev/dtls/lsp/types"
	"bennypowers.dev/dtls/test/integration/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// TestCodeActionFixIncorrectFallback tests code action for fixing incorrect fallback
func TestCodeActionFixIncorrectFallback(t *testing.T) {
	server := testutil.NewTestServer(t)
	testutil.LoadBasicTokens(t, server)
	testutil.OpenCSSFixture(t, server, "file:///test.css", "incorrect-fallback.css")

	// Get diagnostics first
	diagnostics, err := diagnostic.GetDiagnostics(server, "file:///test.css")
	require.NoError(t, err)
	require.Len(t, diagnostics, 1)

	// Request code actions
	req := types.NewRequestContext(server, nil)
	result, err := codeaction.CodeAction(req, &protocol.CodeActionParams{
		TextDocument: protocol.TextDocumentIdentifier{
			URI: "file:///test.css",
		},
		Range: diagnostics[0].Range,
		Context: protocol.CodeActionContext{
			Diagnostics: diagnostics,
		},
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	actions, ok := result.([]protocol.CodeAction)
	require.True(t, ok)

	// Should have at least one action to fix the fallback
	assert.GreaterOrEqual(t, len(actions), 1)

	// Find the fix fallback action
	var fixAction *protocol.CodeAction
	for i := range actions {
		if actions[i].Title == "Fix fallback value to '#0000ff'" {
			fixAction = &actions[i]
			break
		}
	}

	require.NotNil(t, fixAction, "Should have 'Fix fallback value' action")

	// Check that it's a quick fix
	assert.NotNil(t, fixAction.Kind)
	assert.Equal(t, protocol.CodeActionKindQuickFix, *fixAction.Kind)

	// Check that it's marked as preferred
	assert.NotNil(t, fixAction.IsPreferred)
	assert.True(t, *fixAction.IsPreferred)

	// Check that it has the diagnostic
	assert.Len(t, fixAction.Diagnostics, 1)

	// Check that it has the edit
	require.NotNil(t, fixAction.Edit)
	require.NotNil(t, fixAction.Edit.Changes)

	edits := fixAction.Edit.Changes["file:///test.css"]
	require.Len(t, edits, 1)

	// Should replace with correct fallback
	assert.Contains(t, edits[0].NewText, "var(--color-primary, #0000ff)")
}

// TestCodeActionAddFallback tests code action for adding fallback
func TestCodeActionAddFallback(t *testing.T) {
	server := testutil.NewTestServer(t)
	testutil.LoadBasicTokens(t, server)
	testutil.OpenCSSFixture(t, server, "file:///test.css", "basic-var-calls.css")

	// Request code actions at a var call without fallback
	req := types.NewRequestContext(server, nil)
	result, err := codeaction.CodeAction(req, &protocol.CodeActionParams{
		TextDocument: protocol.TextDocumentIdentifier{
			URI: "file:///test.css",
		},
		Range: protocol.Range{
			Start: protocol.Position{Line: 2, Character: 9},
			End:   protocol.Position{Line: 2, Character: 33},
		},
		Context: protocol.CodeActionContext{
			Diagnostics: []protocol.Diagnostic{},
		},
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	actions, ok := result.([]protocol.CodeAction)
	require.True(t, ok)

	// Should have action to add fallback
	assert.GreaterOrEqual(t, len(actions), 1)

	// Find the add fallback action
	var addAction *protocol.CodeAction
	for i := range actions {
		if actions[i].Title == "Add fallback value '#0000ff'" {
			addAction = &actions[i]
			break
		}
	}

	require.NotNil(t, addAction, "Should have 'Add fallback value' action")

	// Check that it has the edit
	require.NotNil(t, addAction.Edit)
	require.NotNil(t, addAction.Edit.Changes)

	edits := addAction.Edit.Changes["file:///test.css"]
	require.Len(t, edits, 1)

	// Should add fallback
	assert.Contains(t, edits[0].NewText, "var(--color-primary, #0000ff)")
}

// TestCodeActionDeprecatedToken tests code action for deprecated token
func TestCodeActionDeprecatedToken(t *testing.T) {
	server := testutil.NewTestServer(t)
	testutil.LoadBasicTokens(t, server)
	testutil.OpenCSSFixture(t, server, "file:///test.css", "deprecated-token.css")

	// Get diagnostics first
	diagnostics, err := diagnostic.GetDiagnostics(server, "file:///test.css")
	require.NoError(t, err)
	require.Len(t, diagnostics, 1)

	// Request code actions
	req := types.NewRequestContext(server, nil)
	result, err := codeaction.CodeAction(req, &protocol.CodeActionParams{
		TextDocument: protocol.TextDocumentIdentifier{
			URI: "file:///test.css",
		},
		Range: diagnostics[0].Range,
		Context: protocol.CodeActionContext{
			Diagnostics: diagnostics,
		},
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	actions, ok := result.([]protocol.CodeAction)
	require.True(t, ok)

	// Should have at least 2 actions: replace with recommended + replace with literal
	assert.GreaterOrEqual(t, len(actions), 2)

	// Find the replace action
	var replaceAction *protocol.CodeAction
	var literalAction *protocol.CodeAction
	for i := range actions {
		if actions[i].Title == "Replace with '--color-primary'" {
			replaceAction = &actions[i]
		}
		if actions[i].Title == "Replace with literal value '#ff0000'" {
			literalAction = &actions[i]
		}
	}

	require.NotNil(t, replaceAction, "Should have 'Replace with recommended' action")
	require.NotNil(t, literalAction, "Should have 'Replace with literal' action")

	// Check the replace action
	assert.NotNil(t, replaceAction.Kind)
	assert.Equal(t, protocol.CodeActionKindQuickFix, *replaceAction.Kind)
	assert.NotNil(t, replaceAction.IsPreferred)
	assert.True(t, *replaceAction.IsPreferred)

	// Check that replace action has the edit
	require.NotNil(t, replaceAction.Edit)
	require.NotNil(t, replaceAction.Edit.Changes)

	edits := replaceAction.Edit.Changes["file:///test.css"]
	require.Len(t, edits, 1)
	assert.Contains(t, edits[0].NewText, "var(--color-primary)")

	// Check the literal action
	require.NotNil(t, literalAction.Edit)
	literalEdits := literalAction.Edit.Changes["file:///test.css"]
	require.Len(t, literalEdits, 1)
	assert.Equal(t, "#ff0000", literalEdits[0].NewText)
}

// TestCodeActionNoActions tests that no actions are returned when not applicable
func TestCodeActionNoActions(t *testing.T) {
	server := testutil.NewTestServer(t)
	testutil.LoadBasicTokens(t, server)
	testutil.OpenCSSFixture(t, server, "file:///test.css", "correct-fallback.css")

	// Request code actions
	req := types.NewRequestContext(server, nil)
	result, err := codeaction.CodeAction(req, &protocol.CodeActionParams{
		TextDocument: protocol.TextDocumentIdentifier{
			URI: "file:///test.css",
		},
		Range: protocol.Range{
			Start: protocol.Position{Line: 2, Character: 0},
			End:   protocol.Position{Line: 2, Character: 50},
		},
		Context: protocol.CodeActionContext{
			Diagnostics: []protocol.Diagnostic{},
		},
	})

	require.NoError(t, err)

	// Should have no actions (or only add fallback suggestions)
	// since the fallback is already correct
	var actions []protocol.CodeAction
	if result != nil {
		var ok bool
		actions, ok = result.([]protocol.CodeAction)
		require.True(t, ok)
	}

	if len(actions) > 0 {
		for _, action := range actions {
			// Should not have any "Fix" actions, only suggestions
			assert.NotContains(t, action.Title, "Fix")
		}
	}
}

// TestCodeActionResolve tests code action resolve
func TestCodeActionResolve(t *testing.T) {
	server := testutil.NewTestServer(t)

	kind := protocol.CodeActionKindQuickFix
	action := protocol.CodeAction{
		Title: "Test action",
		Kind:  &kind,
		Edit: &protocol.WorkspaceEdit{
			Changes: map[string][]protocol.TextEdit{
				"file:///test.css": {
					{
						Range: protocol.Range{
							Start: protocol.Position{Line: 0, Character: 0},
							End:   protocol.Position{Line: 0, Character: 10},
						},
						NewText: "test",
					},
				},
			},
		},
	}

	// Resolve the action
	req := types.NewRequestContext(server, nil)
	resolved, err := codeaction.CodeActionResolve(req, &action)
	require.NoError(t, err)
	require.NotNil(t, resolved)

	// For now, resolve just returns the same action
	assert.Equal(t, action.Title, resolved.Title)
	assert.NotNil(t, resolved.Edit)
}

// TestCodeAction_CompositeTypes tests that composite types (border, shadow) don't offer toggle fallback
func TestCodeAction_CompositeTypes(t *testing.T) {
	server := testutil.NewTestServer(t)
	testutil.LoadBasicTokens(t, server)
	testutil.OpenCSSFixture(t, server, "file:///test.css", "composite-types.css")

	// Request code actions at border var call (line 3, character 10)
	req := types.NewRequestContext(server, nil)
	result, err := codeaction.CodeAction(req, &protocol.CodeActionParams{
		TextDocument: protocol.TextDocumentIdentifier{
			URI: "file:///test.css",
		},
		Range: protocol.Range{
			Start: protocol.Position{Line: 3, Character: 10},
			End:   protocol.Position{Line: 3, Character: 10}, // collapsed cursor
		},
		Context: protocol.CodeActionContext{},
	})

	require.NoError(t, err)

	// Should not crash, but should not offer toggle action for composite types
	if result != nil {
		actions, ok := result.([]protocol.CodeAction)
		if ok {
			for _, action := range actions {
				assert.NotEqual(t, "Toggle design token fallback value", action.Title,
					"Should not offer toggle action for composite type (border)")
				assert.NotEqual(t, "Add fallback value '1px solid #000000'", action.Title,
					"Should not offer add fallback for unsafe composite type")
			}
		}
	}

	// Also test shadow var call (line 4, character 14)
	result, err = codeaction.CodeAction(req, &protocol.CodeActionParams{
		TextDocument: protocol.TextDocumentIdentifier{
			URI: "file:///test.css",
		},
		Range: protocol.Range{
			Start: protocol.Position{Line: 4, Character: 14},
			End:   protocol.Position{Line: 4, Character: 14},
		},
		Context: protocol.CodeActionContext{},
	})

	require.NoError(t, err)

	if result != nil {
		actions, ok := result.([]protocol.CodeAction)
		if ok {
			for _, action := range actions {
				assert.NotEqual(t, "Toggle design token fallback value", action.Title,
					"Should not offer toggle action for composite type (shadow)")
			}
		}
	}
}

// TestCodeAction_FontFamilyFallback tests that font-family fallbacks are properly quoted
func TestCodeAction_FontFamilyFallback(t *testing.T) {
	server := testutil.NewTestServer(t)
	testutil.LoadBasicTokens(t, server)
	testutil.OpenCSSFixture(t, server, "file:///test.css", "font-family-quoting.css")

	// Request code actions at body font (line 4, character 17) - has spaces, should be quoted
	req := types.NewRequestContext(server, nil)
	result, err := codeaction.CodeAction(req, &protocol.CodeActionParams{
		TextDocument: protocol.TextDocumentIdentifier{
			URI: "file:///test.css",
		},
		Range: protocol.Range{
			Start: protocol.Position{Line: 4, Character: 17},
			End:   protocol.Position{Line: 4, Character: 17},
		},
		Context: protocol.CodeActionContext{},
	})

	require.NoError(t, err)
	require.NotNil(t, result)

	actions, ok := result.([]protocol.CodeAction)
	require.True(t, ok)

	// Find toggle or add fallback action
	var toggleAction *protocol.CodeAction
	for i := range actions {
		if actions[i].Title == "Toggle design token fallback value" ||
			actions[i].Title == "Add fallback value '\"Helvetica Neue\"'" {
			toggleAction = &actions[i]
			break
		}
	}

	// Should offer some action for fontFamily token
	require.NotNil(t, toggleAction, "Should offer action for font-family token")

	// If it's an add action, check that the font name is quoted
	if toggleAction.Edit != nil && toggleAction.Edit.Changes != nil {
		edits := toggleAction.Edit.Changes["file:///test.css"]
		if len(edits) > 0 {
			// Should contain quoted font name
			assert.Contains(t, edits[0].NewText, "\"Helvetica Neue\"",
				"Font name with spaces should be quoted in fallback")
		}
	}

	// Test already quoted font (line 24)
	_, err = codeaction.CodeAction(req, &protocol.CodeActionParams{
		TextDocument: protocol.TextDocumentIdentifier{
			URI: "file:///test.css",
		},
		Range: protocol.Range{
			Start: protocol.Position{Line: 24, Character: 17},
			End:   protocol.Position{Line: 24, Character: 17},
		},
		Context: protocol.CodeActionContext{},
	})
	require.NoError(t, err)

	// Test comma-separated font list (line 29)
	_, err = codeaction.CodeAction(req, &protocol.CodeActionParams{
		TextDocument: protocol.TextDocumentIdentifier{
			URI: "file:///test.css",
		},
		Range: protocol.Range{
			Start: protocol.Position{Line: 29, Character: 17},
			End:   protocol.Position{Line: 29, Character: 17},
		},
		Context: protocol.CodeActionContext{},
	})
	require.NoError(t, err)

	// Test quoted list (line 34)
	_, err = codeaction.CodeAction(req, &protocol.CodeActionParams{
		TextDocument: protocol.TextDocumentIdentifier{
			URI: "file:///test.css",
		},
		Range: protocol.Range{
			Start: protocol.Position{Line: 34, Character: 17},
			End:   protocol.Position{Line: 34, Character: 17},
		},
		Context: protocol.CodeActionContext{},
	})
	require.NoError(t, err)
}

// TestCodeAction_DeprecatedMessagePatterns tests extraction of token names from deprecation messages
func TestCodeAction_DeprecatedMessagePatterns(t *testing.T) {
	server := testutil.NewTestServer(t)
	testutil.LoadBasicTokens(t, server)
	testutil.OpenCSSFixture(t, server, "file:///test.css", "deprecated-patterns.css")

	// Test "Use X instead" pattern (line 4, character 9)
	req := types.NewRequestContext(server, nil)
	result, err := codeaction.CodeAction(req, &protocol.CodeActionParams{
		TextDocument: protocol.TextDocumentIdentifier{
			URI: "file:///test.css",
		},
		Range: protocol.Range{
			Start: protocol.Position{Line: 4, Character: 9},
			End:   protocol.Position{Line: 4, Character: 35},
		},
		Context: protocol.CodeActionContext{},
	})

	require.NoError(t, err)
	require.NotNil(t, result)

	actions, ok := result.([]protocol.CodeAction)
	require.True(t, ok)

	// Should offer replacement with --color-primary
	var replacementAction *protocol.CodeAction
	for i := range actions {
		if actions[i].Title == "Replace with '--color-primary'" {
			replacementAction = &actions[i]
			break
		}
	}

	require.NotNil(t, replacementAction, "Should extract 'color.primary' from 'Use color.primary instead'")

	// Test "Replaced by X for better consistency" pattern (line 9, character 10)
	result, err = codeaction.CodeAction(req, &protocol.CodeActionParams{
		TextDocument: protocol.TextDocumentIdentifier{
			URI: "file:///test.css",
		},
		Range: protocol.Range{
			Start: protocol.Position{Line: 9, Character: 10},
			End:   protocol.Position{Line: 9, Character: 40},
		},
		Context: protocol.CodeActionContext{},
	})

	require.NoError(t, err)
	require.NotNil(t, result)

	actions, ok = result.([]protocol.CodeAction)
	require.True(t, ok)

	// Should offer replacement with --spacing-small (extracted from "Replaced by spacing.small for better consistency")
	replacementAction = nil
	for i := range actions {
		if actions[i].Title == "Replace with '--spacing-small'" {
			replacementAction = &actions[i]
			break
		}
	}

	require.NotNil(t, replacementAction, "Should extract 'spacing.small' from 'Replaced by spacing.small for better consistency'")

	// Test no suggestion pattern (line 14, character 13) - should only offer literal value
	result, err = codeaction.CodeAction(req, &protocol.CodeActionParams{
		TextDocument: protocol.TextDocumentIdentifier{
			URI: "file:///test.css",
		},
		Range: protocol.Range{
			Start: protocol.Position{Line: 14, Character: 13},
			End:   protocol.Position{Line: 14, Character: 40},
		},
		Context: protocol.CodeActionContext{},
	})

	require.NoError(t, err)
	require.NotNil(t, result)

	actions, ok = result.([]protocol.CodeAction)
	require.True(t, ok)

	// Should offer literal value but NOT a token replacement
	var literalAction *protocol.CodeAction
	hasReplacementAction := false
	for i := range actions {
		if actions[i].Title == "Replace with literal value '24px'" {
			literalAction = &actions[i]
		}
		if actions[i].Title == "Replace with '--" {
			hasReplacementAction = true
		}
	}

	require.NotNil(t, literalAction, "Should offer literal value for deprecated token")
	assert.False(t, hasReplacementAction, "Should not offer token replacement when no suggestion in message")

	// Test deprecated token with fallback (line 24, character 9) - replacement should preserve fallback
	result, err = codeaction.CodeAction(req, &protocol.CodeActionParams{
		TextDocument: protocol.TextDocumentIdentifier{
			URI: "file:///test.css",
		},
		Range: protocol.Range{
			Start: protocol.Position{Line: 24, Character: 9},
			End:   protocol.Position{Line: 24, Character: 48},
		},
		Context: protocol.CodeActionContext{},
	})

	require.NoError(t, err)
	require.NotNil(t, result)

	actions, ok = result.([]protocol.CodeAction)
	require.True(t, ok)

	// Should offer replacement with --color-primary AND preserve the fallback
	var replacementWithFallback *protocol.CodeAction
	for i := range actions {
		if actions[i].Title == "Replace with '--color-primary'" {
			replacementWithFallback = &actions[i]
			break
		}
	}

	require.NotNil(t, replacementWithFallback, "Should offer replacement for deprecated token with fallback")

	// Check that the replacement includes fallback
	if replacementWithFallback.Edit != nil && replacementWithFallback.Edit.Changes != nil {
		edits := replacementWithFallback.Edit.Changes["file:///test.css"]
		if len(edits) > 0 {
			// Should contain both the new var and a fallback
			assert.Contains(t, edits[0].NewText, "var(--color-primary, ",
				"Replacement should include fallback value")
		}
	}
}

// TestCodeAction_FallbackTypes tests add/fix fallback actions for various token types
func TestCodeAction_FallbackTypes(t *testing.T) {
	server := testutil.NewTestServer(t)
	testutil.LoadBasicTokens(t, server)
	testutil.OpenCSSFixture(t, server, "file:///test.css", "fallback-types.css")

	req := types.NewRequestContext(server, nil)

	// Test color without fallback (line 4, character 15) - should suggest adding
	result, err := codeaction.CodeAction(req, &protocol.CodeActionParams{
		TextDocument: protocol.TextDocumentIdentifier{
			URI: "file:///test.css",
		},
		Range: protocol.Range{
			Start: protocol.Position{Line: 4, Character: 15},
			End:   protocol.Position{Line: 4, Character: 15},
		},
		Context: protocol.CodeActionContext{},
	})

	require.NoError(t, err)
	require.NotNil(t, result)

	actions, ok := result.([]protocol.CodeAction)
	require.True(t, ok)

	// Should have add fallback action for color
	var addAction *protocol.CodeAction
	for i := range actions {
		if actions[i].Title == "Add fallback value '#0000ff'" {
			addAction = &actions[i]
			break
		}
	}
	require.NotNil(t, addAction, "Should offer 'Add fallback' for color token")

	// Test dimension without fallback (line 7, character 16) - should suggest adding
	result, err = codeaction.CodeAction(req, &protocol.CodeActionParams{
		TextDocument: protocol.TextDocumentIdentifier{
			URI: "file:///test.css",
		},
		Range: protocol.Range{
			Start: protocol.Position{Line: 7, Character: 16},
			End:   protocol.Position{Line: 7, Character: 16},
		},
		Context: protocol.CodeActionContext{},
	})

	require.NoError(t, err)
	require.NotNil(t, result)

	actions, ok = result.([]protocol.CodeAction)
	require.True(t, ok)

	// Should have add fallback action for dimension
	addAction = nil
	for i := range actions {
		if actions[i].Title == "Add fallback value '8px'" {
			addAction = &actions[i]
			break
		}
	}
	require.NotNil(t, addAction, "Should offer 'Add fallback' for dimension token")

	// Test incorrect fallback (line 18, character 9) - should suggest fixing
	// First get diagnostics
	diagnostics, err := diagnostic.GetDiagnostics(server, "file:///test.css")
	require.NoError(t, err)

	// Find diagnostic for incorrect fallback
	var incorrectDiag *protocol.Diagnostic
	for i := range diagnostics {
		if diagnostics[i].Range.Start.Line == 18 {
			incorrectDiag = &diagnostics[i]
			break
		}
	}

	if incorrectDiag != nil {
		result, err = codeaction.CodeAction(req, &protocol.CodeActionParams{
			TextDocument: protocol.TextDocumentIdentifier{
				URI: "file:///test.css",
			},
			Range: incorrectDiag.Range,
			Context: protocol.CodeActionContext{
				Diagnostics: diagnostics,
			},
		})

		require.NoError(t, err)
		require.NotNil(t, result)

		actions, ok = result.([]protocol.CodeAction)
		require.True(t, ok)

		// Should have fix fallback action
		var fixAction *protocol.CodeAction
		for i := range actions {
			if actions[i].Title == "Fix fallback value to '#0000ff'" {
				fixAction = &actions[i]
				break
			}
		}
		require.NotNil(t, fixAction, "Should offer 'Fix fallback' for incorrect color fallback")
	}
}

// TestCodeAction_TokenTypeVariations tests add fallback for various token types
func TestCodeAction_TokenTypeVariations(t *testing.T) {
	server := testutil.NewTestServer(t)
	testutil.LoadBasicTokens(t, server)
	testutil.OpenCSSFixture(t, server, "file:///test.css", "token-type-variations.css")

	req := types.NewRequestContext(server, nil)

	// Test rgb color (line 4, character 12)
	result, err := codeaction.CodeAction(req, &protocol.CodeActionParams{
		TextDocument: protocol.TextDocumentIdentifier{
			URI: "file:///test.css",
		},
		Range: protocol.Range{
			Start: protocol.Position{Line: 4, Character: 12},
			End:   protocol.Position{Line: 4, Character: 12},
		},
		Context: protocol.CodeActionContext{},
	})
	require.NoError(t, err)
	require.NotNil(t, result)

	// Test rem dimension (line 11, character 16)
	result, err = codeaction.CodeAction(req, &protocol.CodeActionParams{
		TextDocument: protocol.TextDocumentIdentifier{
			URI: "file:///test.css",
		},
		Range: protocol.Range{
			Start: protocol.Position{Line: 11, Character: 16},
			End:   protocol.Position{Line: 11, Character: 16},
		},
		Context: protocol.CodeActionContext{},
	})
	require.NoError(t, err)
	require.NotNil(t, result)

	// Test number opacity (line 18, character 14)
	result, err = codeaction.CodeAction(req, &protocol.CodeActionParams{
		TextDocument: protocol.TextDocumentIdentifier{
			URI: "file:///test.css",
		},
		Range: protocol.Range{
			Start: protocol.Position{Line: 18, Character: 14},
			End:   protocol.Position{Line: 18, Character: 14},
		},
		Context: protocol.CodeActionContext{},
	})
	require.NoError(t, err)
	require.NotNil(t, result)

	// Test font-weight keyword (line 24, character: 18)
	result, err = codeaction.CodeAction(req, &protocol.CodeActionParams{
		TextDocument: protocol.TextDocumentIdentifier{
			URI: "file:///test.css",
		},
		Range: protocol.Range{
			Start: protocol.Position{Line: 24, Character: 18},
			End:   protocol.Position{Line: 24, Character: 18},
		},
		Context: protocol.CodeActionContext{},
	})
	require.NoError(t, err)
	require.NotNil(t, result)
}
