package textDocument

import (
	"fmt"
	"os"

	"bennypowers.dev/dtls/lsp/types"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// DidOpen handles the textDocument/didOpen notification
func DidOpen(req *types.RequestContext, params *protocol.DidOpenTextDocumentParams) error {
	fmt.Fprintf(os.Stderr, "[DTLS] Document opened: %s (language: %s, version: %d)\n",
		params.TextDocument.URI, params.TextDocument.LanguageID, int(params.TextDocument.Version))

	err := req.Server.DocumentManager().DidOpen(params.TextDocument.URI, params.TextDocument.LanguageID,
		int(params.TextDocument.Version), params.TextDocument.Text)
	if err != nil {
		return err
	}

	// Publish diagnostics for the opened document (only if using push model)
	// If client supports pull diagnostics (LSP 3.17), it will request them via textDocument/diagnostic
	if !req.Server.UsePullDiagnostics() {
		if glspCtx := req.Server.GLSPContext(); glspCtx != nil {
			if err := req.Server.PublishDiagnostics(glspCtx, params.TextDocument.URI); err != nil {
				fmt.Fprintf(os.Stderr, "[DTLS] Warning: failed to publish diagnostics for %s: %v\n", params.TextDocument.URI, err)
			}
		}
	}

	return nil
}

// DidChange handles the textDocument/didChange notification
func DidChange(req *types.RequestContext, params *protocol.DidChangeTextDocumentParams) error {
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

	err := req.Server.DocumentManager().DidChange(uri, version, changes)
	if err != nil {
		return err
	}

	// Publish diagnostics after document change (only if using push model)
	// If client supports pull diagnostics (LSP 3.17), it will request them via textDocument/diagnostic
	if !req.Server.UsePullDiagnostics() {
		if glspCtx := req.Server.GLSPContext(); glspCtx != nil {
			if err := req.Server.PublishDiagnostics(glspCtx, uri); err != nil {
				fmt.Fprintf(os.Stderr, "[DTLS] Warning: failed to publish diagnostics for %s: %v\n", uri, err)
			}
		}
	}

	return nil
}

// DidClose handles the textDocument/didClose notification
func DidClose(req *types.RequestContext, params *protocol.DidCloseTextDocumentParams) error {
	uri := params.TextDocument.URI

	fmt.Fprintf(os.Stderr, "[DTLS] Document closed: %s\n", uri)

	return req.Server.DocumentManager().DidClose(uri)
}
