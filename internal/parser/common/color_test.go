package common_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"bennypowers.dev/dtls/internal/parser/common"
	"bennypowers.dev/dtls/internal/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseColorValue(t *testing.T) {
	t.Run("parse draft color string values", func(t *testing.T) {
		testCases := []struct {
			name     string
			value    string
			expected string
		}{
			{"hex", "#FF6B35", "#FF6B35"},
			{"rgb", "rgb(255, 107, 53)", "rgb(255, 107, 53)"},
			{"rgba", "rgba(255, 107, 53, 0.8)", "rgba(255, 107, 53, 0.8)"},
			{"hsl", "hsl(16, 100%, 60%)", "hsl(16, 100%, 60%)"},
			{"hsla", "hsla(16, 100%, 60%, 0.8)", "hsla(16, 100%, 60%, 0.8)"},
			{"named", "red", "red"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				colorValue, err := common.ParseColorValue(tc.value, schema.Draft)
				require.NoError(t, err)
				assert.Equal(t, schema.Draft, colorValue.SchemaVersion())
				assert.Equal(t, tc.expected, colorValue.ToCSS())
				assert.True(t, colorValue.IsValid())
			})
		}
	})

	t.Run("parse 2025.10 structured color objects", func(t *testing.T) {
		// sRGB color
		value := map[string]interface{}{
			"colorSpace": "srgb",
			"components": []interface{}{1.0, 0.42, 0.21},
			"alpha":      1.0,
		}

		colorValue, err := common.ParseColorValue(value, schema.V2025_10)
		require.NoError(t, err)
		assert.Equal(t, schema.V2025_10, colorValue.SchemaVersion())
		assert.True(t, colorValue.IsValid())

		// Should be able to convert to CSS
		css := colorValue.ToCSS()
		assert.NotEmpty(t, css)
	})

	t.Run("parse 2025.10 color with hex field", func(t *testing.T) {
		value := map[string]interface{}{
			"colorSpace": "srgb",
			"components": []interface{}{1.0, 0.42, 0.21},
			"alpha":      1.0,
			"hex":        "#FF6B35",
		}

		colorValue, err := common.ParseColorValue(value, schema.V2025_10)
		require.NoError(t, err)
		assert.Equal(t, "#FF6B35", colorValue.ToCSS(), "should use hex field if available")
	})

	t.Run("parse 2025.10 color with none keyword", func(t *testing.T) {
		value := map[string]interface{}{
			"colorSpace": "srgb",
			"components": []interface{}{1.0, "none", 0.21},
			"alpha":      0.8,
		}

		colorValue, err := common.ParseColorValue(value, schema.V2025_10)
		require.NoError(t, err)
		assert.True(t, colorValue.IsValid())

		// Should handle "none" keyword in CSS output
		// Note: The "none" keyword relies on CSS Color Module Level 4 syntax and may have
		// limited browser support. This test asserts preservation for spec compliance,
		// but consumers should verify Level 4 compatibility when using colorValue.ToCSS()
		// output in production or UIs.
		css := colorValue.ToCSS()
		assert.NotEmpty(t, css)
		assert.Contains(t, css, "none", "none keyword should be preserved in CSS output")
	})

	t.Run("error on draft schema with structured color", func(t *testing.T) {
		value := map[string]interface{}{
			"colorSpace": "srgb",
			"components": []interface{}{1.0, 0.42, 0.21},
		}

		_, err := common.ParseColorValue(value, schema.Draft)
		assert.Error(t, err)
		assert.ErrorIs(t, err, schema.ErrInvalidColorFormat)
	})

	t.Run("error on 2025.10 schema with string color", func(t *testing.T) {
		_, err := common.ParseColorValue("#FF6B35", schema.V2025_10)
		assert.Error(t, err)
		assert.ErrorIs(t, err, schema.ErrInvalidColorFormat)
	})
}

func TestColorValueFromFixture(t *testing.T) {
	t.Run("parse all draft color formats from fixture", func(t *testing.T) {
		content, err := os.ReadFile(filepath.Join("..", "..", "..", "test", "fixtures", "color", "draft-colors.json"))
		require.NoError(t, err)

		var data map[string]interface{}
		require.NoError(t, json.Unmarshal(content, &data))

		colors := data["color"].(map[string]interface{})

		for name, tokenData := range colors {
			t.Run(name, func(t *testing.T) {
				token := tokenData.(map[string]interface{})
				value := token["$value"]

				colorValue, err := common.ParseColorValue(value, schema.Draft)
				require.NoError(t, err, "failed to parse %s", name)
				assert.NotEmpty(t, colorValue.ToCSS())
			})
		}
	})

	t.Run("parse all 2025.10 color formats from fixture", func(t *testing.T) {
		content, err := os.ReadFile(filepath.Join("..", "..", "..", "test", "fixtures", "color", "2025-colors.json"))
		require.NoError(t, err)

		var data map[string]interface{}
		require.NoError(t, json.Unmarshal(content, &data))

		colors := data["color"].(map[string]interface{})

		for name, tokenData := range colors {
			t.Run(name, func(t *testing.T) {
				token := tokenData.(map[string]interface{})
				value := token["$value"]

				colorValue, err := common.ParseColorValue(value, schema.V2025_10)
				require.NoError(t, err, "failed to parse %s", name)
				assert.NotEmpty(t, colorValue.ToCSS())
			})
		}
	})
}
