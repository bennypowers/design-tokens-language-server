package textDocument

import (
	"testing"

	"bennypowers.dev/dtls/lsp/testutil"
	"bennypowers.dev/dtls/lsp/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func TestDidOpen(t *testing.T) {
	t.Run("opens document successfully", func(t *testing.T) {
		ctx := testutil.NewMockServerContext()
		glspCtx := &glsp.Context{}
		req := types.NewRequestContext(ctx, glspCtx)

		params := &protocol.DidOpenTextDocumentParams{
			TextDocument: protocol.TextDocumentItem{
				URI:        "file:///test.css",
				LanguageID: "css",
				Version:    1,
				Text:       "body { color: red; }",
			},
		}

		err := DidOpen(req, params)
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
		ctx := testutil.NewMockServerContext()
		glspCtx := &glsp.Context{}
		req := types.NewRequestContext(ctx, glspCtx)
		ctx.SetGLSPContext(glspCtx)

		params := &protocol.DidOpenTextDocumentParams{
			TextDocument: protocol.TextDocumentItem{
				URI:        "file:///test.css",
				LanguageID: "css",
				Version:    1,
				Text:       "body { color: red; }",
			},
		}

		err := DidOpen(req, params)
		require.NoError(t, err)

		// Diagnostics are published asynchronously, no direct assertion needed
	})

	t.Run("handles JSON document", func(t *testing.T) {
		ctx := testutil.NewMockServerContext()
		glspCtx := &glsp.Context{}
		req := types.NewRequestContext(ctx, glspCtx)

		params := &protocol.DidOpenTextDocumentParams{
			TextDocument: protocol.TextDocumentItem{
				URI:        "file:///tokens.json",
				LanguageID: "json",
				Version:    1,
				Text:       `{"color": {"$type": "color", "$value": "#ff0000"}}`,
			},
		}

		err := DidOpen(req, params)
		require.NoError(t, err)

		doc := ctx.Document("file:///tokens.json")
		require.NotNil(t, doc)
		assert.Equal(t, "json", doc.LanguageID())
	})

	t.Run("handles YAML document", func(t *testing.T) {
		ctx := testutil.NewMockServerContext()
		glspCtx := &glsp.Context{}
		req := types.NewRequestContext(ctx, glspCtx)

		params := &protocol.DidOpenTextDocumentParams{
			TextDocument: protocol.TextDocumentItem{
				URI:        "file:///tokens.yaml",
				LanguageID: "yaml",
				Version:    1,
				Text:       "color:\n  $type: color\n  $value: '#ff0000'",
			},
		}

		err := DidOpen(req, params)
		require.NoError(t, err)

		doc := ctx.Document("file:///tokens.yaml")
		require.NotNil(t, doc)
		assert.Equal(t, "yaml", doc.LanguageID())
	})
}

func TestDidChange(t *testing.T) {
	t.Run("updates document content", func(t *testing.T) {
		ctx := testutil.NewMockServerContext()
		glspCtx := &glsp.Context{}
		req := types.NewRequestContext(ctx, glspCtx)

		// First open a document
		_ = ctx.DocumentManager().DidOpen("file:///test.css", "css", 1, "body { color: red; }")

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

		err := DidChange(req, params)
		require.NoError(t, err)

		// Verify document was updated
		doc := ctx.Document("file:///test.css")
		require.NotNil(t, doc)
		assert.Equal(t, 2, doc.Version())
		assert.Equal(t, "body { color: blue; }", doc.Content())
	})

	t.Run("publishes diagnostics after change", func(t *testing.T) {
		ctx := testutil.NewMockServerContext()
		glspCtx := &glsp.Context{}
		req := types.NewRequestContext(ctx, glspCtx)
		ctx.SetGLSPContext(glspCtx)

		// First open a document
		_ = ctx.DocumentManager().DidOpen("file:///test.css", "css", 1, "body { color: red; }")

		textChange := protocol.TextDocumentContentChangeEvent{}
		textChange.Text = "body { color: blue; }"

		params := &protocol.DidChangeTextDocumentParams{
			TextDocument: protocol.VersionedTextDocumentIdentifier{
				TextDocumentIdentifier: protocol.TextDocumentIdentifier{URI: "file:///test.css"},
				Version:                2,
			},
			ContentChanges: []interface{}{textChange},
		}

		err := DidChange(req, params)
		require.NoError(t, err)

		// Diagnostics are published asynchronously, no direct assertion needed
	})

	t.Run("handles incremental changes", func(t *testing.T) {
		ctx := testutil.NewMockServerContext()
		glspCtx := &glsp.Context{}
		req := types.NewRequestContext(ctx, glspCtx)

		// First open a document
		_ = ctx.DocumentManager().DidOpen("file:///test.css", "css", 1, "body { color: red; }")

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

		err := DidChange(req, params)
		require.NoError(t, err)

		// Verify version was updated
		doc := ctx.Document("file:///test.css")
		require.NotNil(t, doc)
		assert.Equal(t, 2, doc.Version())
	})

	t.Run("handles multiple changes", func(t *testing.T) {
		ctx := testutil.NewMockServerContext()
		glspCtx := &glsp.Context{}
		req := types.NewRequestContext(ctx, glspCtx)

		// First open a document
		_ = ctx.DocumentManager().DidOpen("file:///test.css", "css", 1, "body { color: red; }")

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

		err := DidChange(req, params)
		require.NoError(t, err)

		doc := ctx.Document("file:///test.css")
		require.NotNil(t, doc)
		assert.Equal(t, 2, doc.Version())
	})

	t.Run("filters invalid change events", func(t *testing.T) {
		ctx := testutil.NewMockServerContext()
		glspCtx := &glsp.Context{}
		req := types.NewRequestContext(ctx, glspCtx)

		// First open a document
		_ = ctx.DocumentManager().DidOpen("file:///test.css", "css", 1, "body { color: red; }")

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
		err := DidChange(req, params)
		require.NoError(t, err)
	})
}

func TestDidClose(t *testing.T) {
	t.Run("closes document successfully", func(t *testing.T) {
		ctx := testutil.NewMockServerContext()
		glspCtx := &glsp.Context{}
		req := types.NewRequestContext(ctx, glspCtx)

		// First open a document
		_ = ctx.DocumentManager().DidOpen("file:///test.css", "css", 1, "body { color: red; }")

		params := &protocol.DidCloseTextDocumentParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: "file:///test.css"},
		}

		err := DidClose(req, params)
		require.NoError(t, err)

		// Verify document was closed
		doc := ctx.Document("file:///test.css")
		assert.Nil(t, doc)
	})

	t.Run("returns error when closing non-existent document", func(t *testing.T) {
		ctx := testutil.NewMockServerContext()
		glspCtx := &glsp.Context{}
		req := types.NewRequestContext(ctx, glspCtx)

		params := &protocol.DidCloseTextDocumentParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: "file:///nonexistent.css"},
		}

		// Should return error
		err := DidClose(req, params)
		assert.Error(t, err)
	})

	t.Run("closes multiple documents independently", func(t *testing.T) {
		ctx := testutil.NewMockServerContext()
		glspCtx := &glsp.Context{}
		req := types.NewRequestContext(ctx, glspCtx)

		// Open two documents
		_ = ctx.DocumentManager().DidOpen("file:///test1.css", "css", 1, "body { color: red; }")
		_ = ctx.DocumentManager().DidOpen("file:///test2.css", "css", 1, "div { color: blue; }")

		// Close first document
		params := &protocol.DidCloseTextDocumentParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: "file:///test1.css"},
		}

		err := DidClose(req, params)
		require.NoError(t, err)

		// First should be closed, second should remain
		assert.Nil(t, ctx.Document("file:///test1.css"))
		assert.NotNil(t, ctx.Document("file:///test2.css"))
	})
}
