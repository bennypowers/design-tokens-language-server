package lifecycle

import (
	"fmt"
	"os"

	"bennypowers.dev/dtls/lsp/types"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// Initialized handles the LSP initialized notification
func Initialized(ctx types.ServerContext, context *glsp.Context, params *protocol.InitializedParams) error {
	fmt.Fprintf(os.Stderr, "[DTLS] Server initialized\n")

	// Store context for later use (diagnostics)
	ctx.SetGLSPContext(context)

	// Load token files from workspace using configuration
	if err := ctx.LoadTokensFromConfig(); err != nil {
		fmt.Fprintf(os.Stderr, "[DTLS] Warning: failed to load token files: %v\n", err)
		// Don't fail initialization, just log the error
	}

	// Register file watchers for token files
	if err := ctx.RegisterFileWatchers(context); err != nil {
		fmt.Fprintf(os.Stderr, "[DTLS] Warning: failed to register file watchers: %v\n", err)
		// Don't fail initialization, just log the error
	}

	return nil
}
