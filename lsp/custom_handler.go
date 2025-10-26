package lsp

import (
	"encoding/json"

	"github.com/bennypowers/design-tokens-language-server/lsp/methods/textDocument/diagnostic"
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
	protocol.Handler
	server *Server
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

		// Call our handler
		result, err := diagnostic.DocumentDiagnostic(h.server, context, &params)
		if err != nil {
			return nil, true, true, err
		}

		return result, true, true, nil
	}

	// Fall through to default protocol.Handler
	return h.Handler.Handle(context)
}
