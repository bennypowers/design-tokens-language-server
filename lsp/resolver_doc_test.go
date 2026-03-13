package lsp

import (
	"os"
	"path/filepath"
	"testing"

	"bennypowers.dev/dtls/lsp/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func loadFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "resolver-doc-extract", name, "resolver.json"))
	require.NoError(t, err)
	return data
}

func TestIsResolverDocument(t *testing.T) {
	t.Run("detects resolver document with resolutionOrder", func(t *testing.T) {
		assert.True(t, isResolverDocument(loadFixture(t, "inline-sources")))
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

func TestExtractResolverSourcePaths(t *testing.T) {
	t.Run("extracts inline sources", func(t *testing.T) {
		paths, err := extractResolverSourcePaths(loadFixture(t, "inline-sources"), "/project/tokens")
		require.NoError(t, err)
		assert.Equal(t, []string{
			filepath.FromSlash("/project/tokens/palette.json"),
			filepath.FromSlash("/project/tokens/colors.json"),
		}, paths)
	})

	t.Run("extracts named set references", func(t *testing.T) {
		paths, err := extractResolverSourcePaths(loadFixture(t, "named-sets"), "/project/tokens")
		require.NoError(t, err)
		assert.Equal(t, []string{filepath.FromSlash("/project/tokens/palette.json")}, paths)
	})

	t.Run("deduplicates paths", func(t *testing.T) {
		paths, err := extractResolverSourcePaths(loadFixture(t, "dedup"), "/project")
		require.NoError(t, err)
		assert.Equal(t, []string{filepath.FromSlash("/project/palette.json")}, paths)
	})

	t.Run("handles multiple sets in order", func(t *testing.T) {
		paths, err := extractResolverSourcePaths(loadFixture(t, "multiple-sets"), "/project")
		require.NoError(t, err)
		assert.Equal(t, []string{
			filepath.FromSlash("/project/palette.json"),
			filepath.FromSlash("/project/overrides.json"),
		}, paths)
	})

	t.Run("ignores JSON pointer refs in sources", func(t *testing.T) {
		paths, err := extractResolverSourcePaths(loadFixture(t, "pointer-refs"), "/project")
		require.NoError(t, err)
		assert.Equal(t, []string{filepath.FromSlash("/project/palette.json")}, paths)
	})

	t.Run("decodes JSON Pointer escaping in set names", func(t *testing.T) {
		paths, err := extractResolverSourcePaths(loadFixture(t, "json-pointer-escaping"), "/project")
		require.NoError(t, err)
		assert.Equal(t, []string{filepath.FromSlash("/project/palette.json")}, paths)
	})

	t.Run("strips fragment identifiers from source refs", func(t *testing.T) {
		paths, err := extractResolverSourcePaths(loadFixture(t, "fragment-stripping"), "/project")
		require.NoError(t, err)
		assert.Equal(t, []string{filepath.FromSlash("/project/palette.json")}, paths)
	})

	t.Run("extracts sources from inline modifier contexts", func(t *testing.T) {
		paths, err := extractResolverSourcePaths(loadFixture(t, "inline-modifier"), "/project")
		require.NoError(t, err)
		assert.Equal(t, []string{
			filepath.FromSlash("/project/palette.json"),
			filepath.FromSlash("/project/dark.json"),
		}, paths)
	})

	t.Run("extracts sources from named modifier ref", func(t *testing.T) {
		paths, err := extractResolverSourcePaths(loadFixture(t, "named-modifier"), "/project")
		require.NoError(t, err)
		assert.Equal(t, []string{
			filepath.FromSlash("/project/palette.json"),
			filepath.FromSlash("/project/dark.json"),
		}, paths)
	})

	t.Run("extracts sources from multiple modifier contexts", func(t *testing.T) {
		paths, err := extractResolverSourcePaths(loadFixture(t, "multi-contexts"), "/project")
		require.NoError(t, err)
		assert.Contains(t, paths, filepath.FromSlash("/project/light.json"))
		assert.Contains(t, paths, filepath.FromSlash("/project/dark.json"))
	})

	t.Run("deduplicates across set and modifier sources", func(t *testing.T) {
		paths, err := extractResolverSourcePaths(loadFixture(t, "dedup-across-types"), "/project")
		require.NoError(t, err)
		assert.Contains(t, paths, filepath.FromSlash("/project/palette.json"))
		assert.Contains(t, paths, filepath.FromSlash("/project/dark.json"))
		count := 0
		for _, p := range paths {
			if p == filepath.FromSlash("/project/palette.json") {
				count++
			}
		}
		assert.Equal(t, 1, count, "palette.json should be deduplicated")
	})

	t.Run("returns error for missing modifier reference", func(t *testing.T) {
		_, err := extractResolverSourcePaths(loadFixture(t, "missing-modifier"), "/project")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "nonexistent")
	})

	t.Run("returns error for invalid JSON", func(t *testing.T) {
		_, err := extractResolverSourcePaths([]byte(`{invalid`), "/project")
		require.Error(t, err)
	})

	t.Run("returns error for unrecognized entry shape", func(t *testing.T) {
		_, err := extractResolverSourcePaths(loadFixture(t, "unrecognized-entry"), "/project")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unrecognized")
	})

	t.Run("returns error for missing set reference", func(t *testing.T) {
		_, err := extractResolverSourcePaths(loadFixture(t, "missing-set"), "/project")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "nonexistent")
	})
}

func TestResolveRefPath(t *testing.T) {
	t.Run("resolves relative path against resolver dir", func(t *testing.T) {
		result := resolveRefPath("./palette.json", "/project/tokens")
		assert.Equal(t, filepath.FromSlash("/project/tokens/palette.json"), result)
	})

	t.Run("cleans absolute path", func(t *testing.T) {
		result := resolveRefPath("/abs/path/tokens.json", "/project/tokens")
		assert.Equal(t, filepath.FromSlash("/abs/path/tokens.json"), result)
	})

	t.Run("passes through npm: URI unchanged", func(t *testing.T) {
		result := resolveRefPath("npm:@scope/tokens/tokens.json", "/project")
		assert.Equal(t, "npm:@scope/tokens/tokens.json", result)
	})

	t.Run("passes through jsr: URI unchanged", func(t *testing.T) {
		result := resolveRefPath("jsr:@scope/tokens/tokens.json", "/project")
		assert.Equal(t, "jsr:@scope/tokens/tokens.json", result)
	})

	t.Run("passes through https:// URI unchanged", func(t *testing.T) {
		result := resolveRefPath("https://cdn.example.com/tokens.json", "/project")
		assert.Equal(t, "https://cdn.example.com/tokens.json", result)
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
