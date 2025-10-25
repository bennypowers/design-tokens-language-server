package lsp

import (
	"testing"

	"github.com/bennypowers/design-tokens-language-server/internal/tokens"
	protocol "github.com/tliron/glsp/protocol_3_16"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
			result := rangesIntersect(tt.a, tt.b)
			assert.Equal(t, tt.expected, result, "rangesIntersect(%+v, %+v)", tt.a, tt.b)
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
			value, safe := formatTokenValueForCSS(tt.token)
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
			value, safe := formatFontFamilyValue(tt.input)
			require.Equal(t, tt.expectedSafe, safe)
			assert.Equal(t, tt.expectedValue, value)
		})
	}
}
