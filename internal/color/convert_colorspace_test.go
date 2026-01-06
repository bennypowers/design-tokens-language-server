package color

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

func TestAdvancedColorSpaceConversions(t *testing.T) {
	// Load fixture file
	fixturePath := filepath.Join("..", "..", "test", "fixtures", "color", "colorspace-advanced.json")
	content, err := os.ReadFile(fixturePath)
	require.NoError(t, err, "Failed to read fixture file")

	// Parse the fixture
	var root map[string]any
	err = json.Unmarshal(content, &root)
	require.NoError(t, err, "Failed to parse fixture")

	tests := []struct {
		name           string
		tokenPath      []string
		expectedCSS    string
		expectedFormat string // The format we expect (e.g., "hwb", "oklab")
	}{
		// HWB tests
		{
			name:           "HWB red opaque",
			tokenPath:      []string{"color", "hwb", "red"},
			expectedCSS:    "hwb(0.0 0.0% 0.0%)",
			expectedFormat: "hwb",
		},
		{
			name:           "HWB with alpha",
			tokenPath:      []string{"color", "hwb", "transparent"},
			expectedCSS:    "hwb(120.0 20.0% 30.0% / 0.50)",
			expectedFormat: "hwb",
		},

		// OKLAB tests
		{
			name:           "OKLAB green opaque",
			tokenPath:      []string{"color", "oklab", "green"},
			expectedCSS:    "oklab(0.50 -0.10 0.20)",
			expectedFormat: "oklab",
		},
		{
			name:           "OKLAB semitransparent",
			tokenPath:      []string{"color", "oklab", "semitransparent"},
			expectedCSS:    "oklab(0.70 0.05 -0.15 / 0.75)",
			expectedFormat: "oklab",
		},

		// OKLCH tests
		{
			name:           "OKLCH primary opaque",
			tokenPath:      []string{"color", "oklch", "primary"},
			expectedCSS:    "oklch(0.65 0.18 240.0)",
			expectedFormat: "oklch",
		},
		{
			name:           "OKLCH with alpha",
			tokenPath:      []string{"color", "oklch", "accent"},
			expectedCSS:    "oklch(0.80 0.12 120.0 / 0.90)",
			expectedFormat: "oklch",
		},

		// LCH tests
		{
			name:           "LCH vibrant opaque",
			tokenPath:      []string{"color", "lch", "vibrant"},
			expectedCSS:    "lch(60.0 80.0 300.0)",
			expectedFormat: "lch",
		},
		{
			name:           "LCH muted with alpha",
			tokenPath:      []string{"color", "lch", "muted"},
			expectedCSS:    "lch(45.0 40.0 180.0 / 0.60)",
			expectedFormat: "lch",
		},

		// LAB tests
		{
			name:           "LAB bright opaque",
			tokenPath:      []string{"color", "lab", "bright"},
			expectedCSS:    "lab(75.0 25.0 -50.0)",
			expectedFormat: "lab",
		},
		{
			name:           "LAB dark with alpha",
			tokenPath:      []string{"color", "lab", "dark"},
			expectedCSS:    "lab(30.0 -15.0 20.0 / 0.85)",
			expectedFormat: "lab",
		},

		// color() function tests
		{
			name:           "display-p3",
			tokenPath:      []string{"color", "displayP3"},
			expectedCSS:    "color(display-p3 1.0000 0.5000 0.2000)",
			expectedFormat: "display-p3",
		},
		{
			name:           "srgb-linear",
			tokenPath:      []string{"color", "srgbLinear"},
			expectedCSS:    "color(srgb-linear 0.8000 0.6000 0.4000)",
			expectedFormat: "srgb-linear",
		},
		{
			name:           "a98-rgb",
			tokenPath:      []string{"color", "a98rgb"},
			expectedCSS:    "color(a98-rgb 0.9000 0.7000 0.5000)",
			expectedFormat: "a98-rgb",
		},
		{
			name:           "prophoto-rgb",
			tokenPath:      []string{"color", "prophotoRgb"},
			expectedCSS:    "color(prophoto-rgb 0.8500 0.6500 0.4500)",
			expectedFormat: "prophoto-rgb",
		},
		{
			name:           "rec2020",
			tokenPath:      []string{"color", "rec2020"},
			expectedCSS:    "color(rec2020 0.9500 0.7500 0.5500)",
			expectedFormat: "rec2020",
		},
		{
			name:           "xyz-d50",
			tokenPath:      []string{"color", "xyzD50"},
			expectedCSS:    "color(xyz-d50 0.4000 0.3000 0.2000)",
			expectedFormat: "xyz-d50",
		},
		{
			name:           "xyz-d65",
			tokenPath:      []string{"color", "xyzD65"},
			expectedCSS:    "color(xyz-d65 0.4500 0.3500 0.2500)",
			expectedFormat: "xyz-d65",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Navigate to the token in the fixture
			tokenData := navigateToPath(root, tt.tokenPath)
			require.NotNil(t, tokenData, "Failed to navigate to token path %v", tt.tokenPath)

			// Extract $value
			tokenMap, ok := tokenData.(map[string]any)
			require.True(t, ok, "Token should be a map")
			colorValue, ok := tokenMap["$value"].(map[string]any)
			require.True(t, ok, "Missing or invalid $value node")

			// Verify colorSpace
			colorSpace, ok := colorValue["colorSpace"].(string)
			require.True(t, ok, "Missing or invalid colorSpace")
			assert.Equal(t, tt.expectedFormat, colorSpace, "Unexpected color space")

			// Test ToCSS conversion
			parsedColor, err := common.ParseColorValue(colorValue, schema.V2025_10)
			require.NoError(t, err, "Failed to parse color value")

			css := ToCSS(parsedColor)
			assert.Equal(t, tt.expectedCSS, css, "CSS output mismatch")
		})
	}
}

func TestColorSpaceEdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		colorValue  map[string]any
		expectedCSS string
	}{
		{
			name: "HWB with insufficient components",
			colorValue: map[string]any{
				"colorSpace": "hwb",
				"components": []any{120.0},
				"alpha":      1.0,
			},
			expectedCSS: "",
		},
		{
			name: "OKLAB with insufficient components",
			colorValue: map[string]any{
				"colorSpace": "oklab",
				"components": []any{0.5, 0.2},
				"alpha":      1.0,
			},
			expectedCSS: "",
		},
		{
			name: "OKLCH with none keyword converts to 0",
			colorValue: map[string]any{
				"colorSpace": "oklch",
				"components": []any{0.65, "none", 240.0},
				"alpha":      1.0,
			},
			expectedCSS: "oklch(0.65 0.00 240.0)",
		},
		{
			name: "LCH with none keyword converts to 0",
			colorValue: map[string]any{
				"colorSpace": "lch",
				"components": []any{60.0, "none", 180.0},
				"alpha":      1.0,
			},
			expectedCSS: "lch(60.0 0.0 180.0)",
		},
		{
			name: "LAB with none keyword converts to 0",
			colorValue: map[string]any{
				"colorSpace": "lab",
				"components": []any{75.0, "none", -50.0},
				"alpha":      1.0,
			},
			expectedCSS: "lab(75.0 0.0 -50.0)",
		},
		{
			name: "color() function with none",
			colorValue: map[string]any{
				"colorSpace": "display-p3",
				"components": []any{1.0, "none", 0.2},
				"alpha":      1.0,
			},
			expectedCSS: "color(display-p3 1.0000 none 0.2000)",
		},
		{
			name: "Unknown color space uses color() function",
			colorValue: map[string]any{
				"colorSpace": "unknown-space",
				"components": []any{0.5, 0.5, 0.5},
				"alpha":      1.0,
			},
			expectedCSS: "color(unknown-space 0.5000 0.5000 0.5000)", // Lets browser handle unknown spaces
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsedColor, err := common.ParseColorValue(tt.colorValue, schema.V2025_10)
			require.NoError(t, err, "Failed to parse color value")

			css := ToCSS(parsedColor)
			assert.Equal(t, tt.expectedCSS, css)
		})
	}
}

// navigateToPath navigates through a JSON map following the path
func navigateToPath(data any, path []string) any {
	current := data
	for _, segment := range path {
		m, ok := current.(map[string]any)
		if !ok {
			return nil
		}
		next, exists := m[segment]
		if !exists {
			return nil
		}
		current = next
	}
	return current
}
