package resolver_test

import (
	"os"
	"path/filepath"
	"testing"

	"bennypowers.dev/dtls/internal/parser/json"
	"bennypowers.dev/dtls/internal/resolver"
	"bennypowers.dev/dtls/internal/schema"
	"bennypowers.dev/dtls/internal/tokens"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveGroupExtensions_Simple(t *testing.T) {
	// Test simple group extension: child inherits from parent
	fixturesDir := "../../test/fixtures/extends"
	content, err := os.ReadFile(filepath.Join(fixturesDir, "simple.json"))
	require.NoError(t, err)

	parser := json.NewParser()
	tokenList, err := parser.ParseWithSchemaVersion(content, "", schema.V2025_10, nil)
	require.NoError(t, err)

	// Debug: print all tokens
	t.Logf("Total tokens: %d", len(tokenList))
	for _, tok := range tokenList {
		t.Logf("Token: %s, Path: %v, Type: %s, Value: %s", tok.Name, tok.Path, tok.Type, tok.Value)
	}

	// Before extension resolution, themeColors should only have "green" (excluding $extends)
	themeTokens := filterByPrefixExcludingExtends(tokenList, "themeColors")
	t.Logf("ThemeColors tokens before: %d", len(themeTokens))
	assert.Equal(t, 1, len(themeTokens), "Should have 1 token in themeColors before extension")

	// Resolve extensions
	tokenList, err = resolver.ResolveGroupExtensions(tokenList)
	require.NoError(t, err)

	// After extension resolution, themeColors should have red, blue, green
	themeTokens = filterByPrefixExcludingExtends(tokenList, "themeColors")
	assert.Equal(t, 3, len(themeTokens), "Should have 3 tokens in themeColors after extension")

	// Check that we have all expected tokens
	tokenNames := []string{}
	for _, tok := range themeTokens {
		tokenNames = append(tokenNames, tok.Name)
	}
	assert.Contains(t, tokenNames, "themeColors-red")
	assert.Contains(t, tokenNames, "themeColors-blue")
	assert.Contains(t, tokenNames, "themeColors-green")
}

func TestResolveGroupExtensions_Nested(t *testing.T) {
	// Test nested group extensions: level2 extends level1 which extends base
	fixturesDir := "../../test/fixtures/extends"
	content, err := os.ReadFile(filepath.Join(fixturesDir, "nested.json"))
	require.NoError(t, err)

	parser := json.NewParser()
	tokenList, err := parser.ParseWithSchemaVersion(content, "", schema.V2025_10, nil)
	require.NoError(t, err)

	// Resolve extensions
	tokenList, err = resolver.ResolveGroupExtensions(tokenList)
	require.NoError(t, err)

	// level2 should have small (from base), medium (from level1), and large (own)
	level2Tokens := filterByPrefixExcludingExtends(tokenList, "level2")
	assert.GreaterOrEqual(t, len(level2Tokens), 3, "level2 should inherit from both base and level1")

	// Check for nested structure
	tokenNames := []string{}
	for _, tok := range level2Tokens {
		tokenNames = append(tokenNames, tok.Name)
	}
	assert.Contains(t, tokenNames, "level2-spacing-small", "Should inherit small from base")
	assert.Contains(t, tokenNames, "level2-spacing-medium", "Should inherit medium from level1")
	assert.Contains(t, tokenNames, "level2-spacing-large", "Should have its own large")
}

func TestResolveGroupExtensions_Override(t *testing.T) {
	// Test that child tokens override parent tokens
	fixturesDir := "../../test/fixtures/extends"
	content, err := os.ReadFile(filepath.Join(fixturesDir, "override.json"))
	require.NoError(t, err)

	parser := json.NewParser()
	tokenList, err := parser.ParseWithSchemaVersion(content, "", schema.V2025_10, nil)
	require.NoError(t, err)

	// Resolve extensions
	tokenList, err = resolver.ResolveGroupExtensions(tokenList)
	require.NoError(t, err)

	// override should have both color (overridden) and spacing (inherited)
	overrideTokens := filterByPrefixExcludingExtends(tokenList, "override")
	assert.Equal(t, 2, len(overrideTokens), "Should have 2 tokens in override group")

	// Find the color token
	var colorToken *tokens.Token
	for _, tok := range tokenList {
		if tok.Name == "override-color" {
			colorToken = tok
			break
		}
	}
	require.NotNil(t, colorToken, "Should find override-color token")

	// The color value should be the child's value (blue), not parent's (red)
	// Check RawValue for structured color values in 2025.10
	require.NotNil(t, colorToken.RawValue, "Color token should have RawValue")

	// RawValue should be a map with components [0, 0, 1.0] for blue
	colorMap, ok := colorToken.RawValue.(map[string]interface{})
	require.True(t, ok, "RawValue should be a map")

	components, ok := colorMap["components"].([]interface{})
	require.True(t, ok, "Should have components array")
	require.Len(t, components, 3, "Should have 3 color components")

	// Check for blue: [0, 0, 1.0]
	// Components can be int or float64 depending on JSON decoding
	assert.Contains(t, []interface{}{0, float64(0)}, components[0], "Red component should be 0")
	assert.Contains(t, []interface{}{0, float64(0)}, components[1], "Green component should be 0")
	assert.Contains(t, []interface{}{1, 1.0, float64(1)}, components[2], "Blue component should be 1")
}

func TestResolveGroupExtensions_CircularDetection(t *testing.T) {
	// Test that circular extensions are detected and return error
	fixturesDir := "../../test/fixtures/extends"
	content, err := os.ReadFile(filepath.Join(fixturesDir, "circular.json"))
	require.NoError(t, err)

	parser := json.NewParser()
	tokenList, err := parser.ParseWithSchemaVersion(content, "", schema.V2025_10, nil)
	require.NoError(t, err)

	// Resolve extensions - should return error
	_, err = resolver.ResolveGroupExtensions(tokenList)
	assert.Error(t, err, "Should detect circular extension")
	assert.Contains(t, err.Error(), "circular", "Error should mention circular reference")
}

func TestResolveGroupExtensions_DraftSchema(t *testing.T) {
	// Test that $extends is only processed for 2025.10 schema
	content := []byte(`{
		"$schema": "https://www.designtokens.org/schemas/draft.json",
		"base": {
			"color": {
				"$type": "color",
				"$value": "#FF0000"
			}
		},
		"theme": {
			"$extends": "#/base",
			"spacing": {
				"$type": "dimension",
				"$value": "16px"
			}
		}
	}`)

	parser := json.NewParser()
	tokenList, err := parser.ParseWithSchemaVersion(content, "", schema.Draft, nil)
	require.NoError(t, err)

	// Resolve extensions - should be no-op for draft schema
	tokenList, err = resolver.ResolveGroupExtensions(tokenList)
	require.NoError(t, err)

	// theme should still only have spacing token (no inheritance in draft)
	themeTokens := filterByPrefixExcludingExtends(tokenList, "theme")
	assert.Equal(t, 1, len(themeTokens), "Draft schema should not process $extends")
}

func TestResolveGroupExtensions_EmptyList(t *testing.T) {
	// Test that empty token list doesn't cause errors
	var tokenList []*tokens.Token

	_, err := resolver.ResolveGroupExtensions(tokenList)
	assert.NoError(t, err)
}

// Helper function to filter tokens by prefix
func filterByPrefix(tokenList []*tokens.Token, prefix string) []*tokens.Token {
	result := []*tokens.Token{}
	for _, tok := range tokenList {
		if len(tok.Path) > 0 && tok.Path[0] == prefix {
			result = append(result, tok)
		}
	}
	return result
}

// Helper function to filter tokens by prefix, excluding $extends metadata tokens
func filterByPrefixExcludingExtends(tokenList []*tokens.Token, prefix string) []*tokens.Token {
	result := []*tokens.Token{}
	for _, tok := range tokenList {
		if len(tok.Path) > 0 && tok.Path[0] == prefix {
			// Exclude $extends metadata tokens
			if len(tok.Path) > 1 && tok.Path[len(tok.Path)-1] == "$extends" {
				continue
			}
			result = append(result, tok)
		}
	}
	return result
}
