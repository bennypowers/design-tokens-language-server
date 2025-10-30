package helpers_test

import (
	"math"
	"testing"

	"bennypowers.dev/dtls/lsp/helpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	sitter "github.com/tree-sitter/go-tree-sitter"
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

// TestPositionToUTF16 tests the PositionToUTF16 function for correct UTF-16 conversion and overflow handling
func TestPositionToUTF16(t *testing.T) {
	t.Run("overflow detection - row exceeds uint32", func(t *testing.T) {
		source := "test"
		point := sitter.Point{
			Row:    math.MaxUint32 + 1,
			Column: 0,
		}

		_, err := helpers.PositionToUTF16(source, point)
		require.Error(t, err, "Should return error when row exceeds uint32")
		assert.Contains(t, err.Error(), "position overflow", "Error should mention overflow")
	})

	t.Run("overflow detection - column exceeds uint32", func(t *testing.T) {
		source := "test"
		point := sitter.Point{
			Row:    0,
			Column: math.MaxUint32 + 1,
		}

		_, err := helpers.PositionToUTF16(source, point)
		require.Error(t, err, "Should return error when column exceeds uint32")
		assert.Contains(t, err.Error(), "position overflow", "Error should mention overflow")
	})

	t.Run("normal ASCII text", func(t *testing.T) {
		source := "hello world\nfoo bar"
		point := sitter.Point{
			Row:    1,
			Column: 4,
		}

		pos, err := helpers.PositionToUTF16(source, point)
		require.NoError(t, err)
		assert.Equal(t, uint32(1), pos.Line, "Line should match row")
		assert.Equal(t, uint32(4), pos.Character, "Character should match column for ASCII")
	})

	t.Run("UTF-8 text with multibyte characters", func(t *testing.T) {
		source := "hello 世界\nfoo"
		// "世" is 3 bytes in UTF-8 (U+4E16), "界" is 3 bytes (U+754C)
		// Position at byte 10 = "hello " (6 bytes) + "世" (3 bytes) + 1 byte into "界"
		point := sitter.Point{
			Row:    0,
			Column: 10, // Byte offset
		}

		pos, err := helpers.PositionToUTF16(source, point)
		require.NoError(t, err)
		assert.Equal(t, uint32(0), pos.Line)
		// "hello " = 6 UTF-16 units, "世" = 1 UTF-16 unit, partial "界" = 1 UTF-16 unit → total 8
		assert.Equal(t, uint32(8), pos.Character, "Should convert to UTF-16 units")
	})

	t.Run("emoji (surrogate pairs)", func(t *testing.T) {
		source := "hello 😀 world"
		// 😀 is U+1F600, requires surrogate pair in UTF-16 (2 units), 4 bytes in UTF-8
		// Position at byte 10 = "hello " (6 bytes) + "😀" (4 bytes) = just after emoji
		point := sitter.Point{
			Row:    0,
			Column: 10, // Byte offset right after emoji
		}

		pos, err := helpers.PositionToUTF16(source, point)
		require.NoError(t, err)
		assert.Equal(t, uint32(0), pos.Line)
		// "hello " = 6 UTF-16 units, "😀" = 2 UTF-16 units → total 8
		assert.Equal(t, uint32(8), pos.Character, "Emoji should count as 2 UTF-16 units (surrogate pair)")
	})

	t.Run("position beyond line length", func(t *testing.T) {
		source := "short"
		point := sitter.Point{
			Row:    0,
			Column: 100, // Beyond line length
		}

		pos, err := helpers.PositionToUTF16(source, point)
		require.NoError(t, err)
		// Should clamp to line length
		assert.Equal(t, uint32(0), pos.Line)
		assert.Equal(t, uint32(5), pos.Character, "Should clamp to line length")
	})

	t.Run("row beyond source lines", func(t *testing.T) {
		source := "line1\nline2"
		point := sitter.Point{
			Row:    10, // Beyond available lines
			Column: 5,
		}

		pos, err := helpers.PositionToUTF16(source, point)
		require.NoError(t, err)
		// Should return position as-is when row is out of bounds
		assert.Equal(t, uint32(10), pos.Line)
		assert.Equal(t, uint32(5), pos.Character)
	})

	t.Run("zero position", func(t *testing.T) {
		source := "test"
		point := sitter.Point{
			Row:    0,
			Column: 0,
		}

		pos, err := helpers.PositionToUTF16(source, point)
		require.NoError(t, err)
		assert.Equal(t, uint32(0), pos.Line)
		assert.Equal(t, uint32(0), pos.Character)
	})

	t.Run("max valid uint32 position", func(t *testing.T) {
		source := "test"
		point := sitter.Point{
			Row:    math.MaxUint32,
			Column: math.MaxUint32,
		}

		pos, err := helpers.PositionToUTF16(source, point)
		require.NoError(t, err, "Max uint32 values should be valid")
		assert.Equal(t, uint32(math.MaxUint32), pos.Line)
	})
}
