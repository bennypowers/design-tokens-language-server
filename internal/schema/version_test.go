package schema_test

import (
	"testing"

	"bennypowers.dev/dtls/internal/schema"
	"github.com/stretchr/testify/assert"
)

func TestSchemaVersion(t *testing.T) {
	t.Run("string representations", func(t *testing.T) {
		assert.Equal(t, "draft", schema.Draft.String())
		assert.Equal(t, "v2025_10", schema.V2025_10.String())
		assert.Equal(t, "unknown", schema.Unknown.String())
	})

	t.Run("schema URLs", func(t *testing.T) {
		assert.Equal(t, "https://www.designtokens.org/schemas/draft.json", schema.Draft.URL())
		assert.Equal(t, "https://www.designtokens.org/schemas/2025.10.json", schema.V2025_10.URL())
		assert.Equal(t, "", schema.Unknown.URL())
	})

	t.Run("from URL", func(t *testing.T) {
		version, err := schema.FromURL("https://www.designtokens.org/schemas/draft.json")
		assert.NoError(t, err)
		assert.Equal(t, schema.Draft, version)

		version, err = schema.FromURL("https://www.designtokens.org/schemas/2025.10.json")
		assert.NoError(t, err)
		assert.Equal(t, schema.V2025_10, version)

		version, err = schema.FromURL("https://unknown.com/schema.json")
		assert.Error(t, err)
		assert.Equal(t, schema.Unknown, version)
	})

	t.Run("from string", func(t *testing.T) {
		version, err := schema.FromString("draft")
		assert.NoError(t, err)
		assert.Equal(t, schema.Draft, version)

		version, err = schema.FromString("v2025_10")
		assert.NoError(t, err)
		assert.Equal(t, schema.V2025_10, version)

		version, err = schema.FromString("invalid")
		assert.Error(t, err)
		assert.Equal(t, schema.Unknown, version)
	})
}
