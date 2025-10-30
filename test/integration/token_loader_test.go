package integration_test

import (
	"os"
	"path/filepath"
	"testing"

	"bennypowers.dev/dtls/lsp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestServer creates a test server instance for token loading tests
func newTestServer() *lsp.Server {
	s, err := lsp.NewServer()
	if err != nil {
		panic("failed to create test server: " + err.Error())
	}
	return s
}

// TestTokenLoader_DiscoveryWithPatterns tests token file discovery with glob patterns
func TestTokenLoader_DiscoveryWithPatterns(t *testing.T) {
	// Create temporary directory structure
	tmpDir := t.TempDir()

	// Create directory structure:
	// tmpDir/
	//   tokens.json (should match)
	//   src/
	//     design-tokens.json (should match)
	//     components/
	//       button.tokens.json (should match)
	//   .hidden/
	//     tokens.json (should be skipped)
	//   node_modules/
	//     tokens.json (should be skipped)
	//   dist/
	//     tokens.json (should be skipped)

	// Create tokens.json at root
	require.NoError(t, os.WriteFile(
		filepath.Join(tmpDir, "tokens.json"),
		[]byte(`{"color": {"primary": {"$value": "#0000ff", "$type": "color"}}}`),
		0o644,
	))

	// Create src directory with design-tokens.json
	srcDir := filepath.Join(tmpDir, "src")
	require.NoError(t, os.MkdirAll(srcDir, 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(srcDir, "design-tokens.json"),
		[]byte(`{"spacing": {"small": {"$value": "8px", "$type": "dimension"}}}`),
		0o644,
	))

	// Create src/components with button.tokens.json
	componentsDir := filepath.Join(srcDir, "components")
	require.NoError(t, os.MkdirAll(componentsDir, 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(componentsDir, "button.tokens.json"),
		[]byte(`{"button": {"bg": {"$value": "#ffffff", "$type": "color"}}}`),
		0o644,
	))

	// Create .hidden directory (should be skipped)
	hiddenDir := filepath.Join(tmpDir, ".hidden")
	require.NoError(t, os.MkdirAll(hiddenDir, 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(hiddenDir, "tokens.json"),
		[]byte(`{"should": {"not": {"$value": "load", "$type": "string"}}}`),
		0o644,
	))

	// Create node_modules directory (should be skipped)
	nodeModulesDir := filepath.Join(tmpDir, "node_modules")
	require.NoError(t, os.MkdirAll(nodeModulesDir, 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(nodeModulesDir, "tokens.json"),
		[]byte(`{"should": {"not": {"$value": "load", "$type": "string"}}}`),
		0o644,
	))

	// Create dist directory (should be skipped)
	distDir := filepath.Join(tmpDir, "dist")
	require.NoError(t, os.MkdirAll(distDir, 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(distDir, "tokens.json"),
		[]byte(`{"should": {"not": {"$value": "load", "$type": "string"}}}`),
		0o644,
	))

	// Create server
	s := newTestServer()

	// Load token files with default patterns
	config := lsp.TokenFileConfig{
		RootDir:  tmpDir,
		Patterns: []string{"**/tokens.json", "**/*.tokens.json", "**/design-tokens.json"},
	}

	err := s.LoadTokenFiles(config)
	require.NoError(t, err)

	// Should have loaded 3 files (tokens.json, design-tokens.json, button.tokens.json)
	// and skipped the ones in .hidden, node_modules, and dist
	tokenCount := s.TokenManager().Count()
	assert.Equal(t, 3, tokenCount, "Should load 3 tokens from 3 files")

	// Verify specific tokens were loaded
	assert.NotNil(t, s.TokenManager().Get("color-primary"), "Should have loaded color-primary from root tokens.json")
	assert.NotNil(t, s.TokenManager().Get("spacing-small"), "Should have loaded spacing-small from design-tokens.json")
	assert.NotNil(t, s.TokenManager().Get("button-bg"), "Should have loaded button-bg from button.tokens.json")

	// Verify tokens from skipped directories were NOT loaded
	assert.Nil(t, s.TokenManager().Get("should-not"), "Should not load tokens from hidden/excluded directories")
}

// TestTokenLoader_CustomPatterns tests loading with custom glob patterns
func TestTokenLoader_CustomPatterns(t *testing.T) {
	tmpDir := t.TempDir()

	// Create files with different naming patterns
	require.NoError(t, os.WriteFile(
		filepath.Join(tmpDir, "theme.json"),
		[]byte(`{"color": {"primary": {"$value": "#ff0000", "$type": "color"}}}`),
		0o644,
	))

	require.NoError(t, os.WriteFile(
		filepath.Join(tmpDir, "colors.tokens"),
		[]byte(`Not a JSON file`),
		0o644,
	))

	s := newTestServer()

	// Load with custom pattern that only matches theme.json
	config := lsp.TokenFileConfig{
		RootDir:  tmpDir,
		Patterns: []string{"theme.json"},
	}

	err := s.LoadTokenFiles(config)
	require.NoError(t, err)

	// Should have loaded 1 token from theme.json
	assert.Equal(t, 1, s.TokenManager().Count())
	assert.NotNil(t, s.TokenManager().Get("color-primary"))
}

// TestTokenLoader_EmptyDirectory tests loading from empty directory
func TestTokenLoader_EmptyDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	s := newTestServer()

	config := lsp.TokenFileConfig{
		RootDir:  tmpDir,
		Patterns: []string{"**/tokens.json"},
	}

	err := s.LoadTokenFiles(config)
	require.NoError(t, err)

	// Should have no tokens
	assert.Equal(t, 0, s.TokenManager().Count())
}

// TestTokenLoader_InvalidRootDir tests error handling for invalid root directory
func TestTokenLoader_InvalidRootDir(t *testing.T) {
	s := newTestServer()

	config := lsp.TokenFileConfig{
		RootDir:  "", // Empty root dir should error
		Patterns: []string{"**/tokens.json"},
	}

	err := s.LoadTokenFiles(config)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "root directory is required")
}

// TestTokenLoader_InvalidJSON tests error handling for invalid JSON files
func TestTokenLoader_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()

	// Create invalid JSON file
	require.NoError(t, os.WriteFile(
		filepath.Join(tmpDir, "tokens.json"),
		[]byte(`{invalid json}`),
		0o644,
	))

	// Create valid JSON file
	require.NoError(t, os.WriteFile(
		filepath.Join(tmpDir, "valid.tokens.json"),
		[]byte(`{"color": {"primary": {"$value": "#0000ff", "$type": "color"}}}`),
		0o644,
	))

	s := newTestServer()

	config := lsp.TokenFileConfig{
		RootDir:  tmpDir,
		Patterns: []string{"**/tokens.json", "**/*.tokens.json"},
	}

	// Load files - loader collects errors but continues
	err := s.LoadTokenFiles(config)

	// Should have an error about the invalid file
	if err != nil {
		assert.Contains(t, err.Error(), "tokens.json", "Error should mention the invalid file")
	}

	// Valid file should still have been loaded
	assert.NotNil(t, s.TokenManager().Get("color-primary"))
}

// TestTokenLoader_ReloadTokens tests reloading functionality
func TestTokenLoader_ReloadTokens(t *testing.T) {
	tmpDir := t.TempDir()

	// Create initial token file
	tokensPath := filepath.Join(tmpDir, "tokens.json")
	require.NoError(t, os.WriteFile(
		tokensPath,
		[]byte(`{"color": {"primary": {"$value": "#0000ff", "$type": "color"}}}`),
		0o644,
	))

	s := newTestServer()

	config := lsp.TokenFileConfig{
		RootDir:  tmpDir,
		Patterns: []string{"**/tokens.json"},
	}

	// Initial load
	err := s.LoadTokenFiles(config)
	require.NoError(t, err)
	assert.Equal(t, 1, s.TokenManager().Count())
	assert.NotNil(t, s.TokenManager().Get("color-primary"))

	// Update the token file
	require.NoError(t, os.WriteFile(
		tokensPath,
		[]byte(`{"spacing": {"small": {"$value": "8px", "$type": "dimension"}}}`),
		0o644,
	))

	// Reload tokens
	err = s.ReloadTokens(config)
	require.NoError(t, err)

	// Should have new token, old token should be gone
	assert.Equal(t, 1, s.TokenManager().Count())
	assert.Nil(t, s.TokenManager().Get("color-primary"), "Old token should be cleared")
	assert.NotNil(t, s.TokenManager().Get("spacing-small"), "New token should be loaded")
}

// TestTokenLoader_WithPrefix tests loading with CSS variable prefix
func TestTokenLoader_WithPrefix(t *testing.T) {
	tmpDir := t.TempDir()

	require.NoError(t, os.WriteFile(
		filepath.Join(tmpDir, "tokens.json"),
		[]byte(`{"color": {"primary": {"$value": "#0000ff", "$type": "color"}}}`),
		0o644,
	))

	s := newTestServer()

	config := lsp.TokenFileConfig{
		RootDir:  tmpDir,
		Patterns: []string{"**/tokens.json"},
		Prefix:   "my-prefix",
	}

	err := s.LoadTokenFiles(config)
	require.NoError(t, err)

	// Token should exist with the correct name and prefix
	token := s.TokenManager().Get("color-primary")
	require.NotNil(t, token, "Token should exist")
	assert.Equal(t, "color-primary", token.Name)
	assert.Equal(t, "my-prefix", token.Prefix)

	// CSS variable name should include the prefix
	assert.Equal(t, "--my-prefix-color-primary", token.CSSVariableName())
}

// TestTokenLoader_NestedDirectories tests deeply nested directory structures
func TestTokenLoader_NestedDirectories(t *testing.T) {
	tmpDir := t.TempDir()

	// Create deeply nested structure
	deepPath := filepath.Join(tmpDir, "a", "b", "c", "d", "e")
	require.NoError(t, os.MkdirAll(deepPath, 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(deepPath, "tokens.json"),
		[]byte(`{"color": {"deep": {"$value": "#000000", "$type": "color"}}}`),
		0o644,
	))

	s := newTestServer()

	config := lsp.TokenFileConfig{
		RootDir:  tmpDir,
		Patterns: []string{"**/tokens.json"},
	}

	err := s.LoadTokenFiles(config)
	require.NoError(t, err)

	assert.Equal(t, 1, s.TokenManager().Count())
	assert.NotNil(t, s.TokenManager().Get("color-deep"))
}

// TestTokenLoader_NonExistentDirectory tests loading from non-existent directory
func TestTokenLoader_NonExistentDirectory(t *testing.T) {
	s := newTestServer()

	config := lsp.TokenFileConfig{
		RootDir:  "/nonexistent/path/that/does/not/exist",
		Patterns: []string{"**/tokens.json"},
	}

	err := s.LoadTokenFiles(config)
	// filepath.Walk behavior varies by OS - on some systems it may succeed but find no files
	// Either way, we should have 0 tokens loaded
	if err != nil {
		assert.Contains(t, err.Error(), "failed to walk directory")
	}
	assert.Equal(t, 0, s.TokenManager().Count())
}

// TestTokenLoader_UnsupportedFileType tests error handling for unsupported file types
func TestTokenLoader_UnsupportedFileType(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a .txt file that matches the pattern
	require.NoError(t, os.WriteFile(
		filepath.Join(tmpDir, "tokens.txt"),
		[]byte(`{"color": {"primary": {"$value": "#0000ff", "$type": "color"}}}`),
		0o644,
	))

	s := newTestServer()

	// Manually add the unsupported file to trigger the error path
	// We can't use LoadTokenFiles because it filters by extension during discovery
	// Instead, we'll create a custom pattern that includes .txt files
	// Actually, LoadTokenFiles uses glob patterns so it will find *.txt files if we specify that pattern
	config := lsp.TokenFileConfig{
		RootDir:  tmpDir,
		Patterns: []string{"**/*.txt"},
	}

	err := s.LoadTokenFiles(config)
	// Should get an error about unsupported file type
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported file type")
}

// TestTokenLoader_BuildDirectory tests that build directory is skipped
func TestTokenLoader_BuildDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	// Create build directory (should be skipped)
	buildDir := filepath.Join(tmpDir, "build")
	require.NoError(t, os.MkdirAll(buildDir, 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(buildDir, "tokens.json"),
		[]byte(`{"should": {"not": {"$value": "load", "$type": "string"}}}`),
		0o644,
	))

	// Create valid token file outside build
	require.NoError(t, os.WriteFile(
		filepath.Join(tmpDir, "tokens.json"),
		[]byte(`{"color": {"primary": {"$value": "#0000ff", "$type": "color"}}}`),
		0o644,
	))

	s := newTestServer()

	config := lsp.TokenFileConfig{
		RootDir:  tmpDir,
		Patterns: []string{"**/tokens.json"},
	}

	err := s.LoadTokenFiles(config)
	require.NoError(t, err)

	// Should only have 1 token from root, not from build/
	assert.Equal(t, 1, s.TokenManager().Count())
	assert.NotNil(t, s.TokenManager().Get("color-primary"))
	assert.Nil(t, s.TokenManager().Get("should-not"))
}
