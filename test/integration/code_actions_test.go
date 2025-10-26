package integration_test

import (
	"testing"

	codeaction "github.com/bennypowers/design-tokens-language-server/lsp/methods/textDocument/codeAction"
	"github.com/bennypowers/design-tokens-language-server/test/integration/testutil"
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
	diagnostics, err := server.GetDiagnostics("file:///test.css")
	require.NoError(t, err)
	require.Len(t, diagnostics, 1)

	// Request code actions
	result, err := codeaction.CodeAction(server, nil, &protocol.CodeActionParams{
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
	result, err := codeaction.CodeAction(server, nil, &protocol.CodeActionParams{
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
	diagnostics, err := server.GetDiagnostics("file:///test.css")
	require.NoError(t, err)
	require.Len(t, diagnostics, 1)

	// Request code actions
	result, err := codeaction.CodeAction(server, nil, &protocol.CodeActionParams{
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
	result, err := codeaction.CodeAction(server, nil, &protocol.CodeActionParams{
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
	resolved, err := codeaction.CodeActionResolve(server, nil, &action)
	require.NoError(t, err)
	require.NotNil(t, resolved)

	// For now, resolve just returns the same action
	assert.Equal(t, action.Title, resolved.Title)
	assert.NotNil(t, resolved.Edit)
}
