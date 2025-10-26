package lsp

import (
	"testing"

	"github.com/bennypowers/design-tokens-language-server/internal/parser/css"
	protocol "github.com/tliron/glsp/protocol_3_16"
	"github.com/stretchr/testify/assert"
)

// TestIsPositionInRange tests the isPositionInRange function with half-open range semantics [start, end)
func TestIsPositionInRange(t *testing.T) {
	tests := []struct {
		name     string
		pos      protocol.Position
		r        css.Range
		expected bool
	}{
		{
			name: "position at start boundary - included",
			pos:  protocol.Position{Line: 0, Character: 5},
			r: css.Range{
				Start: css.Position{Line: 0, Character: 5},
				End:   css.Position{Line: 0, Character: 10},
			},
			expected: true, // Start is inclusive
		},
		{
			name: "position at end boundary - excluded",
			pos:  protocol.Position{Line: 0, Character: 10},
			r: css.Range{
				Start: css.Position{Line: 0, Character: 5},
				End:   css.Position{Line: 0, Character: 10},
			},
			expected: false, // End is exclusive in half-open range [start, end)
		},
		{
			name: "position before range",
			pos:  protocol.Position{Line: 0, Character: 4},
			r: css.Range{
				Start: css.Position{Line: 0, Character: 5},
				End:   css.Position{Line: 0, Character: 10},
			},
			expected: false,
		},
		{
			name: "position after range",
			pos:  protocol.Position{Line: 0, Character: 11},
			r: css.Range{
				Start: css.Position{Line: 0, Character: 5},
				End:   css.Position{Line: 0, Character: 10},
			},
			expected: false,
		},
		{
			name: "position inside range",
			pos:  protocol.Position{Line: 0, Character: 7},
			r: css.Range{
				Start: css.Position{Line: 0, Character: 5},
				End:   css.Position{Line: 0, Character: 10},
			},
			expected: true,
		},
		{
			name: "position on earlier line",
			pos:  protocol.Position{Line: 0, Character: 5},
			r: css.Range{
				Start: css.Position{Line: 1, Character: 0},
				End:   css.Position{Line: 2, Character: 0},
			},
			expected: false,
		},
		{
			name: "position on later line",
			pos:  protocol.Position{Line: 3, Character: 0},
			r: css.Range{
				Start: css.Position{Line: 1, Character: 0},
				End:   css.Position{Line: 2, Character: 0},
			},
			expected: false,
		},
		{
			name: "multi-line range - position at start line, before start char",
			pos:  protocol.Position{Line: 1, Character: 5},
			r: css.Range{
				Start: css.Position{Line: 1, Character: 10},
				End:   css.Position{Line: 3, Character: 5},
			},
			expected: false,
		},
		{
			name: "multi-line range - position at end line, at end char (excluded)",
			pos:  protocol.Position{Line: 3, Character: 5},
			r: css.Range{
				Start: css.Position{Line: 1, Character: 10},
				End:   css.Position{Line: 3, Character: 5},
			},
			expected: false, // End is exclusive
		},
		{
			name: "multi-line range - position inside",
			pos:  protocol.Position{Line: 2, Character: 15},
			r: css.Range{
				Start: css.Position{Line: 1, Character: 10},
				End:   css.Position{Line: 3, Character: 5},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isPositionInRange(tt.pos, tt.r)
			assert.Equal(t, tt.expected, result, "isPositionInRange(%+v, %+v)", tt.pos, tt.r)
		})
	}
}
