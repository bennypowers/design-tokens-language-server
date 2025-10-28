package json

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGroupMarkers(t *testing.T) {
	t.Run("group marker creates parent token", func(t *testing.T) {
		// Structure:
		// {
		//   "color": {
		//     "primary": {
		//       "_": { "$value": "#ff0000" },  // Group marker - should create "color.primary" token
		//       "hover": { "$value": "#cc0000" }  // Regular child - creates "color.primary.hover"
		//     }
		//   }
		// }
		jsonData := `{
  "color": {
    "primary": {
      "_": {
        "$value": "#ff0000",
        "$type": "color",
        "$description": "Primary color"
      },
      "hover": {
        "$value": "#cc0000",
        "$type": "color"
      }
    }
  }
}`

		parser := NewParser()
		tokens, err := parser.ParseWithGroupMarkers([]byte(jsonData), "test", []string{"_", "@", "DEFAULT"})
		require.NoError(t, err)

		// Should create 2 tokens:
		// 1. "color.primary" (from the "_" group marker)
		// 2. "color.primary.hover" (regular token)
		require.Len(t, tokens, 2, "Should create exactly 2 tokens")

		// Find the primary token (from group marker)
		var primaryToken *struct {
			Name        string
			Value       string
			Type        string
			Description string
		}
		var hoverToken *struct {
			Name  string
			Value string
		}

		for _, token := range tokens {
			if token.Name == "color-primary" {
				primaryToken = &struct {
					Name        string
					Value       string
					Type        string
					Description string
				}{
					Name:        token.Name,
					Value:       token.Value,
					Type:        token.Type,
					Description: token.Description,
				}
			} else if token.Name == "color-primary-hover" {
				hoverToken = &struct {
					Name  string
					Value string
				}{
					Name:  token.Name,
					Value: token.Value,
				}
			}
		}

		// The group marker "_" should create a token for the PARENT "color.primary"
		require.NotNil(t, primaryToken, "Should create 'color-primary' token from '_' group marker")
		assert.Equal(t, "color-primary", primaryToken.Name)
		assert.Equal(t, "#ff0000", primaryToken.Value)
		assert.Equal(t, "color", primaryToken.Type)
		assert.Equal(t, "Primary color", primaryToken.Description)

		// Regular child should still work
		require.NotNil(t, hoverToken, "Should create 'color-primary-hover' token")
		assert.Equal(t, "color-primary-hover", hoverToken.Name)
		assert.Equal(t, "#cc0000", hoverToken.Value)
	})

	t.Run("RHDS-style nested group markers", func(t *testing.T) {
		// RHDS structure:
		// {
		//   "color": {
		//     "interactive": {
		//       "primary": {
		//         "default": {
		//           "_": { "$value": "#0066cc" },      // Creates "color.interactive.primary.default"
		//           "on-dark": { "$value": "#92c5f9" } // Creates "color.interactive.primary.default.on-dark"
		//         }
		//       }
		//     }
		//   }
		// }
		jsonData := `{
  "color": {
    "interactive": {
      "primary": {
        "default": {
          "_": {
            "$value": "#0066cc",
            "$type": "color",
            "$description": "Primary interactive color"
          },
          "on-dark": {
            "$value": "#92c5f9",
            "$type": "color"
          }
        }
      }
    }
  }
}`

		parser := NewParser()
		tokens, err := parser.ParseWithGroupMarkers([]byte(jsonData), "rh", []string{"_"})
		require.NoError(t, err)

		// Should create 2 tokens:
		// 1. "color.interactive.primary.default" (from "_")
		// 2. "color.interactive.primary.default.on-dark"
		require.Len(t, tokens, 2)

		// Check that the token from the group marker exists
		var defaultToken, onDarkToken bool
		for _, token := range tokens {
			switch token.Name {
			case "color-interactive-primary-default":
				defaultToken = true
				assert.Equal(t, "#0066cc", token.Value)
				assert.Equal(t, "Primary interactive color", token.Description)
				// Path should be color.interactive.primary.default (NOT color.interactive.primary.default._)
				assert.Equal(t, []string{"color", "interactive", "primary", "default"}, token.Path)
			case "color-interactive-primary-default-on-dark":
				onDarkToken = true
				assert.Equal(t, "#92c5f9", token.Value)
			}
		}

		assert.True(t, defaultToken, "Should create token 'color-interactive-primary-default' from '_' group marker")
		assert.True(t, onDarkToken, "Should create token 'color-interactive-primary-default-on-dark'")
	})

	t.Run("group marker without $value - just a group", func(t *testing.T) {
		// If group marker doesn't have $value, it's just grouping children
		jsonData := `{
  "spacing": {
    "small": {
      "_": {
        "mobile": { "$value": "4px", "$type": "dimension" },
        "desktop": { "$value": "8px", "$type": "dimension" }
      }
    }
  }
}`

		parser := NewParser()
		tokens, err := parser.ParseWithGroupMarkers([]byte(jsonData), "test", []string{"_"})
		require.NoError(t, err)

		// Should NOT create a token for "spacing.small"
		// Should create tokens for the children under "_"
		require.Len(t, tokens, 2)

		var mobileFound, desktopFound bool
		for _, token := range tokens {
			if token.Name == "spacing-small-mobile" {
				mobileFound = true
				assert.Equal(t, "4px", token.Value)
			} else if token.Name == "spacing-small-desktop" {
				desktopFound = true
				assert.Equal(t, "8px", token.Value)
			}
		}

		assert.True(t, mobileFound, "Should find spacing-small-mobile")
		assert.True(t, desktopFound, "Should find spacing-small-desktop")
	})
}
