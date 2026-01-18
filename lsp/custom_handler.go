package lsp

import (
	"encoding/json"

	"bennypowers.dev/dtls/lsp/methods/textDocument/diagnostic"
	semantictokens "bennypowers.dev/dtls/lsp/methods/textDocument/semanticTokens"
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
func (h *CustomHandler) Handle(context *glsp.Context) (r any, validMethod, validParams bool, err error) {
	// WORKAROUND: Intercept initialize to detect diagnostic capability from raw params
	// Since glsp v0.2.2 only supports LSP 3.16, the parsed InitializeParams struct doesn't
	// include the LSP 3.17 "diagnostic" field. We parse the raw JSON here to detect it,
	// then let the normal initialize handler continue.
	if context.Method == "initialize" {
		// Detect pull diagnostics support from raw capabilities JSON
		supportsPullDiagnostics := DetectPullDiagnosticsSupport(context.Params)

		// Store the detected capability in the server for use during initialization
		h.server.SetClientDiagnosticCapability(supportsPullDiagnostics)

		// Fall through to let the normal initialize handler process the request
		// (don't return here - we want the standard initialization to proceed)
	}

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

	// Handle textDocument/semanticTokens/full/delta for incremental semantic token updates
	if context.Method == "textDocument/semanticTokens/full/delta" {
		var params semantictokens.SemanticTokensDeltaParams
		if err := json.Unmarshal(context.Params, &params); err != nil {
			return nil, true, false, err
		}

		req := types.NewRequestContext(h.server, context)
		result, err := semantictokens.SemanticTokensFullDelta(req, &params)
		if err != nil {
			return nil, true, true, err
		}

		return result, true, true, nil
	}

	// Fall through to default protocol.Handler
	return h.Handler.Handle(context)
}
