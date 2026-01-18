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

func TestExtractReferences(t *testing.T) {
	t.Run("extract curly brace references", func(t *testing.T) {
		content := `{color.base}`

		refs, err := common.ExtractReferences(content, schema.Draft)
		require.NoError(t, err)
		require.Len(t, refs, 1)

		assert.Equal(t, common.CurlyBraceReference, refs[0].Type)
		assert.Equal(t, "color.base", refs[0].Path)
	})

	t.Run("extract multiple curly brace references", func(t *testing.T) {
		content := `rgb({color.r}, {color.g}, {color.b})`

		refs, err := common.ExtractReferences(content, schema.Draft)
		require.NoError(t, err)
		require.Len(t, refs, 3)

		assert.Equal(t, "color.r", refs[0].Path)
		assert.Equal(t, "color.g", refs[1].Path)
		assert.Equal(t, "color.b", refs[2].Path)
	})

	t.Run("extract JSON pointer reference", func(t *testing.T) {
		// JSON Pointer references use $ref field, not string interpolation
		// So this tests parsing a structured $ref
		refObj := map[string]interface{}{
			"$ref": "#/color/base",
		}

		refs, err := common.ExtractReferencesFromValue(refObj, schema.V2025_10)
		require.NoError(t, err)
		require.Len(t, refs, 1)

		assert.Equal(t, common.JSONPointerReference, refs[0].Type)
		assert.Equal(t, "color/base", refs[0].Path)
	})

	t.Run("no references in plain value", func(t *testing.T) {
		content := `#FF6B35`

		refs, err := common.ExtractReferences(content, schema.Draft)
		require.NoError(t, err)
		assert.Empty(t, refs)
	})

	t.Run("error on JSON pointer in draft schema", func(t *testing.T) {
		refObj := map[string]interface{}{
			"$ref": "#/color/base",
		}

		_, err := common.ExtractReferencesFromValue(refObj, schema.Draft)
		assert.Error(t, err)
		assert.ErrorIs(t, err, schema.ErrMixedSchemaFeatures)
	})
}

func TestExtractReferencesFromFixture(t *testing.T) {
	t.Run("extract curly brace references from fixture", func(t *testing.T) {
		content, err := os.ReadFile(filepath.Join("..", "..", "..", "test", "fixtures", "references", "curly-braces.json"))
		require.NoError(t, err)

		var data map[string]interface{}
		require.NoError(t, json.Unmarshal(content, &data))

		colorsRaw, ok := data["color"]
		require.True(t, ok, "fixture should have 'color' key")
		colors, ok := colorsRaw.(map[string]interface{})
		require.True(t, ok, "color should be a map")

		// Check "primary" token which references "base"
		primaryTokenRaw, ok := colors["primary"]
		require.True(t, ok, "fixture should have 'primary' token")
		primaryToken, ok := primaryTokenRaw.(map[string]interface{})
		require.True(t, ok, "primary should be a map")
		primaryValueRaw, ok := primaryToken["$value"]
		require.True(t, ok, "primary should have '$value' field")
		primaryValue, ok := primaryValueRaw.(string)
		require.True(t, ok, "primary $value should be a string")

		refs, err := common.ExtractReferences(primaryValue, schema.Draft)
		require.NoError(t, err)
		require.Len(t, refs, 1)
		assert.Equal(t, "color.base", refs[0].Path)

		// Check "secondary" token which references "primary"
		secondaryTokenRaw, ok := colors["secondary"]
		require.True(t, ok, "fixture should have 'secondary' token")
		secondaryToken, ok := secondaryTokenRaw.(map[string]interface{})
		require.True(t, ok, "secondary should be a map")
		secondaryValueRaw, ok := secondaryToken["$value"]
		require.True(t, ok, "secondary should have '$value' field")
		secondaryValue, ok := secondaryValueRaw.(string)
		require.True(t, ok, "secondary $value should be a string")

		refs, err = common.ExtractReferences(secondaryValue, schema.Draft)
		require.NoError(t, err)
		require.Len(t, refs, 1)
		assert.Equal(t, "color.primary", refs[0].Path)
	})

	t.Run("extract JSON pointer references from fixture", func(t *testing.T) {
		content, err := os.ReadFile(filepath.Join("..", "..", "..", "test", "fixtures", "references", "json-pointers.json"))
		require.NoError(t, err)

		var data map[string]interface{}
		require.NoError(t, json.Unmarshal(content, &data))

		colorsRaw, ok := data["color"]
		require.True(t, ok, "fixture should have 'color' key")
		colors, ok := colorsRaw.(map[string]interface{})
		require.True(t, ok, "color should be a map")

		// Check "primary" token which references "base"
		primaryTokenRaw, ok := colors["primary"]
		require.True(t, ok, "fixture should have 'primary' token")
		primaryToken, ok := primaryTokenRaw.(map[string]interface{})
		require.True(t, ok, "primary should be a map")

		refs, err := common.ExtractReferencesFromValue(primaryToken, schema.V2025_10)
		require.NoError(t, err)
		require.Len(t, refs, 1)
		assert.Equal(t, common.JSONPointerReference, refs[0].Type)
		assert.Equal(t, "color/base", refs[0].Path)
	})
}

func TestReferencePathConversion(t *testing.T) {
	t.Run("convert curly brace path to token name", func(t *testing.T) {
		refs, err := common.ExtractReferences("{color.brand.primary}", schema.Draft)
		require.NoError(t, err)
		require.Len(t, refs, 1)

		// Path should be dot-separated
		assert.Equal(t, "color.brand.primary", refs[0].Path)
	})

	t.Run("convert JSON pointer to token path", func(t *testing.T) {
		refObj := map[string]interface{}{
			"$ref": "#/color/brand/primary",
		}

		refs, err := common.ExtractReferencesFromValue(refObj, schema.V2025_10)
		require.NoError(t, err)
		require.Len(t, refs, 1)

		// Path should be slash-separated (JSON Pointer format)
		assert.Equal(t, "color/brand/primary", refs[0].Path)
	})
}
