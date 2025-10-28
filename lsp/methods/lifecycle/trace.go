package lifecycle

import (
	"fmt"
	"os"

	"bennypowers.dev/dtls/lsp/types"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// SetTrace handles the $/setTrace notification
func SetTrace(req *types.RequestContext, params *protocol.SetTraceParams) error {
	fmt.Fprintf(os.Stderr, "[DTLS] Trace level set to: %s\n", params.Value)
	return nil
}
