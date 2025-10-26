package integration_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bennypowers/design-tokens-language-server/lsp"
	"github.com/bennypowers/design-tokens-language-server/lsp/methods/lifecycle"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// TestServerInitialization tests the full server initialization flow
func TestServerInitialization(t *testing.T) {
	t.Run("Initialize with workspace root", func(t *testing.T) {
		server, err := lsp.NewServer()
		require.NoError(t, err)
		defer server.Close()

		// Create temp workspace
		tmpDir := t.TempDir()
		workspaceURI := "file://" + tmpDir
		workspacePath := tmpDir

		// Initialize server
		ctx := &glsp.Context{}
		initParams := &protocol.InitializeParams{
			RootURI:  &workspaceURI,
			RootPath: &workspacePath,
			ClientInfo: &struct {
				Name    string  `json:"name"`
				Version *string `json:"version,omitempty"`
			}{
				Name:    "test-client",
				Version: strPtr("1.0.0"),
			},
		}

		result, err := lifecycle.Initialize(server, ctx, initParams)
		require.NoError(t, err)
		require.NotNil(t, result)

		// Verify capabilities are returned
		resultMap, ok := result.(struct {
			Capabilities any                                      `json:"capabilities"`
			ServerInfo   *protocol.InitializeResultServerInfo `json:"serverInfo,omitempty"`
		})
		require.True(t, ok, "Result should be InitializeResult struct")
		assert.NotNil(t, resultMap.Capabilities)
		assert.NotNil(t, resultMap.ServerInfo)
		assert.Equal(t, "design-tokens-language-server", resultMap.ServerInfo.Name)
	})

	t.Run("Initialize without workspace root", func(t *testing.T) {
		server, err := lsp.NewServer()
		require.NoError(t, err)
		defer server.Close()

		ctx := &glsp.Context{}
		initParams := &protocol.InitializeParams{
			ClientInfo: &struct {
				Name    string  `json:"name"`
				Version *string `json:"version,omitempty"`
			}{
				Name: "test-client",
			},
		}

		result, err := lifecycle.Initialize(server, ctx, initParams)
		require.NoError(t, err)
		require.NotNil(t, result)
	})

	t.Run("Load tokens from workspace configuration", func(t *testing.T) {
		server, err := lsp.NewServer()
		require.NoError(t, err)
		defer server.Close()

		// Create temp workspace with token file
		tmpDir := t.TempDir()
		tokensPath := filepath.Join(tmpDir, "tokens.json")
		tokens := `{
  "color": {
    "primary": {
      "$value": "#0000ff",
      "$type": "color"
    }
  }
}`
		err = os.WriteFile(tokensPath, []byte(tokens), 0644)
		require.NoError(t, err)

		// Load token file directly (simulating what Initialized would do)
		err = server.LoadTokenFile(tokensPath, "")
		require.NoError(t, err)

		// Verify tokens were loaded
		assert.Equal(t, 1, server.TokenCount(), "Should load tokens from file")
	})
}

// TestServerShutdown tests the shutdown flow
func TestServerShutdown(t *testing.T) {
	server, err := lsp.NewServer()
	require.NoError(t, err)

	ctx := &glsp.Context{}

	// Shutdown should not error
	err = lifecycle.Shutdown(server, ctx)
	assert.NoError(t, err)

	// Multiple shutdowns should be safe
	err = lifecycle.Shutdown(server, ctx)
	assert.NoError(t, err)
}

// TestSetTrace tests the setTrace notification
func TestSetTrace(t *testing.T) {
	server, err := lsp.NewServer()
	require.NoError(t, err)
	defer server.Close()

	ctx := &glsp.Context{}

	traces := []protocol.TraceValue{
		protocol.TraceValueOff,
		protocol.TraceValueMessage,
		protocol.TraceValueVerbose,
	}
	for _, trace := range traces {
		t.Run(string(trace), func(t *testing.T) {
			err := lifecycle.SetTrace(server, ctx, &protocol.SetTraceParams{
				Value: trace,
			})
			assert.NoError(t, err)
		})
	}
}

func strPtr(s string) *string {
	return &s
}
