package schema_test

import (
	"testing"

	"bennypowers.dev/dtls/internal/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegistry_Get(t *testing.T) {
	registry := schema.NewRegistry()

	// Should retrieve draft handler
	draftHandler, err := registry.Get(schema.Draft)
	require.NoError(t, err)
	assert.NotNil(t, draftHandler)
	assert.Equal(t, schema.Draft, draftHandler.Version())

	// Should retrieve 2025.10 handler
	v2025Handler, err := registry.Get(schema.V2025_10)
	require.NoError(t, err)
	assert.NotNil(t, v2025Handler)
	assert.Equal(t, schema.V2025_10, v2025Handler.Version())

	// Should error for unknown version
	_, err = registry.Get(schema.Unknown)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no handler registered")
}

func TestRegistry_Versions(t *testing.T) {
	registry := schema.NewRegistry()

	versions := registry.Versions()
	assert.Len(t, versions, 2, "Should have 2 registered handlers")
	assert.Contains(t, versions, schema.Draft)
	assert.Contains(t, versions, schema.V2025_10)
}

func TestDraftSchemaHandler_SupportsFeature(t *testing.T) {
	handler := &schema.DraftSchemaHandler{}

	// Draft supports curly brace references
	assert.True(t, handler.SupportsFeature("curly-brace-references"))

	// Draft does NOT support 2025.10 features
	assert.False(t, handler.SupportsFeature("json-pointer"))
	assert.False(t, handler.SupportsFeature("extends"))
	assert.False(t, handler.SupportsFeature("root"))
	assert.False(t, handler.SupportsFeature("resolution-order"))
}

func TestV2025_10SchemaHandler_SupportsFeature(t *testing.T) {
	handler := &schema.V2025_10SchemaHandler{}

	// 2025.10 supports all reference types
	assert.True(t, handler.SupportsFeature("curly-brace-references"))
	assert.True(t, handler.SupportsFeature("json-pointer"))
	assert.True(t, handler.SupportsFeature("extends"))
	assert.True(t, handler.SupportsFeature("root"))

	// Resolution order is post-MVP
	assert.False(t, handler.SupportsFeature("resolution-order"))
}

func TestDraftSchemaHandler_FormatColorForCSS(t *testing.T) {
	handler := &schema.DraftSchemaHandler{}

	// Draft colors are strings
	assert.Equal(t, "#FF0000", handler.FormatColorForCSS("#FF0000"))
	assert.Equal(t, "rgb(255, 0, 0)", handler.FormatColorForCSS("rgb(255, 0, 0)"))
	assert.Equal(t, "red", handler.FormatColorForCSS("red"))

	// Non-string returns empty
	assert.Equal(t, "", handler.FormatColorForCSS(123))
	assert.Equal(t, "", handler.FormatColorForCSS(nil))
}

func TestV2025_10SchemaHandler_FormatColorForCSS(t *testing.T) {
	handler := &schema.V2025_10SchemaHandler{}

	// 2025.10 colors with hex field
	colorWithHex := map[string]interface{}{
		"colorSpace": "srgb",
		"components": []interface{}{1.0, 0, 0},
		"alpha":      1.0,
		"hex":        "#FF0000",
	}
	assert.Equal(t, "#FF0000", handler.FormatColorForCSS(colorWithHex))

	// 2025.10 colors without hex field
	colorWithoutHex := map[string]interface{}{
		"colorSpace": "oklch",
		"components": []interface{}{0.628, 0.258, 29.234},
		"alpha":      1.0,
	}
	// Returns empty string - callers should use internal/color/convert.ToCSS() for full conversion
	// (circular dependency prevents importing color package here)
	assert.Equal(t, "", handler.FormatColorForCSS(colorWithoutHex))

	// String colors also supported in 2025.10 (for backwards compat)
	assert.Equal(t, "#0000FF", handler.FormatColorForCSS("#0000FF"))
}

func TestRegistry_Register(t *testing.T) {
	registry := schema.NewRegistry()

	// Initially, should have 2 handlers
	versions := registry.Versions()
	assert.Len(t, versions, 2)

	// Note: Testing custom handler registration would require implementing the full interface
	// The built-in handlers are already tested via Get and Versions tests
}

func TestDefaultRegistry(t *testing.T) {
	// Verify the default registry is initialized with handlers
	assert.NotNil(t, schema.DefaultRegistry)

	versions := schema.DefaultRegistry.Versions()
	assert.GreaterOrEqual(t, len(versions), 2, "Default registry should have at least 2 handlers")
}
