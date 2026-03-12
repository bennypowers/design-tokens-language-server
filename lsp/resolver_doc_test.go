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
		data := []byte(`{
			"version": "2025.10",
			"resolutionOrder": [
				{"type": "set", "name": "base", "sources": [{"$ref": "./palette.json"}]}
			]
		}`)
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

func TestExtractResolverSourcePaths(t *testing.T) {
	t.Run("extracts inline sources", func(t *testing.T) {
		data := []byte(`{
			"version": "2025.10",
			"resolutionOrder": [
				{
					"type": "set",
					"name": "base",
					"sources": [
						{"$ref": "./palette.json"},
						{"$ref": "./colors.json"}
					]
				}
			]
		}`)
		paths, err := extractResolverSourcePaths(data, "/project/tokens")
		require.NoError(t, err)
		assert.Equal(t, []string{
			"/project/tokens/palette.json",
			"/project/tokens/colors.json",
		}, paths)
	})

	t.Run("extracts named set references", func(t *testing.T) {
		data := []byte(`{
			"version": "2025.10",
			"sets": {
				"base": {
					"sources": [
						{"$ref": "./palette.json"}
					]
				}
			},
			"resolutionOrder": [
				{"$ref": "#/sets/base"}
			]
		}`)
		paths, err := extractResolverSourcePaths(data, "/project/tokens")
		require.NoError(t, err)
		assert.Equal(t, []string{"/project/tokens/palette.json"}, paths)
	})

	t.Run("deduplicates paths", func(t *testing.T) {
		data := []byte(`{
			"version": "2025.10",
			"resolutionOrder": [
				{"sources": [{"$ref": "./palette.json"}]},
				{"sources": [{"$ref": "./palette.json"}]}
			]
		}`)
		paths, err := extractResolverSourcePaths(data, "/project")
		require.NoError(t, err)
		assert.Equal(t, []string{"/project/palette.json"}, paths)
	})

	t.Run("handles multiple sets in order", func(t *testing.T) {
		data := []byte(`{
			"version": "2025.10",
			"resolutionOrder": [
				{"sources": [{"$ref": "./palette.json"}]},
				{"sources": [{"$ref": "./overrides.json"}]}
			]
		}`)
		paths, err := extractResolverSourcePaths(data, "/project")
		require.NoError(t, err)
		assert.Equal(t, []string{"/project/palette.json", "/project/overrides.json"}, paths)
	})

	t.Run("ignores JSON pointer refs in sources", func(t *testing.T) {
		data := []byte(`{
			"version": "2025.10",
			"resolutionOrder": [
				{"sources": [
					{"$ref": "./palette.json"},
					{"$ref": "#/some/internal/ref"}
				]}
			]
		}`)
		paths, err := extractResolverSourcePaths(data, "/project")
		require.NoError(t, err)
		assert.Equal(t, []string{"/project/palette.json"}, paths)
	})

	t.Run("returns error for invalid JSON", func(t *testing.T) {
		_, err := extractResolverSourcePaths([]byte(`{invalid`), "/project")
		require.Error(t, err)
	})

	t.Run("decodes JSON Pointer escaping in set names", func(t *testing.T) {
		data := []byte(`{
			"version": "2025.10",
			"sets": {
				"brand/core": {
					"sources": [{"$ref": "./palette.json"}]
				}
			},
			"resolutionOrder": [
				{"$ref": "#/sets/brand~1core"}
			]
		}`)
		paths, err := extractResolverSourcePaths(data, "/project")
		require.NoError(t, err)
		assert.Equal(t, []string{"/project/palette.json"}, paths)
	})

	t.Run("strips fragment identifiers from source refs", func(t *testing.T) {
		data := []byte(`{
			"version": "2025.10",
			"resolutionOrder": [
				{"sources": [{"$ref": "./palette.json#/brand"}]}
			]
		}`)
		paths, err := extractResolverSourcePaths(data, "/project")
		require.NoError(t, err)
		assert.Equal(t, []string{"/project/palette.json"}, paths)
	})

	t.Run("handles missing set reference gracefully", func(t *testing.T) {
		data := []byte(`{
			"version": "2025.10",
			"sets": {},
			"resolutionOrder": [
				{"$ref": "#/sets/nonexistent"}
			]
		}`)
		paths, err := extractResolverSourcePaths(data, "/project")
		require.NoError(t, err)
		assert.Empty(t, paths)
	})
}

func TestLoadResolverDocument(t *testing.T) {
	t.Run("loads sources from resolver document", func(t *testing.T) {
		tmpDir := t.TempDir()
		tokensDir := filepath.Join(tmpDir, "tokens")
		require.NoError(t, os.MkdirAll(tokensDir, 0o755))

		// Create source token files
		require.NoError(t, os.WriteFile(filepath.Join(tokensDir, "palette.json"), []byte(`{
			"color": {
				"$type": "color",
				"neutral": {
					"100": {"$value": "#f5f5f5"}
				}
			}
		}`), 0o644))

		require.NoError(t, os.WriteFile(filepath.Join(tokensDir, "colors.json"), []byte(`{
			"color": {
				"surface": {
					"lowered": {"$value": "{color.neutral.100}"}
				}
			}
		}`), 0o644))

		// Create resolver document
		resolverPath := filepath.Join(tokensDir, "tokens.resolver.json")
		require.NoError(t, os.WriteFile(resolverPath, []byte(`{
			"version": "2025.10",
			"resolutionOrder": [
				{
					"type": "set",
					"name": "base",
					"sources": [
						{"$ref": "./palette.json"},
						{"$ref": "./colors.json"}
					]
				}
			]
		}`), 0o644))

		server, err := NewServer()
		require.NoError(t, err)
		defer func() { _ = server.Close() }()

		err = server.loadResolverDocument(resolverPath, &TokenFileOptions{})
		require.NoError(t, err)

		assert.NotNil(t, server.Token("color-neutral-100"), "palette token should be loaded")
		assert.NotNil(t, server.Token("color-surface-lowered"), "colors token should be loaded")
	})

	t.Run("loads from named sets with refs", func(t *testing.T) {
		tmpDir := t.TempDir()

		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "palette.json"), []byte(`{
			"color": {
				"primary": {"$value": "#ff0000", "$type": "color"}
			}
		}`), 0o644))

		resolverPath := filepath.Join(tmpDir, "resolver.json")
		require.NoError(t, os.WriteFile(resolverPath, []byte(`{
			"version": "2025.10",
			"sets": {
				"base": {
					"sources": [{"$ref": "./palette.json"}]
				}
			},
			"resolutionOrder": [
				{"$ref": "#/sets/base"}
			]
		}`), 0o644))

		server, err := NewServer()
		require.NoError(t, err)
		defer func() { _ = server.Close() }()

		err = server.loadResolverDocument(resolverPath, &TokenFileOptions{})
		require.NoError(t, err)

		assert.NotNil(t, server.Token("color-primary"))
	})
}

func TestLoadTokensFromConfig_ResolversFromPackageJson(t *testing.T) {
	t.Run("loads resolvers from package.json config", func(t *testing.T) {
		tmpDir := t.TempDir()
		tokensDir := filepath.Join(tmpDir, "src", "design-tokens")
		require.NoError(t, os.MkdirAll(tokensDir, 0o755))

		// Create source files matching the repro
		require.NoError(t, os.WriteFile(filepath.Join(tokensDir, "palette.json"), []byte(`{
			"color": {
				"$type": "color",
				"neutral": {
					"100": {
						"$value": {
							"colorSpace": "oklch",
							"components": [0.97, 0, 0],
							"alpha": 1,
							"hex": "#f5f5f5"
						}
					}
				}
			}
		}`), 0o644))

		require.NoError(t, os.WriteFile(filepath.Join(tokensDir, "colors.json"), []byte(`{
			"color": {
				"surface": {
					"lowered": {
						"$value": "{color.neutral.100}"
					}
				}
			}
		}`), 0o644))

		require.NoError(t, os.WriteFile(filepath.Join(tokensDir, "tokens.resolver.json"), []byte(`{
			"version": "2025.10",
			"resolutionOrder": [{
				"type": "set",
				"name": "base",
				"sources": [
					{"$ref": "./palette.json"},
					{"$ref": "./colors.json"}
				]
			}]
		}`), 0o644))

		// Create package.json with resolvers
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(`{
			"designTokensLanguageServer": {
				"resolvers": [
					"./src/design-tokens/tokens.resolver.json"
				]
			}
		}`), 0o644))

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

		// Both tokens should be loaded
		assert.NotNil(t, server.Token("color-neutral-100"), "palette token should be loaded")
		assert.NotNil(t, server.Token("color-surface-lowered"), "semantic token should be loaded")

		// The alias should resolve
		tok := server.Token("color-surface-lowered")
		require.NotNil(t, tok)
		assert.True(t, tok.IsResolved, "alias should be resolved across resolver sources")
	})
}

func TestLoadTokensFromConfig_Resolvers(t *testing.T) {
	t.Run("loads tokens from resolver documents", func(t *testing.T) {
		tmpDir := t.TempDir()
		tokensDir := filepath.Join(tmpDir, "tokens")
		require.NoError(t, os.MkdirAll(tokensDir, 0o755))

		// Create source token files
		require.NoError(t, os.WriteFile(filepath.Join(tokensDir, "palette.json"), []byte(`{
			"color": {
				"$type": "color",
				"neutral": {
					"100": {"$value": "#f5f5f5"}
				}
			}
		}`), 0o644))

		require.NoError(t, os.WriteFile(filepath.Join(tokensDir, "colors.json"), []byte(`{
			"color": {
				"surface": {
					"lowered": {"$value": "{color.neutral.100}"}
				}
			}
		}`), 0o644))

		// Create resolver document
		resolverPath := filepath.Join(tokensDir, "tokens.resolver.json")
		require.NoError(t, os.WriteFile(resolverPath, []byte(`{
			"version": "2025.10",
			"resolutionOrder": [
				{
					"type": "set",
					"name": "base",
					"sources": [
						{"$ref": "./palette.json"},
						{"$ref": "./colors.json"}
					]
				}
			]
		}`), 0o644))

		server, err := NewServer()
		require.NoError(t, err)
		defer func() { _ = server.Close() }()

		server.SetRootPath(tmpDir)
		server.SetConfig(types.ServerConfig{
			Resolvers: []string{resolverPath},
		})

		err = server.LoadTokensFromConfig()
		require.NoError(t, err)

		assert.NotNil(t, server.Token("color-neutral-100"), "palette token should be loaded")
		assert.NotNil(t, server.Token("color-surface-lowered"), "colors token should be loaded")
	})

	t.Run("resolves relative resolver paths", func(t *testing.T) {
		tmpDir := t.TempDir()
		tokensDir := filepath.Join(tmpDir, "src", "tokens")
		require.NoError(t, os.MkdirAll(tokensDir, 0o755))

		require.NoError(t, os.WriteFile(filepath.Join(tokensDir, "palette.json"), []byte(`{
			"color": {
				"primary": {"$value": "#ff0000", "$type": "color"}
			}
		}`), 0o644))

		resolverPath := filepath.Join(tokensDir, "tokens.resolver.json")
		require.NoError(t, os.WriteFile(resolverPath, []byte(`{
			"version": "2025.10",
			"resolutionOrder": [
				{"sources": [{"$ref": "./palette.json"}]}
			]
		}`), 0o644))

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
		tmpDir := t.TempDir()

		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "palette.json"), []byte(`{
			"color": {
				"$type": "color",
				"neutral": {"100": {"$value": "#f5f5f5"}}
			}
		}`), 0o644))

		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "semantic.json"), []byte(`{
			"color": {
				"surface": {"lowered": {"$value": "{color.neutral.100}"}}
			}
		}`), 0o644))

		resolverPath := filepath.Join(tmpDir, "resolver.json")
		require.NoError(t, os.WriteFile(resolverPath, []byte(`{
			"version": "2025.10",
			"resolutionOrder": [
				{"sources": [
					{"$ref": "./palette.json"},
					{"$ref": "./semantic.json"}
				]}
			]
		}`), 0o644))

		server, err := NewServer()
		require.NoError(t, err)
		defer func() { _ = server.Close() }()

		server.SetRootPath(tmpDir)
		server.SetConfig(types.ServerConfig{
			Resolvers: []string{resolverPath},
		})

		err = server.LoadTokensFromConfig()
		require.NoError(t, err)

		tok := server.Token("color-surface-lowered")
		require.NotNil(t, tok)
		// After alias resolution, the token should have a resolved value
		assert.True(t, tok.IsResolved, "alias should be resolved across resolver sources")
	})
}
