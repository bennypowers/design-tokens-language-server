package workspace

import (
	"fmt"
	"os"

	"bennypowers.dev/dtls/internal/uriutil"
	"bennypowers.dev/dtls/lsp/types"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// DidChangeWatchedFiles handles the workspace/didChangeWatchedFiles notification
func DidChangeWatchedFiles(req *types.RequestContext, params *protocol.DidChangeWatchedFilesParams) error {
	fmt.Fprintf(os.Stderr, "[DTLS] Watched files changed: %d files\n", len(params.Changes))

	// Track if we need to reload tokens
	needsReload := false
	hasDeletedFile := false

	for _, change := range params.Changes {
		uri := change.URI
		path := uriutil.URIToPath(uri)
		fmt.Fprintf(os.Stderr, "[DTLS] File change: %s (type: %d)\n", path, change.Type)

		// Check if this is a token file we're watching
		if req.Server.IsTokenFile(path) {
			// If the file was deleted, remove it from loaded files
			if change.Type == protocol.FileChangeTypeDeleted {
				fmt.Fprintf(os.Stderr, "[DTLS] Token file deleted: %s\n", path)
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
		fmt.Fprintf(os.Stderr, "[DTLS] Reloading token files due to changes\n")

		// If a file was deleted, we need to force clear tokens even if
		// LoadTokensFromConfig wouldn't normally clear them (e.g., if loadedFiles is now empty)
		if hasDeletedFile {
			req.Server.TokenManager().Clear()
			fmt.Fprintf(os.Stderr, "[DTLS] Cleared all tokens due to file deletion\n")
		}

		if err := req.Server.LoadTokensFromConfig(); err != nil {
			fmt.Fprintf(os.Stderr, "[DTLS] Warning: failed to reload tokens: %v\n", err)
		}

		// Republish diagnostics for all open documents
		glspCtx := req.Server.GLSPContext()
		if glspCtx != nil {
			for _, doc := range req.Server.AllDocuments() {
				if err := req.Server.PublishDiagnostics(glspCtx, doc.URI()); err != nil {
					fmt.Fprintf(os.Stderr, "[DTLS] Warning: failed to publish diagnostics for %s: %v\n", doc.URI(), err)
				}
			}
		}
	}

	return nil
}
