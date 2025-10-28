package lifecycle

import (
	"fmt"
	"os"

	"bennypowers.dev/dtls/internal/uriutil"
	"bennypowers.dev/dtls/internal/version"
	"bennypowers.dev/dtls/lsp/methods/textDocument/diagnostic"
	"bennypowers.dev/dtls/lsp/types"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// Initialize handles the LSP initialize request
func Initialize(req *types.RequestContext, params *protocol.InitializeParams) (any, error) {
	clientName := "unknown"
	if params.ClientInfo != nil {
		clientName = params.ClientInfo.Name
	}

	fmt.Fprintf(os.Stderr, "[DTLS] Initializing for client: %s\n", clientName)

	// Detect if client supports pull diagnostics (LSP 3.17)
	// The CustomHandler intercepts the initialize request and parses the raw capabilities JSON
	// to detect the presence of the textDocument.diagnostic field (LSP 3.17 feature).
	// Since glsp v0.2.2 only supports LSP 3.16, we must parse raw JSON to detect this capability.
	//
	// Conservative approach: Default to push diagnostics (false) unless we can confirm the client
	// explicitly declares diagnostic capability. This prevents breaking LSP 3.16 clients.
	supportsPullDiagnostics := false
	if detectedCapability := req.Server.ClientDiagnosticCapability(); detectedCapability != nil {
		// Use the detected capability from raw JSON parsing
		supportsPullDiagnostics = *detectedCapability
	}
	// else: capability not detected (nil) - default to push diagnostics (false)

	req.Server.SetUsePullDiagnostics(supportsPullDiagnostics)

	if supportsPullDiagnostics {
		fmt.Fprintf(os.Stderr, "[DTLS] Using pull diagnostics model (LSP 3.17) - client will request diagnostics\n")
	} else {
		fmt.Fprintf(os.Stderr, "[DTLS] Using push diagnostics model (LSP 3.0) - server will push diagnostics\n")
	}

	// Store the workspace root
	if params.RootURI != nil {
		req.Server.SetRootURI(*params.RootURI)
		// Convert URI to file path
		req.Server.SetRootPath(uriutil.URIToPath(*params.RootURI))
		fmt.Fprintf(os.Stderr, "[DTLS] Workspace root: %s\n", req.Server.RootPath())
	} else if params.RootPath != nil {
		req.Server.SetRootPath(*params.RootPath)
		req.Server.SetRootURI(uriutil.PathToURI(*params.RootPath))
		fmt.Fprintf(os.Stderr, "[DTLS] Workspace root (from rootPath): %s\n", req.Server.RootPath())
	}

	// Build server capabilities
	//
	// WORKAROUND: We use map[string]any instead of protocol.ServerCapabilities to include
	// LSP 3.17 fields that don't exist in glsp v0.2.2's protocol.ServerCapabilities struct.
	// When glsp is updated to LSP 3.17, we can switch back to using protocol_3_17.ServerCapabilities.
	syncKind := protocol.TextDocumentSyncKindIncremental
	capabilities := map[string]any{
		"textDocumentSync": protocol.TextDocumentSyncOptions{
			OpenClose: boolPtr(true),
			Change:    &syncKind,
		},
		"hoverProvider": true,
		"completionProvider": protocol.CompletionOptions{
			ResolveProvider: boolPtr(true),
		},
		"definitionProvider": true,
		"referencesProvider": true,
		"codeActionProvider": protocol.CodeActionOptions{
			ResolveProvider: boolPtr(true),
		},
		"colorProvider": true,
		"semanticTokensProvider": map[string]any{
			"legend": map[string]any{
				"tokenTypes":     []string{"class", "property"}, // Match TypeScript: class for first part, property for rest
				"tokenModifiers": []string{},
			},
			"full": map[string]any{
				"delta": false, // Disabled: delta implementation needs proper result caching and diffing
			},
		},
	}

	// LSP 3.17: Only advertise pull diagnostics if client supports it
	// For older clients, we'll use push diagnostics (textDocument/publishDiagnostics)
	if supportsPullDiagnostics {
		capabilities["diagnosticProvider"] = diagnostic.DiagnosticOptions{
			InterFileDependencies: false,
			WorkspaceDiagnostics:  false,
		}
	}

	// WORKAROUND: Return custom struct with any type for Capabilities field
	// protocol.InitializeResult expects ServerCapabilities (LSP 3.16), but we need to
	// include LSP 3.17 fields. When glsp is updated, we can use protocol_3_17.InitializeResult.
	return struct {
		Capabilities any                                  `json:"capabilities"`
		ServerInfo   *protocol.InitializeResultServerInfo `json:"serverInfo,omitempty"`
	}{
		Capabilities: capabilities,
		ServerInfo: &protocol.InitializeResultServerInfo{
			Name:    "design-tokens-language-server",
			Version: strPtr(version.GetVersion()),
		},
	}, nil
}

func boolPtr(b bool) *bool {
	return &b
}

func strPtr(s string) *string {
	return &s
}
