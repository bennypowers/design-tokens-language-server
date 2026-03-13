package lsp

import (
	"os"
	"path/filepath"
	"testing"

	"bennypowers.dev/dtls/lsp/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsResolverDocument(t *testing.T) {
	t.Run("detects resolver document with resolutionOrder", func(t *testing.T) {
		data, err := os.ReadFile("testdata/asimonim-integration/resolver-doc/inline-sources/tokens.resolver.json")
		require.NoError(t, err)
		assert.True(t, isResolverDocument(data))
	})

	t.Run("rejects regular token file", func(t *testing.T) {
		data := []byte(`{
			"color": {
				"primary": {"$value": "#ff0000", "$type": "color"}
			}
		}`)
		assert.False(t, isResolverDocument(data))
	})

	t.Run("rejects invalid JSON", func(t *testing.T) {
		assert.False(t, isResolverDocument([]byte(`{invalid`)))
	})
}

func TestLoadResolverDocument(t *testing.T) {
	t.Run("loads sources from resolver document", func(t *testing.T) {
		tmpDir := copyFixture(t, "resolver-doc/inline-sources")

		server, err := NewServer()
		require.NoError(t, err)
		defer func() { _ = server.Close() }()

		resolverPath := filepath.Join(tmpDir, "tokens.resolver.json")
		err = server.loadResolverDocument(resolverPath, &TokenFileOptions{})
		require.NoError(t, err)

		assert.NotNil(t, server.Token("color-neutral-100"), "palette token should be loaded")
		assert.NotNil(t, server.Token("color-surface-lowered"), "colors token should be loaded")
	})

	t.Run("loads from named sets with refs", func(t *testing.T) {
		tmpDir := copyFixture(t, "resolver-doc/named-sets")

		server, err := NewServer()
		require.NoError(t, err)
		defer func() { _ = server.Close() }()

		resolverPath := filepath.Join(tmpDir, "resolver.json")
		err = server.loadResolverDocument(resolverPath, &TokenFileOptions{})
		require.NoError(t, err)

		assert.NotNil(t, server.Token("color-primary"))
	})

	t.Run("loads sources from modifier contexts", func(t *testing.T) {
		tmpDir := copyFixture(t, "resolver-doc/modifier-contexts")

		server, err := NewServer()
		require.NoError(t, err)
		defer func() { _ = server.Close() }()

		resolverPath := filepath.Join(tmpDir, "tokens.resolver.json")
		err = server.loadResolverDocument(resolverPath, &TokenFileOptions{})
		require.NoError(t, err)

		assert.NotNil(t, server.Token("color-neutral-100"), "palette token should be loaded")
		assert.NotNil(t, server.Token("color-surface-lowered"), "colors token should be loaded")
	})

	t.Run("returns error for nonexistent resolver file", func(t *testing.T) {
		server, err := NewServer()
		require.NoError(t, err)
		defer func() { _ = server.Close() }()

		err = server.loadResolverDocument("/nonexistent/resolver.json", &TokenFileOptions{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read resolver document")
	})

	t.Run("returns error for invalid resolver JSON", func(t *testing.T) {
		tmpDir := t.TempDir()
		resolverPath := filepath.Join(tmpDir, "bad.json")
		require.NoError(t, os.WriteFile(resolverPath, []byte(`{not valid`), 0o644))

		server, err := NewServer()
		require.NoError(t, err)
		defer func() { _ = server.Close() }()

		err = server.loadResolverDocument(resolverPath, &TokenFileOptions{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to extract sources")
	})

	t.Run("returns error when source file is missing", func(t *testing.T) {
		tmpDir := t.TempDir()
		resolverPath := filepath.Join(tmpDir, "resolver.json")
		require.NoError(t, os.WriteFile(resolverPath, []byte(`{
			"version": "2025.10",
			"resolutionOrder": [{
				"sources": [{"$ref": "./missing.json"}]
			}]
		}`), 0o644))

		server, err := NewServer()
		require.NoError(t, err)
		defer func() { _ = server.Close() }()

		server.SetRootPath(tmpDir)
		err = server.loadResolverDocument(resolverPath, &TokenFileOptions{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "missing.json")
	})
}

func TestLoadTokensFromConfig_ResolversFromPackageJson(t *testing.T) {
	t.Run("loads resolvers from package.json config", func(t *testing.T) {
		tmpDir := copyFixture(t, "resolver-doc/from-package-json")

		server, err := NewServer()
		require.NoError(t, err)
		defer func() { _ = server.Close() }()

		server.SetRootPath(tmpDir)
		err = server.LoadPackageJsonConfig()
		require.NoError(t, err)

		cfg := server.GetConfig()
		require.Len(t, cfg.Resolvers, 1)

		err = server.LoadTokensFromConfig()
		require.NoError(t, err)

		assert.NotNil(t, server.Token("color-neutral-100"), "palette token should be loaded")
		assert.NotNil(t, server.Token("color-surface-lowered"), "semantic token should be loaded")

		tok := server.Token("color-surface-lowered")
		require.NotNil(t, tok)
		assert.True(t, tok.IsResolved, "alias should be resolved across resolver sources")
	})
}

func TestLoadTokensFromConfig_Resolvers(t *testing.T) {
	t.Run("loads tokens from resolver documents", func(t *testing.T) {
		tmpDir := copyFixture(t, "resolver-doc/inline-sources")

		server, err := NewServer()
		require.NoError(t, err)
		defer func() { _ = server.Close() }()

		server.SetRootPath(tmpDir)
		server.SetConfig(types.ServerConfig{
			Resolvers: []string{filepath.Join(tmpDir, "tokens.resolver.json")},
		})

		err = server.LoadTokensFromConfig()
		require.NoError(t, err)

		assert.NotNil(t, server.Token("color-neutral-100"), "palette token should be loaded")
		assert.NotNil(t, server.Token("color-surface-lowered"), "colors token should be loaded")
	})

	t.Run("resolves relative resolver paths", func(t *testing.T) {
		tmpDir := copyFixture(t, "resolver-doc/relative-resolver")

		server, err := NewServer()
		require.NoError(t, err)
		defer func() { _ = server.Close() }()

		server.SetRootPath(tmpDir)
		server.SetConfig(types.ServerConfig{
			Resolvers: []string{"./src/tokens/tokens.resolver.json"},
		})

		err = server.LoadTokensFromConfig()
		require.NoError(t, err)

		assert.NotNil(t, server.Token("color-primary"))
	})

	t.Run("resolves aliases across resolver sources", func(t *testing.T) {
		tmpDir := copyFixture(t, "resolver-doc/resolver-aliases")

		server, err := NewServer()
		require.NoError(t, err)
		defer func() { _ = server.Close() }()

		server.SetRootPath(tmpDir)
		server.SetConfig(types.ServerConfig{
			Resolvers: []string{filepath.Join(tmpDir, "resolver.json")},
		})

		err = server.LoadTokensFromConfig()
		require.NoError(t, err)

		tok := server.Token("color-surface-lowered")
		require.NotNil(t, tok)
		assert.True(t, tok.IsResolved, "alias should be resolved across resolver sources")
	})

	t.Run("returns error for nonexistent resolver path", func(t *testing.T) {
		server, err := NewServer()
		require.NoError(t, err)
		defer func() { _ = server.Close() }()

		server.SetRootPath(t.TempDir())
		server.SetConfig(types.ServerConfig{
			Resolvers: []string{"/nonexistent/resolver.json"},
		})

		err = server.LoadTokensFromConfig()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "nonexistent")
	})

	t.Run("returns error for resolver with missing sources", func(t *testing.T) {
		tmpDir := t.TempDir()
		resolverPath := filepath.Join(tmpDir, "resolver.json")
		require.NoError(t, os.WriteFile(resolverPath, []byte(`{
			"version": "2025.10",
			"resolutionOrder": [{
				"sources": [{"$ref": "./missing.json"}]
			}]
		}`), 0o644))

		server, err := NewServer()
		require.NoError(t, err)
		defer func() { _ = server.Close() }()

		server.SetRootPath(tmpDir)
		server.SetConfig(types.ServerConfig{
			Resolvers: []string{resolverPath},
		})

		err = server.LoadTokensFromConfig()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "missing.json")
	})
}
