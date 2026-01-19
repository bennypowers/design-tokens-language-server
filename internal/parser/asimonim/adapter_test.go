package asimonim

import (
	"testing"

	"bennypowers.dev/dtls/internal/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseWithSchemaVersion(t *testing.T) {
	t.Run("parse draft schema tokens", func(t *testing.T) {
		data := []byte(`{
			"$schema": "https://www.designtokens.org/schemas/draft.json",
			"color": {
				"primary": {
					"$type": "color",
					"$value": "#ff0000",
					"$description": "Primary brand color"
				}
			}
		}`)

		tokens, err := ParseWithSchemaVersion(data, "test", schema.Draft, nil)

		require.NoError(t, err)
		require.Len(t, tokens, 1)

		token := tokens[0]
		assert.Equal(t, "color-primary", token.Name)
		assert.Equal(t, "#ff0000", token.Value)
		assert.Equal(t, "color", token.Type)
		assert.Equal(t, "Primary brand color", token.Description)
		assert.Equal(t, "test", token.Prefix)
		assert.Equal(t, schema.Draft, token.SchemaVersion)
	})

	t.Run("parse v2025_10 schema tokens", func(t *testing.T) {
		data := []byte(`{
			"$schema": "https://www.designtokens.org/schemas/2025.10.json",
			"spacing": {
				"small": {
					"$type": "dimension",
					"$value": "8px"
				}
			}
		}`)

		tokens, err := ParseWithSchemaVersion(data, "ds", schema.V2025_10, nil)

		require.NoError(t, err)
		require.Len(t, tokens, 1)

		token := tokens[0]
		assert.Equal(t, "spacing-small", token.Name)
		assert.Equal(t, "8px", token.Value)
		assert.Equal(t, "dimension", token.Type)
		assert.Equal(t, "ds", token.Prefix)
		assert.Equal(t, schema.V2025_10, token.SchemaVersion)
	})

	t.Run("parse multiple tokens", func(t *testing.T) {
		data := []byte(`{
			"color": {
				"red": {"$type": "color", "$value": "#ff0000"},
				"green": {"$type": "color", "$value": "#00ff00"},
				"blue": {"$type": "color", "$value": "#0000ff"}
			}
		}`)

		tokens, err := ParseWithSchemaVersion(data, "", schema.Draft, nil)

		require.NoError(t, err)
		assert.Len(t, tokens, 3)
	})

	t.Run("parse with group markers", func(t *testing.T) {
		data := []byte(`{
			"button": {
				"$value": "#000000",
				"$type": "color",
				"hover": {
					"$value": "#111111",
					"$type": "color"
				}
			}
		}`)

		tokens, err := ParseWithSchemaVersion(data, "", schema.Draft, []string{"button"})

		require.NoError(t, err)
		assert.Len(t, tokens, 2)
	})

	t.Run("empty input returns empty slice", func(t *testing.T) {
		data := []byte(`{}`)

		tokens, err := ParseWithSchemaVersion(data, "", schema.Draft, nil)

		require.NoError(t, err)
		assert.Empty(t, tokens)
	})
}

func TestParseWithOptions(t *testing.T) {
	t.Run("skipSort option works", func(t *testing.T) {
		data := []byte(`{
			"color": {
				"zebra": {"$type": "color", "$value": "#000"},
				"apple": {"$type": "color", "$value": "#fff"}
			}
		}`)

		// With sorting (default)
		sortedTokens, err := ParseWithOptions(data, "", schema.Draft, nil, false)
		require.NoError(t, err)
		require.Len(t, sortedTokens, 2)
		// Sorted: apple comes before zebra
		assert.Equal(t, "color-apple", sortedTokens[0].Name)
		assert.Equal(t, "color-zebra", sortedTokens[1].Name)

		// Without sorting
		unsortedTokens, err := ParseWithOptions(data, "", schema.Draft, nil, true)
		require.NoError(t, err)
		require.Len(t, unsortedTokens, 2)
		// Order depends on map iteration - just verify both exist
		names := []string{unsortedTokens[0].Name, unsortedTokens[1].Name}
		assert.Contains(t, names, "color-apple")
		assert.Contains(t, names, "color-zebra")
	})
}
