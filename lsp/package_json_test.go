package lsp

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadPackageJsonConfig(t *testing.T) {
	t.Run("reads designTokensLanguageServer config", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create package.json with config
		packageJSON := map[string]any{
			"name": "test-project",
			"designTokensLanguageServer": map[string]any{
				"prefix": "rh",
				"tokensFiles": []any{
					"npm:@rhds/tokens/json/rhds.tokens.json",
				},
				"groupMarkers": []any{"_", "@", "GROUP"},
			},
		}

		data, err := json.Marshal(packageJSON)
		require.NoError(t, err)

		packageJSONPath := filepath.Join(tmpDir, "package.json")
		err = os.WriteFile(packageJSONPath, data, 0o644)
		require.NoError(t, err)

		// Read config
		config, err := ReadPackageJsonConfig(tmpDir)
		require.NoError(t, err)
		require.NotNil(t, config)

		// Verify fields
		assert.Equal(t, "rh", config.Prefix)
		assert.Len(t, config.TokensFiles, 1)
		assert.Equal(t, "npm:@rhds/tokens/json/rhds.tokens.json", config.TokensFiles[0])
		assert.Equal(t, []string{"_", "@", "GROUP"}, config.GroupMarkers)
	})

	t.Run("returns nil when package.json doesn't exist", func(t *testing.T) {
		tmpDir := t.TempDir()

		config, err := ReadPackageJsonConfig(tmpDir)
		require.NoError(t, err)
		assert.Nil(t, config)
	})

	t.Run("returns nil when designTokensLanguageServer field is missing", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create package.json without config
		packageJSON := map[string]any{
			"name": "test-project",
		}

		data, err := json.Marshal(packageJSON)
		require.NoError(t, err)

		packageJSONPath := filepath.Join(tmpDir, "package.json")
		err = os.WriteFile(packageJSONPath, data, 0o644)
		require.NoError(t, err)

		config, err := ReadPackageJsonConfig(tmpDir)
		require.NoError(t, err)
		assert.Nil(t, config)
	})

	t.Run("handles tokensFiles as single string", func(t *testing.T) {
		tmpDir := t.TempDir()

		packageJSON := map[string]any{
			"name": "test-project",
			"designTokensLanguageServer": map[string]any{
				"tokensFiles": "tokens/design-tokens.json",
			},
		}

		data, err := json.Marshal(packageJSON)
		require.NoError(t, err)

		packageJSONPath := filepath.Join(tmpDir, "package.json")
		err = os.WriteFile(packageJSONPath, data, 0o644)
		require.NoError(t, err)

		config, err := ReadPackageJsonConfig(tmpDir)
		require.NoError(t, err)
		require.NotNil(t, config)

		assert.Len(t, config.TokensFiles, 1)
		assert.Equal(t, "tokens/design-tokens.json", config.TokensFiles[0])
	})

	t.Run("handles JSONC comments", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create package.json with comments (JSONC)
		packageJSONContent := `{
			"name": "test-project",
			// This is a comment
			"designTokensLanguageServer": {
				"prefix": "ds", // inline comment
				"tokensFiles": [
					"tokens.json"
				]
			}
		}`

		packageJSONPath := filepath.Join(tmpDir, "package.json")
		err := os.WriteFile(packageJSONPath, []byte(packageJSONContent), 0o644)
		require.NoError(t, err)

		config, err := ReadPackageJsonConfig(tmpDir)
		require.NoError(t, err)
		require.NotNil(t, config)

		assert.Equal(t, "ds", config.Prefix)
	})
}
