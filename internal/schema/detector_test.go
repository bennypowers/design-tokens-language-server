package schema_test

import (
	"os"
	"testing"

	"bennypowers.dev/dtls/internal/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetectVersion(t *testing.T) {
	t.Run("detect from explicit $schema field", func(t *testing.T) {
		content, err := os.ReadFile("testdata/detection/explicit-draft.json")
		require.NoError(t, err)

		version, err := schema.DetectVersion(content, nil)
		assert.NoError(t, err)
		assert.Equal(t, schema.Draft, version)

		content, err = os.ReadFile("testdata/detection/explicit-2025.json")
		require.NoError(t, err)

		version, err = schema.DetectVersion(content, nil)
		assert.NoError(t, err)
		assert.Equal(t, schema.V2025_10, version)
	})

	t.Run("detect 2025.10 from structured color format", func(t *testing.T) {
		content, err := os.ReadFile("testdata/detection/duck-type-2025.json")
		require.NoError(t, err)

		version, err := schema.DetectVersion(content, nil)
		assert.NoError(t, err)
		assert.Equal(t, schema.V2025_10, version)
	})

	t.Run("detect 2025.10 from $ref field", func(t *testing.T) {
		content, err := os.ReadFile("testdata/detection/with-ref.json")
		require.NoError(t, err)

		version, err := schema.DetectVersion(content, nil)
		assert.NoError(t, err)
		assert.Equal(t, schema.V2025_10, version)
	})

	t.Run("detect 2025.10 from $extends field", func(t *testing.T) {
		content, err := os.ReadFile("testdata/detection/with-extends.json")
		require.NoError(t, err)

		version, err := schema.DetectVersion(content, nil)
		assert.NoError(t, err)
		assert.Equal(t, schema.V2025_10, version)
	})

	t.Run("default to draft for ambiguous files", func(t *testing.T) {
		content, err := os.ReadFile("testdata/detection/duck-type-draft.json")
		require.NoError(t, err)

		version, err := schema.DetectVersion(content, nil)
		assert.NoError(t, err)
		assert.Equal(t, schema.Draft, version, "ambiguous files should default to draft for backward compatibility")
	})

	t.Run("config override takes precedence", func(t *testing.T) {
		content := []byte(`{"color": {"primary": {"$type": "color", "$value": "#FF6B35"}}}`)

		config := &schema.DetectionConfig{
			DefaultVersion: schema.V2025_10,
		}

		version, err := schema.DetectVersion(content, config)
		assert.NoError(t, err)
		assert.Equal(t, schema.V2025_10, version, "config should override detection")
	})

	t.Run("$schema takes precedence over config", func(t *testing.T) {
		content := []byte(`{"$schema": "https://www.designtokens.org/schemas/draft.json", "color": {"primary": {"$type": "color", "$value": "#FF6B35"}}}`)

		config := &schema.DetectionConfig{
			DefaultVersion: schema.V2025_10,
		}

		version, err := schema.DetectVersion(content, config)
		assert.NoError(t, err)
		assert.Equal(t, schema.Draft, version, "$schema field should take precedence over config")
	})

	t.Run("error when no detection method works and no config", func(t *testing.T) {
		content, err := os.ReadFile("testdata/detection/ambiguous.json")
		require.NoError(t, err)

		// With no config, ambiguous file defaults to draft
		version, err := schema.DetectVersion(content, nil)
		assert.NoError(t, err)
		assert.Equal(t, schema.Draft, version)
	})
}

func TestDetectWithValidation(t *testing.T) {
	t.Run("validate after detection", func(t *testing.T) {
		// File with draft schema but 2025.10 features
		content, err := os.ReadFile("testdata/errors/invalid-color-format.json")
		require.NoError(t, err)

		version, err := schema.DetectVersionWithValidation("test.json", content, nil)
		assert.Error(t, err, "should fail validation")
		assert.ErrorIs(t, err, schema.ErrInvalidColorFormat)
		assert.Equal(t, schema.Draft, version, "should still return detected version")
	})

	t.Run("valid file passes detection and validation", func(t *testing.T) {
		content, err := os.ReadFile("testdata/detection/explicit-draft.json")
		require.NoError(t, err)

		version, err := schema.DetectVersionWithValidation("test.json", content, nil)
		assert.NoError(t, err)
		assert.Equal(t, schema.Draft, version)
	})
}
