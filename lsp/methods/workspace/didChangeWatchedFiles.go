package workspace

import (
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/bennypowers/design-tokens-language-server/lsp/types"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// DidChangeWatchedFiles handles the workspace/didChangeWatchedFiles notification
func DidChangeWatchedFiles(ctx types.ServerContext, context *glsp.Context, params *protocol.DidChangeWatchedFilesParams) error {
	fmt.Fprintf(os.Stderr, "[DTLS] Watched files changed: %d files\n", len(params.Changes))

	// Track if we need to reload tokens
	needsReload := false

	for _, change := range params.Changes {
		uri := change.URI
		path := uriToPath(uri)
		fmt.Fprintf(os.Stderr, "[DTLS] File change: %s (type: %d)\n", path, change.Type)

		// Check if this is a token file we're watching
		if ctx.IsTokenFile(path) {
			needsReload = true

			// If the file was deleted, we might want to handle it differently
			if change.Type == protocol.FileChangeTypeDeleted {
				fmt.Fprintf(os.Stderr, "[DTLS] Token file deleted: %s\n", path)
			}
		}
	}

	// Reload all token files if any token file changed
	if needsReload {
		fmt.Fprintf(os.Stderr, "[DTLS] Reloading token files due to changes\n")
		if err := ctx.LoadTokensFromConfig(); err != nil {
			fmt.Fprintf(os.Stderr, "[DTLS] Warning: failed to reload tokens: %v\n", err)
		}

		// Republish diagnostics for all open documents
		glspCtx := ctx.GLSPContext()
		if glspCtx != nil {
			for _, doc := range ctx.AllDocuments() {
				if err := ctx.PublishDiagnostics(glspCtx, doc.URI()); err != nil {
					fmt.Fprintf(os.Stderr, "[DTLS] Warning: failed to publish diagnostics for %s: %v\n", doc.URI(), err)
				}
			}
		}
	}

	return nil
}

// uriToPath converts a URI to a file path
func uriToPath(uri string) string {
	// Parse the URI
	u, err := url.Parse(uri)
	if err != nil {
		return uri
	}

	// Return the path component
	path := u.Path

	// On Windows, file:///C:/path becomes /C:/path, so we need to trim the leading slash
	if len(path) > 2 && path[0] == '/' && path[2] == ':' {
		path = path[1:]
	}

	// Decode percent-encoded characters
	decoded, err := url.PathUnescape(path)
	if err != nil {
		return path
	}

	// Convert forward slashes to backslashes on Windows if needed
	if strings.Contains(decoded, ":") {
		decoded = strings.ReplaceAll(decoded, "/", "\\")
	}

	return decoded
}
