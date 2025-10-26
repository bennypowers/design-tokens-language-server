package lifecycle

import (
	"fmt"
	"os"

	"bennypowers.dev/dtls/internal/uriutil"
	"bennypowers.dev/dtls/lsp/methods/textDocument/diagnostic"
	"bennypowers.dev/dtls/lsp/types"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// Initialize handles the LSP initialize request
func Initialize(ctx types.ServerContext, context *glsp.Context, params *protocol.InitializeParams) (any, error) {
	clientName := "unknown"
	if params.ClientInfo != nil {
		clientName = params.ClientInfo.Name
	}

	fmt.Fprintf(os.Stderr, "[DTLS] Initializing for client: %s\n", clientName)

	// Store the workspace root
	if params.RootURI != nil {
		ctx.SetRootURI(*params.RootURI)
		// Convert URI to file path
		ctx.SetRootPath(uriutil.URIToPath(*params.RootURI))
		fmt.Fprintf(os.Stderr, "[DTLS] Workspace root: %s\n", ctx.RootPath())
	} else if params.RootPath != nil {
		ctx.SetRootPath(*params.RootPath)
		ctx.SetRootURI(uriutil.PathToURI(*params.RootPath))
		fmt.Fprintf(os.Stderr, "[DTLS] Workspace root (from rootPath): %s\n", ctx.RootPath())
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
		"hoverProvider":      true,
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
				"delta": true,
			},
		},
		// LSP 3.17: Pull diagnostics support
		"diagnosticProvider": diagnostic.DiagnosticOptions{
			InterFileDependencies: false,
			WorkspaceDiagnostics:  false,
		},
	}

	// WORKAROUND: Return custom struct with any type for Capabilities field
	// protocol.InitializeResult expects ServerCapabilities (LSP 3.16), but we need to
	// include LSP 3.17 fields. When glsp is updated, we can use protocol_3_17.InitializeResult.
	return struct {
		Capabilities any                                      `json:"capabilities"`
		ServerInfo   *protocol.InitializeResultServerInfo `json:"serverInfo,omitempty"`
	}{
		Capabilities: capabilities,
		ServerInfo: &protocol.InitializeResultServerInfo{
			Name:    "design-tokens-language-server",
			Version: strPtr("1.0.0-alpha"),
		},
	}, nil
}

func boolPtr(b bool) *bool {
	return &b
}

func strPtr(s string) *string {
	return &s
}
