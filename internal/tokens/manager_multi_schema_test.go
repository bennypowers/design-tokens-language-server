package tokens_test

import (
	"testing"

	"bennypowers.dev/dtls/internal/schema"
	"bennypowers.dev/dtls/internal/tokens"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestManager_MultiSchema_TrackSchemaPerToken(t *testing.T) {
	// Test that tokens from different schema versions can coexist
	manager := tokens.NewManager()

	// Add draft schema token
	draftToken := &tokens.Token{
		Name:          "color-primary",
		Value:         "#FF0000",
		SchemaVersion: schema.Draft,
		FilePath:      "draft-tokens.json",
	}
	err := manager.Add(draftToken)
	require.NoError(t, err)

	// Add 2025.10 schema token with same name from different file
	token2025 := &tokens.Token{
		Name:          "color-primary",
		Value:         "oklch(0.68 0.19 25)",
		SchemaVersion: schema.V2025_10,
		FilePath:      "2025-tokens.json",
	}
	err = manager.Add(token2025)
	require.NoError(t, err)

	// Should have 2 tokens total
	assert.Equal(t, 2, manager.Count())

	// Should be able to retrieve tokens by schema version
	draftTokens := manager.GetBySchemaVersion(schema.Draft)
	assert.Len(t, draftTokens, 1)
	assert.Equal(t, "#FF0000", draftTokens[0].Value)

	token2025s := manager.GetBySchemaVersion(schema.V2025_10)
	assert.Len(t, token2025s, 1)
	assert.Equal(t, "oklch(0.68 0.19 25)", token2025s[0].Value)
}

func TestManager_MultiSchema_GetBySourceFile(t *testing.T) {
	// Test that tokens can be retrieved by source file
	manager := tokens.NewManager()

	// Add tokens from different files
	token1 := &tokens.Token{
		Name:          "color-primary",
		Value:         "#FF0000",
		SchemaVersion: schema.Draft,
		FilePath:      "draft-tokens.json",
	}
	token2 := &tokens.Token{
		Name:          "spacing-base",
		Value:         "16px",
		SchemaVersion: schema.Draft,
		FilePath:      "draft-tokens.json",
	}
	token3 := &tokens.Token{
		Name:          "color-success",
		Value:         "oklch(0.68 0.19 145)",
		SchemaVersion: schema.V2025_10,
		FilePath:      "2025-tokens.json",
	}

	require.NoError(t, manager.Add(token1))
	require.NoError(t, manager.Add(token2))
	require.NoError(t, manager.Add(token3))

	// Get tokens from draft file
	draftFileTokens := manager.GetBySourceFile("draft-tokens.json")
	assert.Len(t, draftFileTokens, 2)

	// Get tokens from 2025.10 file
	token2025FileTokens := manager.GetBySourceFile("2025-tokens.json")
	assert.Len(t, token2025FileTokens, 1)
	assert.Equal(t, "color-success", token2025FileTokens[0].Name)
}

func TestManager_MultiSchema_QualifiedLookup(t *testing.T) {
	// Test that ambiguous token names can be resolved by file path
	manager := tokens.NewManager()

	// Add two tokens with same name from different files
	draftToken := &tokens.Token{
		Name:          "color-primary",
		Value:         "#FF0000",
		SchemaVersion: schema.Draft,
		FilePath:      "draft-tokens.json",
	}
	token2025 := &tokens.Token{
		Name:          "color-primary",
		Value:         "oklch(0.68 0.19 25)",
		SchemaVersion: schema.V2025_10,
		FilePath:      "2025-tokens.json",
	}

	require.NoError(t, manager.Add(draftToken))
	require.NoError(t, manager.Add(token2025))

	// Qualified lookup by file path
	token := manager.GetQualified("color-primary", "draft-tokens.json")
	require.NotNil(t, token)
	assert.Equal(t, "#FF0000", token.Value)
	assert.Equal(t, schema.Draft, token.SchemaVersion)

	token = manager.GetQualified("color-primary", "2025-tokens.json")
	require.NotNil(t, token)
	assert.Equal(t, "oklch(0.68 0.19 25)", token.Value)
	assert.Equal(t, schema.V2025_10, token.SchemaVersion)
}

func TestManager_MultiSchema_GetReturnsFirstMatch(t *testing.T) {
	// Test that Get() returns first match when multiple tokens have same name
	manager := tokens.NewManager()

	// Add two tokens with same name
	token1 := &tokens.Token{
		Name:          "color-primary",
		Value:         "#FF0000",
		SchemaVersion: schema.Draft,
		FilePath:      "file1.json",
	}
	token2 := &tokens.Token{
		Name:          "color-primary",
		Value:         "#00FF00",
		SchemaVersion: schema.V2025_10,
		FilePath:      "file2.json",
	}

	require.NoError(t, manager.Add(token1))
	require.NoError(t, manager.Add(token2))

	// Get should return one of them (behavior is undefined but shouldn't crash)
	token := manager.Get("color-primary")
	require.NotNil(t, token)
	assert.Contains(t, []string{"#FF0000", "#00FF00"}, token.Value)
}

func TestManager_MultiSchema_RemoveByFilePath(t *testing.T) {
	// Test removing all tokens from a specific file
	manager := tokens.NewManager()

	// Add tokens from two files
	token1 := &tokens.Token{
		Name:          "color-primary",
		Value:         "#FF0000",
		SchemaVersion: schema.Draft,
		FilePath:      "draft-tokens.json",
	}
	token2 := &tokens.Token{
		Name:          "spacing-base",
		Value:         "16px",
		SchemaVersion: schema.Draft,
		FilePath:      "draft-tokens.json",
	}
	token3 := &tokens.Token{
		Name:          "color-success",
		Value:         "oklch(0.68 0.19 145)",
		SchemaVersion: schema.V2025_10,
		FilePath:      "2025-tokens.json",
	}

	require.NoError(t, manager.Add(token1))
	require.NoError(t, manager.Add(token2))
	require.NoError(t, manager.Add(token3))

	assert.Equal(t, 3, manager.Count())

	// Remove tokens from draft file
	removed := manager.RemoveBySourceFile("draft-tokens.json")
	assert.Equal(t, 2, removed, "Should remove 2 tokens")
	assert.Equal(t, 1, manager.Count(), "Should have 1 token remaining")

	// Verify only 2025 token remains
	remaining := manager.GetAll()
	require.Len(t, remaining, 1)
	assert.Equal(t, "color-success", remaining[0].Name)
	assert.Equal(t, schema.V2025_10, remaining[0].SchemaVersion)
}

func TestManager_MultiSchema_GetSourceFiles(t *testing.T) {
	// Test getting list of all source files
	manager := tokens.NewManager()

	// Add tokens from multiple files
	token1 := &tokens.Token{
		Name:          "color-primary",
		Value:         "#FF0000",
		SchemaVersion: schema.Draft,
		FilePath:      "draft-tokens.json",
	}
	token2 := &tokens.Token{
		Name:          "color-success",
		Value:         "oklch(0.68 0.19 145)",
		SchemaVersion: schema.V2025_10,
		FilePath:      "2025-tokens.json",
	}
	token3 := &tokens.Token{
		Name:          "spacing-base",
		Value:         "16px",
		SchemaVersion: schema.Draft,
		FilePath:      "draft-tokens.json",
	}

	require.NoError(t, manager.Add(token1))
	require.NoError(t, manager.Add(token2))
	require.NoError(t, manager.Add(token3))

	// Get list of source files
	files := manager.GetSourceFiles()
	assert.Len(t, files, 2)
	assert.Contains(t, files, "draft-tokens.json")
	assert.Contains(t, files, "2025-tokens.json")
}

func TestManager_MultiSchema_GetSchemaVersionForFile(t *testing.T) {
	// Test getting the schema version for a specific file
	manager := tokens.NewManager()

	// Add tokens from files with different schemas
	draftToken := &tokens.Token{
		Name:          "color-primary",
		Value:         "#FF0000",
		SchemaVersion: schema.Draft,
		FilePath:      "draft-tokens.json",
	}
	token2025 := &tokens.Token{
		Name:          "color-success",
		Value:         "oklch(0.68 0.19 145)",
		SchemaVersion: schema.V2025_10,
		FilePath:      "2025-tokens.json",
	}

	require.NoError(t, manager.Add(draftToken))
	require.NoError(t, manager.Add(token2025))

	// Get schema version for each file
	version := manager.GetSchemaVersionForFile("draft-tokens.json")
	assert.Equal(t, schema.Draft, version)

	version = manager.GetSchemaVersionForFile("2025-tokens.json")
	assert.Equal(t, schema.V2025_10, version)

	// Unknown file should return Unknown
	version = manager.GetSchemaVersionForFile("nonexistent.json")
	assert.Equal(t, schema.Unknown, version)
}
