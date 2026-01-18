package workspace

import (
	"bennypowers.dev/dtls/internal/log"

	"bennypowers.dev/dtls/internal/uriutil"
	"bennypowers.dev/dtls/lsp/types"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// DidChangeWatchedFiles handles the workspace/didChangeWatchedFiles notification
func DidChangeWatchedFiles(req *types.RequestContext, params *protocol.DidChangeWatchedFilesParams) error {
	log.Info("Watched files changed: %d files", len(params.Changes))

	// Track if we need to reload tokens
	needsReload := false
	hasDeletedFile := false

	for _, change := range params.Changes {
		uri := change.URI
		path := uriutil.URIToPath(uri)
		log.Info("File change: %s (type: %d)", path, change.Type)

		// Check if this is a token file we're watching
		if req.Server.IsTokenFile(path) {
			// If the file was deleted, remove it from loaded files
			if change.Type == protocol.FileChangeTypeDeleted {
				log.Info("Token file deleted: %s", path)
				req.Server.RemoveLoadedFile(path)
				hasDeletedFile = true
				// Still trigger reload to clear tokens from the deleted file
				// The reload will re-scan remaining files, excluding the deleted one
			}

			// File was created, modified, or deleted - trigger reload
			needsReload = true
		}
	}

	// Reload all token files if any token file changed
	if needsReload {
		log.Info("Reloading token files due to changes")

		// If a file was deleted, we need to force clear tokens even if
		// LoadTokensFromConfig wouldn't normally clear them (e.g., if loadedFiles is now empty)
		if hasDeletedFile {
			req.Server.TokenManager().Clear()
			log.Info("Cleared all tokens due to file deletion")
		}

		if err := req.Server.LoadTokensFromConfig(); err != nil {
			log.Info("Warning: failed to reload tokens: %v", err)
		}

		// Republish diagnostics for all open documents (only if using push model)
		// If client supports pull diagnostics (LSP 3.17), it will request them via textDocument/diagnostic
		if !req.Server.UsePullDiagnostics() {
			glspCtx := req.Server.GLSPContext()
			if glspCtx != nil {
				for _, doc := range req.Server.AllDocuments() {
					if err := req.Server.PublishDiagnostics(glspCtx, doc.URI()); err != nil {
						log.Info("Warning: failed to publish diagnostics for %s: %v", doc.URI(), err)
					}
				}
			}
		}
	}

	return nil
}
