package lsp

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"bennypowers.dev/asimonim/load"
	"bennypowers.dev/dtls/lsp/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadTokensFromConfig(t *testing.T) {
	t.Run("explicit token files", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create test token files
		tokens1 := filepath.Join(tmpDir, "tokens1.json")
		require.NoError(t, os.WriteFile(tokens1, []byte(`{
			"color": {
				"primary": {
					"$value": "#ff0000",
					"$type": "color"
				}
			}
		}`), 0o644))

		tokens2 := filepath.Join(tmpDir, "tokens2.json")
		require.NoError(t, os.WriteFile(tokens2, []byte(`{
			"spacing": {
				"small": {
					"$value": "8px",
					"$type": "dimension"
				}
			}
		}`), 0o644))

		server, err := NewServer()
		require.NoError(t, err)
		defer func() { _ = server.Close() }()

		// Set workspace root and explicit token files
		server.SetRootPath(tmpDir)
		server.SetConfig(types.ServerConfig{
			TokensFiles: []any{
				tokens1,
				tokens2,
			},
			Prefix:       "ds",
			GroupMarkers: []string{"_"},
		})

		err = server.LoadTokensFromConfig()
		require.NoError(t, err)

		// Verify tokens loaded
		assert.NotNil(t, server.Token("color-primary"))
		assert.NotNil(t, server.Token("spacing-small"))
		assert.Equal(t, 2, server.TokenCount())
	})

	t.Run("explicit token files with objects", func(t *testing.T) {
		tmpDir := t.TempDir()

		tokens := filepath.Join(tmpDir, "tokens.json")
		require.NoError(t, os.WriteFile(tokens, []byte(`{
			"color": {
				"primary": {
					"$value": "#ff0000",
					"$type": "color"
				}
			}
		}`), 0o644))

		server, err := NewServer()
		require.NoError(t, err)
		defer func() { _ = server.Close() }()

		server.SetRootPath(tmpDir)
		server.SetConfig(types.ServerConfig{
			TokensFiles: []any{
				map[string]any{
					"path":   tokens,
					"prefix": "custom",
				},
			},
		})

		err = server.LoadTokensFromConfig()
		require.NoError(t, err)

		assert.Equal(t, 1, server.TokenCount())
	})

	t.Run("empty tokensFiles loads nothing", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create a token file (that won't be loaded)
		tokensFile := filepath.Join(tmpDir, "tokens.json")
		require.NoError(t, os.WriteFile(tokensFile, []byte(`{
			"color": {
				"primary": {
					"$value": "#ff0000",
					"$type": "color"
				}
			}
		}`), 0o644))

		server, err := NewServer()
		require.NoError(t, err)
		defer func() { _ = server.Close() }()

		server.SetRootPath(tmpDir)
		server.SetConfig(types.ServerConfig{
			TokensFiles: []any{}, // Empty does NOT trigger auto-discovery
		})

		err = server.LoadTokensFromConfig()
		require.NoError(t, err)

		// Should NOT load any tokens (matches TypeScript behavior)
		assert.Equal(t, 0, server.TokenCount())
	})

	t.Run("nil tokensFiles loads nothing", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create a token file (that won't be loaded)
		tokensFile := filepath.Join(tmpDir, "tokens.yaml")
		require.NoError(t, os.WriteFile(tokensFile, []byte(`
color:
  primary:
    $value: "#ff0000"
    $type: "color"
`), 0o644))

		server, err := NewServer()
		require.NoError(t, err)
		defer func() { _ = server.Close() }()

		server.SetRootPath(tmpDir)
		server.SetConfig(types.ServerConfig{
			TokensFiles: nil, // nil does NOT trigger auto-discovery
		})

		err = server.LoadTokensFromConfig()
		require.NoError(t, err)

		// Should NOT load any tokens (matches TypeScript behavior)
		assert.Equal(t, 0, server.TokenCount())
	})

	t.Run("npm: protocol support", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create mock npm package
		pkgDir := filepath.Join(tmpDir, "node_modules", "@design", "tokens")
		require.NoError(t, os.MkdirAll(pkgDir, 0o755))

		tokensFile := filepath.Join(pkgDir, "tokens.json")
		require.NoError(t, os.WriteFile(tokensFile, []byte(`{
			"color": {
				"brand": {
					"$value": "#0000ff",
					"$type": "color"
				}
			}
		}`), 0o644))

		server, err := NewServer()
		require.NoError(t, err)
		defer func() { _ = server.Close() }()

		server.SetRootPath(tmpDir)
		server.SetConfig(types.ServerConfig{
			TokensFiles: []any{
				"npm:@design/tokens/tokens.json",
			},
		})

		err = server.LoadTokensFromConfig()
		require.NoError(t, err)

		assert.Equal(t, 1, server.TokenCount())
		assert.NotNil(t, server.Token("color-brand"))
	})

	t.Run("relative paths resolved", func(t *testing.T) {
		tmpDir := t.TempDir()

		tokensFile := filepath.Join(tmpDir, "design", "tokens.json")
		require.NoError(t, os.MkdirAll(filepath.Dir(tokensFile), 0o755))
		require.NoError(t, os.WriteFile(tokensFile, []byte(`{
			"color": {
				"accent": {
					"$value": "#00ff00",
					"$type": "color"
				}
			}
		}`), 0o644))

		server, err := NewServer()
		require.NoError(t, err)
		defer func() { _ = server.Close() }()

		server.SetRootPath(tmpDir)
		server.SetConfig(types.ServerConfig{
			TokensFiles: []any{
				"./design/tokens.json",
			},
		})

		err = server.LoadTokensFromConfig()
		require.NoError(t, err)

		assert.Equal(t, 1, server.TokenCount())
		assert.NotNil(t, server.Token("color-accent"))
	})

	t.Run("error on invalid path", func(t *testing.T) {
		tmpDir := t.TempDir()

		server, err := NewServer()
		require.NoError(t, err)
		defer func() { _ = server.Close() }()

		server.SetRootPath(tmpDir)
		server.SetConfig(types.ServerConfig{
			TokensFiles: []any{
				"/nonexistent/tokens.json",
			},
		})

		err = server.LoadTokensFromConfig()
		assert.Error(t, err)
	})

	t.Run("reload previously loaded files", func(t *testing.T) {
		tmpDir := t.TempDir()

		tokensFile := filepath.Join(tmpDir, "tokens.json")
		require.NoError(t, os.WriteFile(tokensFile, []byte(`{
			"color": {
				"primary": {
					"$value": "#ff0000",
					"$type": "color"
				}
			}
		}`), 0o644))

		server, err := NewServer()
		require.NoError(t, err)
		defer func() { _ = server.Close() }()

		// Load file programmatically (simulates test usage)
		err = server.LoadTokenFile(tokensFile, "")
		require.NoError(t, err)
		assert.Equal(t, 1, server.TokenCount())

		// Clear tokens
		server.tokens.Clear()
		assert.Equal(t, 0, server.TokenCount())

		// Reload should restore from loadedFiles
		err = server.LoadTokensFromConfig()
		require.NoError(t, err)
		assert.Equal(t, 1, server.TokenCount())
	})

	t.Run("groupMarkers per-file", func(t *testing.T) {
		tmpDir := t.TempDir()

		tokensFile := filepath.Join(tmpDir, "tokens.json")
		require.NoError(t, os.WriteFile(tokensFile, []byte(`{
			"color": {
				"primary": {
					"DEFAULT": {
						"$value": "#ff0000",
						"$type": "color"
					}
				}
			}
		}`), 0o644))

		server, err := NewServer()
		require.NoError(t, err)
		defer func() { _ = server.Close() }()

		server.SetRootPath(tmpDir)
		server.SetConfig(types.ServerConfig{
			TokensFiles: []any{
				map[string]any{
					"path":         tokensFile,
					"groupMarkers": []any{"DEFAULT"},
				},
			},
		})

		err = server.LoadTokensFromConfig()
		require.NoError(t, err)

		// Should have both color-primary (group) and color-primary-DEFAULT (value)
		assert.Greater(t, server.TokenCount(), 0)
	})

	t.Run("error on empty string path", func(t *testing.T) {
		server, err := NewServer()
		require.NoError(t, err)
		defer func() { _ = server.Close() }()

		server.SetConfig(types.ServerConfig{
			TokensFiles: []any{
				"", // Empty string path
			},
		})

		err = server.LoadTokensFromConfig()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "must not be empty")
	})

	t.Run("error on empty path in object", func(t *testing.T) {
		server, err := NewServer()
		require.NoError(t, err)
		defer func() { _ = server.Close() }()

		server.SetConfig(types.ServerConfig{
			TokensFiles: []any{
				map[string]any{
					"path":   "", // Empty path in object
					"prefix": "custom",
				},
			},
		})

		err = server.LoadTokensFromConfig()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "must not be empty")
	})
}

func TestSetRootPath(t *testing.T) {
	server, err := NewServer()
	require.NoError(t, err)
	defer func() { _ = server.Close() }()

	// Initially empty
	assert.Empty(t, server.GetState().RootPath)

	// Set path using public API
	server.SetRootPath("/test/path")
	assert.Equal(t, "/test/path", server.GetState().RootPath)
	assert.Equal(t, "/test/path", server.RootPath())
}

// mockFetcher implements load.Fetcher for testing
type mockFetcher struct {
	data map[string][]byte
	err  error
}

func (m *mockFetcher) Fetch(_ context.Context, url string) ([]byte, error) {
	if m.err != nil {
		return nil, m.err
	}
	data, ok := m.data[url]
	if !ok {
		return nil, fmt.Errorf("not found: %s", url)
	}
	return data, nil
}

func TestLoadFromCDN(t *testing.T) {
	colorBrandJSON, err := os.ReadFile("testdata/tokens/color_brand.json")
	require.NoError(t, err)

	t.Run("successful CDN fallback", func(t *testing.T) {
		server, err := NewServer()
		require.NoError(t, err)
		defer func() { _ = server.Close() }()

		fetcher := &mockFetcher{
			data: map[string][]byte{
				"https://unpkg.com/@design/tokens/tokens.json": colorBrandJSON,
			},
		}

		cfg := types.ServerConfig{NetworkTimeout: 30}
		opts := &TokenFileOptions{}
		count, err := server.loadFromCDN(fetcher, "npm:@design/tokens/tokens.json", opts, cfg)
		require.NoError(t, err)

		assert.Equal(t, 1, count)
		assert.Equal(t, 1, server.TokenCount())
		assert.NotNil(t, server.Token("color-brand"))
	})

	t.Run("CDN fetch error", func(t *testing.T) {
		server, err := NewServer()
		require.NoError(t, err)
		defer func() { _ = server.Close() }()

		fetcher := &mockFetcher{
			err: fmt.Errorf("network error"),
		}

		cfg := types.ServerConfig{NetworkTimeout: 30}
		opts := &TokenFileOptions{}
		count, err := server.loadFromCDN(fetcher, "npm:@design/tokens/tokens.json", opts, cfg)
		require.Error(t, err)
		assert.Equal(t, 0, count)
		assert.Contains(t, err.Error(), "CDN fetch failed")
	})

	t.Run("invalid npm specifier for CDN", func(t *testing.T) {
		server, err := NewServer()
		require.NoError(t, err)
		defer func() { _ = server.Close() }()

		fetcher := &mockFetcher{}
		cfg := types.ServerConfig{}
		opts := &TokenFileOptions{}
		// npm:package without a file component can't map to CDN
		count, err := server.loadFromCDN(fetcher, "npm:@design/tokens", opts, cfg)
		require.Error(t, err)
		assert.Equal(t, 0, count)
		assert.Contains(t, err.Error(), "cannot determine CDN URL")
	})

	t.Run("CDN fallback with prefix", func(t *testing.T) {
		server, err := NewServer()
		require.NoError(t, err)
		defer func() { _ = server.Close() }()

		fetcher := &mockFetcher{
			data: map[string][]byte{
				"https://unpkg.com/@design/tokens/tokens.json": colorBrandJSON,
			},
		}

		cfg := types.ServerConfig{NetworkTimeout: 30}
		opts := &TokenFileOptions{Prefix: "ds"}
		count, err := server.loadFromCDN(fetcher, "npm:@design/tokens/tokens.json", opts, cfg)
		require.NoError(t, err)

		assert.Equal(t, 1, count)
		assert.Equal(t, 1, server.TokenCount())
	})

	t.Run("configurable CDN provider", func(t *testing.T) {
		server, err := NewServer()
		require.NoError(t, err)
		defer func() { _ = server.Close() }()

		fetcher := &mockFetcher{
			data: map[string][]byte{
				"https://cdn.jsdelivr.net/npm/@design/tokens/tokens.json": colorBrandJSON,
			},
		}

		cfg := types.ServerConfig{NetworkTimeout: 30, CDN: "jsdelivr"}
		opts := &TokenFileOptions{}
		count, err := server.loadFromCDN(fetcher, "npm:@design/tokens/tokens.json", opts, cfg)
		require.NoError(t, err)

		assert.Equal(t, 1, count)
		assert.Equal(t, 1, server.TokenCount())
		assert.NotNil(t, server.Token("color-brand"))
	})

	t.Run("defaults to unpkg when CDN is empty", func(t *testing.T) {
		server, err := NewServer()
		require.NoError(t, err)
		defer func() { _ = server.Close() }()

		fetcher := &mockFetcher{
			data: map[string][]byte{
				"https://unpkg.com/@design/tokens/tokens.json": colorBrandJSON,
			},
		}

		cfg := types.ServerConfig{NetworkTimeout: 30, CDN: ""}
		opts := &TokenFileOptions{}
		count, err := server.loadFromCDN(fetcher, "npm:@design/tokens/tokens.json", opts, cfg)
		require.NoError(t, err)

		assert.Equal(t, 1, count)
		assert.Equal(t, 1, server.TokenCount())
	})
}

func TestLoadPackageJsonConfig_NetworkFields(t *testing.T) {
	t.Run("merges networkFallback and CDN from package.json", func(t *testing.T) {
		tmpDir := t.TempDir()

		packageJSON := `{
			"name": "test",
			"designTokensLanguageServer": {
				"networkFallback": true,
				"networkTimeout": 45,
				"cdn": "jsdelivr",
				"tokensFiles": ["tokens.json"]
			}
		}`
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(packageJSON), 0o644))

		server, err := NewServer()
		require.NoError(t, err)
		defer func() { _ = server.Close() }()

		server.SetRootPath(tmpDir)
		err = server.LoadPackageJsonConfig()
		require.NoError(t, err)

		cfg := server.GetConfig()
		assert.Len(t, cfg.TokensFiles, 1)
		assert.True(t, cfg.NetworkFallback)
		assert.Equal(t, 45, cfg.NetworkTimeout)
		assert.Equal(t, "jsdelivr", cfg.CDN)
	})
}

func TestNetworkFallbackInLoadExplicitTokenFiles(t *testing.T) {
	t.Run("fallback disabled - npm error propagates", func(t *testing.T) {
		tmpDir := t.TempDir()

		server, err := NewServer()
		require.NoError(t, err)
		defer func() { _ = server.Close() }()

		// No node_modules, networkFallback disabled
		server.SetRootPath(tmpDir)
		server.SetConfig(types.ServerConfig{
			TokensFiles:     []any{"npm:@missing/tokens/tokens.json"},
			NetworkFallback: false,
		})

		err = server.LoadTokensFromConfig()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to resolve path")
		assert.Equal(t, 0, server.TokenCount())
	})

	t.Run("fallback disabled - package specifier error propagates", func(t *testing.T) {
		tmpDir := t.TempDir()

		server, err := NewServer()
		require.NoError(t, err)
		defer func() { _ = server.Close() }()

		server.SetRootPath(tmpDir)
		server.SetConfig(types.ServerConfig{
			TokensFiles:     []any{"npm:@test/tokens/tokens.json"},
			NetworkFallback: false,
		})

		err = server.LoadTokensFromConfig()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to resolve path")
		assert.Equal(t, 0, server.TokenCount())
	})

	t.Run("non-npm path - no fallback attempted", func(t *testing.T) {
		tmpDir := t.TempDir()

		server, err := NewServer()
		require.NoError(t, err)
		defer func() { _ = server.Close() }()

		server.SetRootPath(tmpDir)
		server.SetConfig(types.ServerConfig{
			TokensFiles:     []any{"/nonexistent/tokens.json"},
			NetworkFallback: true,
		})

		err = server.LoadTokensFromConfig()
		require.Error(t, err)
		// Should fail with a regular file error, not a CDN error
		assert.NotContains(t, err.Error(), "CDN")
	})
}

func TestNetworkTimeout(t *testing.T) {
	t.Run("uses configured timeout", func(t *testing.T) {
		cfg := types.ServerConfig{NetworkTimeout: 60}
		d := networkTimeout(cfg)
		assert.Equal(t, 60*time.Second, d)
	})

	t.Run("falls back to default when zero", func(t *testing.T) {
		cfg := types.ServerConfig{NetworkTimeout: 0}
		d := networkTimeout(cfg)
		assert.Equal(t, load.DefaultTimeout, d)
	})

	t.Run("falls back to default when negative", func(t *testing.T) {
		cfg := types.ServerConfig{NetworkTimeout: -1}
		d := networkTimeout(cfg)
		assert.Equal(t, load.DefaultTimeout, d)
	})
}

func TestLoadTokensFromJSON(t *testing.T) {
	colorPrimaryJSON, err := os.ReadFile("testdata/tokens/color_primary.json")
	require.NoError(t, err)

	t.Run("loads tokens successfully", func(t *testing.T) {
		server, err := NewServer()
		require.NoError(t, err)
		defer func() { _ = server.Close() }()

		err = server.LoadTokensFromJSON(colorPrimaryJSON, "")
		require.NoError(t, err)

		assert.Equal(t, 1, server.TokenCount())
		assert.NotNil(t, server.Token("color-primary"))
	})

	t.Run("loads tokens with prefix", func(t *testing.T) {
		server, err := NewServer()
		require.NoError(t, err)
		defer func() { _ = server.Close() }()

		err = server.LoadTokensFromJSON(colorPrimaryJSON, "ds")
		require.NoError(t, err)

		tok := server.Token("color-primary")
		require.NotNil(t, tok)
		assert.Equal(t, "ds", tok.Prefix)
	})

	t.Run("returns error for invalid JSON", func(t *testing.T) {
		server, err := NewServer()
		require.NoError(t, err)
		defer func() { _ = server.Close() }()

		err = server.LoadTokensFromJSON([]byte(`{invalid`), "")
		require.Error(t, err)
	})

	t.Run("resolves aliases", func(t *testing.T) {
		server, err := NewServer()
		require.NoError(t, err)
		defer func() { _ = server.Close() }()

		aliasJSON := []byte(`{
			"color": {
				"primary": {"$value": "#ff0000", "$type": "color"},
				"alias": {"$value": "{color.primary}", "$type": "color"}
			}
		}`)
		err = server.LoadTokensFromJSON(aliasJSON, "")
		require.NoError(t, err)
		assert.Equal(t, 2, server.TokenCount())
	})
}

func TestLoadTokensFromDocumentContent(t *testing.T) {
	t.Run("loads JSON content", func(t *testing.T) {
		server, err := NewServer()
		require.NoError(t, err)
		defer func() { _ = server.Close() }()

		content := `{"color": {"primary": {"$value": "#ff0000", "$type": "color"}}}`
		err = server.LoadTokensFromDocumentContent("file:///tmp/tokens.json", "json", content)
		require.NoError(t, err)

		assert.Equal(t, 1, server.TokenCount())
		tok := server.Token("color-primary")
		require.NotNil(t, tok)
		assert.Equal(t, "file:///tmp/tokens.json", tok.DefinitionURI)
	})

	t.Run("loads YAML content", func(t *testing.T) {
		server, err := NewServer()
		require.NoError(t, err)
		defer func() { _ = server.Close() }()

		content := "color:\n  primary:\n    $value: \"#ff0000\"\n    $type: color\n"
		err = server.LoadTokensFromDocumentContent("file:///tmp/tokens.yaml", "yaml", content)
		require.NoError(t, err)

		assert.Equal(t, 1, server.TokenCount())
	})

	t.Run("skips unsupported language", func(t *testing.T) {
		server, err := NewServer()
		require.NoError(t, err)
		defer func() { _ = server.Close() }()

		err = server.LoadTokensFromDocumentContent("file:///tmp/style.css", "css", "body { color: red; }")
		require.NoError(t, err)
		assert.Equal(t, 0, server.TokenCount())
	})

	t.Run("returns error for invalid content", func(t *testing.T) {
		server, err := NewServer()
		require.NoError(t, err)
		defer func() { _ = server.Close() }()

		err = server.LoadTokensFromDocumentContent("file:///tmp/tokens.json", "json", `{invalid`)
		require.Error(t, err)
	})
}

func TestParseAndAddTokens(t *testing.T) {
	colorPrimaryJSON, err := os.ReadFile("testdata/tokens/color_primary.json")
	require.NoError(t, err)

	t.Run("adds tokens with file path and URI", func(t *testing.T) {
		server, err := NewServer()
		require.NoError(t, err)
		defer func() { _ = server.Close() }()

		count, err := server.parseAndAddTokens(colorPrimaryJSON, "/tmp/tokens.json", "file:///tmp/tokens.json", &TokenFileOptions{})
		require.NoError(t, err)
		assert.Equal(t, 1, count)

		tok := server.Token("color-primary")
		require.NotNil(t, tok)
		assert.Equal(t, "/tmp/tokens.json", tok.FilePath)
		assert.Equal(t, "file:///tmp/tokens.json", tok.DefinitionURI)
	})

	t.Run("adds tokens with empty file path (CDN source)", func(t *testing.T) {
		server, err := NewServer()
		require.NoError(t, err)
		defer func() { _ = server.Close() }()

		count, err := server.parseAndAddTokens(colorPrimaryJSON, "", "https://unpkg.com/@design/tokens/tokens.json", &TokenFileOptions{})
		require.NoError(t, err)
		assert.Equal(t, 1, count)

		tok := server.Token("color-primary")
		require.NotNil(t, tok)
		assert.Empty(t, tok.FilePath)
		assert.Equal(t, "https://unpkg.com/@design/tokens/tokens.json", tok.DefinitionURI)
	})

	t.Run("applies prefix", func(t *testing.T) {
		server, err := NewServer()
		require.NoError(t, err)
		defer func() { _ = server.Close() }()

		count, err := server.parseAndAddTokens(colorPrimaryJSON, "", "", &TokenFileOptions{Prefix: "ds"})
		require.NoError(t, err)
		assert.Equal(t, 1, count)

		tok := server.Token("color-primary")
		require.NotNil(t, tok, "token should exist by name")
		assert.Equal(t, "ds", tok.Prefix, "token should have prefix set")
	})

	t.Run("returns error for invalid JSON", func(t *testing.T) {
		server, err := NewServer()
		require.NoError(t, err)
		defer func() { _ = server.Close() }()

		_, err = server.parseAndAddTokens([]byte(`{invalid`), "", "", &TokenFileOptions{})
		require.Error(t, err)
	})

	t.Run("nil opts defaults to empty", func(t *testing.T) {
		server, err := NewServer()
		require.NoError(t, err)
		defer func() { _ = server.Close() }()

		count, err := server.parseAndAddTokens(colorPrimaryJSON, "", "", nil)
		require.NoError(t, err)
		assert.Equal(t, 1, count)
	})

	t.Run("validates schema consistency with file path", func(t *testing.T) {
		server, err := NewServer()
		require.NoError(t, err)
		defer func() { _ = server.Close() }()

		// This file uses 2025.10 schema with draft group markers, triggering validation warnings
		mixedJSON, err := os.ReadFile("testdata/tokens/mixed_schema.json")
		require.NoError(t, err)
		count, err := server.parseAndAddTokens(mixedJSON, "/tmp/mixed.json", "file:///tmp/mixed.json", &TokenFileOptions{})
		require.NoError(t, err)
		assert.Greater(t, count, 0)
	})

	t.Run("validates schema consistency without file path", func(t *testing.T) {
		server, err := NewServer()
		require.NoError(t, err)
		defer func() { _ = server.Close() }()

		mixedJSON, err := os.ReadFile("testdata/tokens/mixed_schema.json")
		require.NoError(t, err)
		count, err := server.parseAndAddTokens(mixedJSON, "", "", &TokenFileOptions{})
		require.NoError(t, err)
		assert.Greater(t, count, 0)
	})

	t.Run("logs with groupMarkers", func(t *testing.T) {
		server, err := NewServer()
		require.NoError(t, err)
		defer func() { _ = server.Close() }()

		count, err := server.parseAndAddTokens(colorPrimaryJSON, "/tmp/tokens.json", "file:///tmp/tokens.json", &TokenFileOptions{
			Prefix:       "ds",
			GroupMarkers: []string{"DEFAULT"},
		})
		require.NoError(t, err)
		assert.Equal(t, 1, count)
	})

	t.Run("duplicate tokens are overwritten", func(t *testing.T) {
		server, err := NewServer()
		require.NoError(t, err)
		defer func() { _ = server.Close() }()

		// Load tokens once
		count, err := server.parseAndAddTokens(colorPrimaryJSON, "/a.json", "file:///a.json", &TokenFileOptions{})
		require.NoError(t, err)
		assert.Equal(t, 1, count)

		// Load same tokens again from a different file - manager overwrites without error
		count, err = server.parseAndAddTokens(colorPrimaryJSON, "/b.json", "file:///b.json", &TokenFileOptions{})
		require.NoError(t, err)
		assert.Equal(t, 1, count)
	})
}
