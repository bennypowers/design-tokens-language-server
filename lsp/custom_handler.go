package lsp

import (
	"encoding/json"

	"bennypowers.dev/dtls/lsp/methods/textDocument/diagnostic"
	"bennypowers.dev/dtls/lsp/types"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// CustomHandler wraps protocol.Handler to add custom method support
//
// WORKAROUND: This wrapper is needed to support LSP 3.17 methods while using glsp v0.2.2
// which only implements LSP 3.16. The protocol.Handler struct doesn't have fields for
// LSP 3.17 methods like textDocument/diagnostic, so we intercept them here.
//
// When glsp is updated to support LSP 3.17, this wrapper can be removed and we can
// register handlers directly in protocol.Handler (protocol_3_17.Handler).
type CustomHandler struct {
	*protocol.Handler // Pointer to avoid copying embedded mutex
	server            *Server
}

// Handle implements glsp.Handler interface
func (h *CustomHandler) Handle(context *glsp.Context) (r any, validMethod bool, validParams bool, err error) {
	// WORKAROUND: Intercept textDocument/diagnostic for LSP 3.17 pull diagnostics
	// This method doesn't exist in protocol.Handler (LSP 3.16), so we handle it manually
	if context.Method == "textDocument/diagnostic" {
		// Parse params manually since protocol.Handler doesn't know about this method
		var params diagnostic.DocumentDiagnosticParams
		if err := json.Unmarshal(context.Params, &params); err != nil {
			return nil, true, false, err
		}

		// Create request context and call our handler
		req := types.NewRequestContext(h.server, context)
		result, err := diagnostic.DocumentDiagnostic(req, &params)
		if err != nil {
			return nil, true, true, err
		}

		return result, true, true, nil
	}

	// NOTE: textDocument/semanticTokens/delta handler removed
	// Delta support is disabled in capabilities (see initialize.go) because the implementation
	// lacks proper result caching and diffing, which would corrupt client state.
	// If delta support is re-enabled in the future, implement proper resultId bookkeeping:
	//   1. Store token arrays by resultID when full responses are produced
	//   2. Look up previous array using params.PreviousResultID
	//   3. Compute correct SemanticTokensEdit(s) with proper Start/DeleteCount/Data
	//   4. Update and return new resultID atomically

	// Fall through to default protocol.Handler
	return h.Handler.Handle(context)
}
