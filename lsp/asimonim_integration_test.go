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

	t.Run("merges resolvers from both package.json and asimonim config", func(t *testing.T) {
		tmpDir := t.TempDir()

		configDir := filepath.Join(tmpDir, ".config")
		require.NoError(t, os.MkdirAll(configDir, 0o755))

		// First resolver: design tokens (from package.json)
		colorsDir := filepath.Join(tmpDir, "src", "design-tokens")
		require.NoError(t, os.MkdirAll(colorsDir, 0o755))

		require.NoError(t, os.WriteFile(filepath.Join(colorsDir, "palette.json"), []byte(`{
			"color": {
				"$type": "color",
				"neutral": {"100": {"$value": "#f5f5f5"}}
			}
		}`), 0o644))

		require.NoError(t, os.WriteFile(filepath.Join(colorsDir, "colors.json"), []byte(`{
			"color": {
				"surface": {"lowered": {"$value": "{color.neutral.100}"}}
			}
		}`), 0o644))

		require.NoError(t, os.WriteFile(filepath.Join(colorsDir, "tokens.resolver.json"), []byte(`{
			"version": "2025.10",
			"resolutionOrder": [{
				"type": "set", "name": "base",
				"sources": [
					{"$ref": "./palette.json"},
					{"$ref": "./colors.json"}
				]
			}]
		}`), 0o644))

		// Second resolver: typography (from asimonim config)
		typoDir := filepath.Join(tmpDir, "src", "typography")
		require.NoError(t, os.MkdirAll(typoDir, 0o755))

		require.NoError(t, os.WriteFile(filepath.Join(typoDir, "fonts.json"), []byte(`{
			"font": {
				"$type": "fontFamily",
				"body": {"$value": "Inter, sans-serif"}
			}
		}`), 0o644))

		require.NoError(t, os.WriteFile(filepath.Join(typoDir, "typography.resolver.json"), []byte(`{
			"version": "2025.10",
			"resolutionOrder": [{
				"type": "set", "name": "base",
				"sources": [{"$ref": "./fonts.json"}]
			}]
		}`), 0o644))

		// package.json has one resolver
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(`{
			"designTokensLanguageServer": {
				"resolvers": [
					"./src/design-tokens/tokens.resolver.json"
				]
			}
		}`), 0o644))

		// asimonim config has a different resolver
		require.NoError(t, os.WriteFile(filepath.Join(configDir, "design-tokens.yaml"),
			[]byte("resolvers:\n  - ./src/typography/typography.resolver.json\n"), 0o644))

		server, err := NewServer()
		require.NoError(t, err)
		defer func() { _ = server.Close() }()

		server.SetRootPath(tmpDir)
		err = server.LoadPackageJsonConfig()
		require.NoError(t, err)

		cfg := server.GetConfig()
		// package.json resolver wins; asimonim resolver is not merged because
		// package.json already has resolvers set
		require.Len(t, cfg.Resolvers, 1, "package.json resolvers should take precedence")

		err = server.LoadTokensFromConfig()
		require.NoError(t, err)

		// Only color tokens from the package.json resolver should be loaded
		assert.NotNil(t, server.Token("color-neutral-100"), "palette token should be loaded")
		assert.NotNil(t, server.Token("color-surface-lowered"), "colors token should be loaded")
		assert.Nil(t, server.Token("font-body"), "typography token should NOT be loaded (asimonim resolver not merged)")
	})

	t.Run("merges asimonim resolvers when package.json has no resolvers", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create .config directory with resolvers config
		configDir := filepath.Join(tmpDir, ".config")
		require.NoError(t, os.MkdirAll(configDir, 0o755))

		tokensDir := filepath.Join(tmpDir, "src", "design-tokens")
		require.NoError(t, os.MkdirAll(tokensDir, 0o755))

		// Create source token files
		require.NoError(t, os.WriteFile(filepath.Join(tokensDir, "palette.json"), []byte(`{
			"color": {
				"$type": "color",
				"neutral": {"100": {"$value": "#f5f5f5"}}
			}
		}`), 0o644))

		require.NoError(t, os.WriteFile(filepath.Join(tokensDir, "colors.json"), []byte(`{
			"color": {
				"surface": {"lowered": {"$value": "{color.neutral.100}"}}
			}
		}`), 0o644))

		// Create resolver document
		resolverPath := filepath.Join(tokensDir, "tokens.resolver.json")
		require.NoError(t, os.WriteFile(resolverPath, []byte(`{
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

		// package.json has designTokensLanguageServer but no resolvers
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(`{
			"designTokensLanguageServer": {
				"tokensFiles": []
			}
		}`), 0o644))

		// asimonim config has resolvers
		configContent := "resolvers:\n  - ./src/design-tokens/tokens.resolver.json\n"
		require.NoError(t, os.WriteFile(filepath.Join(configDir, "design-tokens.yaml"), []byte(configContent), 0o644))

		server, err := NewServer()
		require.NoError(t, err)
		defer func() { _ = server.Close() }()

		server.SetRootPath(tmpDir)
		err = server.LoadPackageJsonConfig()
		require.NoError(t, err)

		cfg := server.GetConfig()
		require.Len(t, cfg.Resolvers, 1, "resolvers from asimonim config should be merged")

		// Load tokens
		err = server.LoadTokensFromConfig()
		require.NoError(t, err)

		assert.NotNil(t, server.Token("color-neutral-100"), "palette token should be loaded")
		assert.NotNil(t, server.Token("color-surface-lowered"), "colors token should be loaded")
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
