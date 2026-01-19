package lsp

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadAsimonimConfig(t *testing.T) {
	t.Run("returns nil when rootPath is empty", func(t *testing.T) {
		config, err := ReadAsimonimConfig("")
		require.NoError(t, err)
		assert.Nil(t, config)
	})

	t.Run("returns nil when config file doesn't exist", func(t *testing.T) {
		tmpDir := t.TempDir()
		config, err := ReadAsimonimConfig(tmpDir)
		require.NoError(t, err)
		assert.Nil(t, config)
	})

	t.Run("reads simple config", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create .config directory and config file
		configDir := filepath.Join(tmpDir, ".config")
		require.NoError(t, os.MkdirAll(configDir, 0o755))

		// Create a token file so it can be found
		tokensPath := filepath.Join(tmpDir, "tokens.json")
		require.NoError(t, os.WriteFile(tokensPath, []byte(`{"color":{"$value":"#fff"}}`), 0o644))

		configContent := `prefix: rh
files:
  - tokens.json
`
		require.NoError(t, os.WriteFile(filepath.Join(configDir, "design-tokens.yaml"), []byte(configContent), 0o644))

		config, err := ReadAsimonimConfig(tmpDir)
		require.NoError(t, err)
		require.NotNil(t, config)

		assert.Equal(t, "rh", config.Prefix)
		assert.Len(t, config.TokensFiles, 1)
	})

	t.Run("expands glob patterns", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create .config directory and config file
		configDir := filepath.Join(tmpDir, ".config")
		require.NoError(t, os.MkdirAll(configDir, 0o755))

		// Create directory structure with multiple token files
		tokensDir := filepath.Join(tmpDir, "tokens", "color")
		require.NoError(t, os.MkdirAll(tokensDir, 0o755))

		// Create token files
		require.NoError(t, os.WriteFile(filepath.Join(tokensDir, "blue.yaml"), []byte("color:\n  blue:\n    $value: \"#00f\""), 0o644))
		require.NoError(t, os.WriteFile(filepath.Join(tokensDir, "red.yaml"), []byte("color:\n  red:\n    $value: \"#f00\""), 0o644))
		require.NoError(t, os.WriteFile(filepath.Join(tokensDir, "green.yml"), []byte("color:\n  green:\n    $value: \"#0f0\""), 0o644))

		configContent := `prefix: rh
files:
  - tokens/**/*.yaml
  - tokens/**/*.yml
`
		require.NoError(t, os.WriteFile(filepath.Join(configDir, "design-tokens.yaml"), []byte(configContent), 0o644))

		config, err := ReadAsimonimConfig(tmpDir)
		require.NoError(t, err)
		require.NotNil(t, config)

		assert.Equal(t, "rh", config.Prefix)
		// Should find all 3 files (blue.yaml, red.yaml, green.yml)
		assert.Len(t, config.TokensFiles, 3, "glob should expand to 3 files: blue.yaml, red.yaml, green.yml")

		// Verify the paths are absolute
		for _, tokenFile := range config.TokensFiles {
			path, ok := tokenFile.(string)
			require.True(t, ok, "tokenFile should be a string")
			assert.True(t, filepath.IsAbs(path), "expanded path should be absolute: %s", path)
		}
	})

	t.Run("preserves prefix and groupMarkers from config", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create .config directory and config file
		configDir := filepath.Join(tmpDir, ".config")
		require.NoError(t, os.MkdirAll(configDir, 0o755))

		// Create a token file
		tokensPath := filepath.Join(tmpDir, "tokens.json")
		require.NoError(t, os.WriteFile(tokensPath, []byte(`{"color":{"$value":"#fff"}}`), 0o644))

		configContent := `prefix: custom
groupMarkers:
  - _
  - DEFAULT
files:
  - tokens.json
`
		require.NoError(t, os.WriteFile(filepath.Join(configDir, "design-tokens.yaml"), []byte(configContent), 0o644))

		config, err := ReadAsimonimConfig(tmpDir)
		require.NoError(t, err)
		require.NotNil(t, config)

		assert.Equal(t, "custom", config.Prefix)
		assert.Equal(t, []string{"_", "DEFAULT"}, config.GroupMarkers)
	})
}
