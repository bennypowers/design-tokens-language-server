package lifecycle

import (
	"fmt"
	"os"

	"bennypowers.dev/dtls/internal/parser/css"
	"bennypowers.dev/dtls/lsp/types"
	"github.com/tliron/glsp"
)

// Shutdown handles the LSP shutdown request
func Shutdown(ctx types.ServerContext, context *glsp.Context) error {
	fmt.Fprintf(os.Stderr, "[DTLS] Server shutting down\n")

	// Clean up the CSS parser pool
	// Note: This is currently handled by server.Close() but we put it here
	// for completeness in case we need other cleanup logic
	css.ClosePool()

	return nil
}
