package textDocument

import (
	"fmt"
	"os"

	"github.com/bennypowers/design-tokens-language-server/lsp/types"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// DidOpen handles the textDocument/didOpen notification
func DidOpen(ctx types.ServerContext, context *glsp.Context, params *protocol.DidOpenTextDocumentParams) error {
	fmt.Fprintf(os.Stderr, "[DTLS] Document opened: %s (language: %s, version: %d)\n",
		params.TextDocument.URI, params.TextDocument.LanguageID, int(params.TextDocument.Version))

	err := ctx.DocumentManager().DidOpen(params.TextDocument.URI, params.TextDocument.LanguageID,
		int(params.TextDocument.Version), params.TextDocument.Text)
	if err != nil {
		return err
	}

	// Publish diagnostics for the opened document
	if glspCtx := ctx.GLSPContext(); glspCtx != nil {
		ctx.PublishDiagnostics(glspCtx, params.TextDocument.URI)
	}

	return nil
}

// DidChange handles the textDocument/didChange notification
func DidChange(ctx types.ServerContext, context *glsp.Context, params *protocol.DidChangeTextDocumentParams) error {
	uri := params.TextDocument.URI
	version := int(params.TextDocument.Version)

	fmt.Fprintf(os.Stderr, "[DTLS] Document changed: %s (version: %d, changes: %d)\n", uri, version, len(params.ContentChanges))

	// Convert any[] to proper type, filtering out invalid entries
	changes := make([]protocol.TextDocumentContentChangeEvent, 0, len(params.ContentChanges))
	for _, change := range params.ContentChanges {
		if changeEvent, ok := change.(protocol.TextDocumentContentChangeEvent); ok {
			changes = append(changes, changeEvent)
		}
	}

	err := ctx.DocumentManager().DidChange(uri, version, changes)
	if err != nil {
		return err
	}

	// Publish diagnostics after document change
	if glspCtx := ctx.GLSPContext(); glspCtx != nil {
		ctx.PublishDiagnostics(glspCtx, uri)
	}

	return nil
}

// DidClose handles the textDocument/didClose notification
func DidClose(ctx types.ServerContext, context *glsp.Context, params *protocol.DidCloseTextDocumentParams) error {
	uri := params.TextDocument.URI

	fmt.Fprintf(os.Stderr, "[DTLS] Document closed: %s\n", uri)

	return ctx.DocumentManager().DidClose(uri)
}
