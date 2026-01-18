package schema_test

import (
	"os"
	"path/filepath"
	"testing"

	"bennypowers.dev/dtls/internal/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateSchemaConsistency(t *testing.T) {
	t.Run("valid draft schema passes", func(t *testing.T) {
		content := []byte(`{
			"$schema": "https://www.designtokens.org/schemas/draft.json",
			"color": {
				"primary": {
					"$type": "color",
					"$value": "#FF6B35"
				}
			}
		}`)

		err := schema.ValidateSchemaConsistency(content, schema.Draft)
		assert.NoError(t, err)
	})

	t.Run("valid 2025.10 schema passes", func(t *testing.T) {
		content := []byte(`{
			"$schema": "https://www.designtokens.org/schemas/2025.10.json",
			"color": {
				"primary": {
					"$type": "color",
					"$value": {
						"colorSpace": "srgb",
						"components": [1.0, 0.42, 0.21]
					}
				}
			}
		}`)

		err := schema.ValidateSchemaConsistency(content, schema.V2025_10)
		assert.NoError(t, err)
	})

	t.Run("mixed schema features fail validation", func(t *testing.T) {
		fixtureData, err := os.ReadFile(filepath.Join("..", "..", "test", "fixtures", "errors", "mixed-schema-features.json"))
		require.NoError(t, err)

		err = schema.ValidateSchemaConsistency(fixtureData, schema.Draft)
		assert.Error(t, err)
		assert.ErrorIs(t, err, schema.ErrMixedSchemaFeatures)
	})

	t.Run("draft schema with 2025.10 color objects fails", func(t *testing.T) {
		fixtureData, err := os.ReadFile(filepath.Join("..", "..", "test", "fixtures", "errors", "invalid-color-format.json"))
		require.NoError(t, err)

		err = schema.ValidateSchemaConsistency(fixtureData, schema.Draft)
		assert.Error(t, err)
		assert.ErrorIs(t, err, schema.ErrInvalidColorFormat)
	})

	t.Run("draft schema with $extends fails", func(t *testing.T) {
		fixtureData, err := os.ReadFile(filepath.Join("..", "..", "test", "fixtures", "errors", "draft-with-2025-features.json"))
		require.NoError(t, err)

		err = schema.ValidateSchemaConsistency(fixtureData, schema.Draft)
		assert.Error(t, err)
		assert.ErrorIs(t, err, schema.ErrMixedSchemaFeatures)
		assert.Contains(t, err.Error(), "$extends")
	})

	t.Run("2025.10 schema with group markers fails", func(t *testing.T) {
		fixtureData, err := os.ReadFile(filepath.Join("..", "..", "test", "fixtures", "errors", "2025-with-draft-root.json"))
		require.NoError(t, err)

		// This should fail because '_' is not $root in 2025.10
		err = schema.ValidateSchemaConsistency(fixtureData, schema.V2025_10)
		assert.Error(t, err)
	})

	t.Run("both $root and group markers fails", func(t *testing.T) {
		fixtureData, err := os.ReadFile(filepath.Join("..", "..", "test", "fixtures", "errors", "root-and-markers.json"))
		require.NoError(t, err)

		err = schema.ValidateSchemaConsistency(fixtureData, schema.V2025_10)
		assert.Error(t, err)
		assert.ErrorIs(t, err, schema.ErrConflictingRootTokens)
	})

	t.Run("2025.10 with string color values fails", func(t *testing.T) {
		content := []byte(`{
			"$schema": "https://www.designtokens.org/schemas/2025.10.json",
			"color": {
				"primary": {
					"$type": "color",
					"$value": "#FF6B35"
				}
			}
		}`)

		err := schema.ValidateSchemaConsistency(content, schema.V2025_10)
		assert.Error(t, err)
		assert.ErrorIs(t, err, schema.ErrInvalidColorFormat)
	})

	t.Run("draft with $ref fails", func(t *testing.T) {
		content := []byte(`{
			"$schema": "https://www.designtokens.org/schemas/draft.json",
			"color": {
				"primary": {
					"$type": "color",
					"$value": "#FF6B35"
				},
				"secondary": {
					"$ref": "#/color/primary"
				}
			}
		}`)

		err := schema.ValidateSchemaConsistency(content, schema.Draft)
		assert.Error(t, err)
		assert.ErrorIs(t, err, schema.ErrMixedSchemaFeatures)
		assert.Contains(t, err.Error(), "$ref")
	})
}

func TestValidateWithFilePath(t *testing.T) {
	t.Run("error includes file path in message", func(t *testing.T) {
		content := []byte(`{
			"$schema": "https://www.designtokens.org/schemas/draft.json",
			"color": {
				"primary": {
					"$ref": "#/base/color"
				}
			}
		}`)

		err := schema.ValidateSchemaConsistencyWithPath("tokens/colors.json", content, schema.Draft)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "tokens/colors.json")
	})
}
