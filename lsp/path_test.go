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

// TestResolveNpmPath_PathTraversal tests security fixes for path traversal vulnerabilities
func TestResolveNpmPath_PathTraversal(t *testing.T) {
	tmpDir := t.TempDir()
	workspaceRoot := tmpDir

	// Create a minimal node_modules structure
	nodeModulesDir := filepath.Join(tmpDir, "node_modules")
	require.NoError(t, os.MkdirAll(nodeModulesDir, 0o755))

	// Create a legitimate package for comparison tests
	legitimatePkgDir := filepath.Join(nodeModulesDir, "legitimate-package")
	require.NoError(t, os.MkdirAll(legitimatePkgDir, 0o755))
	legitimateFile := filepath.Join(legitimatePkgDir, "tokens.json")
	require.NoError(t, os.WriteFile(legitimateFile, []byte(`{}`), 0o644))

	tests := []struct {
		name        string
		npmPath     string
		shouldError bool
		errContains string
	}{
		{
			name:        "valid unscoped package",
			npmPath:     "legitimate-package/tokens.json",
			shouldError: false,
		},
		{
			name:        "path traversal in package name - dotdot",
			npmPath:     "../../etc/passwd",
			shouldError: true,
			errContains: "path traversal not allowed",
		},
		{
			name:        "path traversal in package name - single dotdot",
			npmPath:     "../sensitive-file.txt",
			shouldError: true,
			errContains: "path traversal not allowed",
		},
		{
			name:        "path traversal with scoped package format",
			npmPath:     "@../../../etc/passwd",
			shouldError: true,
			errContains: "path traversal not allowed",
		},
		{
			name:        "path traversal in scope name",
			npmPath:     "@../../etc/passwd",
			shouldError: true,
			errContains: "path traversal not allowed",
		},
		{
			name:        "dots in legitimate package name (valid)",
			npmPath:     "my.package.name/tokens.json",
			shouldError: true, // Will fail because package doesn't exist, but NOT because of traversal
			errContains: "not found",
		},
		{
			name:        "dotdot in subpath (not package name)",
			npmPath:     "legitimate-package/../../../etc/passwd",
			shouldError: true,
			errContains: "not found", // This tests that subpath traversal is blocked by file system checks
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolveNpmPath(tt.npmPath, workspaceRoot)

			if tt.shouldError {
				require.Error(t, err, "Expected error for npm path: %s", tt.npmPath)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains,
						"Error message should contain '%s' for npm path: %s", tt.errContains, tt.npmPath)
				}
				assert.Empty(t, got, "Should return empty path on error")
				return
			}

			require.NoError(t, err, "Should not error for valid npm path: %s", tt.npmPath)
			assert.NotEmpty(t, got, "Should return non-empty path for valid npm path")
		})
	}
}

// TestResolveNpmPath_BoundaryValidation tests that npm: paths are restricted to node_modules
func TestResolveNpmPath_BoundaryValidation(t *testing.T) {
	tmpDir := t.TempDir()
	workspaceRoot := tmpDir

	// Create node_modules
	nodeModulesDir := filepath.Join(tmpDir, "node_modules")
	require.NoError(t, os.MkdirAll(nodeModulesDir, 0o755))

	tests := []struct {
		name        string
		npmPath     string
		shouldError bool
		errContains string
	}{
		{
			name:        "absolute path disguised as package name",
			npmPath:     "../../../../../../../etc/passwd",
			shouldError: true,
			errContains: "path traversal not allowed",
		},
		{
			name:        "package name with multiple dotdot sequences",
			npmPath:     "../../../../../../etc/shadow",
			shouldError: true,
			errContains: "path traversal not allowed",
		},
		{
			name:        "scoped package with dotdot in scope",
			npmPath:     "@../evil/package/file.json",
			shouldError: true,
			errContains: "path traversal not allowed",
		},
		{
			name:        "scoped package with dotdot in package name",
			npmPath:     "@scope/../evil/file.json",
			shouldError: true,
			errContains: "path traversal not allowed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolveNpmPath(tt.npmPath, workspaceRoot)

			if tt.shouldError {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				assert.Empty(t, got)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
