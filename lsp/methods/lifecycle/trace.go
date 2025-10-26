package lifecycle

import (
	"fmt"
	"os"

	"github.com/bennypowers/design-tokens-language-server/lsp/types"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// SetTrace handles the $/setTrace notification
func SetTrace(ctx types.ServerContext, context *glsp.Context, params *protocol.SetTraceParams) error {
	fmt.Fprintf(os.Stderr, "[DTLS] Trace level set to: %s\n", params.Value)
	return nil
}
