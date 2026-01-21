package lsp

import (
	"os"
	"path/filepath"
	"testing"

	"bennypowers.dev/dtls/lsp/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAsimonimConfigIntegration tests the full flow from config loading to token resolution
func TestAsimonimConfigIntegration(t *testing.T) {
	t.Run("loads tokens from glob patterns in config", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create .config directory and config file
		configDir := filepath.Join(tmpDir, ".config")
		require.NoError(t, os.MkdirAll(configDir, 0o755))

		// Create directory structure mimicking red-hat-design-tokens
		colorDir := filepath.Join(tmpDir, "tokens", "color")
		crayonDir := filepath.Join(colorDir, "crayon")
		require.NoError(t, os.MkdirAll(crayonDir, 0o755))

		// Create token files similar to red-hat-design-tokens
		// accent.yml - should be loaded
		require.NoError(t, os.WriteFile(filepath.Join(colorDir, "accent.yml"), []byte(`
color:
  accent:
    base:
      on-light:
        $value: "#0066cc"
`), 0o644))

		// blue.yaml in crayon - this is what was missing
		require.NoError(t, os.WriteFile(filepath.Join(crayonDir, "blue.yaml"), []byte(`
color:
  blue:
    "10":
      $value: "#e6f1ff"
    "20":
      $value: "#cce3ff"
`), 0o644))

		// brand.yml
		require.NoError(t, os.WriteFile(filepath.Join(colorDir, "brand.yml"), []byte(`
color:
  brand:
    red:
      $value: "#ee0000"
`), 0o644))

		// Config file with glob patterns
		configContent := `prefix: rh
files:
  - tokens/**/*.yaml
  - tokens/**/*.yml
`
		require.NoError(t, os.WriteFile(filepath.Join(configDir, "design-tokens.yaml"), []byte(configContent), 0o644))

		// Create server and simulate the initialization flow
		server, err := NewServer()
		require.NoError(t, err)
		server.SetRootPath(tmpDir)

		// Load config (this is what initialized.go does)
		err = server.LoadPackageJsonConfig()
		require.NoError(t, err)

		// Check config was loaded with expanded paths
		cfg := server.GetConfig()
		t.Logf("Config loaded: prefix=%s, tokensFiles=%d", cfg.Prefix, len(cfg.TokensFiles))
		for i, f := range cfg.TokensFiles {
			t.Logf("  [%d] %v", i, f)
		}
		assert.Equal(t, "rh", cfg.Prefix)
		assert.Equal(t, 3, len(cfg.TokensFiles), "should have 3 expanded file paths")

		// Load tokens (this is what initialized.go does next)
		err = server.LoadTokensFromConfig()
		require.NoError(t, err)

		// Check that all tokens were loaded
		tokenCount := server.tokens.Count()
		t.Logf("Loaded %d tokens", tokenCount)
		assert.Greater(t, tokenCount, 0, "should have loaded some tokens")

		// Check specific tokens exist
		// color.accent.base.on-light from accent.yml
		accentToken := server.tokens.Get("color.accent.base.on-light")
		assert.NotNil(t, accentToken, "color.accent.base.on-light should exist")

		// color.blue.10 from crayon/blue.yaml - this was previously missing
		blueToken := server.tokens.Get("color.blue.10")
		assert.NotNil(t, blueToken, "color.blue.10 should exist (from crayon/blue.yaml)")

		// color.brand.red from brand.yml - this was previously missing
		brandToken := server.tokens.Get("color.brand.red")
		assert.NotNil(t, brandToken, "color.brand.red should exist (from brand.yml)")
	})

	t.Run("SetConfig does not clear tokensFiles from asimonim config", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create .config directory and config file
		configDir := filepath.Join(tmpDir, ".config")
		require.NoError(t, os.MkdirAll(configDir, 0o755))

		// Create a token file
		tokensDir := filepath.Join(tmpDir, "tokens")
		require.NoError(t, os.MkdirAll(tokensDir, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(tokensDir, "test.yaml"), []byte(`
color:
  test:
    $value: "#fff"
`), 0o644))

		configContent := `prefix: test
files:
  - tokens/*.yaml
`
		require.NoError(t, os.WriteFile(filepath.Join(configDir, "design-tokens.yaml"), []byte(configContent), 0o644))

		// Create server and load config
		server, err := NewServer()
		require.NoError(t, err)
		server.SetRootPath(tmpDir)

		err = server.LoadPackageJsonConfig()
		require.NoError(t, err)

		cfg := server.GetConfig()
		require.Equal(t, 1, len(cfg.TokensFiles), "should have 1 token file")

		// Now simulate client sending config update with empty tokensFiles
		// This should NOT clear the asimonim config
		server.SetConfig(types.ServerConfig{
			Prefix: "client-prefix",
			// Note: no TokensFiles set
		})

		// Reload from package.json - this should restore the asimonim config
		err = server.LoadPackageJsonConfig()
		require.NoError(t, err)

		cfg = server.GetConfig()
		// The prefix should be the client prefix (takes precedence)
		assert.Equal(t, "client-prefix", cfg.Prefix)
		// But tokensFiles should be restored from asimonim config
		assert.Equal(t, 1, len(cfg.TokensFiles), "tokensFiles should be restored from asimonim config")
	})
}
