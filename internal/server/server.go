package server

import (
	"fmt"

	protocol "github.com/tliron/glsp/protocol_3_16"
)

// Server represents the Design Tokens Language Server
type Server struct {
	initialized bool
	version     string
}

// New creates a new Design Tokens Language Server instance
func New() *Server {
	return &Server{
		version: "1.0.0-go", // TODO: Get from build info
	}
}

// Initialize handles the LSP initialize request
func (s *Server) Initialize(params *protocol.InitializeParams) (*protocol.InitializeResult, error) {
	if params == nil {
		return nil, fmt.Errorf("initialize params cannot be nil")
	}

	// Build server capabilities
	syncKind := protocol.TextDocumentSyncKindIncremental
	capabilities := protocol.ServerCapabilities{
		// Text document sync - incremental
		TextDocumentSync: protocol.TextDocumentSyncOptions{
			OpenClose: boolPtr(true),
			Change:    &syncKind,
		},

		// Hover support
		HoverProvider: true,

		// Completion with resolve support
		CompletionProvider: &protocol.CompletionOptions{
			ResolveProvider: boolPtr(true),
		},

		// Go to definition
		DefinitionProvider: true,

		// Find references
		ReferencesProvider: true,

		// Code actions with resolve
		CodeActionProvider: &protocol.CodeActionOptions{
			ResolveProvider: boolPtr(true),
		},

		// Document color
		ColorProvider: true,

		// Semantic tokens
		SemanticTokensProvider: &protocol.SemanticTokensOptions{
			Legend: protocol.SemanticTokensLegend{
				TokenTypes:     []string{"variable", "property"},
				TokenModifiers: []string{"declaration", "definition", "readonly"},
			},
			Full: true,
		},

		// Note: DiagnosticProvider is LSP 3.17, glsp uses 3.16
		// We'll implement diagnostics via textDocument/diagnostic request
	}

	result := &protocol.InitializeResult{
		Capabilities: capabilities,
	}

	return result, nil
}

// Initialized handles the LSP initialized notification
func (s *Server) Initialized(params *protocol.InitializedParams) error {
	s.initialized = true
	return nil
}

func boolPtr(b bool) *bool {
	return &b
}
