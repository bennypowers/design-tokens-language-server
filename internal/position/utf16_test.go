package position

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUTF16ToByteOffset(t *testing.T) {
	tests := []struct {
		name       string
		s          string
		utf16Col   int
		expectByte int
	}{
		{
			name:       "empty string",
			s:          "",
			utf16Col:   0,
			expectByte: 0,
		},
		{
			name:       "ASCII only",
			s:          "hello world",
			utf16Col:   5,
			expectByte: 5,
		},
		{
			name:       "ASCII - beyond end",
			s:          "hello",
			utf16Col:   100,
			expectByte: 5,
		},
		{
			name:       "emoji at start (surrogate pair)",
			s:          "👍 hello",
			utf16Col:   2, // Emoji counts as 2 UTF-16 units
			expectByte: 4, // Emoji is 4 bytes in UTF-8
		},
		{
			name:       "emoji in middle",
			s:          "hello 👍 world",
			utf16Col:   8,  // 6 (hello ) + 2 (👍)
			expectByte: 10, // 6 bytes + 4 bytes
		},
		{
			name:       "CJK characters (BMP)",
			s:          "颜色",
			utf16Col:   2,
			expectByte: 6, // Each CJK char is 3 bytes in UTF-8, 1 UTF-16 unit
		},
		{
			name:       "mixed emoji and CJK",
			s:          "👍颜色🎨",
			utf16Col:   6,  // 2 (👍) + 2 (颜色) + 2 (🎨)
			expectByte: 14, // 4 (👍) + 6 (颜色) + 4 (🎨)
		},
		{
			name:       "CSS variable with emoji comment",
			s:          "/* 👍 */ --color-primary",
			utf16Col:   7, // /* 👍 */  (4 + 2 + 1)
			expectByte: 9,
		},
		{
			name:       "negative offset",
			s:          "hello",
			utf16Col:   -1,
			expectByte: 0,
		},
		{
			name:       "zero offset",
			s:          "hello",
			utf16Col:   0,
			expectByte: 0,
		},
		{
			name:       "invalid UTF-8 byte sequence",
			s:          "hello\xFFworld",
			utf16Col:   7, // After "hello\xFF" (invalid byte treated as 1 unit)
			expectByte: 7,
		},
		{
			name:       "invalid UTF-8 at start",
			s:          "\xFFhello",
			utf16Col:   1, // After invalid byte
			expectByte: 1,
		},
		{
			name:       "surrogate pair boundary - clamp to start",
			s:          "👍hello",
			utf16Col:   1, // Middle of surrogate pair (between high and low)
			expectByte: 0, // Should clamp to start of emoji, not advance past it
		},
		{
			name:       "surrogate pair boundary - after emoji",
			s:          "👍hello",
			utf16Col:   2, // After full surrogate pair
			expectByte: 4, // Should be after emoji (4 UTF-8 bytes)
		},
		{
			name:       "multiple surrogates - clamp to second emoji start",
			s:          "👍🎨hello",
			utf16Col:   3, // 2 for first emoji + 1 (middle of second)
			expectByte: 4, // Should clamp to start of second emoji
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := UTF16ToByteOffset(tt.s, tt.utf16Col)
			assert.Equal(t, tt.expectByte, result)
		})
	}
}

func TestByteOffsetToUTF16(t *testing.T) {
	tests := []struct {
		name        string
		s           string
		byteOffset  int
		expectUTF16 int
	}{
		{
			name:        "empty string",
			s:           "",
			byteOffset:  0,
			expectUTF16: 0,
		},
		{
			name:        "ASCII only",
			s:           "hello world",
			byteOffset:  5,
			expectUTF16: 5,
		},
		{
			name:        "ASCII - beyond end",
			s:           "hello",
			byteOffset:  100,
			expectUTF16: 5,
		},
		{
			name:        "emoji at start",
			s:           "👍 hello",
			byteOffset:  4, // After emoji (4 bytes)
			expectUTF16: 2, // Emoji is 2 UTF-16 units
		},
		{
			name:        "emoji in middle",
			s:           "hello 👍 world",
			byteOffset:  10, // After "hello 👍"
			expectUTF16: 8,  // 6 + 2
		},
		{
			name:        "CJK characters",
			s:           "颜色",
			byteOffset:  6,
			expectUTF16: 2,
		},
		{
			name:        "negative offset",
			s:           "hello",
			byteOffset:  -1,
			expectUTF16: 0,
		},
		{
			name:        "zero offset",
			s:           "hello",
			byteOffset:  0,
			expectUTF16: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ByteOffsetToUTF16(tt.s, tt.byteOffset)
			assert.Equal(t, tt.expectUTF16, result)
		})
	}
}

func TestStringLengthUTF16(t *testing.T) {
	tests := []struct {
		name      string
		s         string
		expectLen int
	}{
		{
			name:      "empty string",
			s:         "",
			expectLen: 0,
		},
		{
			name:      "ASCII only",
			s:         "hello world",
			expectLen: 11,
		},
		{
			name:      "single emoji",
			s:         "👍",
			expectLen: 2,
		},
		{
			name:      "multiple emoji",
			s:         "👍🎨",
			expectLen: 4,
		},
		{
			name:      "CJK characters",
			s:         "颜色",
			expectLen: 2,
		},
		{
			name:      "mixed content",
			s:         "hello 👍 world",
			expectLen: 14, // 6 + 2 + 6
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := StringLengthUTF16(tt.s)
			assert.Equal(t, tt.expectLen, result)
		})
	}
}

// TestRoundTrip verifies that UTF16ToByteOffset and ByteOffsetToUTF16 are inverses
// for valid character boundaries (not in the middle of surrogate pairs)
func TestRoundTrip(t *testing.T) {
	tests := []struct {
		name      string
		s         string
		positions []int // Valid UTF-16 positions (at character boundaries)
	}{
		{
			name:      "ASCII",
			s:         "hello world",
			positions: []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11},
		},
		{
			name:      "emoji",
			s:         "👍 emoji",
			positions: []int{0, 2, 3, 4, 5, 6, 7, 8}, // Skip 1 (middle of surrogate pair)
		},
		{
			name:      "CJK",
			s:         "颜色 CJK",
			positions: []int{0, 1, 2, 3, 4, 5, 6},
		},
		{
			name:      "mixed",
			s:         "mixed 👍颜色🎨 content",
			positions: []int{0, 1, 2, 3, 4, 5, 6, 8, 9, 10, 12, 13, 14, 15, 16, 17, 18, 19}, // Skip surrogate pair middles
		},
		{
			name:      "empty",
			s:         "",
			positions: []int{0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, utf16Pos := range tt.positions {
				bytePos := UTF16ToByteOffset(tt.s, utf16Pos)
				backToUTF16 := ByteOffsetToUTF16(tt.s, bytePos)
				assert.Equal(t, utf16Pos, backToUTF16,
					"Round trip failed for position %d in string %q", utf16Pos, tt.s)
			}
		})
	}
}

func TestByteOffsetToUTF16Uint32(t *testing.T) {
	tests := []struct {
		name        string
		s           string
		byteOffset  int
		expectUTF16 uint32
	}{
		{
			name:        "empty string",
			s:           "",
			byteOffset:  0,
			expectUTF16: 0,
		},
		{
			name:        "ASCII only",
			s:           "hello world",
			byteOffset:  5,
			expectUTF16: 5,
		},
		{
			name:        "emoji at start",
			s:           "👍 hello",
			byteOffset:  4,
			expectUTF16: 2,
		},
		{
			name:        "negative offset returns 0",
			s:           "hello",
			byteOffset:  -1,
			expectUTF16: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ByteOffsetToUTF16Uint32(tt.s, tt.byteOffset)
			assert.Equal(t, tt.expectUTF16, result)
		})
	}
}

func TestStringLengthUTF16Uint32(t *testing.T) {
	tests := []struct {
		name      string
		s         string
		expectLen uint32
	}{
		{
			name:      "empty string",
			s:         "",
			expectLen: 0,
		},
		{
			name:      "ASCII only",
			s:         "hello world",
			expectLen: 11,
		},
		{
			name:      "single emoji",
			s:         "👍",
			expectLen: 2,
		},
		{
			name:      "CJK characters",
			s:         "颜色",
			expectLen: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := StringLengthUTF16Uint32(tt.s)
			assert.Equal(t, tt.expectLen, result)
		})
	}
}

// BenchmarkUTF16ToByteOffset benchmarks the UTF-16 conversion
func BenchmarkUTF16ToByteOffset(b *testing.B) {
	tests := []struct {
		name string
		s    string
		col  int
	}{
		{"ASCII", "color: var(--color-primary);", 15},
		{"WithEmoji", "/* 👍 */ color: var(--color-primary);", 20},
		{"WithCJK", "/* 颜色 */ color: var(--color-primary);", 20},
		{"Mixed", "/* 👍颜色🎨 */ color: var(--color-primary);", 25},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				UTF16ToByteOffset(tt.s, tt.col)
			}
		})
	}
}
