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
