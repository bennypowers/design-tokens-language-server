package textDocument

import (
	"bennypowers.dev/dtls/internal/documents"
	"bennypowers.dev/dtls/internal/log"

	"bennypowers.dev/dtls/lsp/types"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// DidOpen handles the textDocument/didOpen notification
func DidOpen(req *types.RequestContext, params *protocol.DidOpenTextDocumentParams) error {
	log.Info("Document opened: %s (language: %s, version: %d)",
		params.TextDocument.URI, params.TextDocument.LanguageID, int(params.TextDocument.Version))

	err := req.Server.DocumentManager().DidOpen(params.TextDocument.URI, params.TextDocument.LanguageID,
		int(params.TextDocument.Version), params.TextDocument.Text)
	if err != nil {
		return err
	}

	// Auto-load tokens from files that look like DTCG token files
	// This enables semantic tokens and other features for token files not in config
	languageID := params.TextDocument.LanguageID
	content := params.TextDocument.Text
	if (languageID == "json" || languageID == "yaml") &&
		(documents.IsDesignTokensSchema(content) || documents.LooksLikeDTCGContent(content)) {
		if err := req.Server.LoadTokensFromDocumentContent(
			params.TextDocument.URI,
			languageID,
			content,
		); err != nil {
			log.Warn("Failed to auto-load tokens from %s: %v", params.TextDocument.URI, err)
		}
	}

	// Publish diagnostics for the opened document (only if using push model)
	// If client supports pull diagnostics (LSP 3.17), it will request them via textDocument/diagnostic
	if !req.Server.UsePullDiagnostics() {
		if glspCtx := req.Server.GLSPContext(); glspCtx != nil {
			if err := req.Server.PublishDiagnostics(glspCtx, params.TextDocument.URI); err != nil {
				log.Warn("Failed to publish diagnostics for %s: %v", params.TextDocument.URI, err)
			}
		}
	}

	return nil
}

// DidChange handles the textDocument/didChange notification
func DidChange(req *types.RequestContext, params *protocol.DidChangeTextDocumentParams) error {
	uri := params.TextDocument.URI
	version := int(params.TextDocument.Version)

	log.Info("Document changed: %s (version: %d, changes: %d)", uri, version, len(params.ContentChanges))

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
				log.Warn("Failed to publish diagnostics for %s: %v", uri, err)
			}
		}
	}

	return nil
}

// DidClose handles the textDocument/didClose notification
func DidClose(req *types.RequestContext, params *protocol.DidCloseTextDocumentParams) error {
	uri := params.TextDocument.URI

	log.Info("Document closed: %s", uri)

	// Invalidate semantic token cache for this document
	req.Server.SemanticTokenCache().Invalidate(uri)

	return req.Server.DocumentManager().DidClose(uri)
}
