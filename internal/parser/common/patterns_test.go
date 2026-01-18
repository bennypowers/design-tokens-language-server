package common_test

import (
	"testing"

	"bennypowers.dev/dtls/internal/parser/common"
	"github.com/stretchr/testify/assert"
)

func TestCurlyBraceReferenceRegexp(t *testing.T) {
	t.Run("matches simple reference", func(t *testing.T) {
		matches := common.CurlyBraceReferenceRegexp.FindAllStringSubmatch("{color.primary}", -1)
		assert.Len(t, matches, 1)
		assert.Equal(t, "color.primary", matches[0][1])
	})

	t.Run("matches multiple references", func(t *testing.T) {
		matches := common.CurlyBraceReferenceRegexp.FindAllStringSubmatch("rgb({color.r}, {color.g}, {color.b})", -1)
		assert.Len(t, matches, 3)
		assert.Equal(t, "color.r", matches[0][1])
		assert.Equal(t, "color.g", matches[1][1])
		assert.Equal(t, "color.b", matches[2][1])
	})

	t.Run("matches nested path", func(t *testing.T) {
		matches := common.CurlyBraceReferenceRegexp.FindAllStringSubmatch("{color.brand.primary.base}", -1)
		assert.Len(t, matches, 1)
		assert.Equal(t, "color.brand.primary.base", matches[0][1])
	})

	t.Run("no match for plain text", func(t *testing.T) {
		matches := common.CurlyBraceReferenceRegexp.FindAllStringSubmatch("#FF6B35", -1)
		assert.Empty(t, matches)
	})

	t.Run("no match for empty braces", func(t *testing.T) {
		matches := common.CurlyBraceReferenceRegexp.FindAllStringSubmatch("{}", -1)
		assert.Empty(t, matches)
	})

	t.Run("matches reference with hyphens", func(t *testing.T) {
		matches := common.CurlyBraceReferenceRegexp.FindAllStringSubmatch("{color-palette.brand-blue}", -1)
		assert.Len(t, matches, 1)
		assert.Equal(t, "color-palette.brand-blue", matches[0][1])
	})

	t.Run("matches reference with underscores", func(t *testing.T) {
		matches := common.CurlyBraceReferenceRegexp.FindAllStringSubmatch("{color_primary}", -1)
		assert.Len(t, matches, 1)
		assert.Equal(t, "color_primary", matches[0][1])
	})
}

func TestJSONPointerReferenceRegexp(t *testing.T) {
	t.Run("matches JSON format", func(t *testing.T) {
		matches := common.JSONPointerReferenceRegexp.FindAllStringSubmatch(`"$ref": "#/color/primary"`, -1)
		assert.Len(t, matches, 1)
		assert.Equal(t, "#/color/primary", matches[0][1])
	})

	t.Run("matches YAML format with double quotes", func(t *testing.T) {
		matches := common.JSONPointerReferenceRegexp.FindAllStringSubmatch(`$ref: "#/color/primary"`, -1)
		assert.Len(t, matches, 1)
		assert.Equal(t, "#/color/primary", matches[0][1])
	})

	t.Run("matches YAML format with single quotes", func(t *testing.T) {
		matches := common.JSONPointerReferenceRegexp.FindAllStringSubmatch(`$ref: '#/color/primary'`, -1)
		assert.Len(t, matches, 1)
		assert.Equal(t, "#/color/primary", matches[0][1])
	})

	t.Run("matches nested path", func(t *testing.T) {
		matches := common.JSONPointerReferenceRegexp.FindAllStringSubmatch(`"$ref": "#/color/brand/primary/base"`, -1)
		assert.Len(t, matches, 1)
		assert.Equal(t, "#/color/brand/primary/base", matches[0][1])
	})

	t.Run("no match without hash prefix", func(t *testing.T) {
		matches := common.JSONPointerReferenceRegexp.FindAllStringSubmatch(`"$ref": "color/primary"`, -1)
		assert.Empty(t, matches)
	})
}

func TestRootKeywordRegexp(t *testing.T) {
	t.Run("matches JSON $root", func(t *testing.T) {
		assert.True(t, common.RootKeywordRegexp.MatchString(`"$root": {`))
	})

	t.Run("matches YAML $root", func(t *testing.T) {
		assert.True(t, common.RootKeywordRegexp.MatchString(`$root:`))
	})

	t.Run("matches with spaces", func(t *testing.T) {
		assert.True(t, common.RootKeywordRegexp.MatchString(`"$root"   :`))
	})

	t.Run("no match for $rootValue", func(t *testing.T) {
		// Should not match when followed by other characters before colon
		assert.False(t, common.RootKeywordRegexp.MatchString(`$rootValue: value`))
	})
}

func TestSchemaFieldRegexp(t *testing.T) {
	t.Run("matches JSON $schema field", func(t *testing.T) {
		matches := common.SchemaFieldRegexp.FindStringSubmatch(`"$schema": "https://designtokens.org/schemas/draft.json"`)
		assert.Len(t, matches, 2)
		assert.Equal(t, "https://designtokens.org/schemas/draft.json", matches[1])
	})

	t.Run("matches YAML $schema with double quotes", func(t *testing.T) {
		matches := common.SchemaFieldRegexp.FindStringSubmatch(`$schema: "https://designtokens.org/schemas/2025.10.json"`)
		assert.Len(t, matches, 2)
		assert.Equal(t, "https://designtokens.org/schemas/2025.10.json", matches[1])
	})

	t.Run("matches YAML $schema with single quotes", func(t *testing.T) {
		matches := common.SchemaFieldRegexp.FindStringSubmatch(`$schema: 'https://designtokens.org/schemas/draft.json'`)
		assert.Len(t, matches, 2)
		assert.Equal(t, "https://designtokens.org/schemas/draft.json", matches[1])
	})

	t.Run("matches with leading whitespace", func(t *testing.T) {
		matches := common.SchemaFieldRegexp.FindStringSubmatch(`  "$schema": "https://example.com/schema.json"`)
		assert.Len(t, matches, 2)
		assert.Equal(t, "https://example.com/schema.json", matches[1])
	})

	t.Run("does not match nested $schema", func(t *testing.T) {
		// Should only match when anchored to line start
		content := `other: { "$schema": "nested" }`
		matches := common.SchemaFieldRegexp.FindStringSubmatch(content)
		assert.Empty(t, matches)
	})
}
