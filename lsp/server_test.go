package lsp

import (
	"testing"

	"github.com/bennypowers/design-tokens-language-server/internal/documents"
	"github.com/bennypowers/design-tokens-language-server/internal/tokens"
	"github.com/stretchr/testify/assert"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// TestHandlers_WrappersSmokeTest verifies that protocol handler wrappers
// are properly connected to their business logic methods.
// This provides coverage for the 1-3 line wrapper functions without
// duplicating the comprehensive business logic tests in integration/.
func TestHandlers_WrappersSmokeTest(t *testing.T) {
	// Create minimal server for smoke tests
	server := &Server{
		documents:   documents.NewManager(),
		tokens:      tokens.NewManager(),
		config:      ServerConfig{},
		loadedFiles: make(map[string]string),
	}

	// Dummy context (nil is fine for these simple wrappers)
	var ctx *glsp.Context

	t.Run("handleHover", func(t *testing.T) {
		params := &protocol.HoverParams{
			TextDocumentPositionParams: protocol.TextDocumentPositionParams{
				TextDocument: protocol.TextDocumentIdentifier{URI: "file:///test.css"},
				Position:     protocol.Position{Line: 0, Character: 0},
			},
		}
		// Should not panic, returns nil for non-existent document
		result, err := Hover(server, ctx, params)
		assert.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("handleCompletion", func(t *testing.T) {
		params := &protocol.CompletionParams{
			TextDocumentPositionParams: protocol.TextDocumentPositionParams{
				TextDocument: protocol.TextDocumentIdentifier{URI: "file:///test.css"},
				Position:     protocol.Position{Line: 0, Character: 0},
			},
		}
		result, err := Completion(server, ctx, params)
		assert.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("handleCompletionResolve", func(t *testing.T) {
		item := &protocol.CompletionItem{Label: "test"}
		result, err := CompletionResolve(server, ctx, item)
		assert.NoError(t, err)
		assert.Equal(t, item, result) // Returns same item if no data
	})

	t.Run("handleDefinition", func(t *testing.T) {
		params := &protocol.DefinitionParams{
			TextDocumentPositionParams: protocol.TextDocumentPositionParams{
				TextDocument: protocol.TextDocumentIdentifier{URI: "file:///test.css"},
				Position:     protocol.Position{Line: 0, Character: 0},
			},
		}
		result, err := Definition(server, ctx, params)
		assert.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("handleReferences", func(t *testing.T) {
		params := &protocol.ReferenceParams{
			TextDocumentPositionParams: protocol.TextDocumentPositionParams{
				TextDocument: protocol.TextDocumentIdentifier{URI: "file:///test.css"},
				Position:     protocol.Position{Line: 0, Character: 0},
			},
		}
		result, err := References(server, ctx, params)
		assert.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("handleCodeAction", func(t *testing.T) {
		params := &protocol.CodeActionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: "file:///test.css"},
			Range: protocol.Range{
				Start: protocol.Position{Line: 0, Character: 0},
				End:   protocol.Position{Line: 0, Character: 5},
			},
		}
		result, err := CodeAction(server, ctx, params)
		assert.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("handleCodeActionResolve", func(t *testing.T) {
		action := &protocol.CodeAction{Title: "test"}
		result, err := CodeActionResolve(server, ctx, action)
		assert.NoError(t, err)
		assert.Equal(t, action, result)
	})

	t.Run("handleDocumentColor", func(t *testing.T) {
		params := &protocol.DocumentColorParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: "file:///test.css"},
		}
		result, err := DocumentColor(server, ctx, params)
		assert.NoError(t, err)
		assert.Empty(t, result)
	})

	t.Run("handleColorPresentation", func(t *testing.T) {
		params := &protocol.ColorPresentationParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: "file:///test.css"},
			Color: protocol.Color{
				Red:   1.0,
				Green: 0.0,
				Blue:  0.0,
				Alpha: 1.0,
			},
		}
		result, err := ColorPresentation(server, ctx, params)
		assert.NoError(t, err)
		assert.NotEmpty(t, result) // Returns format options even without document
	})

	t.Run("handleDocumentDiagnostic", func(t *testing.T) {
		params := &DocumentDiagnosticParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: "file:///test.css"},
		}
		result, err := server.handleDocumentDiagnostic(ctx, params)
		assert.NoError(t, err)
		assert.NotNil(t, result)
	})

	t.Run("handleDidOpen", func(t *testing.T) {
		params := &protocol.DidOpenTextDocumentParams{
			TextDocument: protocol.TextDocumentItem{
				URI:        "file:///test.css",
				LanguageID: "css",
				Version:    1,
				Text:       "body { color: red; }",
			},
		}
		err := server.handleDidOpen(ctx, params)
		assert.NoError(t, err)
	})

	t.Run("handleDidChange", func(t *testing.T) {
		// First open a document
		server.documents.DidOpen("file:///test.css", "css", 1, "body { color: red; }")

		textChange := protocol.TextDocumentContentChangeEvent{}
		textChange.Text = "body { color: blue; }"

		params := &protocol.DidChangeTextDocumentParams{
			TextDocument: protocol.VersionedTextDocumentIdentifier{
				TextDocumentIdentifier: protocol.TextDocumentIdentifier{URI: "file:///test.css"},
				Version:                2,
			},
			ContentChanges: []any{textChange},
		}
		err := server.handleDidChange(ctx, params)
		assert.NoError(t, err)
	})

	t.Run("handleDidClose", func(t *testing.T) {
		// Ensure document exists
		server.documents.DidOpen("file:///test2.css", "css", 1, "")

		params := &protocol.DidCloseTextDocumentParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: "file:///test2.css"},
		}
		err := server.handleDidClose(ctx, params)
		assert.NoError(t, err)
	})

	t.Run("handleShutdown", func(t *testing.T) {
		err := server.handleShutdown(ctx)
		assert.NoError(t, err)
	})

	t.Run("handleSetTrace", func(t *testing.T) {
		params := &protocol.SetTraceParams{Value: "off"}
		err := server.handleSetTrace(ctx, params)
		assert.NoError(t, err)
	})
}

// TestHelperFunctions tests the utility helper functions
func TestHelperFunctions(t *testing.T) {
	t.Run("boolPtr", func(t *testing.T) {
		truePtr := boolPtr(true)
		assert.NotNil(t, truePtr)
		assert.True(t, *truePtr)

		falsePtr := boolPtr(false)
		assert.NotNil(t, falsePtr)
		assert.False(t, *falsePtr)
	})

	t.Run("strPtr", func(t *testing.T) {
		str := strPtr("test-string")
		assert.NotNil(t, str)
		assert.Equal(t, "test-string", *str)

		emptyStr := strPtr("")
		assert.NotNil(t, emptyStr)
		assert.Equal(t, "", *emptyStr)
	})
}

// TestServer_Close tests that Close() properly releases resources
func TestServer_Close(t *testing.T) {
	t.Run("Close releases CSS parser pool", func(t *testing.T) {
		server, err := NewServer()
		assert.NoError(t, err)
		assert.NotNil(t, server)

		// Close should not panic and should clean up resources
		assert.NotPanics(t, func() {
			err := server.Close()
			assert.NoError(t, err)
		})
	})

	t.Run("Close can be called multiple times", func(t *testing.T) {
		server, err := NewServer()
		assert.NoError(t, err)

		// First close
		err = server.Close()
		assert.NoError(t, err)

		// Second close should not panic or error
		err = server.Close()
		assert.NoError(t, err)
	})

	t.Run("Close works with nil server fields", func(t *testing.T) {
		// Minimal server with no initialization
		server := &Server{
			documents:   documents.NewManager(),
			tokens:      tokens.NewManager(),
			config:      ServerConfig{},
			loadedFiles: make(map[string]string),
		}

		// Should not panic
		assert.NotPanics(t, func() {
			err := server.Close()
			assert.NoError(t, err)
		})
	})
}
