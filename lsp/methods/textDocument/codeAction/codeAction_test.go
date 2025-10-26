package codeaction_test

import (
	"testing"

	"bennypowers.dev/dtls/internal/parser/css"
	"bennypowers.dev/dtls/internal/tokens"
	"bennypowers.dev/dtls/lsp"
	codeaction "bennypowers.dev/dtls/lsp/methods/textDocument/codeAction"
	protocol "github.com/tliron/glsp/protocol_3_16"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ptrString returns a pointer to the given string
func ptrString(s string) *string {
	return &s
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
	s.TokenManager().Add(oldToken)
	s.TokenManager().Add(newToken)

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
	s.TokenManager().Add(spacingBase)

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
