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

	t.Run("returns nil when rootPath is empty", func(t *testing.T) {
		config, err := ReadPackageJsonConfig("")
		require.NoError(t, err)
		assert.Nil(t, config)
	})

	t.Run("expands glob patterns in tokensFiles", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create token files
		tokensDir := filepath.Join(tmpDir, "tokens")
		require.NoError(t, os.MkdirAll(tokensDir, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(tokensDir, "color.json"), []byte(`{}`), 0o644))
		require.NoError(t, os.WriteFile(filepath.Join(tokensDir, "spacing.json"), []byte(`{}`), 0o644))

		packageJSON := map[string]any{
			"name": "test-project",
			"designTokensLanguageServer": map[string]any{
				"tokensFiles": []any{"tokens/*.json"},
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

		assert.Len(t, config.TokensFiles, 2, "glob should expand to 2 files")
	})

	t.Run("handles brace expansion patterns", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create token files
		tokensDir := filepath.Join(tmpDir, "tokens")
		require.NoError(t, os.MkdirAll(tokensDir, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(tokensDir, "color.yaml"), []byte(`color: {}`), 0o644))
		require.NoError(t, os.WriteFile(filepath.Join(tokensDir, "spacing.yml"), []byte(`spacing: {}`), 0o644))

		packageJSON := map[string]any{
			"name": "test-project",
			"designTokensLanguageServer": map[string]any{
				"tokensFiles": []any{"tokens/*.{yaml,yml}"},
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

		assert.Len(t, config.TokensFiles, 2, "brace expansion should match 2 files")
	})

	t.Run("handles object form for tokensFiles", func(t *testing.T) {
		tmpDir := t.TempDir()

		packageJSON := map[string]any{
			"name": "test-project",
			"designTokensLanguageServer": map[string]any{
				"tokensFiles": []any{
					map[string]any{
						"path":   "tokens.json",
						"prefix": "custom",
					},
				},
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
	})

	t.Run("returns error for invalid designTokensLanguageServer type", func(t *testing.T) {
		tmpDir := t.TempDir()

		packageJSON := map[string]any{
			"name":                        "test-project",
			"designTokensLanguageServer": "not an object",
		}

		data, err := json.Marshal(packageJSON)
		require.NoError(t, err)

		packageJSONPath := filepath.Join(tmpDir, "package.json")
		err = os.WriteFile(packageJSONPath, data, 0o644)
		require.NoError(t, err)

		_, err = ReadPackageJsonConfig(tmpDir)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "must be an object")
	})
}

func TestContainsGlobChars(t *testing.T) {
	tests := []struct {
		pattern  string
		expected bool
	}{
		{"tokens.json", false},
		{"tokens/*.json", true},
		{"tokens/?.json", true},
		{"tokens/[abc].json", true},
		{"tokens/*.{yaml,yml}", true},
		{"simple/path/file.json", false},
	}

	for _, tt := range tests {
		t.Run(tt.pattern, func(t *testing.T) {
			result := containsGlobChars(tt.pattern)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExpandGlobPattern(t *testing.T) {
	t.Run("expands simple glob", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create files
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "a.json"), []byte(`{}`), 0o644))
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "b.json"), []byte(`{}`), 0o644))

		matches, err := expandGlobPattern("*.json", tmpDir)
		require.NoError(t, err)
		assert.Len(t, matches, 2)
	})

	t.Run("expands absolute pattern", func(t *testing.T) {
		tmpDir := t.TempDir()

		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "test.json"), []byte(`{}`), 0o644))

		pattern := filepath.Join(tmpDir, "*.json")
		matches, err := expandGlobPattern(pattern, tmpDir)
		require.NoError(t, err)
		assert.Len(t, matches, 1)
	})

	t.Run("handles leading ./", func(t *testing.T) {
		tmpDir := t.TempDir()

		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "test.json"), []byte(`{}`), 0o644))

		matches, err := expandGlobPattern("./test.json", tmpDir)
		require.NoError(t, err)
		assert.Len(t, matches, 1)
	})

	t.Run("returns empty slice for no matches", func(t *testing.T) {
		tmpDir := t.TempDir()

		matches, err := expandGlobPattern("*.nonexistent", tmpDir)
		require.NoError(t, err)
		assert.Len(t, matches, 0)
	})
}

func TestExpandTokensFileGlobs(t *testing.T) {
	t.Run("expands glob patterns to files", func(t *testing.T) {
		tmpDir := t.TempDir()

		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "a.json"), []byte(`{}`), 0o644))
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "b.json"), []byte(`{}`), 0o644))

		tokensFiles := []any{"*.json"}
		result := expandTokensFileGlobs(tokensFiles, tmpDir)
		assert.Len(t, result, 2)
	})

	t.Run("preserves non-glob paths", func(t *testing.T) {
		tmpDir := t.TempDir()

		tokensFiles := []any{"tokens.json", "npm:@rhds/tokens/file.json"}
		result := expandTokensFileGlobs(tokensFiles, tmpDir)
		assert.Len(t, result, 2)
		assert.Equal(t, "tokens.json", result[0])
		assert.Equal(t, "npm:@rhds/tokens/file.json", result[1])
	})

	t.Run("preserves object items as-is", func(t *testing.T) {
		tmpDir := t.TempDir()

		objItem := map[string]any{"path": "tokens.json", "prefix": "custom"}
		tokensFiles := []any{objItem}
		result := expandTokensFileGlobs(tokensFiles, tmpDir)
		assert.Len(t, result, 1)
		assert.Equal(t, objItem, result[0])
	})

	t.Run("falls back to original pattern when glob fails", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Use an invalid glob pattern that will fail
		tokensFiles := []any{"[invalid"}
		result := expandTokensFileGlobs(tokensFiles, tmpDir)
		assert.Len(t, result, 1)
		assert.Equal(t, "[invalid", result[0])
	})
}

func TestParseTokensFilesField(t *testing.T) {
	t.Run("handles []string type", func(t *testing.T) {
		configMap := map[string]any{
			"tokensFiles": []string{"a.json", "b.json"},
		}
		result := parseTokensFilesField(configMap)
		assert.Len(t, result, 2)
	})

	t.Run("returns nil when field is missing", func(t *testing.T) {
		configMap := map[string]any{}
		result := parseTokensFilesField(configMap)
		assert.Nil(t, result)
	})

	t.Run("handles invalid type gracefully", func(t *testing.T) {
		configMap := map[string]any{
			"tokensFiles": 12345, // invalid type
		}
		result := parseTokensFilesField(configMap)
		assert.Nil(t, result)
	})
}

func TestReadPackageJsonConfig_NetworkFallback(t *testing.T) {
	t.Run("parses networkFallback and networkTimeout", func(t *testing.T) {
		tmpDir := t.TempDir()

		packageJSON := map[string]any{
			"name": "test-project",
			"designTokensLanguageServer": map[string]any{
				"tokensFiles":     []any{"tokens.json"},
				"networkFallback": true,
				"networkTimeout":  60,
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

		assert.True(t, config.NetworkFallback)
		assert.Equal(t, 60, config.NetworkTimeout)
	})

	t.Run("parses cdn provider", func(t *testing.T) {
		tmpDir := t.TempDir()

		packageJSON := map[string]any{
			"name": "test-project",
			"designTokensLanguageServer": map[string]any{
				"tokensFiles":     []any{"tokens.json"},
				"networkFallback": true,
				"cdn":             "jsdelivr",
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

		assert.Equal(t, "jsdelivr", config.CDN)
	})

	t.Run("defaults to false when not specified", func(t *testing.T) {
		tmpDir := t.TempDir()

		packageJSON := map[string]any{
			"name": "test-project",
			"designTokensLanguageServer": map[string]any{
				"tokensFiles": []any{"tokens.json"},
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

		assert.False(t, config.NetworkFallback)
		assert.Equal(t, 0, config.NetworkTimeout)
		assert.Equal(t, "", config.CDN)
	})
}

func TestParseGroupMarkersField(t *testing.T) {
	t.Run("handles []string type", func(t *testing.T) {
		configMap := map[string]any{
			"groupMarkers": []string{"_", "DEFAULT"},
		}
		result := parseGroupMarkersField(configMap)
		assert.Equal(t, []string{"_", "DEFAULT"}, result)
	})

	t.Run("returns nil when field is missing", func(t *testing.T) {
		configMap := map[string]any{}
		result := parseGroupMarkersField(configMap)
		assert.Nil(t, result)
	})

	t.Run("handles invalid type gracefully", func(t *testing.T) {
		configMap := map[string]any{
			"groupMarkers": 12345,
		}
		result := parseGroupMarkersField(configMap)
		assert.Nil(t, result)
	})
}
