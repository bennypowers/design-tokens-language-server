package schema_test

import (
	"errors"
	"testing"

	"bennypowers.dev/dtls/internal/schema"
	"github.com/stretchr/testify/assert"
)

func TestSchemaErrors(t *testing.T) {
	t.Run("SchemaDetectionError", func(t *testing.T) {
		err := schema.NewSchemaDetectionError("test.json", "no $schema field and no config default")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "test.json")
		assert.Contains(t, err.Error(), "schema")

		// Should be unwrappable to ErrSchemaDetectionFailed
		assert.True(t, errors.Is(err, schema.ErrSchemaDetectionFailed))
	})

	t.Run("InvalidSchemaError", func(t *testing.T) {
		err := schema.NewInvalidSchemaError("test.json", "draft", "unknown schema URL")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "test.json")
		assert.Contains(t, err.Error(), "draft")

		assert.True(t, errors.Is(err, schema.ErrInvalidSchema))
	})

	t.Run("MixedSchemaFeaturesError", func(t *testing.T) {
		err := schema.NewMixedSchemaFeaturesError("test.json", "draft", []string{"$extends", "structured color objects"})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "test.json")
		assert.Contains(t, err.Error(), "draft")
		assert.Contains(t, err.Error(), "$extends")
		assert.Contains(t, err.Error(), "structured color")

		assert.True(t, errors.Is(err, schema.ErrMixedSchemaFeatures))
	})

	t.Run("ConflictingRootTokensError", func(t *testing.T) {
		err := schema.NewConflictingRootTokensError("test.json", "color", "$root", "_")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "test.json")
		assert.Contains(t, err.Error(), "color")
		assert.Contains(t, err.Error(), "$root")
		assert.Contains(t, err.Error(), "_")

		assert.True(t, errors.Is(err, schema.ErrConflictingRootTokens))
	})

	t.Run("InvalidColorFormatError", func(t *testing.T) {
		err := schema.NewInvalidColorFormatError("test.json", "color.primary", "draft", "structured object", "string value")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "test.json")
		assert.Contains(t, err.Error(), "color.primary")
		assert.Contains(t, err.Error(), "draft")

		assert.True(t, errors.Is(err, schema.ErrInvalidColorFormat))
	})

	t.Run("CircularReferenceError", func(t *testing.T) {
		err := schema.NewCircularReferenceError("test.json", []string{"color.a", "color.b", "color.a"})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "test.json")
		assert.Contains(t, err.Error(), "circular")
		assert.Contains(t, err.Error(), "color.a")
		assert.Contains(t, err.Error(), "color.b")

		assert.True(t, errors.Is(err, schema.ErrCircularReference))
	})
}

func TestErrorContextFields(t *testing.T) {
	t.Run("error includes file path", func(t *testing.T) {
		err := schema.NewSchemaDetectionError("/path/to/tokens.json", "test reason")
		assert.Contains(t, err.Error(), "/path/to/tokens.json")
	})

	t.Run("error includes schema version context", func(t *testing.T) {
		err := schema.NewMixedSchemaFeaturesError("test.json", "v2025_10", []string{"$extends"})
		assert.Contains(t, err.Error(), "v2025_10")
	})

	t.Run("error includes suggested fix", func(t *testing.T) {
		err := schema.NewSchemaDetectionError("test.json", "cannot detect schema version")
		// Should suggest adding $schema field
		assert.Contains(t, err.Error(), "$schema")
	})
}
