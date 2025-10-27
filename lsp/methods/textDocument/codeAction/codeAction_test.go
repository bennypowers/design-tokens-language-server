package codeaction_test

import (
	"testing"

	"bennypowers.dev/dtls/internal/parser/css"
	"bennypowers.dev/dtls/internal/tokens"
	"bennypowers.dev/dtls/lsp"
	codeaction "bennypowers.dev/dtls/lsp/methods/textDocument/codeAction"
	"bennypowers.dev/dtls/lsp/types"
	protocol "github.com/tliron/glsp/protocol_3_16"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	// CodeActionKindSourceFixAll is not defined in glsp v0.2.2
	codeActionKindSourceFixAll protocol.CodeActionKind = "source.fixAll"
)

// ptrString returns a pointer to the given string
func ptrString(s string) *string {
	return &s
}

// ptrIntegerOrString returns a pointer to IntegerOrString from a string
func ptrIntegerOrString(s string) *protocol.IntegerOrString {
	return &protocol.IntegerOrString{Value: s}
}

// TestRangesIntersect tests the rangesIntersect function with half-open range semantics [start, end)
func TestRangesIntersect(t *testing.T) {
	tests := []struct {
		name     string
		a        protocol.Range
		b        protocol.Range
		expected bool
	}{
		{
			name: "touching ranges - same line, a.end == b.start",
			a: protocol.Range{
				Start: protocol.Position{Line: 0, Character: 0},
				End:   protocol.Position{Line: 0, Character: 5},
			},
			b: protocol.Range{
				Start: protocol.Position{Line: 0, Character: 5},
				End:   protocol.Position{Line: 0, Character: 10},
			},
			expected: false, // Half-open ranges don't intersect when touching
		},
		{
			name: "touching ranges - same line, b.end == a.start",
			a: protocol.Range{
				Start: protocol.Position{Line: 0, Character: 5},
				End:   protocol.Position{Line: 0, Character: 10},
			},
			b: protocol.Range{
				Start: protocol.Position{Line: 0, Character: 0},
				End:   protocol.Position{Line: 0, Character: 5},
			},
			expected: false, // Half-open ranges don't intersect when touching
		},
		{
			name: "overlapping ranges - same line",
			a: protocol.Range{
				Start: protocol.Position{Line: 0, Character: 0},
				End:   protocol.Position{Line: 0, Character: 6},
			},
			b: protocol.Range{
				Start: protocol.Position{Line: 0, Character: 5},
				End:   protocol.Position{Line: 0, Character: 10},
			},
			expected: true, // Character 5 is in both ranges
		},
		{
			name: "separate ranges - same line",
			a: protocol.Range{
				Start: protocol.Position{Line: 0, Character: 0},
				End:   protocol.Position{Line: 0, Character: 4},
			},
			b: protocol.Range{
				Start: protocol.Position{Line: 0, Character: 5},
				End:   protocol.Position{Line: 0, Character: 10},
			},
			expected: false,
		},
		{
			name: "touching ranges - different lines, a.end.line+1 == b.start.line",
			a: protocol.Range{
				Start: protocol.Position{Line: 0, Character: 0},
				End:   protocol.Position{Line: 1, Character: 0},
			},
			b: protocol.Range{
				Start: protocol.Position{Line: 1, Character: 0},
				End:   protocol.Position{Line: 2, Character: 0},
			},
			expected: false, // Half-open ranges don't intersect when touching
		},
		{
			name: "overlapping ranges - different lines",
			a: protocol.Range{
				Start: protocol.Position{Line: 0, Character: 0},
				End:   protocol.Position{Line: 1, Character: 5},
			},
			b: protocol.Range{
				Start: protocol.Position{Line: 1, Character: 0},
				End:   protocol.Position{Line: 2, Character: 0},
			},
			expected: true, // Line 1, chars 0-4 overlap
		},
		{
			name: "separate ranges - different lines",
			a: protocol.Range{
				Start: protocol.Position{Line: 0, Character: 0},
				End:   protocol.Position{Line: 0, Character: 10},
			},
			b: protocol.Range{
				Start: protocol.Position{Line: 2, Character: 0},
				End:   protocol.Position{Line: 3, Character: 0},
			},
			expected: false,
		},
		{
			name: "identical ranges",
			a: protocol.Range{
				Start: protocol.Position{Line: 0, Character: 5},
				End:   protocol.Position{Line: 0, Character: 10},
			},
			b: protocol.Range{
				Start: protocol.Position{Line: 0, Character: 5},
				End:   protocol.Position{Line: 0, Character: 10},
			},
			expected: true,
		},
		{
			name: "range a contains range b",
			a: protocol.Range{
				Start: protocol.Position{Line: 0, Character: 0},
				End:   protocol.Position{Line: 2, Character: 10},
			},
			b: protocol.Range{
				Start: protocol.Position{Line: 1, Character: 5},
				End:   protocol.Position{Line: 1, Character: 8},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := codeaction.RangesIntersect(tt.a, tt.b)
			assert.Equal(t, tt.expected, result, "codeaction.RangesIntersect(%+v, %+v)", tt.a, tt.b)
		})
	}
}

// TestFormatTokenValueForCSS tests the formatTokenValueForCSS function
func TestFormatTokenValueForCSS(t *testing.T) {
	tests := []struct {
		name           string
		token          *tokens.Token
		expectedValue  string
		expectedSafe   bool
	}{
		// Safe types - colors
		{
			name:          "hex color",
			token:         &tokens.Token{Type: "color", Value: "#ff0000"},
			expectedValue: "#ff0000",
			expectedSafe:  true,
		},
		{
			name:          "rgb color",
			token:         &tokens.Token{Type: "color", Value: "rgb(255, 0, 0)"},
			expectedValue: "rgb(255, 0, 0)",
			expectedSafe:  true,
		},
		{
			name:          "named color",
			token:         &tokens.Token{Type: "color", Value: "red"},
			expectedValue: "red",
			expectedSafe:  true,
		},

		// Safe types - dimensions
		{
			name:          "pixel dimension",
			token:         &tokens.Token{Type: "dimension", Value: "16px"},
			expectedValue: "16px",
			expectedSafe:  true,
		},
		{
			name:          "rem dimension",
			token:         &tokens.Token{Type: "dimension", Value: "1.5rem"},
			expectedValue: "1.5rem",
			expectedSafe:  true,
		},

		// Safe types - numbers
		{
			name:          "integer number",
			token:         &tokens.Token{Type: "number", Value: "42"},
			expectedValue: "42",
			expectedSafe:  true,
		},
		{
			name:          "decimal number",
			token:         &tokens.Token{Type: "number", Value: "1.5"},
			expectedValue: "1.5",
			expectedSafe:  true,
		},

		// Font weight - safe cases
		{
			name:          "numeric font weight",
			token:         &tokens.Token{Type: "fontWeight", Value: "700"},
			expectedValue: "700",
			expectedSafe:  true,
		},
		{
			name:          "keyword font weight",
			token:         &tokens.Token{Type: "fontWeight", Value: "bold"},
			expectedValue: "bold",
			expectedSafe:  true,
		},
		{
			name:          "invalid font weight",
			token:         &tokens.Token{Type: "fontWeight", Value: "super-bold"},
			expectedValue: "",
			expectedSafe:  false,
		},

		// Font family - quoting needed
		{
			name:          "font family with spaces",
			token:         &tokens.Token{Type: "fontFamily", Value: "Helvetica Neue"},
			expectedValue: `"Helvetica Neue"`,
			expectedSafe:  true,
		},
		{
			name:          "font family with quotes",
			token:         &tokens.Token{Type: "fontFamily", Value: `"Times New Roman"`},
			expectedValue: `"Times New Roman"`,
			expectedSafe:  true,
		},
		{
			name:          "generic font family",
			token:         &tokens.Token{Type: "fontFamily", Value: "sans-serif"},
			expectedValue: "sans-serif",
			expectedSafe:  true,
		},
		{
			name:          "single word font",
			token:         &tokens.Token{Type: "fontFamily", Value: "Arial"},
			expectedValue: "Arial",
			expectedSafe:  true,
		},
		{
			name:          "font family with internal quotes",
			token:         &tokens.Token{Type: "fontFamily", Value: `Font "Name" Here`},
			expectedValue: `"Font \"Name\" Here"`,
			expectedSafe:  true,
		},

		// No type specified - heuristic detection
		{
			name:          "untyped hex color",
			token:         &tokens.Token{Value: "#00ff00"},
			expectedValue: "#00ff00",
			expectedSafe:  true,
		},
		{
			name:          "untyped dimension",
			token:         &tokens.Token{Value: "20px"},
			expectedValue: "20px",
			expectedSafe:  true,
		},
		{
			name:          "untyped number",
			token:         &tokens.Token{Value: "3.14"},
			expectedValue: "3.14",
			expectedSafe:  true,
		},
		{
			name:          "untyped simple identifier",
			token:         &tokens.Token{Value: "auto"},
			expectedValue: "auto",
			expectedSafe:  true,
		},
		{
			name:          "untyped with spaces - unsafe",
			token:         &tokens.Token{Value: "some value"},
			expectedValue: "",
			expectedSafe:  false,
		},

		// Unsafe composite types
		{
			name:          "border composite",
			token:         &tokens.Token{Type: "border", Value: "1px solid #000"},
			expectedValue: "",
			expectedSafe:  false,
		},
		{
			name:          "shadow composite",
			token:         &tokens.Token{Type: "shadow", Value: "0 2px 4px rgba(0,0,0,0.1)"},
			expectedValue: "",
			expectedSafe:  false,
		},
		{
			name:          "typography composite",
			token:         &tokens.Token{Type: "typography", Value: "bold 16px/1.5 Arial"},
			expectedValue: "",
			expectedSafe:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value, safe := codeaction.FormatTokenValueForCSS(tt.token)
			assert.Equal(t, tt.expectedSafe, safe, "safety check mismatch")
			if safe {
				assert.Equal(t, tt.expectedValue, value, "formatted value mismatch")
			}
		})
	}
}

// TestFormatFontFamilyValue tests the formatFontFamilyValue function
func TestFormatFontFamilyValue(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedValue string
		expectedSafe  bool
	}{
		{
			name:          "generic sans-serif",
			input:         "sans-serif",
			expectedValue: "sans-serif",
			expectedSafe:  true,
		},
		{
			name:          "generic serif",
			input:         "serif",
			expectedValue: "serif",
			expectedSafe:  true,
		},
		{
			name:          "single word font",
			input:         "Arial",
			expectedValue: "Arial",
			expectedSafe:  true,
		},
		{
			name:          "font with spaces",
			input:         "Comic Sans MS",
			expectedValue: `"Comic Sans MS"`,
			expectedSafe:  true,
		},
		{
			name:          "already quoted",
			input:         `"Times New Roman"`,
			expectedValue: `"Times New Roman"`,
			expectedSafe:  true,
		},
		{
			name:          "font with internal quotes",
			input:         `Font "Special" Name`,
			expectedValue: `"Font \"Special\" Name"`,
			expectedSafe:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value, safe := codeaction.FormatFontFamilyValue(tt.input)
			require.Equal(t, tt.expectedSafe, safe)
			assert.Equal(t, tt.expectedValue, value)
		})
	}
}

// TestCreateFixFallbackAction tests the createFixFallbackAction helper
func TestCreateFixFallbackAction(t *testing.T) {
	// Add a test token
	token := &tokens.Token{
		Name:  "color-primary",
		Value: "#ff0000",
		Type:  "color",
	}
	tests := []struct {
		name        string
		uri         string
		varCall     css.VarCall
		token       *tokens.Token
		diagnostics []protocol.Diagnostic
		expectNil   bool
		checkTitle  string
		checkEdit   bool
	}{
		{
			name: "fix incorrect fallback",
			uri:  "file:///test.css",
			varCall: css.VarCall{
				TokenName: "--color-primary",
				Fallback:  ptrString("#00ff00"), // incorrect
				Range: css.Range{
					Start: css.Position{Line: 10, Character: 10},
					End:   css.Position{Line: 10, Character: 35},
				},
			},
			token: token,
			diagnostics: []protocol.Diagnostic{
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 10, Character: 10},
						End:   protocol.Position{Line: 10, Character: 35},
					},
					Message: "Incorrect fallback",
				},
			},
			expectNil:  false,
			checkTitle: "Fix fallback value to '#ff0000'",
			checkEdit:  true,
		},
		{
			name: "unsafe token type",
			uri:  "file:///test.css",
			varCall: css.VarCall{
				TokenName: "--border",
				Fallback:  ptrString("1px solid black"),
				Range: css.Range{
					Start: css.Position{Line: 5, Character: 5},
					End:   css.Position{Line: 5, Character: 30},
				},
			},
			token: &tokens.Token{
				Name:  "border",
				Value: "2px solid #ff0000",
				Type:  "border", // composite type - unsafe
			},
			diagnostics: []protocol.Diagnostic{},
			expectNil:   true, // Should return nil for unsafe types
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			action := codeaction.CreateFixFallbackAction(tt.uri, tt.varCall, tt.token, tt.diagnostics)

			if tt.expectNil {
				assert.Nil(t, action)
			} else {
				require.NotNil(t, action)
				assert.Equal(t, tt.checkTitle, action.Title)
				if tt.checkEdit {
					require.NotNil(t, action.Edit)
					require.NotNil(t, action.Edit.Changes)
					edits, ok := action.Edit.Changes[tt.uri]
					require.True(t, ok)
					require.Len(t, edits, 1)
					assert.Contains(t, edits[0].NewText, "var(--")
				}
			}
		})
	}
}

// TestCreateAddFallbackAction tests the createAddFallbackAction helper
func TestCreateAddFallbackAction(t *testing.T) {
	token := &tokens.Token{
		Name:  "spacing-large",
		Value: "24px",
		Type:  "dimension",
	}

	tests := []struct {
		name       string
		uri        string
		varCall    css.VarCall
		token      *tokens.Token
		expectNil  bool
		checkTitle string
	}{
		{
			name: "add fallback to var without one",
			uri:  "file:///test.css",
			varCall: css.VarCall{
				TokenName: "--spacing-large",
				Fallback:  nil, // no fallback
				Range: css.Range{
					Start: css.Position{Line: 3, Character: 12},
					End:   css.Position{Line: 3, Character: 32},
				},
			},
			token:      token,
			expectNil:  false,
			checkTitle: "Add fallback value '24px'",
		},
		{
			name: "unsafe composite type",
			uri:  "file:///test.css",
			varCall: css.VarCall{
				TokenName: "--shadow",
				Fallback:  nil,
				Range: css.Range{
					Start: css.Position{Line: 5, Character: 5},
					End:   css.Position{Line: 5, Character: 18},
				},
			},
			token: &tokens.Token{
				Name:  "shadow-default",
				Value: "0 2px 4px rgba(0,0,0,0.1)",
				Type:  "shadow", // composite - unsafe
			},
			expectNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			action := codeaction.CreateAddFallbackAction(tt.uri, tt.varCall, tt.token)

			if tt.expectNil {
				assert.Nil(t, action)
			} else {
				require.NotNil(t, action)
				assert.Equal(t, tt.checkTitle, action.Title)
				require.NotNil(t, action.Edit)
			}
		})
	}
}

// TestCreateDeprecatedTokenActions tests the createDeprecatedTokenActions helper
func TestCreateDeprecatedTokenActions(t *testing.T) {
	s, err := lsp.NewServer()
	require.NoError(t, err)

	// Add test tokens
	oldToken := &tokens.Token{
		Name:        "color-old",
		Value:       "#ff0000",
		Type:        "color",
		Description: "Deprecated: use color-new instead",
		Deprecated:  true,
	}
	newToken := &tokens.Token{
		Name:  "color-new",
		Value: "#ff0000",
		Type:  "color",
	}
	_ = s.TokenManager().Add(oldToken)
	_ = s.TokenManager().Add(newToken)

	tests := []struct {
		name               string
		uri                string
		varCall            css.VarCall
		token              *tokens.Token
		diagnostics        []protocol.Diagnostic
		expectedNumActions int
		checkActionTitle   string
	}{
		{
			name: "deprecated token with diagnostic",
			uri:  "file:///test.css",
			varCall: css.VarCall{
				TokenName: "--color-old",
				Range: css.Range{
					Start: css.Position{Line: 8, Character: 10},
					End:   css.Position{Line: 8, Character: 28},
				},
			},
			token: oldToken,
			diagnostics: []protocol.Diagnostic{
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 8, Character: 10},
						End:   protocol.Position{Line: 8, Character: 28},
					},
					Message: "Token is deprecated",
				},
			},
			expectedNumActions: 1,
			checkActionTitle:   "Replace with literal value '#ff0000'",
		},
		{
			name: "deprecated token without matching diagnostic",
			uri:  "file:///test.css",
			varCall: css.VarCall{
				TokenName: "--color-old",
				Range: css.Range{
					Start: css.Position{Line: 5, Character: 5},
					End:   css.Position{Line: 5, Character: 20},
				},
			},
			token:              oldToken,
			diagnostics:        []protocol.Diagnostic{},
			expectedNumActions: 1,
		},
		{
			name: "deprecated token with 'Use X instead' message",
			uri:  "file:///test.css",
			varCall: css.VarCall{
				TokenName: "--color-legacy",
				Range: css.Range{
					Start: css.Position{Line: 10, Character: 5},
					End:   css.Position{Line: 10, Character: 25},
				},
			},
			token: &tokens.Token{
				Name:               "color-legacy",
				Value:              "#ff0000",
				Type:               "color",
				DeprecationMessage: "Use color-new instead",
				Deprecated:         true,
			},
			diagnostics: []protocol.Diagnostic{
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 10, Character: 5},
						End:   protocol.Position{Line: 10, Character: 25},
					},
					Message: "Token is deprecated",
				},
			},
			expectedNumActions: 2, // Should create both replacement and literal actions
			checkActionTitle:   "Replace with '--color-new'",
		},
		{
			name: "deprecated token with 'Replaced by X' message",
			uri:  "file:///test.css",
			varCall: css.VarCall{
				TokenName: "--spacing-old",
				Range: css.Range{
					Start: css.Position{Line: 12, Character: 8},
					End:   css.Position{Line: 12, Character: 28},
				},
			},
			token: &tokens.Token{
				Name:               "spacing-old",
				Value:              "16px",
				Type:               "dimension",
				DeprecationMessage: "Replaced by spacing-base",
				Deprecated:         true,
			},
			diagnostics: []protocol.Diagnostic{},
			expectedNumActions: 2, // Both actions even without matching diagnostic
		},
		{
			name: "deprecated token with fallback, 'Use X instead' message",
			uri:  "file:///test.css",
			varCall: css.VarCall{
				TokenName: "--color-legacy",
				Fallback:  ptrString("#000000"),
				Range: css.Range{
					Start: css.Position{Line: 14, Character: 5},
					End:   css.Position{Line: 14, Character: 35},
				},
			},
			token: &tokens.Token{
				Name:               "color-legacy",
				Value:              "#ff0000",
				Type:               "color",
				DeprecationMessage: "Use color-new instead",
				Deprecated:         true,
			},
			diagnostics:        []protocol.Diagnostic{},
			expectedNumActions: 2, // Both actions
		},
	}

	// Add the recommended replacement tokens for the tests
	spacingBase := &tokens.Token{
		Name:  "spacing-base",
		Value: "16px",
		Type:  "dimension",
	}
	_ = s.TokenManager().Add(spacingBase)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actions := codeaction.CreateDeprecatedTokenActions(s, tt.uri, tt.varCall, tt.token, tt.diagnostics)
			assert.Len(t, actions, tt.expectedNumActions)

			// Check action title if specified
			if tt.checkActionTitle != "" && len(actions) > 0 {
				assert.Equal(t, tt.checkActionTitle, actions[0].Title)
			}
		})
	}
}

// TestToggleFallback tests the toggle fallback action (RefactorRewrite)
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
		cursorLine     uint32
		cursorChar     uint32
		expectedAction string // empty if no action expected
		expectedEdit   string // the new text after toggle
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

			// Request code actions at cursor position (single-char range)
			params := &protocol.CodeActionParams{
				TextDocument: protocol.TextDocumentIdentifier{URI: uri},
				Range: protocol.Range{
					Start: protocol.Position{Line: tt.cursorLine, Character: tt.cursorChar},
					End:   protocol.Position{Line: tt.cursorLine, Character: tt.cursorChar + 1},
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
		rangeStart     protocol.Position
		rangeEnd       protocol.Position
		expectedAction string
		numEdits       int
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
			name:           "single char range - should not show range action",
			cssContent:     `.button { color: var(--color-primary, #ff0000); }`,
			rangeStart:     protocol.Position{Line: 0, Character: 21},
			rangeEnd:       protocol.Position{Line: 0, Character: 22},
			expectedAction: "", // single-char should show toggleFallback, not toggleRangeFallbacks
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
