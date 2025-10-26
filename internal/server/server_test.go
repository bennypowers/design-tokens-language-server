package server_test

import (
	"testing"

	"bennypowers.dev/dtls/internal/server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// TestServerInitialization tests that the LSP server can be initialized
// and returns the correct capabilities
func TestServerInitialization(t *testing.T) {
	// Create a new server
	srv := server.New()
	require.NotNil(t, srv, "Server should be created")

	// Create initialize params
	params := protocol.InitializeParams{
		ProcessID: nil,
		RootURI:   strPtr("file:///test"),
		Capabilities: protocol.ClientCapabilities{
			TextDocument: &protocol.TextDocumentClientCapabilities{
				Hover: &protocol.HoverClientCapabilities{
					ContentFormat: []protocol.MarkupKind{
						protocol.MarkupKindMarkdown,
						protocol.MarkupKindPlainText,
					},
				},
			},
		},
	}

	// Initialize the server
	result, err := srv.Initialize(&params)
	require.NoError(t, err, "Initialization should not error")
	require.NotNil(t, result, "Initialization result should not be nil")

	// Verify capabilities
	caps := result.Capabilities

	// Text document sync should be incremental
	assert.NotNil(t, caps.TextDocumentSync)
	syncOptions, ok := caps.TextDocumentSync.(protocol.TextDocumentSyncOptions)
	if ok && syncOptions.Change != nil {
		assert.Equal(t, protocol.TextDocumentSyncKindIncremental, *syncOptions.Change)
	}

	// Hover support
	assert.NotNil(t, caps.HoverProvider, "Should support hover")

	// Completion support with resolve
	assert.NotNil(t, caps.CompletionProvider, "Should support completion")
	if caps.CompletionProvider != nil {
		if caps.CompletionProvider.ResolveProvider != nil {
			assert.True(t, *caps.CompletionProvider.ResolveProvider, "Should support completion resolve")
		}
	}

	// Definition support
	assert.NotNil(t, caps.DefinitionProvider, "Should support go to definition")

	// References support
	assert.NotNil(t, caps.ReferencesProvider, "Should support find references")

	// Code action support with resolve
	assert.NotNil(t, caps.CodeActionProvider, "Should support code actions")

	// Document color support
	assert.NotNil(t, caps.ColorProvider, "Should support document color")

	// Semantic tokens support
	assert.NotNil(t, caps.SemanticTokensProvider, "Should support semantic tokens")

	// Note: DiagnosticProvider is LSP 3.17, we're using 3.16
	// Diagnostics will be implemented via textDocument/diagnostic request
}

// TestServerInitialized tests that the server properly handles the initialized notification
func TestServerInitialized(t *testing.T) {
	srv := server.New()
	require.NotNil(t, srv)

	// Initialize first
	params := protocol.InitializeParams{
		ProcessID: nil,
		RootURI:   strPtr("file:///test"),
	}
	_, err := srv.Initialize(&params)
	require.NoError(t, err)

	// Send initialized notification
	err = srv.Initialized(&protocol.InitializedParams{})
	assert.NoError(t, err, "Initialized notification should not error")
}

func strPtr(s string) *string {
	return &s
}
