package helpers_test

import (
	"testing"

	"bennypowers.dev/dtls/lsp/helpers"
	"github.com/stretchr/testify/assert"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// TestRangesIntersect tests the RangesIntersect function with half-open range semantics [start, end)
func TestRangesIntersect(t *testing.T) {
	tests := []struct {
		name     string
		a        protocol.Range
		b        protocol.Range
		expected bool
	}{
		// Same line cases
		{
			name: "same line - overlapping ranges",
			a: protocol.Range{
				Start: protocol.Position{Line: 0, Character: 0},
				End:   protocol.Position{Line: 0, Character: 5},
			},
			b: protocol.Range{
				Start: protocol.Position{Line: 0, Character: 3},
				End:   protocol.Position{Line: 0, Character: 7},
			},
			expected: true,
		},
		{
			name: "same line - adjacent ranges (no overlap)",
			a: protocol.Range{
				Start: protocol.Position{Line: 0, Character: 0},
				End:   protocol.Position{Line: 0, Character: 5},
			},
			b: protocol.Range{
				Start: protocol.Position{Line: 0, Character: 5},
				End:   protocol.Position{Line: 0, Character: 10},
			},
			expected: false, // [0,5) and [5,10) don't overlap (half-open)
		},
		{
			name: "same line - b before a (no overlap)",
			a: protocol.Range{
				Start: protocol.Position{Line: 0, Character: 5},
				End:   protocol.Position{Line: 0, Character: 10},
			},
			b: protocol.Range{
				Start: protocol.Position{Line: 0, Character: 0},
				End:   protocol.Position{Line: 0, Character: 4},
			},
			expected: false,
		},
		{
			name: "same line - a contains b",
			a: protocol.Range{
				Start: protocol.Position{Line: 0, Character: 0},
				End:   protocol.Position{Line: 0, Character: 10},
			},
			b: protocol.Range{
				Start: protocol.Position{Line: 0, Character: 3},
				End:   protocol.Position{Line: 0, Character: 7},
			},
			expected: true,
		},
		{
			name: "same line - b contains a",
			a: protocol.Range{
				Start: protocol.Position{Line: 0, Character: 3},
				End:   protocol.Position{Line: 0, Character: 7},
			},
			b: protocol.Range{
				Start: protocol.Position{Line: 0, Character: 0},
				End:   protocol.Position{Line: 0, Character: 10},
			},
			expected: true,
		},
		{
			name: "same line - identical ranges",
			a: protocol.Range{
				Start: protocol.Position{Line: 0, Character: 0},
				End:   protocol.Position{Line: 0, Character: 5},
			},
			b: protocol.Range{
				Start: protocol.Position{Line: 0, Character: 0},
				End:   protocol.Position{Line: 0, Character: 5},
			},
			expected: true,
		},
		{
			name: "same line - zero-width ranges at same position",
			a: protocol.Range{
				Start: protocol.Position{Line: 0, Character: 5},
				End:   protocol.Position{Line: 0, Character: 5},
			},
			b: protocol.Range{
				Start: protocol.Position{Line: 0, Character: 5},
				End:   protocol.Position{Line: 0, Character: 5},
			},
			expected: false, // Two empty ranges don't intersect
		},
		{
			name: "same line - zero-width range at start of non-zero range",
			a: protocol.Range{
				Start: protocol.Position{Line: 0, Character: 5},
				End:   protocol.Position{Line: 0, Character: 5},
			},
			b: protocol.Range{
				Start: protocol.Position{Line: 0, Character: 5},
				End:   protocol.Position{Line: 0, Character: 10},
			},
			expected: false, // Cursor at start doesn't intersect [5,10) - practical for LSP
		},
		{
			name: "same line - zero-width range inside non-zero range",
			a: protocol.Range{
				Start: protocol.Position{Line: 0, Character: 5},
				End:   protocol.Position{Line: 0, Character: 5},
			},
			b: protocol.Range{
				Start: protocol.Position{Line: 0, Character: 0},
				End:   protocol.Position{Line: 0, Character: 10},
			},
			expected: true, // Cursor at position 5 should trigger actions for [0,10) - practical for LSP
		},

		// Multi-line cases
		{
			name: "multi-line - overlapping ranges",
			a: protocol.Range{
				Start: protocol.Position{Line: 0, Character: 0},
				End:   protocol.Position{Line: 2, Character: 5},
			},
			b: protocol.Range{
				Start: protocol.Position{Line: 1, Character: 0},
				End:   protocol.Position{Line: 3, Character: 0},
			},
			expected: true,
		},
		{
			name: "multi-line - a before b (no overlap)",
			a: protocol.Range{
				Start: protocol.Position{Line: 0, Character: 0},
				End:   protocol.Position{Line: 1, Character: 0},
			},
			b: protocol.Range{
				Start: protocol.Position{Line: 2, Character: 0},
				End:   protocol.Position{Line: 3, Character: 0},
			},
			expected: false,
		},
		{
			name: "multi-line - b before a (no overlap)",
			a: protocol.Range{
				Start: protocol.Position{Line: 2, Character: 0},
				End:   protocol.Position{Line: 3, Character: 0},
			},
			b: protocol.Range{
				Start: protocol.Position{Line: 0, Character: 0},
				End:   protocol.Position{Line: 1, Character: 0},
			},
			expected: false,
		},
		{
			name: "multi-line - adjacent ranges at line boundary",
			a: protocol.Range{
				Start: protocol.Position{Line: 0, Character: 0},
				End:   protocol.Position{Line: 1, Character: 0},
			},
			b: protocol.Range{
				Start: protocol.Position{Line: 1, Character: 0},
				End:   protocol.Position{Line: 2, Character: 0},
			},
			expected: false, // Adjacent at line 1:0 (half-open)
		},
		{
			name: "multi-line - ranges touch at character boundary",
			a: protocol.Range{
				Start: protocol.Position{Line: 0, Character: 0},
				End:   protocol.Position{Line: 1, Character: 5},
			},
			b: protocol.Range{
				Start: protocol.Position{Line: 1, Character: 5},
				End:   protocol.Position{Line: 2, Character: 0},
			},
			expected: false, // Adjacent at 1:5 (half-open)
		},
		{
			name: "multi-line - one character overlap",
			a: protocol.Range{
				Start: protocol.Position{Line: 0, Character: 0},
				End:   protocol.Position{Line: 1, Character: 5},
			},
			b: protocol.Range{
				Start: protocol.Position{Line: 1, Character: 4},
				End:   protocol.Position{Line: 2, Character: 0},
			},
			expected: true, // Overlap at 1:4
		},

		// Edge cases
		{
			name: "entire document - both ranges span full file",
			a: protocol.Range{
				Start: protocol.Position{Line: 0, Character: 0},
				End:   protocol.Position{Line: 100, Character: 0},
			},
			b: protocol.Range{
				Start: protocol.Position{Line: 0, Character: 0},
				End:   protocol.Position{Line: 100, Character: 0},
			},
			expected: true,
		},
		{
			name: "single character range",
			a: protocol.Range{
				Start: protocol.Position{Line: 5, Character: 10},
				End:   protocol.Position{Line: 5, Character: 11},
			},
			b: protocol.Range{
				Start: protocol.Position{Line: 5, Character: 10},
				End:   protocol.Position{Line: 5, Character: 15},
			},
			expected: true,
		},
		{
			name: "single line range vs multi-line range",
			a: protocol.Range{
				Start: protocol.Position{Line: 1, Character: 0},
				End:   protocol.Position{Line: 1, Character: 10},
			},
			b: protocol.Range{
				Start: protocol.Position{Line: 0, Character: 0},
				End:   protocol.Position{Line: 5, Character: 0},
			},
			expected: true,
		},

		// Reversed parameter order (symmetry tests)
		{
			name: "symmetry - same line overlapping (reversed)",
			a: protocol.Range{
				Start: protocol.Position{Line: 0, Character: 3},
				End:   protocol.Position{Line: 0, Character: 7},
			},
			b: protocol.Range{
				Start: protocol.Position{Line: 0, Character: 0},
				End:   protocol.Position{Line: 0, Character: 5},
			},
			expected: true,
		},
		{
			name: "symmetry - adjacent ranges (reversed)",
			a: protocol.Range{
				Start: protocol.Position{Line: 0, Character: 5},
				End:   protocol.Position{Line: 0, Character: 10},
			},
			b: protocol.Range{
				Start: protocol.Position{Line: 0, Character: 0},
				End:   protocol.Position{Line: 0, Character: 5},
			},
			expected: false,
		},

		// Real-world LSP scenarios
		{
			name: "cursor selection vs var call range - before range",
			a: protocol.Range{ // Cursor position (collapsed)
				Start: protocol.Position{Line: 5, Character: 5},
				End:   protocol.Position{Line: 5, Character: 5},
			},
			b: protocol.Range{ // var(--token) call
				Start: protocol.Position{Line: 5, Character: 8},
				End:   protocol.Position{Line: 5, Character: 25},
			},
			expected: false, // Collapsed range at character 5 doesn't intersect [8,25)
		},
		{
			name: "cursor inside var call range",
			a: protocol.Range{ // Cursor position
				Start: protocol.Position{Line: 5, Character: 15},
				End:   protocol.Position{Line: 5, Character: 16},
			},
			b: protocol.Range{ // var(--token) call
				Start: protocol.Position{Line: 5, Character: 8},
				End:   protocol.Position{Line: 5, Character: 25},
			},
			expected: true,
		},
		{
			name: "multi-line selection vs single line diagnostic",
			a: protocol.Range{ // User selection
				Start: protocol.Position{Line: 3, Character: 0},
				End:   protocol.Position{Line: 7, Character: 0},
			},
			b: protocol.Range{ // Diagnostic on line 5
				Start: protocol.Position{Line: 5, Character: 10},
				End:   protocol.Position{Line: 5, Character: 30},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := helpers.RangesIntersect(tt.a, tt.b)
			assert.Equal(t, tt.expected, result, "helpers.RangesIntersect(%+v, %+v)", tt.a, tt.b)

			// Test symmetry: RangesIntersect(a, b) should equal RangesIntersect(b, a)
			resultReversed := helpers.RangesIntersect(tt.b, tt.a)
			assert.Equal(t, result, resultReversed, "RangesIntersect should be symmetric")
		})
	}
}
