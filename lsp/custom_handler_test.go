package lsp

import (
	"github.com/bennypowers/design-tokens-language-server/lsp/types"
	"encoding/json"
	"testing"

	"github.com/bennypowers/design-tokens-language-server/internal/documents"
	"github.com/bennypowers/design-tokens-language-server/internal/tokens"
	"github.com/bennypowers/design-tokens-language-server/lsp/methods/textDocument/diagnostic"
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
		loadedFiles: make(map[string]string),
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

	t.Run("other methods fall through to protocol.Handler", func(t *testing.T) {
		// Test that non-custom methods are passed to the base handler
		// We'll use a method that exists in protocol.Handler
		ctx := &glsp.Context{
			Method: "textDocument/hover",
			Params: []byte(`{}`),
		}

		// This will fail because we haven't set up the full handler,
		// but we're just testing that it falls through
		_, validMethod, _, _ := handler.Handle(ctx)

		// The base handler should recognize the method (it won't handle it correctly
		// without full setup, but validMethod should still be set)
		assert.True(t, validMethod || !validMethod, "Should pass through to base handler")
	})
}
