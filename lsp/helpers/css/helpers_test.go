package css_test

import (
	"testing"

	"bennypowers.dev/dtls/internal/tokens"
	"bennypowers.dev/dtls/lsp/helpers/css"
	"github.com/stretchr/testify/assert"
)

func TestFormatTokenValueForCSS(t *testing.T) {
	tests := []struct {
		name          string
		token         *tokens.Token
		expectedValue string
		expectError   bool
	}{
		// Safe types - colors
		{
			name:          "hex color",
			token:         &tokens.Token{Type: "color", Value: "#ff0000"},
			expectedValue: "#ff0000",
			expectError:   false,
		},
		{
			name:          "rgb color",
			token:         &tokens.Token{Type: "color", Value: "rgb(255, 0, 0)"},
			expectedValue: "rgb(255, 0, 0)",
			expectError:   false,
		},
		{
			name:          "named color",
			token:         &tokens.Token{Type: "color", Value: "red"},
			expectedValue: "red",
			expectError:   false,
		},

		// Safe types - dimensions
		{
			name:          "pixel dimension",
			token:         &tokens.Token{Type: "dimension", Value: "16px"},
			expectedValue: "16px",
			expectError:   false,
		},
		{
			name:          "rem dimension",
			token:         &tokens.Token{Type: "dimension", Value: "1.5rem"},
			expectedValue: "1.5rem",
			expectError:   false,
		},

		// Safe types - numbers
		{
			name:          "integer number",
			token:         &tokens.Token{Type: "number", Value: "42"},
			expectedValue: "42",
			expectError:   false,
		},
		{
			name:          "decimal number",
			token:         &tokens.Token{Type: "number", Value: "1.5"},
			expectedValue: "1.5",
			expectError:   false,
		},

		// Font weight - safe cases
		{
			name:          "numeric font weight",
			token:         &tokens.Token{Type: "fontWeight", Value: "700"},
			expectedValue: "700",
			expectError:   false,
		},
		{
			name:          "keyword font weight bold",
			token:         &tokens.Token{Type: "fontWeight", Value: "bold"},
			expectedValue: "bold",
			expectError:   false,
		},
		{
			name:          "keyword font weight normal",
			token:         &tokens.Token{Type: "fontWeight", Value: "normal"},
			expectedValue: "normal",
			expectError:   false,
		},
		{
			name:          "keyword font weight bolder",
			token:         &tokens.Token{Type: "fontWeight", Value: "bolder"},
			expectedValue: "bolder",
			expectError:   false,
		},
		{
			name:          "keyword font weight lighter",
			token:         &tokens.Token{Type: "fontWeight", Value: "lighter"},
			expectedValue: "lighter",
			expectError:   false,
		},
		{
			name:          "keyword font weight inherit",
			token:         &tokens.Token{Type: "fontWeight", Value: "inherit"},
			expectedValue: "inherit",
			expectError:   false,
		},
		{
			name:          "keyword font weight initial",
			token:         &tokens.Token{Type: "fontWeight", Value: "initial"},
			expectedValue: "initial",
			expectError:   false,
		},
		{
			name:          "keyword font weight unset",
			token:         &tokens.Token{Type: "fontWeight", Value: "unset"},
			expectedValue: "unset",
			expectError:   false,
		},
		{
			name:          "font weight minimum valid (1)",
			token:         &tokens.Token{Type: "fontWeight", Value: "1"},
			expectedValue: "1",
			expectError:   false,
		},
		{
			name:          "font weight maximum valid (1000)",
			token:         &tokens.Token{Type: "fontWeight", Value: "1000"},
			expectedValue: "1000",
			expectError:   false,
		},
		{
			name:          "font weight mid-range (500)",
			token:         &tokens.Token{Type: "fontWeight", Value: "500"},
			expectedValue: "500",
			expectError:   false,
		},
		{
			name:          "font weight zero (invalid)",
			token:         &tokens.Token{Type: "fontWeight", Value: "0"},
			expectedValue: "",
			expectError:   true,
		},
		{
			name:          "font weight over 1000 (invalid)",
			token:         &tokens.Token{Type: "fontWeight", Value: "1001"},
			expectedValue: "",
			expectError:   true,
		},
		{
			name:          "font weight way over range (invalid)",
			token:         &tokens.Token{Type: "fontWeight", Value: "9999"},
			expectedValue: "",
			expectError:   true,
		},
		{
			name:          "invalid font weight keyword",
			token:         &tokens.Token{Type: "fontWeight", Value: "super-bold"},
			expectedValue: "",
			expectError:   true,
		},

		// Font family - quoting needed
		{
			name:          "font family with spaces",
			token:         &tokens.Token{Type: "fontFamily", Value: "Helvetica Neue"},
			expectedValue: `"Helvetica Neue"`,
			expectError:   false,
		},
		{
			name:          "font family with quotes",
			token:         &tokens.Token{Type: "fontFamily", Value: `"Times New Roman"`},
			expectedValue: `"Times New Roman"`,
			expectError:   false,
		},
		{
			name:          "generic font family",
			token:         &tokens.Token{Type: "fontFamily", Value: "sans-serif"},
			expectedValue: "sans-serif",
			expectError:   false,
		},
		{
			name:          "single word font",
			token:         &tokens.Token{Type: "fontFamily", Value: "Arial"},
			expectedValue: "Arial",
			expectError:   false,
		},
		{
			name:          "font family with internal quotes",
			token:         &tokens.Token{Type: "fontFamily", Value: `Font "Name" Here`},
			expectedValue: `"Font \"Name\" Here"`,
			expectError:   false,
		},

		// No type specified - heuristic detection
		{
			name:          "untyped hex color",
			token:         &tokens.Token{Value: "#00ff00"},
			expectedValue: "#00ff00",
			expectError:   false,
		},
		{
			name:          "untyped dimension",
			token:         &tokens.Token{Value: "20px"},
			expectedValue: "20px",
			expectError:   false,
		},
		{
			name:          "untyped number",
			token:         &tokens.Token{Value: "3.14"},
			expectedValue: "3.14",
			expectError:   false,
		},
		{
			name:          "untyped simple identifier",
			token:         &tokens.Token{Value: "auto"},
			expectedValue: "auto",
			expectError:   false,
		},
		{
			name:          "untyped with spaces - unsafe",
			token:         &tokens.Token{Value: "some value"},
			expectedValue: "",
			expectError:   true,
		},

		// Unsafe composite types
		{
			name:          "border composite",
			token:         &tokens.Token{Type: "border", Value: "1px solid #000"},
			expectedValue: "",
			expectError:   true,
		},
		{
			name:          "shadow composite",
			token:         &tokens.Token{Type: "shadow", Value: "0 2px 4px rgba(0,0,0,0.1)"},
			expectedValue: "",
			expectError:   true,
		},
		{
			name:          "typography composite",
			token:         &tokens.Token{Type: "typography", Value: "bold 16px/1.5 Arial"},
			expectedValue: "",
			expectError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value, err := css.FormatTokenValueForCSS(tt.token)
			if tt.expectError {
				assert.Error(t, err, "expected error but got none")
				assert.Equal(t, "", value, "value should be empty on error")
			} else {
				assert.NoError(t, err, "unexpected error")
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
		expectError   bool
	}{
		{
			name:          "generic sans-serif",
			input:         "sans-serif",
			expectedValue: "sans-serif",
			expectError:   false,
		},
		{
			name:          "generic serif",
			input:         "serif",
			expectedValue: "serif",
			expectError:   false,
		},
		{
			name:          "single word font",
			input:         "Arial",
			expectedValue: "Arial",
			expectError:   false,
		},
		{
			name:          "font with spaces",
			input:         "Comic Sans MS",
			expectedValue: `"Comic Sans MS"`,
			expectError:   false,
		},
		{
			name:          "already quoted",
			input:         `"Times New Roman"`,
			expectedValue: `"Times New Roman"`,
			expectError:   false,
		},
		{
			name:          "font with internal quotes",
			input:         `Font "Special" Name`,
			expectedValue: `"Font \"Special\" Name"`,
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value, err := css.FormatFontFamilyValue(tt.input)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedValue, value)
			}
		})
	}
}

// TestCreateFixFallbackAction tests the createFixFallbackAction helper

// TestIsCSSValueSemanticallyEquivalent tests CSS value equivalence checking
func TestIsCSSValueSemanticallyEquivalent(t *testing.T) {
	tests := []struct {
		name     string
		a        string
		b        string
		expected bool
	}{
		// Exact matches
		{
			name:     "identical simple values",
			a:        "#ff0000",
			b:        "#ff0000",
			expected: true,
		},
		{
			name:     "identical complex values",
			a:        "rgb(255, 0, 0)",
			b:        "rgb(255, 0, 0)",
			expected: true,
		},

		// Case differences
		{
			name:     "hex color - different case",
			a:        "#FF0000",
			b:        "#ff0000",
			expected: true,
		},
		{
			name:     "hex color - mixed case",
			a:        "#Ff0000",
			b:        "#fF0000",
			expected: true,
		},
		{
			name:     "dimension unit - different case",
			a:        "1.5REM",
			b:        "1.5rem",
			expected: true,
		},
		{
			name:     "function name - different case",
			a:        "RGB(255, 0, 0)",
			b:        "rgb(255, 0, 0)",
			expected: true,
		},

		// Whitespace differences
		{
			name:     "rgb - extra spaces",
			a:        "rgb(255, 0, 0)",
			b:        "rgb(255,0,0)",
			expected: true,
		},
		{
			name:     "rgb - spaces vs tabs",
			a:        "rgb(255, 0, 0)",
			b:        "rgb(255,\t0,\t0)",
			expected: true,
		},
		{
			name:     "rgb - with newlines",
			a:        "rgb(255, 0, 0)",
			b:        "rgb(255,\n0,\n0)",
			expected: true,
		},
		{
			name:     "rgba - extra spaces",
			a:        "rgba(0, 255, 0, 0.5)",
			b:        "rgba(0,255,0,0.5)",
			expected: true,
		},
		{
			name:     "calc - with spaces",
			a:        "calc(100% - 20px)",
			b:        "calc(100%-20px)",
			expected: true,
		},
		{
			name:     "multiple spaces",
			a:        "1px  solid  #000",
			b:        "1px solid #000",
			expected: true,
		},

		// Combined case and whitespace
		{
			name:     "rgb - case and whitespace",
			a:        "RGB(255, 0, 0)",
			b:        "rgb(255,0,0)",
			expected: true,
		},
		{
			name:     "hsl - case and whitespace",
			a:        "HSL(120, 100%, 50%)",
			b:        "hsl(120,100%,50%)",
			expected: true,
		},

		// Different values (should not be equivalent)
		{
			name:     "different hex colors",
			a:        "#ff0000",
			b:        "#00ff00",
			expected: false,
		},
		{
			name:     "different rgb values",
			a:        "rgb(255, 0, 0)",
			b:        "rgb(0, 255, 0)",
			expected: false,
		},
		{
			name:     "different dimensions",
			a:        "16px",
			b:        "20px",
			expected: false,
		},
		{
			name:     "different units",
			a:        "1.5rem",
			b:        "1.5em",
			expected: false,
		},
		{
			name:     "different functions",
			a:        "rgb(255, 0, 0)",
			b:        "hsl(0, 100%, 50%)",
			expected: false,
		},

		// Edge cases
		{
			name:     "empty strings",
			a:        "",
			b:        "",
			expected: true,
		},
		{
			name:     "empty vs non-empty",
			a:        "",
			b:        "#ff0000",
			expected: false,
		},
		{
			name:     "only whitespace vs empty",
			a:        "   ",
			b:        "",
			expected: true, // Both normalize to empty
		},
		{
			name:     "only whitespace - spaces vs tabs",
			a:        "   ",
			b:        "\t\t\t",
			expected: true, // Both normalize to empty
		},

		// Real-world CSS values
		{
			name:     "border shorthand",
			a:        "1px solid #000000",
			b:        "1px solid #000000",
			expected: true,
		},
		{
			name:     "border - with spaces",
			a:        "1px solid #000000",
			b:        "1px  solid  #000000",
			expected: true,
		},
		{
			name:     "box-shadow",
			a:        "0 2px 4px rgba(0,0,0,0.1)",
			b:        "0 2px 4px rgba(0, 0, 0, 0.1)",
			expected: true,
		},
		{
			name:     "gradient",
			a:        "linear-gradient(to right, #ff0000, #00ff00)",
			b:        "linear-gradient(to right,#ff0000,#00ff00)",
			expected: true,
		},
		{
			name:     "var() function",
			a:        "var(--color-primary)",
			b:        "var(--color-primary)",
			expected: true,
		},
		{
			name:     "var() with fallback - spaces",
			a:        "var(--color-primary, #0000ff)",
			b:        "var(--color-primary,#0000ff)",
			expected: true,
		},
		{
			name:     "transform",
			a:        "translate(10px, 20px)",
			b:        "translate(10px,20px)",
			expected: true,
		},

		// Numeric precision (note: this compares strings, not numeric values)
		{
			name:     "decimal - same",
			a:        "0.5",
			b:        "0.5",
			expected: true,
		},
		{
			name:     "decimal - different precision",
			a:        "0.5",
			b:        "0.50",
			expected: false, // String comparison, not numeric
		},
		{
			name:     "leading zero",
			a:        "0.5",
			b:        ".5",
			expected: false, // String comparison
		},

		// Symmetry tests (a==b implies b==a)
		{
			name:     "symmetry - rgb reversed",
			a:        "rgb(255,0,0)",
			b:        "rgb(255, 0, 0)",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := css.IsCSSValueSemanticallyEquivalent(tt.a, tt.b)
			assert.Equal(t, tt.expected, result, "IsCSSValueSemanticallyEquivalent(%q, %q)", tt.a, tt.b)

			// Test symmetry: f(a,b) should equal f(b,a)
			resultReversed := css.IsCSSValueSemanticallyEquivalent(tt.b, tt.a)
			assert.Equal(t, result, resultReversed, "Function should be symmetric")
		})
	}
}
