package lsp

import (
	"testing"

	protocol "github.com/tliron/glsp/protocol_3_16"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestParseColor tests the parseColor helper function
func TestParseColor(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    *protocol.Color
		expectError bool
	}{
		{
			name:  "6-digit hex color",
			input: "#ff0000",
			expected: &protocol.Color{
				Red:   1.0,
				Green: 0.0,
				Blue:  0.0,
				Alpha: 1.0,
			},
			expectError: false,
		},
		{
			name:  "6-digit hex color uppercase",
			input: "#FF0000",
			expected: &protocol.Color{
				Red:   1.0,
				Green: 0.0,
				Blue:  0.0,
				Alpha: 1.0,
			},
			expectError: false,
		},
		{
			name:  "6-digit hex color with whitespace",
			input: "  #00ff00  ",
			expected: &protocol.Color{
				Red:   0.0,
				Green: 1.0,
				Blue:  0.0,
				Alpha: 1.0,
			},
			expectError: false,
		},
		{
			name:  "3-digit hex color",
			input: "#f00",
			expected: &protocol.Color{
				Red:   1.0,
				Green: 0.0,
				Blue:  0.0,
				Alpha: 1.0,
			},
			expectError: false,
		},
		{
			name:  "3-digit hex color - blue",
			input: "#00f",
			expected: &protocol.Color{
				Red:   0.0,
				Green: 0.0,
				Blue:  1.0,
				Alpha: 1.0,
			},
			expectError: false,
		},
		{
			name:  "8-digit hex color with alpha",
			input: "#ff000080",
			expected: &protocol.Color{
				Red:   1.0,
				Green: 0.0,
				Blue:  0.0,
				Alpha: protocol.Decimal(128.0 / 255.0), // ~0.502
			},
			expectError: false,
		},
		{
			name:  "8-digit hex color with full alpha",
			input: "#0000ffff",
			expected: &protocol.Color{
				Red:   0.0,
				Green: 0.0,
				Blue:  1.0,
				Alpha: 1.0,
			},
			expectError: false,
		},
		{
			name:  "8-digit hex color with zero alpha",
			input: "#ff000000",
			expected: &protocol.Color{
				Red:   1.0,
				Green: 0.0,
				Blue:  0.0,
				Alpha: 0.0,
			},
			expectError: false,
		},
		{
			name:        "invalid hex - wrong length",
			input:       "#ff00",
			expected:    nil,
			expectError: true,
		},
		{
			name:        "invalid hex - non-hex characters",
			input:       "#gggggg",
			expected:    nil,
			expectError: true,
		},
		{
			name:        "rgb() format not supported yet",
			input:       "rgb(255, 0, 0)",
			expected:    nil,
			expectError: true,
		},
		{
			name:        "named color not supported",
			input:       "red",
			expected:    nil,
			expectError: true,
		},
		{
			name:        "empty string",
			input:       "",
			expected:    nil,
			expectError: true,
		},
		{
			name:        "just hash",
			input:       "#",
			expected:    nil,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseColor(tt.input)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)

				// Compare with small tolerance for floating point
				const tolerance = 0.001
				assert.InDelta(t, tt.expected.Red, result.Red, tolerance, "Red channel mismatch")
				assert.InDelta(t, tt.expected.Green, result.Green, tolerance, "Green channel mismatch")
				assert.InDelta(t, tt.expected.Blue, result.Blue, tolerance, "Blue channel mismatch")
				assert.InDelta(t, tt.expected.Alpha, result.Alpha, tolerance, "Alpha channel mismatch")
			}
		})
	}
}
