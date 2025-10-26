package lsp

import (
	"bennypowers.dev/dtls/lsp/types"
	"encoding/json"
	"testing"

	"bennypowers.dev/dtls/internal/documents"
	"bennypowers.dev/dtls/internal/tokens"
	"bennypowers.dev/dtls/lsp/methods/textDocument/diagnostic"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// TestCustomHandler_DiagnosticMethod tests the custom handler for textDocument/diagnostic
func TestCustomHandler_DiagnosticMethod(t *testing.T) {
	// Create server
	server := &Server{
		documents:   documents.NewManager(),
		tokens:      tokens.NewManager(),
		config:      types.ServerConfig{},
		loadedFiles: make(map[string]*TokenFileOptions),
	}

	// Create custom handler
	handler := &CustomHandler{
		Handler: protocol.Handler{},
		server:  server,
	}

	t.Run("textDocument/diagnostic with valid params", func(t *testing.T) {
		params := diagnostic.DocumentDiagnosticParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: "file:///test.css"},
		}
		paramsJSON, err := json.Marshal(params)
		require.NoError(t, err)

		ctx := &glsp.Context{
			Method: "textDocument/diagnostic",
			Params: paramsJSON,
		}

		result, validMethod, validParams, err := handler.Handle(ctx)
		assert.True(t, validMethod, "Should recognize textDocument/diagnostic as valid method")
		assert.True(t, validParams, "Should parse params successfully")
		assert.NoError(t, err)
		assert.NotNil(t, result)
	})

	t.Run("textDocument/diagnostic with invalid JSON", func(t *testing.T) {
		// Actually malformed JSON
		invalidJSON := []byte(`{invalid json`)

		ctx := &glsp.Context{
			Method: "textDocument/diagnostic",
			Params: invalidJSON,
		}

		_, validMethod, validParams, err := handler.Handle(ctx)
		assert.True(t, validMethod, "Should recognize method even with invalid JSON")
		assert.False(t, validParams, "Should fail to parse malformed JSON")
		assert.Error(t, err)
	})

	t.Run("standard LSP methods not intercepted by CustomHandler", func(t *testing.T) {
		// Test that CustomHandler doesn't intercept standard LSP methods
		// (i.e., they fall through to the base handler, not handled by our custom code)

		// Send a hover request with valid params
		params := protocol.HoverParams{
			TextDocumentPositionParams: protocol.TextDocumentPositionParams{
				TextDocument: protocol.TextDocumentIdentifier{URI: "file:///test.css"},
				Position:     protocol.Position{Line: 0, Character: 0},
			},
		}
		paramsJSON, err := json.Marshal(params)
		require.NoError(t, err)

		ctx := &glsp.Context{
			Method: "textDocument/hover",
			Params: paramsJSON,
		}

		// Call the handler - it should fall through to protocol.Handler
		_, validMethod, _, _ := handler.Handle(ctx)

		// The method should be recognized by the base handler
		// If CustomHandler tried to intercept it, we'd get different behavior
		assert.True(t, validMethod, "Should pass through to base handler and recognize the method")
	})

	// NOTE: semanticTokens/delta test removed
	// Delta support was disabled because the implementation lacks proper result caching
	// and would corrupt client state. See custom_handler.go for details.
}
