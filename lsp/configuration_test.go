package lsp

import (
	"os"
	"path/filepath"
	"testing"

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
		}`), 0644))

		tokens2 := filepath.Join(tmpDir, "tokens2.json")
		require.NoError(t, os.WriteFile(tokens2, []byte(`{
			"spacing": {
				"small": {
					"$value": "8px",
					"$type": "dimension"
				}
			}
		}`), 0644))

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
		}`), 0644))

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
		}`), 0644))

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
`), 0644))

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
		require.NoError(t, os.MkdirAll(pkgDir, 0755))

		tokensFile := filepath.Join(pkgDir, "tokens.json")
		require.NoError(t, os.WriteFile(tokensFile, []byte(`{
			"color": {
				"brand": {
					"$value": "#0000ff",
					"$type": "color"
				}
			}
		}`), 0644))

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
		require.NoError(t, os.MkdirAll(filepath.Dir(tokensFile), 0755))
		require.NoError(t, os.WriteFile(tokensFile, []byte(`{
			"color": {
				"accent": {
					"$value": "#00ff00",
					"$type": "color"
				}
			}
		}`), 0644))

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
		}`), 0644))

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
		}`), 0644))

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

