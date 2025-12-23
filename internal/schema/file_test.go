package schema_test

import (
	"testing"
	"time"

	"bennypowers.dev/dtls/internal/schema"
	"github.com/stretchr/testify/assert"
)

func TestTokenFile(t *testing.T) {
	t.Run("create token file", func(t *testing.T) {
		content := []byte(`{"color": {"primary": {"$type": "color", "$value": "#FF6B35"}}}`)

		file, err := schema.NewTokenFile("tokens.json", content, nil)
		assert.NoError(t, err)
		assert.NotNil(t, file)
		assert.Equal(t, "tokens.json", file.Path)
		assert.Equal(t, content, file.Content)
		assert.Equal(t, schema.Draft, file.SchemaVersion, "should default to draft")
		assert.WithinDuration(t, time.Now(), file.LoadedAt, 1*time.Second)
	})

	t.Run("detect schema version on creation", func(t *testing.T) {
		content := []byte(`{
			"$schema": "https://www.designtokens.org/schemas/2025.10.json",
			"color": {
				"primary": {
					"$type": "color",
					"$value": {"colorSpace": "srgb", "components": [1.0, 0, 0]}
				}
			}
		}`)

		file, err := schema.NewTokenFile("tokens.json", content, nil)
		assert.NoError(t, err)
		assert.Equal(t, schema.V2025_10, file.SchemaVersion)
	})

	t.Run("config overrides detection with valid content", func(t *testing.T) {
		// Config says v2025_10, content must match that schema
		content := []byte(`{
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

		config := &schema.DetectionConfig{
			DefaultVersion: schema.V2025_10,
		}

		file, err := schema.NewTokenFile("tokens.json", content, config)
		assert.NoError(t, err)
		assert.Equal(t, schema.V2025_10, file.SchemaVersion)
	})

	t.Run("config override with incompatible content fails validation", func(t *testing.T) {
		// Config says v2025_10, but content is draft-style (should fail)
		content := []byte(`{"color": {"primary": {"$type": "color", "$value": "#FF6B35"}}}`)

		config := &schema.DetectionConfig{
			DefaultVersion: schema.V2025_10,
		}

		file, err := schema.NewTokenFile("tokens.json", content, config)
		assert.Error(t, err, "should fail validation")
		assert.ErrorIs(t, err, schema.ErrInvalidColorFormat)
		assert.Nil(t, file)
	})

	t.Run("fail on invalid schema", func(t *testing.T) {
		content := []byte(`{
			"$schema": "https://www.designtokens.org/schemas/draft.json",
			"color": {
				"primary": {
					"$type": "color",
					"$value": {"colorSpace": "srgb", "components": [1.0, 0, 0]}
				}
			}
		}`)

		file, err := schema.NewTokenFile("tokens.json", content, nil)
		assert.Error(t, err, "should fail validation")
		assert.Nil(t, file)
	})
}
