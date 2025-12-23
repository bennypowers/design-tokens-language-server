package color_test

import (
	"testing"

	"bennypowers.dev/dtls/internal/color"
	"bennypowers.dev/dtls/internal/parser/common"
	"bennypowers.dev/dtls/internal/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToCSS(t *testing.T) {
	t.Run("draft string colors pass through", func(t *testing.T) {
		tests := []struct {
			name     string
			input    string
			expected string
		}{
			{"hex", "#FF6B35", "#FF6B35"},
			{"rgb", "rgb(255, 107, 53)", "rgb(255, 107, 53)"},
			{"rgba", "rgba(255, 107, 53, 0.8)", "rgba(255, 107, 53, 0.8)"},
			{"hsl", "hsl(16, 100%, 60%)", "hsl(16, 100%, 60%)"},
			{"named", "rebeccapurple", "rebeccapurple"},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				colorValue, err := common.ParseColorValue(tt.input, schema.Draft)
				require.NoError(t, err)

				css := color.ToCSS(colorValue)
				assert.Equal(t, tt.expected, css)
			})
		}
	})

	t.Run("2025.10 color with hex field", func(t *testing.T) {
		value := map[string]interface{}{
			"colorSpace": "srgb",
			"components": []interface{}{1.0, 0.42, 0.21},
			"alpha":      1.0,
			"hex":        "#FF6B35",
		}

		colorValue, err := common.ParseColorValue(value, schema.V2025_10)
		require.NoError(t, err)

		css := color.ToCSS(colorValue)
		// Should use hex if available
		assert.Equal(t, "#FF6B35", css)
	})

	t.Run("2025.10 srgb without hex converts to hex", func(t *testing.T) {
		value := map[string]interface{}{
			"colorSpace": "srgb",
			"components": []interface{}{1.0, 0.42, 0.21},
			"alpha":      1.0,
		}

		colorValue, err := common.ParseColorValue(value, schema.V2025_10)
		require.NoError(t, err)

		css := color.ToCSS(colorValue)
		// Should convert to hex (or rgb if alpha < 1)
		assert.Contains(t, css, "#")
	})

	t.Run("2025.10 srgb with alpha < 1 uses rgba", func(t *testing.T) {
		value := map[string]interface{}{
			"colorSpace": "srgb",
			"components": []interface{}{1.0, 0.42, 0.21},
			"alpha":      0.8,
		}

		colorValue, err := common.ParseColorValue(value, schema.V2025_10)
		require.NoError(t, err)

		css := color.ToCSS(colorValue)
		// Should use rgba for transparency
		assert.Contains(t, css, "rgba(")
	})

	t.Run("2025.10 hsl color space", func(t *testing.T) {
		value := map[string]interface{}{
			"colorSpace": "hsl",
			"components": []interface{}{16.0, 100.0, 60.0},
			"alpha":      1.0,
		}

		colorValue, err := common.ParseColorValue(value, schema.V2025_10)
		require.NoError(t, err)

		css := color.ToCSS(colorValue)
		// Should use CSS hsl() function
		assert.Contains(t, css, "hsl(")
	})

	t.Run("2025.10 modern color space uses color() function", func(t *testing.T) {
		value := map[string]interface{}{
			"colorSpace": "oklch",
			"components": []interface{}{0.68, 0.19, 25.0},
			"alpha":      1.0,
		}

		colorValue, err := common.ParseColorValue(value, schema.V2025_10)
		require.NoError(t, err)

		css := color.ToCSS(colorValue)
		// Should use CSS color() function
		assert.Contains(t, css, "oklch(")
	})

	t.Run("2025.10 component with 'none' keyword", func(t *testing.T) {
		value := map[string]interface{}{
			"colorSpace": "srgb",
			"components": []interface{}{1.0, "none", 0.21},
			"alpha":      1.0,
		}

		colorValue, err := common.ParseColorValue(value, schema.V2025_10)
		require.NoError(t, err)

		css := color.ToCSS(colorValue)
		// Should handle 'none' keyword (treat as 0 for hex, or preserve for color())
		assert.NotEmpty(t, css)
	})
}

func TestToHex(t *testing.T) {
	t.Run("srgb to hex", func(t *testing.T) {
		value := map[string]interface{}{
			"colorSpace": "srgb",
			"components": []interface{}{1.0, 0.42, 0.21},
		}

		colorValue, err := common.ParseColorValue(value, schema.V2025_10)
		require.NoError(t, err)

		objColor, ok := colorValue.(*common.ObjectColorValue)
		require.True(t, ok)

		hex, err := color.ToHex(objColor)
		assert.NoError(t, err)
		assert.Equal(t, "#ff6b35", hex)
	})

	t.Run("components with 'none' treat as 0", func(t *testing.T) {
		value := map[string]interface{}{
			"colorSpace": "srgb",
			"components": []interface{}{1.0, "none", 0.0},
		}

		colorValue, err := common.ParseColorValue(value, schema.V2025_10)
		require.NoError(t, err)

		objColor, ok := colorValue.(*common.ObjectColorValue)
		require.True(t, ok)

		hex, err := color.ToHex(objColor)
		assert.NoError(t, err)
		// 'none' treated as 0: rgb(255, 0, 0) = #ff0000
		assert.Equal(t, "#ff0000", hex)
	})
}
