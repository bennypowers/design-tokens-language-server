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

func TestResolveAliases(t *testing.T) {
	t.Run("resolve simple curly brace aliases", func(t *testing.T) {
		content, err := os.ReadFile(filepath.Join("..", "..", "test", "fixtures", "resolver", "simple-alias.json"))
		require.NoError(t, err)

		parser := json.NewParser()
		tokenList, err := parser.ParseWithSchemaVersion(content, "", schema.Draft, nil)
		require.NoError(t, err)

		err = resolver.ResolveAliases(tokenList, schema.Draft)
		assert.NoError(t, err)

		// Find color.primary token
		var primaryToken *tokens.Token
		for _, tok := range tokenList {
			if tok.Name == "color-primary" {
				primaryToken = tok
				break
			}
		}
		require.NotNil(t, primaryToken)

		// Should be resolved
		assert.True(t, primaryToken.IsResolved, "color-primary should be resolved")
		assert.NotNil(t, primaryToken.ResolvedValue, "color-primary should have ResolvedValue")
		assert.Equal(t, "#FF6B35", primaryToken.ResolvedValue, "color-primary should resolve to #FF6B35")

		// Find spacing.medium token
		var mediumToken *tokens.Token
		for _, tok := range tokenList {
			if tok.Name == "spacing-medium" {
				mediumToken = tok
				break
			}
		}
		require.NotNil(t, mediumToken)

		assert.True(t, mediumToken.IsResolved)
		assert.Equal(t, "8px", mediumToken.ResolvedValue)
	})

	t.Run("resolve chained aliases", func(t *testing.T) {
		content, err := os.ReadFile(filepath.Join("..", "..", "test", "fixtures", "resolver", "chained-alias.json"))
		require.NoError(t, err)

		parser := json.NewParser()
		tokenList, err := parser.ParseWithSchemaVersion(content, "", schema.Draft, nil)
		require.NoError(t, err)

		err = resolver.ResolveAliases(tokenList, schema.Draft)
		assert.NoError(t, err)

		// All tokens should resolve to #FF0000
		tokenMap := make(map[string]*tokens.Token)
		for _, tok := range tokenList {
			tokenMap[tok.Name] = tok
		}

		// color-red is the base (not an alias)
		assert.True(t, tokenMap["color-red"].IsResolved)
		assert.Equal(t, "#FF0000", tokenMap["color-red"].ResolvedValue)

		// color-brand references color-red
		assert.True(t, tokenMap["color-brand"].IsResolved)
		assert.Equal(t, "#FF0000", tokenMap["color-brand"].ResolvedValue)

		// color-primary references color-brand
		assert.True(t, tokenMap["color-primary"].IsResolved)
		assert.Equal(t, "#FF0000", tokenMap["color-primary"].ResolvedValue)

		// color-accent references color-primary
		assert.True(t, tokenMap["color-accent"].IsResolved)
		assert.Equal(t, "#FF0000", tokenMap["color-accent"].ResolvedValue)
	})

	t.Run("detect circular references", func(t *testing.T) {
		content, err := os.ReadFile(filepath.Join("..", "..", "test", "fixtures", "resolver", "circular-alias.json"))
		require.NoError(t, err)

		parser := json.NewParser()
		tokenList, err := parser.ParseWithSchemaVersion(content, "", schema.Draft, nil)
		require.NoError(t, err)

		err = resolver.ResolveAliases(tokenList, schema.Draft)
		assert.Error(t, err, "should detect circular reference")
		assert.ErrorIs(t, err, schema.ErrCircularReference)
	})

	t.Run("resolve JSON Pointer aliases (2025.10)", func(t *testing.T) {
		content, err := os.ReadFile(filepath.Join("..", "..", "test", "fixtures", "resolver", "json-pointer-alias.json"))
		require.NoError(t, err)

		parser := json.NewParser()
		tokenList, err := parser.ParseWithSchemaVersion(content, "", schema.V2025_10, nil)
		require.NoError(t, err)

		err = resolver.ResolveAliases(tokenList, schema.V2025_10)
		assert.NoError(t, err)

		tokenMap := make(map[string]*tokens.Token)
		for _, tok := range tokenList {
			tokenMap[tok.Name] = tok
		}

		// color-base has structured value
		baseToken := tokenMap["color-base"]
		require.NotNil(t, baseToken)
		assert.True(t, baseToken.IsResolved)
		assert.NotNil(t, baseToken.ResolvedValue)

		// color-primary references color-base
		primaryToken := tokenMap["color-primary"]
		require.NotNil(t, primaryToken)
		assert.True(t, primaryToken.IsResolved)
		assert.Equal(t, baseToken.ResolvedValue, primaryToken.ResolvedValue, "should resolve to same value as base")

		// color-secondary references color-primary
		secondaryToken := tokenMap["color-secondary"]
		require.NotNil(t, secondaryToken)
		assert.True(t, secondaryToken.IsResolved)
		assert.Equal(t, baseToken.ResolvedValue, secondaryToken.ResolvedValue, "should resolve to same value as base")
	})

	t.Run("non-alias tokens are marked resolved", func(t *testing.T) {
		content, err := os.ReadFile(filepath.Join("..", "..", "test", "fixtures", "resolver", "simple-alias.json"))
		require.NoError(t, err)

		parser := json.NewParser()
		tokenList, err := parser.ParseWithSchemaVersion(content, "", schema.Draft, nil)
		require.NoError(t, err)

		err = resolver.ResolveAliases(tokenList, schema.Draft)
		assert.NoError(t, err)

		// Find color.base (not an alias)
		var baseToken *tokens.Token
		for _, tok := range tokenList {
			if tok.Name == "color-base" {
				baseToken = tok
				break
			}
		}
		require.NotNil(t, baseToken)

		assert.True(t, baseToken.IsResolved)
		assert.Equal(t, "#FF6B35", baseToken.ResolvedValue, "non-alias should resolve to its own value")
	})
}
