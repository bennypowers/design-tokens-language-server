package lsp

import (
	"testing"

	protocol "github.com/tliron/glsp/protocol_3_16"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGetWordAtPosition tests the getWordAtPosition helper function
func TestGetWordAtPosition(t *testing.T) {
	s, err := NewServer()
	require.NoError(t, err)

	tests := []struct {
		name     string
		content  string
		position protocol.Position
		expected string
	}{
		{
			name:     "word at start of line",
			content:  "color-primary: #ff0000;",
			position: protocol.Position{Line: 0, Character: 5},
			expected: "color-primary",
		},
		{
			name:     "word in middle of line",
			content:  "  var(--color-primary, #ff0000);",
			position: protocol.Position{Line: 0, Character: 12},
			expected: "--color-primary",
		},
		{
			name:     "cursor at end of word",
			content:  "spacing-large",
			position: protocol.Position{Line: 0, Character: 13},
			expected: "spacing-large",
		},
		{
			name:     "cursor on whitespace before word",
			content:  "  color-primary",
			position: protocol.Position{Line: 0, Character: 1},
			expected: "", // cursor is on space, not touching the word
		},
		{
			name:     "cursor at start of word",
			content:  "color-primary",
			position: protocol.Position{Line: 0, Character: 0},
			expected: "color-primary",
		},
		{
			name:     "empty line",
			content:  "",
			position: protocol.Position{Line: 0, Character: 0},
			expected: "",
		},
		{
			name:     "position out of bounds",
			content:  "color",
			position: protocol.Position{Line: 5, Character: 0},
			expected: "",
		},
		{
			name:     "word with underscores",
			content:  "color_primary_500",
			position: protocol.Position{Line: 0, Character: 8},
			expected: "color_primary_500",
		},
		{
			name:     "word with numbers",
			content:  "spacing-16px",
			position: protocol.Position{Line: 0, Character: 10},
			expected: "spacing-16px",
		},
		{
			name:     "multiline content",
			content:  "line1\ncolor-primary\nline3",
			position: protocol.Position{Line: 1, Character: 5},
			expected: "color-primary",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := s.getWordAtPosition(tt.content, tt.position)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestIsWordChar tests the isWordChar helper function
func TestIsWordChar(t *testing.T) {
	tests := []struct {
		name     string
		char     byte
		expected bool
	}{
		{"lowercase letter", 'a', true},
		{"uppercase letter", 'Z', true},
		{"digit", '5', true},
		{"hyphen", '-', true},
		{"underscore", '_', true},
		{"space", ' ', false},
		{"dot", '.', false},
		{"colon", ':', false},
		{"semicolon", ';', false},
		{"comma", ',', false},
		{"paren", '(', false},
		{"bracket", '{', false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isWordChar(tt.char)
			assert.Equal(t, tt.expected, result, "character: %c (%d)", tt.char, tt.char)
		})
	}
}

// TestIsInCompletionContext tests the isInCompletionContext helper function
func TestIsInCompletionContext(t *testing.T) {
	s, err := NewServer()
	require.NoError(t, err)

	// For now, this function always returns true
	// It's a placeholder for future enhancement
	result := s.isInCompletionContext(nil, protocol.Position{Line: 0, Character: 0})
	assert.True(t, result)
}

// TestNormalizeTokenName tests the normalizeTokenName helper function
func TestNormalizeTokenName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "CSS variable with dashes",
			input:    "--color-primary",
			expected: "colorprimary",
		},
		{
			name:     "token name without prefix",
			input:    "color-primary",
			expected: "colorprimary",
		},
		{
			name:     "uppercase token name",
			input:    "COLOR-PRIMARY",
			expected: "colorprimary",
		},
		{
			name:     "mixed case with dashes",
			input:    "--Color-Primary-500",
			expected: "colorprimary500",
		},
		{
			name:     "token with multiple hyphens",
			input:    "--spacing-large-xl",
			expected: "spacinglargexl",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "just dashes",
			input:    "--",
			expected: "",
		},
		{
			name:     "single word",
			input:    "primary",
			expected: "primary",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeTokenName(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
