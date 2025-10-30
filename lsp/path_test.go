package lsp

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizePath(t *testing.T) {
	// Create a temporary directory structure for testing
	tmpDir := t.TempDir()
	workspaceRoot := tmpDir

	// Create a mock node_modules structure
	nodeModulesDir := filepath.Join(tmpDir, "node_modules")
	require.NoError(t, os.MkdirAll(nodeModulesDir, 0o755))

	// Create a mock package with tokens and package.json with exports
	mockPkgDir := filepath.Join(nodeModulesDir, "@design-system", "tokens")
	require.NoError(t, os.MkdirAll(mockPkgDir, 0o755))
	tokensFile := filepath.Join(mockPkgDir, "tokens.json")
	require.NoError(t, os.WriteFile(tokensFile, []byte(`{"color": {}}`), 0o644))

	// Create package.json with exports
	packageJSON := `{
		"name": "@design-system/tokens",
		"version": "1.0.0",
		"exports": {
			".": "./tokens.json",
			"./tokens": "./tokens.json",
			"./dist/*": "./dist/*.json"
		}
	}`
	require.NoError(t, os.WriteFile(filepath.Join(mockPkgDir, "package.json"), []byte(packageJSON), 0o644))

	// Create dist directory for pattern matching
	distDir := filepath.Join(mockPkgDir, "dist")
	require.NoError(t, os.MkdirAll(distDir, 0o755))
	colorsFile := filepath.Join(distDir, "colors.json")
	require.NoError(t, os.WriteFile(colorsFile, []byte(`{"primary": "#ff0000"}`), 0o644))

	tests := []struct {
		name          string
		path          string
		workspaceRoot string
		expected      string
		wantErr       bool
		skipOnCI      bool // Skip tests that require HOME env var
	}{
		{
			name:          "absolute path unchanged",
			path:          "/absolute/path/to/tokens.json",
			workspaceRoot: workspaceRoot,
			expected:      "/absolute/path/to/tokens.json",
		},
		{
			name:          "relative path resolved",
			path:          "./relative/tokens.json",
			workspaceRoot: workspaceRoot,
			expected:      filepath.Join(workspaceRoot, "relative", "tokens.json"),
		},
		{
			name:          "relative path without dot",
			path:          "relative/tokens.json",
			workspaceRoot: workspaceRoot,
			expected:      filepath.Join(workspaceRoot, "relative", "tokens.json"),
		},
		{
			name:          "home directory expansion",
			path:          "~/my-tokens.json",
			workspaceRoot: workspaceRoot,
			expected:      filepath.Join(os.Getenv("HOME"), "my-tokens.json"),
			skipOnCI:      os.Getenv("HOME") == "",
		},
		{
			name:          "npm: scoped package with direct path",
			path:          "npm:@design-system/tokens/tokens.json",
			workspaceRoot: workspaceRoot,
			expected:      tokensFile,
		},
		{
			name:          "npm: package main entry (uses exports)",
			path:          "npm:@design-system/tokens",
			workspaceRoot: workspaceRoot,
			expected:      tokensFile, // Resolves via exports "." field
		},
		{
			name:          "npm: package with export path",
			path:          "npm:@design-system/tokens/tokens",
			workspaceRoot: workspaceRoot,
			expected:      tokensFile, // Resolves via exports "./tokens" field
		},
		{
			name:          "npm: package with wildcard export",
			path:          "npm:@design-system/tokens/dist/colors",
			workspaceRoot: workspaceRoot,
			expected:      colorsFile, // Resolves via exports "./dist/*" pattern
		},
		{
			name:          "npm: unscoped package",
			path:          "npm:design-tokens/tokens.json",
			workspaceRoot: workspaceRoot,
			wantErr:       true, // Package doesn't exist in our test setup
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skipOnCI {
				t.Skip("Skipping test that requires HOME environment variable")
			}

			got, err := normalizePath(tt.path, tt.workspaceRoot)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestNormalizePathErrors(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name          string
		path          string
		workspaceRoot string
		errContains   string
		wantErr       bool
	}{
		{
			name:          "npm: package not found",
			path:          "npm:nonexistent-package/tokens.json",
			workspaceRoot: tmpDir,
			wantErr:       true,
			errContains:   "not found",
		},
		{
			name:          "npm: invalid package name",
			path:          "npm:",
			workspaceRoot: tmpDir,
			wantErr:       true,
			errContains:   "invalid npm package",
		},
		{
			name:          "npm: empty package name",
			path:          "npm:/tokens.json",
			workspaceRoot: tmpDir,
			wantErr:       true,
			errContains:   "invalid npm package",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := normalizePath(tt.path, tt.workspaceRoot)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				assert.Empty(t, got)
				return
			}

			require.NoError(t, err)
		})
	}
}
