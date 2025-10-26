package textDocument

import (
	"github.com/bennypowers/design-tokens-language-server/lsp/types"
	"testing"

	"github.com/bennypowers/design-tokens-language-server/internal/documents"
	"github.com/bennypowers/design-tokens-language-server/internal/tokens"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// mockServerContext implements types.ServerContext for testing
type mockServerContext struct {
	docs              *documents.Manager
	tokens            *tokens.Manager
	context           *glsp.Context
	diagnosticsPublished map[string]bool
}

func (m *mockServerContext) Document(uri string) *documents.Document {
	return m.docs.Get(uri)
}

func (m *mockServerContext) DocumentManager() *documents.Manager {
	return m.docs
}

func (m *mockServerContext) AllDocuments() []*documents.Document {
	return m.docs.GetAll()
}

func (m *mockServerContext) Token(name string) *tokens.Token {
	return m.tokens.Get(name)
}

func (m *mockServerContext) TokenManager() *tokens.Manager {
	return m.tokens
}

func (m *mockServerContext) TokenCount() int {
	return m.tokens.Count()
}

func (m *mockServerContext) RootURI() string {
	return "file:///workspace"
}

func (m *mockServerContext) RootPath() string {
	return "/workspace"
}

func (m *mockServerContext) SetRootURI(uri string) {}

func (m *mockServerContext) SetRootPath(path string) {}

func (m *mockServerContext) LoadTokensFromConfig() error {
	return nil
}

func (m *mockServerContext) RegisterFileWatchers(ctx *glsp.Context) error {
	return nil
}

func (m *mockServerContext) GLSPContext() *glsp.Context {
	return m.context
}

func (m *mockServerContext) SetGLSPContext(ctx *glsp.Context) {
	m.context = ctx
}



func (m *mockServerContext) GetConfig() types.ServerConfig {
	return types.DefaultConfig()
}

func (m *mockServerContext) SetConfig(config types.ServerConfig) {}

func (m *mockServerContext) IsTokenFile(path string) bool {
	return false
}

func (m *mockServerContext) PublishDiagnostics(context *glsp.Context, uri string) error {
	if m.diagnosticsPublished == nil {
		m.diagnosticsPublished = make(map[string]bool)
	}
	m.diagnosticsPublished[uri] = true
	return nil
}

func newMockServerContext() *mockServerContext {
	return &mockServerContext{
		docs:   documents.NewManager(),
		tokens: tokens.NewManager(),
	}
}

func TestDidOpen(t *testing.T) {
	t.Run("opens document successfully", func(t *testing.T) {
		ctx := newMockServerContext()
		glspCtx := &glsp.Context{}

		params := &protocol.DidOpenTextDocumentParams{
			TextDocument: protocol.TextDocumentItem{
				URI:        "file:///test.css",
				LanguageID: "css",
				Version:    1,
				Text:       "body { color: red; }",
			},
		}

		err := DidOpen(ctx, glspCtx, params)
		require.NoError(t, err)

		// Verify document was opened
		doc := ctx.Document("file:///test.css")
		require.NotNil(t, doc)
		assert.Equal(t, "file:///test.css", doc.URI())
		assert.Equal(t, "css", doc.LanguageID())
		assert.Equal(t, 1, doc.Version())
		assert.Equal(t, "body { color: red; }", doc.Content())
	})

	t.Run("publishes diagnostics after opening", func(t *testing.T) {
		ctx := newMockServerContext()
		glspCtx := &glsp.Context{}
		ctx.SetGLSPContext(glspCtx)

		params := &protocol.DidOpenTextDocumentParams{
			TextDocument: protocol.TextDocumentItem{
				URI:        "file:///test.css",
				LanguageID: "css",
				Version:    1,
				Text:       "body { color: red; }",
			},
		}

		err := DidOpen(ctx, glspCtx, params)
		require.NoError(t, err)

		// Verify diagnostics were published
		assert.True(t, ctx.diagnosticsPublished["file:///test.css"])
	})

	t.Run("handles JSON document", func(t *testing.T) {
		ctx := newMockServerContext()
		glspCtx := &glsp.Context{}

		params := &protocol.DidOpenTextDocumentParams{
			TextDocument: protocol.TextDocumentItem{
				URI:        "file:///tokens.json",
				LanguageID: "json",
				Version:    1,
				Text:       `{"color": {"$type": "color", "$value": "#ff0000"}}`,
			},
		}

		err := DidOpen(ctx, glspCtx, params)
		require.NoError(t, err)

		doc := ctx.Document("file:///tokens.json")
		require.NotNil(t, doc)
		assert.Equal(t, "json", doc.LanguageID())
	})

	t.Run("handles YAML document", func(t *testing.T) {
		ctx := newMockServerContext()
		glspCtx := &glsp.Context{}

		params := &protocol.DidOpenTextDocumentParams{
			TextDocument: protocol.TextDocumentItem{
				URI:        "file:///tokens.yaml",
				LanguageID: "yaml",
				Version:    1,
				Text:       "color:\n  $type: color\n  $value: '#ff0000'",
			},
		}

		err := DidOpen(ctx, glspCtx, params)
		require.NoError(t, err)

		doc := ctx.Document("file:///tokens.yaml")
		require.NotNil(t, doc)
		assert.Equal(t, "yaml", doc.LanguageID())
	})
}

func TestDidChange(t *testing.T) {
	t.Run("updates document content", func(t *testing.T) {
		ctx := newMockServerContext()
		glspCtx := &glsp.Context{}

		// First open a document
		ctx.DocumentManager().DidOpen("file:///test.css", "css", 1, "body { color: red; }")

		// Change the document
		textChange := protocol.TextDocumentContentChangeEvent{}
		textChange.Text = "body { color: blue; }"

		params := &protocol.DidChangeTextDocumentParams{
			TextDocument: protocol.VersionedTextDocumentIdentifier{
				TextDocumentIdentifier: protocol.TextDocumentIdentifier{URI: "file:///test.css"},
				Version:                2,
			},
			ContentChanges: []interface{}{textChange},
		}

		err := DidChange(ctx, glspCtx, params)
		require.NoError(t, err)

		// Verify document was updated
		doc := ctx.Document("file:///test.css")
		require.NotNil(t, doc)
		assert.Equal(t, 2, doc.Version())
		assert.Equal(t, "body { color: blue; }", doc.Content())
	})

	t.Run("publishes diagnostics after change", func(t *testing.T) {
		ctx := newMockServerContext()
		glspCtx := &glsp.Context{}
		ctx.SetGLSPContext(glspCtx)

		// First open a document
		ctx.DocumentManager().DidOpen("file:///test.css", "css", 1, "body { color: red; }")

		textChange := protocol.TextDocumentContentChangeEvent{}
		textChange.Text = "body { color: blue; }"

		params := &protocol.DidChangeTextDocumentParams{
			TextDocument: protocol.VersionedTextDocumentIdentifier{
				TextDocumentIdentifier: protocol.TextDocumentIdentifier{URI: "file:///test.css"},
				Version:                2,
			},
			ContentChanges: []interface{}{textChange},
		}

		err := DidChange(ctx, glspCtx, params)
		require.NoError(t, err)

		// Verify diagnostics were published
		assert.True(t, ctx.diagnosticsPublished["file:///test.css"])
	})

	t.Run("handles incremental changes", func(t *testing.T) {
		ctx := newMockServerContext()
		glspCtx := &glsp.Context{}

		// First open a document
		ctx.DocumentManager().DidOpen("file:///test.css", "css", 1, "body { color: red; }")

		// Incremental change with range
		textChange := protocol.TextDocumentContentChangeEvent{}
		textChange.Range = &protocol.Range{
			Start: protocol.Position{Line: 0, Character: 7},
			End:   protocol.Position{Line: 0, Character: 18},
		}
		textChange.Text = "background: blue"

		params := &protocol.DidChangeTextDocumentParams{
			TextDocument: protocol.VersionedTextDocumentIdentifier{
				TextDocumentIdentifier: protocol.TextDocumentIdentifier{URI: "file:///test.css"},
				Version:                2,
			},
			ContentChanges: []interface{}{textChange},
		}

		err := DidChange(ctx, glspCtx, params)
		require.NoError(t, err)

		// Verify version was updated
		doc := ctx.Document("file:///test.css")
		require.NotNil(t, doc)
		assert.Equal(t, 2, doc.Version())
	})

	t.Run("handles multiple changes", func(t *testing.T) {
		ctx := newMockServerContext()
		glspCtx := &glsp.Context{}

		// First open a document
		ctx.DocumentManager().DidOpen("file:///test.css", "css", 1, "body { color: red; }")

		change1 := protocol.TextDocumentContentChangeEvent{}
		change1.Text = "body { color: blue; }"

		change2 := protocol.TextDocumentContentChangeEvent{}
		change2.Text = "body { background: green; }"

		params := &protocol.DidChangeTextDocumentParams{
			TextDocument: protocol.VersionedTextDocumentIdentifier{
				TextDocumentIdentifier: protocol.TextDocumentIdentifier{URI: "file:///test.css"},
				Version:                2,
			},
			ContentChanges: []interface{}{change1, change2},
		}

		err := DidChange(ctx, glspCtx, params)
		require.NoError(t, err)

		doc := ctx.Document("file:///test.css")
		require.NotNil(t, doc)
		assert.Equal(t, 2, doc.Version())
	})

	t.Run("filters invalid change events", func(t *testing.T) {
		ctx := newMockServerContext()
		glspCtx := &glsp.Context{}

		// First open a document
		ctx.DocumentManager().DidOpen("file:///test.css", "css", 1, "body { color: red; }")

		validChange := protocol.TextDocumentContentChangeEvent{}
		validChange.Text = "body { color: blue; }"

		params := &protocol.DidChangeTextDocumentParams{
			TextDocument: protocol.VersionedTextDocumentIdentifier{
				TextDocumentIdentifier: protocol.TextDocumentIdentifier{URI: "file:///test.css"},
				Version:                2,
			},
			ContentChanges: []interface{}{
				validChange,
				"invalid change", // Should be filtered out
				42,               // Should be filtered out
			},
		}

		// Should not error, just skip invalid changes
		err := DidChange(ctx, glspCtx, params)
		require.NoError(t, err)
	})
}

func TestDidClose(t *testing.T) {
	t.Run("closes document successfully", func(t *testing.T) {
		ctx := newMockServerContext()
		glspCtx := &glsp.Context{}

		// First open a document
		ctx.DocumentManager().DidOpen("file:///test.css", "css", 1, "body { color: red; }")

		params := &protocol.DidCloseTextDocumentParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: "file:///test.css"},
		}

		err := DidClose(ctx, glspCtx, params)
		require.NoError(t, err)

		// Verify document was closed
		doc := ctx.Document("file:///test.css")
		assert.Nil(t, doc)
	})

	t.Run("returns error when closing non-existent document", func(t *testing.T) {
		ctx := newMockServerContext()
		glspCtx := &glsp.Context{}

		params := &protocol.DidCloseTextDocumentParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: "file:///nonexistent.css"},
		}

		// Should return error
		err := DidClose(ctx, glspCtx, params)
		assert.Error(t, err)
	})

	t.Run("closes multiple documents independently", func(t *testing.T) {
		ctx := newMockServerContext()
		glspCtx := &glsp.Context{}

		// Open two documents
		ctx.DocumentManager().DidOpen("file:///test1.css", "css", 1, "body { color: red; }")
		ctx.DocumentManager().DidOpen("file:///test2.css", "css", 1, "div { color: blue; }")

		// Close first document
		params := &protocol.DidCloseTextDocumentParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: "file:///test1.css"},
		}

		err := DidClose(ctx, glspCtx, params)
		require.NoError(t, err)

		// First should be closed, second should remain
		assert.Nil(t, ctx.Document("file:///test1.css"))
		assert.NotNil(t, ctx.Document("file:///test2.css"))
	})
}
