package documents_test

import (
	"testing"

	"bennypowers.dev/dtls/internal/documents"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDocument(t *testing.T) {
	t.Run("creates document with correct fields", func(t *testing.T) {
		doc := documents.NewDocument("file:///test.json", "json", 1, "content")

		assert.Equal(t, "file:///test.json", doc.URI())
		assert.Equal(t, "json", doc.LanguageID())
		assert.Equal(t, 1, doc.Version())
		assert.Equal(t, "content", doc.Content())
	})

	t.Run("handles empty content", func(t *testing.T) {
		doc := documents.NewDocument("file:///empty.json", "json", 0, "")

		assert.Equal(t, "", doc.Content())
		assert.Equal(t, 0, doc.Version())
	})
}

func TestDocument_SetContent(t *testing.T) {
	t.Run("accepts newer version", func(t *testing.T) {
		doc := documents.NewDocument("file:///test.json", "json", 1, "original")

		err := doc.SetContent("updated", 2)
		require.NoError(t, err)
		assert.Equal(t, "updated", doc.Content())
		assert.Equal(t, 2, doc.Version())
	})

	t.Run("accepts same version", func(t *testing.T) {
		doc := documents.NewDocument("file:///test.json", "json", 1, "original")

		err := doc.SetContent("updated", 1)
		require.NoError(t, err)
		assert.Equal(t, "updated", doc.Content())
		assert.Equal(t, 1, doc.Version())
	})

	t.Run("rejects stale update", func(t *testing.T) {
		doc := documents.NewDocument("file:///test.json", "json", 5, "original")

		err := doc.SetContent("stale update", 3)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "stale")
		// Content should remain unchanged
		assert.Equal(t, "original", doc.Content())
		assert.Equal(t, 5, doc.Version())
	})

	t.Run("error message includes version numbers", func(t *testing.T) {
		doc := documents.NewDocument("file:///test.json", "json", 10, "original")

		err := doc.SetContent("stale", 5)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "10")
		assert.Contains(t, err.Error(), "5")
	})
}

func TestDocument_Getters(t *testing.T) {
	doc := documents.NewDocument("file:///path/to/tokens.json", "json", 42, "token content")

	t.Run("URI returns correct value", func(t *testing.T) {
		assert.Equal(t, "file:///path/to/tokens.json", doc.URI())
	})

	t.Run("LanguageID returns correct value", func(t *testing.T) {
		assert.Equal(t, "json", doc.LanguageID())
	})

	t.Run("Version returns correct value", func(t *testing.T) {
		assert.Equal(t, 42, doc.Version())
	})

	t.Run("Content returns correct value", func(t *testing.T) {
		assert.Equal(t, "token content", doc.Content())
	})
}
