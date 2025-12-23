package json_test

import (
	"os"
	"path/filepath"
	"testing"

	"bennypowers.dev/dtls/internal/parser/json"
	"bennypowers.dev/dtls/internal/schema"
	"bennypowers.dev/dtls/internal/tokens"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseWithSchemaVersion(t *testing.T) {
	t.Run("parse draft schema with group markers", func(t *testing.T) {
		content, err := os.ReadFile(filepath.Join("..", "..", "..", "test", "fixtures", "parser", "draft-tokens.json"))
		require.NoError(t, err)

		parser := json.NewParser()
		groupMarkers := []string{"_", "@", "DEFAULT"}

		tokenList, err := parser.ParseWithSchemaVersion(content, "", schema.Draft, groupMarkers)
		assert.NoError(t, err)
		assert.NotEmpty(t, tokenList)

		// Verify all tokens have correct schema version
		for _, tok := range tokenList {
			assert.Equal(t, schema.Draft, tok.SchemaVersion, "token %s should have Draft schema version", tok.Name)
		}

		// Verify group marker creates correct token
		var rootToken *tokens.Token
		for _, tok := range tokenList {
			if tok.Name == "color" {
				rootToken = tok
				break
			}
		}
		assert.NotNil(t, rootToken, "should find color root token from _ marker")
		assert.Equal(t, "#FF0000", rootToken.Value)
		assert.Equal(t, "color", rootToken.Type)
	})

	t.Run("parse 2025.10 schema with $root", func(t *testing.T) {
		content, err := os.ReadFile(filepath.Join("..", "..", "..", "test", "fixtures", "parser", "2025-tokens.json"))
		require.NoError(t, err)

		parser := json.NewParser()

		tokenList, err := parser.ParseWithSchemaVersion(content, "", schema.V2025_10, nil)
		assert.NoError(t, err)
		assert.NotEmpty(t, tokenList)

		// Verify all tokens have correct schema version
		for _, tok := range tokenList {
			assert.Equal(t, schema.V2025_10, tok.SchemaVersion, "token %s should have V2025_10 schema version", tok.Name)
		}

		// Verify $root creates correct token
		var rootToken *tokens.Token
		for _, tok := range tokenList {
			if tok.Name == "color" {
				rootToken = tok
				break
			}
		}
		assert.NotNil(t, rootToken, "should find color root token from $root")
		assert.Equal(t, "color", rootToken.Type)

		// RawValue should be a map for structured color
		assert.NotNil(t, rootToken.RawValue)
		colorMap, ok := rootToken.RawValue.(map[string]interface{})
		assert.True(t, ok, "RawValue should be a map for structured color")
		assert.Equal(t, "srgb", colorMap["colorSpace"])
	})

	t.Run("skip $schema field in 2025.10", func(t *testing.T) {
		content, err := os.ReadFile(filepath.Join("..", "..", "..", "test", "fixtures", "parser", "2025-tokens.json"))
		require.NoError(t, err)

		parser := json.NewParser()

		tokenList, err := parser.ParseWithSchemaVersion(content, "", schema.V2025_10, nil)
		assert.NoError(t, err)

		// $schema should not appear as a token
		for _, tok := range tokenList {
			assert.NotEqual(t, "$schema", tok.Name, "$schema field should be skipped")
			assert.NotContains(t, tok.Path, "$schema", "token path should not contain $schema")
		}
	})

	t.Run("handle $ref in 2025.10 (don't parse as child token)", func(t *testing.T) {
		content, err := os.ReadFile(filepath.Join("..", "..", "..", "test", "fixtures", "parser", "2025-tokens.json"))
		require.NoError(t, err)

		parser := json.NewParser()

		tokenList, err := parser.ParseWithSchemaVersion(content, "", schema.V2025_10, nil)
		assert.NoError(t, err)

		// Find the secondary token (which has $ref)
		var secondaryToken *tokens.Token
		for _, tok := range tokenList {
			if tok.Name == "color-secondary" {
				secondaryToken = tok
				break
			}
		}
		assert.NotNil(t, secondaryToken, "should find color-secondary token")

		// $ref should not create a child token
		for _, tok := range tokenList {
			assert.NotContains(t, tok.Name, "$ref", "$ref should not create a token")
		}
	})

	t.Run("groupMarkers ignored for 2025.10 schema", func(t *testing.T) {
		// Create content with _ marker (which should be treated as regular token in 2025.10)
		content := []byte(`{
			"$schema": "https://www.designtokens.org/schemas/2025.10.json",
			"color": {
				"_": {
					"$type": "color",
					"$value": {
						"colorSpace": "srgb",
						"components": [1.0, 0, 0]
					}
				}
			}
		}`)

		parser := json.NewParser()
		groupMarkers := []string{"_", "@", "DEFAULT"}

		tokenList, err := parser.ParseWithSchemaVersion(content, "", schema.V2025_10, groupMarkers)
		assert.NoError(t, err)

		// _ should create a token named "color-_", NOT promote to "color"
		var underscoreToken *tokens.Token
		for _, tok := range tokenList {
			if tok.Name == "color-_" {
				underscoreToken = tok
				break
			}
		}
		assert.NotNil(t, underscoreToken, "_ should create color-_ token in 2025.10, not be treated as group marker")

		// Should NOT create a token named just "color"
		for _, tok := range tokenList {
			if tok.Name == "color" {
				assert.Fail(t, "_ should not promote to parent token in 2025.10 schema")
			}
		}
	})

	t.Run("draft schema with no groupMarkers", func(t *testing.T) {
		content, err := os.ReadFile(filepath.Join("..", "..", "..", "test", "fixtures", "parser", "draft-no-schema.json"))
		require.NoError(t, err)

		parser := json.NewParser()

		// Parse without group markers - @ should create regular token
		tokenList, err := parser.ParseWithSchemaVersion(content, "", schema.Draft, nil)
		assert.NoError(t, err)

		// @ should create a token named "color-@"
		var atToken *tokens.Token
		for _, tok := range tokenList {
			if tok.Name == "color-@" {
				atToken = tok
				break
			}
		}
		assert.NotNil(t, atToken, "@ should create color-@ token when not in groupMarkers list")
	})
}

func TestParseFileWithSchemaVersion(t *testing.T) {
	t.Run("parse draft file", func(t *testing.T) {
		parser := json.NewParser()
		groupMarkers := []string{"_", "@", "DEFAULT"}

		tokenList, err := parser.ParseFileWithSchemaVersion(
			filepath.Join("..", "..", "..", "test", "fixtures", "parser", "draft-tokens.json"),
			"",
			schema.Draft,
			groupMarkers,
		)
		assert.NoError(t, err)
		assert.NotEmpty(t, tokenList)

		for _, tok := range tokenList {
			assert.Equal(t, schema.Draft, tok.SchemaVersion)
		}
	})

	t.Run("parse 2025.10 file", func(t *testing.T) {
		parser := json.NewParser()

		tokenList, err := parser.ParseFileWithSchemaVersion(
			filepath.Join("..", "..", "..", "test", "fixtures", "parser", "2025-tokens.json"),
			"",
			schema.V2025_10,
			nil,
		)
		assert.NoError(t, err)
		assert.NotEmpty(t, tokenList)

		for _, tok := range tokenList {
			assert.Equal(t, schema.V2025_10, tok.SchemaVersion)
		}
	})
}
