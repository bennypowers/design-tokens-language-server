package lsp

import (
	"github.com/bennypowers/design-tokens-language-server/lsp/types"
	"testing"

	"github.com/bennypowers/design-tokens-language-server/internal/documents"
	"github.com/bennypowers/design-tokens-language-server/internal/tokens"
	"github.com/bennypowers/design-tokens-language-server/lsp/methods/lifecycle"
	"github.com/bennypowers/design-tokens-language-server/lsp/methods/textDocument"
	"github.com/bennypowers/design-tokens-language-server/lsp/methods/textDocument/codeAction"
	"github.com/bennypowers/design-tokens-language-server/lsp/methods/textDocument/completion"
	"github.com/bennypowers/design-tokens-language-server/lsp/methods/textDocument/definition"
	"github.com/bennypowers/design-tokens-language-server/lsp/methods/textDocument/diagnostic"
	documentcolor "github.com/bennypowers/design-tokens-language-server/lsp/methods/textDocument/documentColor"
	"github.com/bennypowers/design-tokens-language-server/lsp/methods/textDocument/hover"
	"github.com/bennypowers/design-tokens-language-server/lsp/methods/textDocument/references"
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
		config:      types.ServerConfig{},
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
		result, err := hover.Hover(server, ctx, params)
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
		result, err := completion.Completion(server, ctx, params)
		assert.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("handleCompletionResolve", func(t *testing.T) {
		item := &protocol.CompletionItem{Label: "test"}
		result, err := completion.CompletionResolve(server, ctx, item)
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
		result, err := definition.Definition(server, ctx, params)
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
		result, err := references.References(server, ctx, params)
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
		result, err := codeaction.CodeAction(server, ctx, params)
		assert.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("handleCodeActionResolve", func(t *testing.T) {
		action := &protocol.CodeAction{Title: "test"}
		result, err := codeaction.CodeActionResolve(server, ctx, action)
		assert.NoError(t, err)
		assert.Equal(t, action, result)
	})

	t.Run("handleDocumentColor", func(t *testing.T) {
		params := &protocol.DocumentColorParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: "file:///test.css"},
		}
		result, err := documentcolor.DocumentColor(server, ctx, params)
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
		result, err := documentcolor.ColorPresentation(server, ctx, params)
		assert.NoError(t, err)
		assert.NotEmpty(t, result) // Returns format options even without document
	})

	t.Run("handleDocumentDiagnostic", func(t *testing.T) {
		params := &diagnostic.DocumentDiagnosticParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: "file:///test.css"},
		}
		result, err := diagnostic.DocumentDiagnostic(server, ctx, params)
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
		err := textDocument.DidOpen(server, ctx, params)
		assert.NoError(t, err)
	})

	t.Run("didChange", func(t *testing.T) {
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
		err := textDocument.DidChange(server, ctx, params)
		assert.NoError(t, err)
	})

	t.Run("didClose", func(t *testing.T) {
		// Ensure document exists
		server.documents.DidOpen("file:///test2.css", "css", 1, "")

		params := &protocol.DidCloseTextDocumentParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: "file:///test2.css"},
		}
		err := textDocument.DidClose(server, ctx, params)
		assert.NoError(t, err)
	})

	t.Run("shutdown", func(t *testing.T) {
		err := lifecycle.Shutdown(server, ctx)
		assert.NoError(t, err)
	})

	t.Run("setTrace", func(t *testing.T) {
		params := &protocol.SetTraceParams{Value: "off"}
		err := lifecycle.SetTrace(server, ctx, params)
		assert.NoError(t, err)
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
			config:      types.ServerConfig{},
			loadedFiles: make(map[string]string),
		}

		// Should not panic
		assert.NotPanics(t, func() {
			err := server.Close()
			assert.NoError(t, err)
		})
	})
}

func TestIsTokenFile(t *testing.T) {
	tests := []struct {
		name           string
		path           string
		configFiles    []any
		rootPath       string
		expectedResult bool
	}{
		{
			name:           "Explicit token file - JSON",
			path:           "/workspace/tokens.json",
			rootPath:       "/workspace",
			configFiles:    []any{"tokens.json"},
			expectedResult: true,
		},
		{
			name:           "Explicit token file - absolute path",
			path:           "/workspace/design-system/tokens.json",
			rootPath:       "/workspace",
			configFiles:    []any{"/workspace/design-system/tokens.json"},
			expectedResult: true,
		},
		{
			name:     "Explicit token file - relative path",
			path:     "/workspace/design-system/tokens.json",
			rootPath: "/workspace",
			configFiles: []any{
				map[string]any{
					"path": "design-system/tokens.json",
				},
			},
			expectedResult: true,
		},
		{
			name:           "Non-token file",
			path:           "/workspace/package.json",
			rootPath:       "/workspace",
			configFiles:    []any{"tokens.json"},
			expectedResult: false,
		},
		{
			name:           "Auto-discover - tokens.json",
			path:           "/workspace/tokens.json",
			rootPath:       "/workspace",
			configFiles:    []any{}, // Empty = auto-discover
			expectedResult: true,
		},
		{
			name:           "Auto-discover - design-tokens.json",
			path:           "/workspace/design-tokens.json",
			rootPath:       "/workspace",
			configFiles:    []any{},
			expectedResult: true,
		},
		{
			name:           "Auto-discover - custom.tokens.json",
			path:           "/workspace/custom.tokens.json",
			rootPath:       "/workspace",
			configFiles:    []any{},
			expectedResult: true,
		},
		{
			name:           "Auto-discover - YAML",
			path:           "/workspace/tokens.yaml",
			rootPath:       "/workspace",
			configFiles:    []any{},
			expectedResult: true,
		},
		{
			name:           "Auto-discover - non-token file",
			path:           "/workspace/package.json",
			rootPath:       "/workspace",
			configFiles:    []any{},
			expectedResult: false,
		},
		{
			name:           "Non-JSON/YAML file",
			path:           "/workspace/tokens.txt",
			rootPath:       "/workspace",
			configFiles:    []any{},
			expectedResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, err := NewServer()
			if err != nil {
				t.Fatalf("Failed to create server: %v", err)
			}

			s.rootPath = tt.rootPath
			s.config.TokensFiles = tt.configFiles

			result := s.IsTokenFile(tt.path)
			if result != tt.expectedResult {
				t.Errorf("Expected %v, got %v", tt.expectedResult, result)
			}
		})
	}
}
