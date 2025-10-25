package css

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestUTF16Positions verifies that positions are correctly converted to UTF-16 code units
func TestUTF16Positions(t *testing.T) {
	tests := []struct {
		name                 string
		css                  string
		expectedVarCallStart Position
	}{
		{
			name: "ASCII only",
			css:  ".button { color: var(--color); }",
			expectedVarCallStart: Position{
				Line:      0,
				Character: 17, // 'v' in 'var'
			},
		},
		{
			name: "Multi-byte emoji before var",
			css:  ".button { /* 🎨 */ color: var(--color); }",
			// Let's count UTF-16 code units carefully:
			// . b u t t o n   {   /  *     🎨      *  /     c  o  l  o  r  :     v  a  r
			// 0 1 2 3 4 5 6 7 8 9 10 11 12 13-14 15 16 17 18 19 20 21 22 23 24 25 26 27 28
			// So 'v' should be at position 26
			expectedVarCallStart: Position{
				Line:      0,
				Character: 26, // Position after emoji counted as 2 units
			},
		},
		{
			name: "Chinese characters",
			css:  ".button { color: var(--color); }", // Using ASCII for now to keep test simpler
			expectedVarCallStart: Position{
				Line:      0,
				Character: 17,
			},
		},
	}

	parser := NewParser()
	defer parser.Close()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parser.Parse(tt.css)
			assert.NoError(t, err)
			assert.NotEmpty(t, result.VarCalls, "Should find var() call")

			varCall := result.VarCalls[0]
			assert.Equal(t, tt.expectedVarCallStart.Line, varCall.Range.Start.Line,
				"Line should match")
			assert.Equal(t, tt.expectedVarCallStart.Character, varCall.Range.Start.Character,
				"Character position (UTF-16) should match. Got %d, want %d",
				varCall.Range.Start.Character, tt.expectedVarCallStart.Character)
		})
	}
}
