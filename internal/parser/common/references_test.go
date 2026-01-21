package common_test

import (
	"testing"

	"bennypowers.dev/dtls/internal/parser/common"
	"github.com/stretchr/testify/assert"
)

func TestFindReferenceAtPosition(t *testing.T) {
	t.Run("finds curly brace reference at position", func(t *testing.T) {
		content := `{
  "color": {
    "primary": {
      "$type": "color",
      "$value": "{color.base}"
    }
  }
}`
		// Line 4 (0-indexed), character position within "{color.base}"
		ref := common.FindReferenceAtPosition(content, 4, 18)
		assert.NotNil(t, ref)
		assert.Equal(t, "color-base", ref.TokenName)
		assert.Equal(t, "color.base", ref.RawReference)
		assert.Equal(t, common.CurlyBraceReference, ref.Type)
	})

	t.Run("finds JSON pointer reference at position", func(t *testing.T) {
		content := `{
  "color": {
    "primary": {
      "$ref": "#/color/base"
    }
  }
}`
		// Line 3, within the $ref value
		ref := common.FindReferenceAtPosition(content, 3, 18)
		assert.NotNil(t, ref)
		assert.Equal(t, "color-base", ref.TokenName)
		assert.Equal(t, "#/color/base", ref.RawReference)
		assert.Equal(t, common.JSONPointerReference, ref.Type)
	})

	t.Run("returns nil when no reference at position", func(t *testing.T) {
		content := `{
  "color": {
    "primary": {
      "$type": "color",
      "$value": "#FF6B35"
    }
  }
}`
		// Line 4, within the hex value (not a reference)
		ref := common.FindReferenceAtPosition(content, 4, 18)
		assert.Nil(t, ref)
	})

	t.Run("returns nil for out of bounds line", func(t *testing.T) {
		content := `{"color": {"$value": "{color.base}"}}`
		ref := common.FindReferenceAtPosition(content, 100, 0)
		assert.Nil(t, ref)
	})

	t.Run("handles CRLF line endings", func(t *testing.T) {
		content := "{\r\n  \"$value\": \"{color.base}\"\r\n}"
		ref := common.FindReferenceAtPosition(content, 1, 18)
		assert.NotNil(t, ref)
		assert.Equal(t, "color-base", ref.TokenName)
	})

	t.Run("handles nested path reference", func(t *testing.T) {
		content := `{"$value": "{color.brand.primary.base}"}`
		ref := common.FindReferenceAtPosition(content, 0, 20)
		assert.NotNil(t, ref)
		assert.Equal(t, "color-brand-primary-base", ref.TokenName)
		assert.Equal(t, "color.brand.primary.base", ref.RawReference)
	})

	t.Run("handles JSON pointer with nested path", func(t *testing.T) {
		content := `{"$ref": "#/color/brand/primary"}`
		ref := common.FindReferenceAtPosition(content, 0, 15)
		assert.NotNil(t, ref)
		assert.Equal(t, "color-brand-primary", ref.TokenName)
		assert.Equal(t, "#/color/brand/primary", ref.RawReference)
	})

	t.Run("returns nil when cursor is exactly at end of curly brace reference (half-open range)", func(t *testing.T) {
		// Content: {"$value": "{color.base}"}
		// Indices:  0123456789...
		// The reference "{color.base}" is at positions 12-24 (StartChar:12, EndChar:24)
		// With half-open range semantics, position 24 should NOT match (exclusive end)
		content := `{"$value": "{color.base}"}`
		ref := common.FindReferenceAtPosition(content, 0, 24)
		assert.Nil(t, ref, "cursor at exclusive end boundary (position 24) should return nil")
	})

	t.Run("returns reference when cursor is at last character of curly brace reference", func(t *testing.T) {
		// Position 23 is the closing brace "}", which should still be inside the range [12, 24)
		content := `{"$value": "{color.base}"}`
		ref := common.FindReferenceAtPosition(content, 0, 23)
		assert.NotNil(t, ref, "cursor at last character of reference (position 23) should return reference")
		assert.Equal(t, "color-base", ref.TokenName)
	})

	t.Run("returns nil when cursor is exactly at end of JSON pointer reference (half-open range)", func(t *testing.T) {
		// Content: {"$ref": "#/color/base"}
		// The reference starts at position 1, ends at position 23 (StartChar:1, EndChar:23)
		// With half-open semantics, position 23 should NOT match
		content := `{"$ref": "#/color/base"}`
		ref := common.FindReferenceAtPosition(content, 0, 23)
		assert.Nil(t, ref, "cursor at exclusive end boundary (position 23) should return nil")
	})

	t.Run("returns reference when cursor is at last character of JSON pointer reference", func(t *testing.T) {
		// Position 22 is the 'e' in 'base', which should still be inside the range [1, 23)
		content := `{"$ref": "#/color/base"}`
		ref := common.FindReferenceAtPosition(content, 0, 22)
		assert.NotNil(t, ref, "cursor at last character of reference (position 22) should return reference")
		assert.Equal(t, "color-base", ref.TokenName)
	})
}

func TestNormalizeLineEndings(t *testing.T) {
	t.Run("normalizes CRLF to LF", func(t *testing.T) {
		input := "line1\r\nline2\r\nline3"
		expected := "line1\nline2\nline3"
		assert.Equal(t, expected, common.NormalizeLineEndings(input))
	})

	t.Run("normalizes CR to LF", func(t *testing.T) {
		input := "line1\rline2\rline3"
		expected := "line1\nline2\nline3"
		assert.Equal(t, expected, common.NormalizeLineEndings(input))
	})

	t.Run("preserves LF", func(t *testing.T) {
		input := "line1\nline2\nline3"
		assert.Equal(t, input, common.NormalizeLineEndings(input))
	})

	t.Run("handles mixed line endings", func(t *testing.T) {
		input := "line1\r\nline2\rline3\nline4"
		expected := "line1\nline2\nline3\nline4"
		assert.Equal(t, expected, common.NormalizeLineEndings(input))
	})
}
