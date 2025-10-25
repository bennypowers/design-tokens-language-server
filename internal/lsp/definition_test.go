package lsp

import (
	"testing"

	"github.com/bennypowers/design-tokens-language-server/internal/parser/css"
	protocol "github.com/tliron/glsp/protocol_3_16"
	"github.com/stretchr/testify/assert"
)

// TestIsPositionInVarCall tests the isPositionInVarCall function with half-open range semantics [start, end)
func TestIsPositionInVarCall(t *testing.T) {
	server := &Server{} // Minimal server for method call

	tests := []struct {
		name     string
		pos      protocol.Position
		varCall  *css.VarCall
		expected bool
	}{
		{
			name: "position at start boundary - included",
			pos:  protocol.Position{Line: 0, Character: 10},
			varCall: &css.VarCall{
				TokenName: "color-primary",
				Range: css.Range{
					Start: css.Position{Line: 0, Character: 10},
					End:   css.Position{Line: 0, Character: 30},
				},
			},
			expected: true, // Start is inclusive
		},
		{
			name: "position at end boundary - excluded",
			pos:  protocol.Position{Line: 0, Character: 30},
			varCall: &css.VarCall{
				TokenName: "color-primary",
				Range: css.Range{
					Start: css.Position{Line: 0, Character: 10},
					End:   css.Position{Line: 0, Character: 30},
				},
			},
			expected: false, // End is exclusive in half-open range [start, end)
		},
		{
			name: "position before var call",
			pos:  protocol.Position{Line: 0, Character: 9},
			varCall: &css.VarCall{
				TokenName: "color-primary",
				Range: css.Range{
					Start: css.Position{Line: 0, Character: 10},
					End:   css.Position{Line: 0, Character: 30},
				},
			},
			expected: false,
		},
		{
			name: "position after var call",
			pos:  protocol.Position{Line: 0, Character: 31},
			varCall: &css.VarCall{
				TokenName: "color-primary",
				Range: css.Range{
					Start: css.Position{Line: 0, Character: 10},
					End:   css.Position{Line: 0, Character: 30},
				},
			},
			expected: false,
		},
		{
			name: "position inside var call",
			pos:  protocol.Position{Line: 0, Character: 20},
			varCall: &css.VarCall{
				TokenName: "color-primary",
				Range: css.Range{
					Start: css.Position{Line: 0, Character: 10},
					End:   css.Position{Line: 0, Character: 30},
				},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := server.isPositionInVarCall(tt.pos, tt.varCall)
			assert.Equal(t, tt.expected, result, "isPositionInVarCall(%+v, %+v)", tt.pos, tt.varCall)
		})
	}
}
