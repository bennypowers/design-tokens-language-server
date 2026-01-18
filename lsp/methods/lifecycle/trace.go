package lifecycle

import (
	"bennypowers.dev/dtls/internal/log"

	"bennypowers.dev/dtls/lsp/types"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// SetTrace handles the $/setTrace notification
func SetTrace(req *types.RequestContext, params *protocol.SetTraceParams) error {
	log.Info("Trace level set to: %s", params.Value)
	return nil
}
