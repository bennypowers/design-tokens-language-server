package tokens

import "testing"

// TestCSSVariableName tests that CSSVariableName properly converts token names to valid CSS custom properties
func TestCSSVariableName(t *testing.T) {
	tests := []struct {
		name     string
		token    Token
		expected string
	}{
		{
			name:     "simple name without prefix",
			token:    Token{Name: "primary"},
			expected: "--primary",
		},
		{
			name:     "name with single dot",
			token:    Token{Name: "color.primary"},
			expected: "--color-primary",
		},
		{
			name:     "name with multiple dots",
			token:    Token{Name: "color.brand.primary"},
			expected: "--color-brand-primary",
		},
		{
			name:     "name with many dots",
			token:    Token{Name: "color.brand.primary.500"},
			expected: "--color-brand-primary-500",
		},
		{
			name:     "simple name with simple prefix",
			token:    Token{Name: "primary", Prefix: "my"},
			expected: "--my-primary",
		},
		{
			name:     "dotted name with simple prefix",
			token:    Token{Name: "color.primary", Prefix: "my"},
			expected: "--my-color-primary",
		},
		{
			name:     "simple name with dotted prefix",
			token:    Token{Name: "primary", Prefix: "my.brand"},
			expected: "--my-brand-primary",
		},
		{
			name:     "dotted name with dotted prefix",
			token:    Token{Name: "color.primary.500", Prefix: "my.brand"},
			expected: "--my-brand-color-primary-500",
		},
		{
			name:     "complex nested token path",
			token:    Token{Name: "semantic.color.background.primary.light", Prefix: "ds.v2"},
			expected: "--ds-v2-semantic-color-background-primary-light",
		},
		{
			name:     "empty name",
			token:    Token{Name: ""},
			expected: "",
		},
		{
			name:     "empty name with prefix",
			token:    Token{Name: "", Prefix: "my"},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.token.CSSVariableName()
			if got != tt.expected {
				t.Errorf("CSSVariableName() = %q, want %q", got, tt.expected)
			}
		})
	}
}
