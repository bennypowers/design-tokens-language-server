package integration_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"bennypowers.dev/dtls/lsp"
	"bennypowers.dev/dtls/lsp/methods/lifecycle"
	"bennypowers.dev/dtls/lsp/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// TestPackageJsonConfiguration tests that the server reads configuration from package.json
func TestPackageJsonConfiguration(t *testing.T) {
	t.Run("loads tokens from npm: path in package.json", func(t *testing.T) {
		// Create a fixture project structure:
		// project/
		//   package.json (with designTokensLanguageServer config)
		//   node_modules/@test/tokens/tokens.json
		tmpDir := t.TempDir()

		// Create package.json with npm: protocol
		packageJSON := map[string]any{
			"name": "test-project",
			"designTokensLanguageServer": map[string]any{
				"prefix": "test",
				"tokensFiles": []any{
					"npm:@test/tokens/tokens.json",
				},
			},
		}
		packageJSONData, err := json.MarshalIndent(packageJSON, "", "  ")
		require.NoError(t, err)
		err = os.WriteFile(filepath.Join(tmpDir, "package.json"), packageJSONData, 0o644)
		require.NoError(t, err)

		// Create node_modules structure
		nodeModulesDir := filepath.Join(tmpDir, "node_modules", "@test", "tokens")
		err = os.MkdirAll(nodeModulesDir, 0o755)
		require.NoError(t, err)

		// Create tokens file
		tokensFile := filepath.Join(nodeModulesDir, "tokens.json")
		tokensData := `{
  "color": {
    "primary": {
      "$value": "#ff0000",
      "$type": "color"
    }
  }
}`
		err = os.WriteFile(tokensFile, []byte(tokensData), 0o644)
		require.NoError(t, err)

		// Initialize server with workspace
		server, err := lsp.NewServer()
		require.NoError(t, err)
		defer func() { _ = server.Close() }()

		workspaceURI := "file://" + tmpDir
		workspacePath := tmpDir

		ctx := &glsp.Context{}
		initParams := &protocol.InitializeParams{
			RootURI:  &workspaceURI,
			RootPath: &workspacePath,
		}

		req := types.NewRequestContext(server, ctx)
		_, err = lifecycle.Initialize(req, initParams)
		require.NoError(t, err)

		// Call Initialized which should trigger package.json loading
		err = lifecycle.Initialized(req, &protocol.InitializedParams{})
		require.NoError(t, err)

		// Verify tokens were loaded
		require.Greater(t, server.TokenCount(), 0, "Should load tokens from package.json config")

		// The token was loaded - let's verify it exists
		// Note: Tokens can be accessed by DTCG path (dots) or hyphenated name
		token := server.Token("color.primary")
		require.NotNil(t, token, "Should load color.primary token")
		assert.Equal(t, "#ff0000", token.Value)
	})

	t.Run("loads tokens from relative path in package.json", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create package.json with relative path
		packageJSON := map[string]any{
			"name": "test-project",
			"designTokensLanguageServer": map[string]any{
				"tokensFiles": []any{
					"tokens/design-tokens.json",
				},
			},
		}
		packageJSONData, err := json.MarshalIndent(packageJSON, "", "  ")
		require.NoError(t, err)
		err = os.WriteFile(filepath.Join(tmpDir, "package.json"), packageJSONData, 0o644)
		require.NoError(t, err)

		// Create tokens directory and file
		tokensDir := filepath.Join(tmpDir, "tokens")
		err = os.MkdirAll(tokensDir, 0o755)
		require.NoError(t, err)

		tokensFile := filepath.Join(tokensDir, "design-tokens.json")
		tokensData := `{
  "spacing": {
    "small": {
      "$value": "8px",
      "$type": "dimension"
    }
  }
}`
		err = os.WriteFile(tokensFile, []byte(tokensData), 0o644)
		require.NoError(t, err)

		// Initialize server
		server, err := lsp.NewServer()
		require.NoError(t, err)
		defer func() { _ = server.Close() }()

		workspaceURI := "file://" + tmpDir
		workspacePath := tmpDir

		ctx := &glsp.Context{}
		initParams := &protocol.InitializeParams{
			RootURI:  &workspaceURI,
			RootPath: &workspacePath,
		}

		req := types.NewRequestContext(server, ctx)
		_, err = lifecycle.Initialize(req, initParams)
		require.NoError(t, err)

		err = lifecycle.Initialized(req, &protocol.InitializedParams{})
		require.NoError(t, err)

		// Verify tokens were loaded
		assert.Greater(t, server.TokenCount(), 0)
		token := server.Token("spacing.small")
		require.NotNil(t, token)
		assert.Equal(t, "8px", token.Value)
	})

	t.Run("works without package.json", func(t *testing.T) {
		tmpDir := t.TempDir()

		// No package.json created

		server, err := lsp.NewServer()
		require.NoError(t, err)
		defer func() { _ = server.Close() }()

		workspaceURI := "file://" + tmpDir
		workspacePath := tmpDir

		ctx := &glsp.Context{}
		initParams := &protocol.InitializeParams{
			RootURI:  &workspaceURI,
			RootPath: &workspacePath,
		}

		req := types.NewRequestContext(server, ctx)
		_, err = lifecycle.Initialize(req, initParams)
		require.NoError(t, err)

		// Should not error when package.json doesn't exist
		err = lifecycle.Initialized(req, &protocol.InitializedParams{})
		require.NoError(t, err)

		// No tokens loaded
		assert.Equal(t, 0, server.TokenCount())
	})

	t.Run("client config overrides package.json", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create package.json
		packageJSON := map[string]any{
			"name": "test-project",
			"designTokensLanguageServer": map[string]any{
				"prefix": "pkg",
				"tokensFiles": []any{
					"tokens/from-package.json",
				},
			},
		}
		packageJSONData, err := json.MarshalIndent(packageJSON, "", "  ")
		require.NoError(t, err)
		err = os.WriteFile(filepath.Join(tmpDir, "package.json"), packageJSONData, 0o644)
		require.NoError(t, err)

		// Initialize server
		server, err := lsp.NewServer()
		require.NoError(t, err)
		defer func() { _ = server.Close() }()

		// Set config via client (simulates didChangeConfiguration)
		clientConfig := types.ServerConfig{
			Prefix: "client",
		}
		server.SetConfig(clientConfig)

		workspaceURI := "file://" + tmpDir
		workspacePath := tmpDir

		ctx := &glsp.Context{}
		initParams := &protocol.InitializeParams{
			RootURI:  &workspaceURI,
			RootPath: &workspacePath,
		}

		req := types.NewRequestContext(server, ctx)
		_, err = lifecycle.Initialize(req, initParams)
		require.NoError(t, err)

		err = lifecycle.Initialized(req, &protocol.InitializedParams{})
		require.NoError(t, err)

		// Client config should take precedence
		config := server.GetConfig()
		assert.Equal(t, "client", config.Prefix, "Client config should override package.json")
	})

	t.Run("RHDS real-world example", func(t *testing.T) {
		// Simulate Red Hat Design System package.json structure
		tmpDir := t.TempDir()

		packageJSON := map[string]any{
			"name": "red-hat-design-system",
			"designTokensLanguageServer": map[string]any{
				"prefix": "rh",
				"tokensFiles": []any{
					"npm:@rhds/tokens/json/rhds.tokens.json",
				},
				"groupMarkers": []any{"_", "@", "GROUP", "HOOLI"},
			},
		}
		packageJSONData, err := json.MarshalIndent(packageJSON, "", "  ")
		require.NoError(t, err)
		err = os.WriteFile(filepath.Join(tmpDir, "package.json"), packageJSONData, 0o644)
		require.NoError(t, err)

		// Create @rhds/tokens structure
		nodeModulesDir := filepath.Join(tmpDir, "node_modules", "@rhds", "tokens", "json")
		err = os.MkdirAll(nodeModulesDir, 0o755)
		require.NoError(t, err)

		tokensFile := filepath.Join(nodeModulesDir, "rhds.tokens.json")
		tokensData := `{
  "color": {
    "interactive": {
      "primary": {
        "default": {
          "$value": "#0066cc",
          "$type": "color",
          "$description": "Primary interactive color",
          "name": "rh-color-interactive-primary-default"
        }
      }
    }
  }
}`
		err = os.WriteFile(tokensFile, []byte(tokensData), 0o644)
		require.NoError(t, err)

		// Initialize server
		server, err := lsp.NewServer()
		require.NoError(t, err)
		defer func() { _ = server.Close() }()

		workspaceURI := "file://" + tmpDir
		workspacePath := tmpDir

		ctx := &glsp.Context{}
		initParams := &protocol.InitializeParams{
			RootURI:  &workspaceURI,
			RootPath: &workspacePath,
		}

		req := types.NewRequestContext(server, ctx)
		_, err = lifecycle.Initialize(req, initParams)
		require.NoError(t, err)

		err = lifecycle.Initialized(req, &protocol.InitializedParams{})
		require.NoError(t, err)

		// Verify tokens were loaded with correct prefix
		assert.Greater(t, server.TokenCount(), 0)

		// Token should be accessible by its DTCG path
		token := server.Token("color.interactive.primary.default")
		require.NotNil(t, token, "Should load RHDS token")
		assert.Equal(t, "#0066cc", token.Value)

		// Verify config was loaded
		config := server.GetConfig()
		assert.Equal(t, "rh", config.Prefix)
		assert.Equal(t, []string{"_", "@", "GROUP", "HOOLI"}, config.GroupMarkers)
	})
}
