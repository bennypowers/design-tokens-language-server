package tokens_test

import (
	"os"
	"testing"

	asimonimParser "bennypowers.dev/asimonim/parser"
	"bennypowers.dev/dtls/internal/schema"
	"bennypowers.dev/dtls/internal/tokens"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMultiSchemaWorkspace_Integration tests loading and managing tokens from multiple schema versions
func TestMultiSchemaWorkspace_Integration(t *testing.T) {
	manager := tokens.NewManager()
	parser := asimonimParser.NewJSONParser()

	// Load draft schema tokens
	draftContent, err := os.ReadFile("testdata/multi-schema/mixed-workspace/draft-tokens.json")
	require.NoError(t, err)

	draftTokens, err := parser.Parse(draftContent, asimonimParser.Options{Prefix: "draft", SchemaVersion: schema.Draft, GroupMarkers: []string{"_"}})
	require.NoError(t, err)

	// Load 2025.10 schema tokens
	v2025Content, err := os.ReadFile("testdata/multi-schema/mixed-workspace/2025-tokens.json")
	require.NoError(t, err)

	v2025Tokens, err := parser.Parse(v2025Content, asimonimParser.Options{Prefix: "2025", SchemaVersion: schema.V2025_10})
	require.NoError(t, err)

	// Add tokens to manager
	for _, tok := range draftTokens {
		tok.FilePath = "draft-tokens.json"
		manager.Add(tok)
	}

	for _, tok := range v2025Tokens {
		tok.FilePath = "2025-tokens.json"
		manager.Add(tok)
	}

	// Verify both schemas are loaded
	allTokens := manager.GetAll()
	assert.Greater(t, len(allTokens), 0, "Should have loaded tokens from both files")

	// Verify tokens from each schema version
	draftSchemaTokens := manager.GetBySchemaVersion(schema.Draft)
	v2025SchemaTokens := manager.GetBySchemaVersion(schema.V2025_10)

	assert.Greater(t, len(draftSchemaTokens), 0, "Should have draft schema tokens")
	assert.Greater(t, len(v2025SchemaTokens), 0, "Should have 2025.10 schema tokens")

	// Verify tokens from each source file
	draftFileTokens := manager.GetBySourceFile("draft-tokens.json")
	v2025FileTokens := manager.GetBySourceFile("2025-tokens.json")

	assert.Equal(t, len(draftSchemaTokens), len(draftFileTokens), "Draft schema tokens should match draft file tokens")
	assert.Equal(t, len(v2025SchemaTokens), len(v2025FileTokens), "2025.10 schema tokens should match 2025 file tokens")

	// Verify schema version is tracked correctly
	for _, tok := range draftFileTokens {
		assert.Equal(t, schema.Draft, tok.SchemaVersion, "Draft file tokens should have Draft schema version")
	}

	for _, tok := range v2025FileTokens {
		assert.Equal(t, schema.V2025_10, tok.SchemaVersion, "2025.10 file tokens should have V2025_10 schema version")
	}
}

// TestMultiSchemaWorkspace_TokenIsolation tests that tokens from different schemas don't interfere
func TestMultiSchemaWorkspace_TokenIsolation(t *testing.T) {
	manager := tokens.NewManager()
	parser := asimonimParser.NewJSONParser()

	// Load draft schema tokens
	draftContent, err := os.ReadFile("testdata/multi-schema/mixed-workspace/draft-tokens.json")
	require.NoError(t, err)

	draftTokens, err := parser.Parse(draftContent, asimonimParser.Options{Prefix: "draft", SchemaVersion: schema.Draft, GroupMarkers: []string{"_"}})
	require.NoError(t, err)

	// Load 2025.10 schema tokens
	v2025Content, err := os.ReadFile("testdata/multi-schema/mixed-workspace/2025-tokens.json")
	require.NoError(t, err)

	v2025Tokens, err := parser.Parse(v2025Content, asimonimParser.Options{Prefix: "2025", SchemaVersion: schema.V2025_10})
	require.NoError(t, err)

	// Add tokens with file paths
	for _, tok := range draftTokens {
		tok.FilePath = "draft-tokens.json"
		manager.Add(tok)
	}

	for _, tok := range v2025Tokens {
		tok.FilePath = "2025-tokens.json"
		manager.Add(tok)
	}

	// Test that removing one file doesn't affect the other
	removedCount := manager.RemoveBySourceFile("draft-tokens.json")
	assert.Greater(t, removedCount, 0, "Should have removed draft tokens")

	// Verify 2025.10 tokens are still present
	remaining := manager.GetBySchemaVersion(schema.V2025_10)
	assert.Greater(t, len(remaining), 0, "2025.10 tokens should still be present after removing draft tokens")

	// Verify draft tokens are gone
	draftRemaining := manager.GetBySchemaVersion(schema.Draft)
	assert.Equal(t, 0, len(draftRemaining), "Draft tokens should be removed")
}

// TestMultiSchemaWorkspace_QualifiedLookup tests looking up tokens by name and file path
func TestMultiSchemaWorkspace_QualifiedLookup(t *testing.T) {
	manager := tokens.NewManager()

	// Create tokens with same name but different schemas
	draftToken := &tokens.Token{
		Name:          "color-brand-primary",
		Value:         "#FF0000",
		SchemaVersion: schema.Draft,
		FilePath:      "draft.json",
	}

	v2025Token := &tokens.Token{
		Name:          "color-brand-primary",
		SchemaVersion: schema.V2025_10,
		FilePath:      "2025.json",
		RawValue: map[string]interface{}{
			"colorSpace": "srgb",
			"components": []interface{}{1.0, 0.0, 0.0},
			"alpha":      1.0,
		},
	}

	manager.Add(draftToken)
	manager.Add(v2025Token)

	// Qualified lookup by file path
	draftLookup := manager.GetQualified("color-brand-primary", "draft.json")
	require.NotNil(t, draftLookup, "Should find draft token by qualified name")
	assert.Equal(t, schema.Draft, draftLookup.SchemaVersion)

	v2025Lookup := manager.GetQualified("color-brand-primary", "2025.json")
	require.NotNil(t, v2025Lookup, "Should find 2025.10 token by qualified name")
	assert.Equal(t, schema.V2025_10, v2025Lookup.SchemaVersion)

	// Regular lookup returns first found (could be either)
	regularLookup := manager.Get("color-brand-primary")
	require.NotNil(t, regularLookup, "Should find token by regular lookup")
	assert.Contains(t, []schema.SchemaVersion{schema.Draft, schema.V2025_10}, regularLookup.SchemaVersion)
}

// TestMultiSchemaWorkspace_SchemaVersionPerFile tests schema version tracking per file
func TestMultiSchemaWorkspace_SchemaVersionPerFile(t *testing.T) {
	manager := tokens.NewManager()

	// Add tokens from different files with different schemas
	manager.Add(&tokens.Token{
		Name:          "draft-token",
		SchemaVersion: schema.Draft,
		FilePath:      "draft.json",
	})

	manager.Add(&tokens.Token{
		Name:          "v2025-token",
		SchemaVersion: schema.V2025_10,
		FilePath:      "2025.json",
	})

	manager.Add(&tokens.Token{
		Name:          "another-draft-token",
		SchemaVersion: schema.Draft,
		FilePath:      "draft.json",
	})

	// Get schema version for each file
	draftSchemaVersion := manager.GetSchemaVersionForFile("draft.json")
	assert.Equal(t, schema.Draft, draftSchemaVersion, "draft.json should be Draft schema")

	v2025SchemaVersion := manager.GetSchemaVersionForFile("2025.json")
	assert.Equal(t, schema.V2025_10, v2025SchemaVersion, "2025.json should be V2025_10 schema")

	// Unknown file should return Unknown
	unknownSchemaVersion := manager.GetSchemaVersionForFile("nonexistent.json")
	assert.Equal(t, schema.Unknown, unknownSchemaVersion, "Unknown file should return Unknown schema")
}

// TestMultiSchemaWorkspace_SourceFiles tests getting list of all source files
func TestMultiSchemaWorkspace_SourceFiles(t *testing.T) {
	manager := tokens.NewManager()

	// Add tokens from multiple files
	manager.Add(&tokens.Token{Name: "token1", FilePath: "file1.json"})
	manager.Add(&tokens.Token{Name: "token2", FilePath: "file2.json"})
	manager.Add(&tokens.Token{Name: "token3", FilePath: "file1.json"}) // Duplicate file
	manager.Add(&tokens.Token{Name: "token4", FilePath: "file3.json"})

	sourceFiles := manager.GetSourceFiles()

	// Should return unique file paths
	assert.Len(t, sourceFiles, 3, "Should have 3 unique source files")
	assert.Contains(t, sourceFiles, "file1.json")
	assert.Contains(t, sourceFiles, "file2.json")
	assert.Contains(t, sourceFiles, "file3.json")
}
