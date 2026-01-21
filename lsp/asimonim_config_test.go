package lsp

import (
	"os"
	"path/filepath"
	"testing"

	asimonimconfig "bennypowers.dev/asimonim/config"
	"bennypowers.dev/dtls/lsp/types"
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

		// Verify the resolved file path
		if len(config.TokensFiles) > 0 {
			path, ok := config.TokensFiles[0].(string)
			assert.True(t, ok, "tokenFile should be a string")
			assert.Equal(t, tokensPath, path)
		}
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

	t.Run("preserves per-file overrides when expanding globs", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create .config directory and config file
		configDir := filepath.Join(tmpDir, ".config")
		require.NoError(t, os.MkdirAll(configDir, 0o755))

		// Create token files
		tokensDir := filepath.Join(tmpDir, "tokens")
		require.NoError(t, os.MkdirAll(tokensDir, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(tokensDir, "a.yaml"), []byte("a: {}"), 0o644))
		require.NoError(t, os.WriteFile(filepath.Join(tokensDir, "b.yaml"), []byte("b: {}"), 0o644))

		// Config with per-file override on the glob pattern
		configContent := `prefix: global
files:
  - path: tokens/*.yaml
    prefix: custom-prefix
    groupMarkers:
      - _
`
		require.NoError(t, os.WriteFile(filepath.Join(configDir, "design-tokens.yaml"), []byte(configContent), 0o644))

		config, err := ReadAsimonimConfig(tmpDir)
		require.NoError(t, err)
		require.NotNil(t, config)

		assert.Equal(t, "global", config.Prefix)
		assert.Len(t, config.TokensFiles, 2)

		// Both files should have the per-file overrides
		for _, tf := range config.TokensFiles {
			spec, ok := tf.(types.TokenFileSpec)
			require.True(t, ok, "expected TokenFileSpec, got %T", tf)
			assert.Equal(t, "custom-prefix", spec.Prefix)
			assert.Equal(t, []string{"_"}, spec.GroupMarkers)
		}
	})
}

func TestAsimonimConfigToServerConfigWithPaths(t *testing.T) {
	t.Run("returns nil for nil config", func(t *testing.T) {
		result := asimonimConfigToServerConfigWithPaths(nil, []string{"/path/to/file.json"})
		assert.Nil(t, result)
	})

	t.Run("handles paths without matching FileSpec", func(t *testing.T) {
		cfg := &asimonimconfig.Config{
			Prefix: "test",
			Files: []asimonimconfig.FileSpec{
				{Path: "other/*.json", Prefix: "other"},
			},
		}
		result := asimonimConfigToServerConfigWithPaths(cfg, []string{"/path/to/unmatched.json"})
		require.NotNil(t, result)
		assert.Len(t, result.TokensFiles, 1)
		// Should be a string, not TokenFileSpec, since it doesn't match
		_, isString := result.TokensFiles[0].(string)
		assert.True(t, isString)
	})
}

func TestAsimonimConfigToServerConfig(t *testing.T) {
	t.Run("returns nil for nil config", func(t *testing.T) {
		result := AsimonimConfigToServerConfig(nil)
		assert.Nil(t, result)
	})

	t.Run("converts config with simple file paths", func(t *testing.T) {
		cfg := &asimonimconfig.Config{
			Prefix:       "test",
			GroupMarkers: []string{"_"},
			Files: []asimonimconfig.FileSpec{
				{Path: "tokens.json"},
				{Path: "colors.yaml"},
			},
		}

		result := AsimonimConfigToServerConfig(cfg)
		require.NotNil(t, result)

		assert.Equal(t, "test", result.Prefix)
		assert.Equal(t, []string{"_"}, result.GroupMarkers)
		assert.Len(t, result.TokensFiles, 2)

		// Simple paths should be strings
		assert.Equal(t, "tokens.json", result.TokensFiles[0])
		assert.Equal(t, "colors.yaml", result.TokensFiles[1])
	})

	t.Run("converts config with per-file overrides", func(t *testing.T) {
		cfg := &asimonimconfig.Config{
			Prefix: "global",
			Files: []asimonimconfig.FileSpec{
				{Path: "simple.json"},
				{Path: "custom.json", Prefix: "custom", GroupMarkers: []string{"_", "DEFAULT"}},
			},
		}

		result := AsimonimConfigToServerConfig(cfg)
		require.NotNil(t, result)

		assert.Len(t, result.TokensFiles, 2)

		// First should be simple string
		assert.Equal(t, "simple.json", result.TokensFiles[0])

		// Second should be TokenFileSpec
		spec, ok := result.TokensFiles[1].(types.TokenFileSpec)
		require.True(t, ok)
		assert.Equal(t, "custom.json", spec.Path)
		assert.Equal(t, "custom", spec.Prefix)
		assert.Equal(t, []string{"_", "DEFAULT"}, spec.GroupMarkers)
	})
}

func TestFindMatchingFileSpec(t *testing.T) {
	files := []asimonimconfig.FileSpec{
		{Path: "exact.json"},
		{Path: "tokens/*.json", Prefix: "glob-prefix"},
		{Path: "colors/**/*.yaml", GroupMarkers: []string{"_"}},
	}

	t.Run("finds exact match", func(t *testing.T) {
		result := findMatchingFileSpec(files, "exact.json")
		require.NotNil(t, result)
		assert.Equal(t, "exact.json", result.Path)
	})

	t.Run("finds glob match", func(t *testing.T) {
		result := findMatchingFileSpec(files, "tokens/color.json")
		require.NotNil(t, result)
		assert.Equal(t, "tokens/*.json", result.Path)
		assert.Equal(t, "glob-prefix", result.Prefix)
	})

	t.Run("finds doublestar match", func(t *testing.T) {
		result := findMatchingFileSpec(files, "colors/brand/primary.yaml")
		require.NotNil(t, result)
		assert.Equal(t, "colors/**/*.yaml", result.Path)
		assert.Equal(t, []string{"_"}, result.GroupMarkers)
	})

	t.Run("returns nil for no match", func(t *testing.T) {
		result := findMatchingFileSpec(files, "unknown/path.json")
		assert.Nil(t, result)
	})

	t.Run("finds match for absolute path against relative pattern", func(t *testing.T) {
		result := findMatchingFileSpec(files, "/tmp/project/tokens/color.json")
		require.NotNil(t, result)
		assert.Equal(t, "tokens/*.json", result.Path)
		assert.Equal(t, "glob-prefix", result.Prefix)
	})

	t.Run("finds match for deeply nested absolute path", func(t *testing.T) {
		result := findMatchingFileSpec(files, "/home/user/project/colors/brand/primary.yaml")
		require.NotNil(t, result)
		assert.Equal(t, "colors/**/*.yaml", result.Path)
	})
}

func TestSplitPath(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected []string
	}{
		{
			name:     "simple path",
			path:     "tokens/color.json",
			expected: []string{"tokens", "color.json"},
		},
		{
			name:     "absolute path",
			path:     "/home/user/project/file.json",
			expected: []string{"home", "user", "project", "file.json"},
		},
		{
			name:     "single file",
			path:     "file.json",
			expected: []string{"file.json"},
		},
		{
			name:     "empty path",
			path:     "",
			expected: nil,
		},
		{
			name:     "root path",
			path:     "/",
			expected: nil,
		},
		{
			name:     "current dir",
			path:     ".",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := splitPath(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}
