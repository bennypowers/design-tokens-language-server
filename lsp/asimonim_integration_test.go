package lsp

import (
	"os"
	"path/filepath"
	"testing"

	"bennypowers.dev/dtls/lsp/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// copyFixture copies a testdata fixture directory into a temporary directory
// and returns the path to the copy.
func copyFixture(t *testing.T, fixtureName string) string {
	t.Helper()
	tmpDir := t.TempDir()
	src := filepath.Join("testdata", "asimonim-integration", fixtureName)
	require.NoError(t, os.CopyFS(tmpDir, os.DirFS(src)))
	return tmpDir
}

// TestAsimonimConfigIntegration tests the full flow from config loading to token resolution
func TestAsimonimConfigIntegration(t *testing.T) {
	t.Run("loads tokens from glob patterns in config", func(t *testing.T) {
		tmpDir := copyFixture(t, "glob-patterns")

		server, err := NewServer()
		require.NoError(t, err)
		server.SetRootPath(tmpDir)

		err = server.LoadPackageJsonConfig()
		require.NoError(t, err)

		cfg := server.GetConfig()
		t.Logf("Config loaded: prefix=%s, tokensFiles=%d", cfg.Prefix, len(cfg.TokensFiles))
		assert.Equal(t, "rh", cfg.Prefix)
		assert.Equal(t, 3, len(cfg.TokensFiles), "should have 3 expanded file paths")

		err = server.LoadTokensFromConfig()
		require.NoError(t, err)

		tokenCount := server.tokens.Count()
		t.Logf("Loaded %d tokens", tokenCount)
		assert.Greater(t, tokenCount, 0, "should have loaded some tokens")

		assert.NotNil(t, server.tokens.Get("color.accent.base.on-light"),
			"color.accent.base.on-light should exist")
		assert.NotNil(t, server.tokens.Get("color.blue.10"),
			"color.blue.10 should exist (from crayon/blue.yaml)")
		assert.NotNil(t, server.tokens.Get("color.brand.red"),
			"color.brand.red should exist (from brand.yml)")
	})

	t.Run("merges resolvers from both package.json and asimonim config", func(t *testing.T) {
		tmpDir := copyFixture(t, "merges-resolvers")

		server, err := NewServer()
		require.NoError(t, err)
		defer func() { _ = server.Close() }()

		server.SetRootPath(tmpDir)
		err = server.LoadPackageJsonConfig()
		require.NoError(t, err)

		cfg := server.GetConfig()
		require.Len(t, cfg.Resolvers, 1, "package.json resolvers should take precedence")

		err = server.LoadTokensFromConfig()
		require.NoError(t, err)

		assert.NotNil(t, server.Token("color-neutral-100"), "palette token should be loaded")
		assert.NotNil(t, server.Token("color-surface-lowered"), "colors token should be loaded")
		assert.Nil(t, server.Token("font-body"),
			"typography token should NOT be loaded (asimonim resolver not merged)")
	})

	t.Run("merges asimonim resolvers when package.json has no resolvers", func(t *testing.T) {
		tmpDir := copyFixture(t, "merges-asimonim-resolvers")

		server, err := NewServer()
		require.NoError(t, err)
		defer func() { _ = server.Close() }()

		server.SetRootPath(tmpDir)
		err = server.LoadPackageJsonConfig()
		require.NoError(t, err)

		cfg := server.GetConfig()
		require.Len(t, cfg.Resolvers, 1, "resolvers from asimonim config should be merged")

		err = server.LoadTokensFromConfig()
		require.NoError(t, err)

		assert.NotNil(t, server.Token("color-neutral-100"), "palette token should be loaded")
		assert.NotNil(t, server.Token("color-surface-lowered"), "colors token should be loaded")
	})

	t.Run("SetConfig does not clear tokensFiles from asimonim config", func(t *testing.T) {
		tmpDir := copyFixture(t, "setconfig-preserves")

		server, err := NewServer()
		require.NoError(t, err)
		server.SetRootPath(tmpDir)

		err = server.LoadPackageJsonConfig()
		require.NoError(t, err)

		cfg := server.GetConfig()
		require.Equal(t, 1, len(cfg.TokensFiles), "should have 1 token file")

		server.SetConfig(types.ServerConfig{
			Prefix: "client-prefix",
		})

		err = server.LoadPackageJsonConfig()
		require.NoError(t, err)

		cfg = server.GetConfig()
		assert.Equal(t, "client-prefix", cfg.Prefix)
		assert.Equal(t, 1, len(cfg.TokensFiles),
			"tokensFiles should be restored from asimonim config")
	})
}
