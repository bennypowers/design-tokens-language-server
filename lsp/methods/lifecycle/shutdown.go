package lifecycle

import (
	"fmt"
	"os"

	"github.com/bennypowers/design-tokens-language-server/internal/parser/css"
	"github.com/bennypowers/design-tokens-language-server/lsp/types"
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
