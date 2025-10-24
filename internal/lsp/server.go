package lsp

import (
	"fmt"
	"os"

	"github.com/bennypowers/design-tokens-language-server/internal/documents"
	"github.com/bennypowers/design-tokens-language-server/internal/tokens"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
	"github.com/tliron/glsp/server"
)

// Server represents the Design Tokens Language Server
type Server struct {
	documents  *documents.Manager
	tokens     *tokens.Manager
	glspServer *server.Server
	context    *glsp.Context
}

// NewServer creates a new Design Tokens LSP server
func NewServer() (*Server, error) {
	s := &Server{
		documents: documents.NewManager(),
		tokens:    tokens.NewManager(),
	}

	// Create the GLSP server with our handlers
	handler := protocol.Handler{
		Initialize:                    s.handleInitialize,
		Initialized:                   s.handleInitialized,
		Shutdown:                      s.handleShutdown,
		SetTrace:                      s.handleSetTrace,
		TextDocumentDidOpen:           s.handleDidOpen,
		TextDocumentDidChange:         s.handleDidChange,
		TextDocumentDidClose:          s.handleDidClose,
		TextDocumentHover:             s.handleHover,
		TextDocumentCompletion:        s.handleCompletion,
		CompletionItemResolve:         s.handleCompletionResolve,
		TextDocumentDefinition:        s.handleDefinition,
		TextDocumentReferences:        s.handleReferences,
		TextDocumentColor:             s.handleDocumentColor,
		TextDocumentColorPresentation: s.handleColorPresentation,
	}

	// Create GLSP server with debug enabled for stdio
	s.glspServer = server.NewServer(&handler, "design-tokens-lsp", true)

	return s, nil
}

// RunStdio starts the LSP server using stdio transport
func (s *Server) RunStdio() error {
	return s.glspServer.RunStdio()
}

// handleInitialize handles the initialize request
func (s *Server) handleInitialize(context *glsp.Context, params *protocol.InitializeParams) (interface{}, error) {
	clientName := "unknown"
	if params.ClientInfo != nil {
		clientName = params.ClientInfo.Name
	}

	fmt.Fprintf(os.Stderr, "[DTLS] Initializing for client: %s\n", clientName)

	// Build server capabilities
	syncKind := protocol.TextDocumentSyncKindIncremental
	capabilities := protocol.ServerCapabilities{
		TextDocumentSync: protocol.TextDocumentSyncOptions{
			OpenClose: boolPtr(true),
			Change:    &syncKind,
		},
		HoverProvider:      true,
		CompletionProvider: &protocol.CompletionOptions{
			ResolveProvider: boolPtr(true),
		},
		DefinitionProvider: true,
		ReferencesProvider: true,
		CodeActionProvider: &protocol.CodeActionOptions{
			ResolveProvider: boolPtr(true),
		},
		ColorProvider: true,
		SemanticTokensProvider: &protocol.SemanticTokensOptions{
			Legend: protocol.SemanticTokensLegend{
				TokenTypes:     []string{"variable", "property"},
				TokenModifiers: []string{"declaration", "definition", "readonly"},
			},
			Full: true,
		},
	}

	return protocol.InitializeResult{
		Capabilities: capabilities,
		ServerInfo: &protocol.InitializeResultServerInfo{
			Name:    "design-tokens-language-server",
			Version: strPtr("1.0.0-alpha"),
		},
	}, nil
}

// handleInitialized handles the initialized notification
func (s *Server) handleInitialized(context *glsp.Context, params *protocol.InitializedParams) error {
	fmt.Fprintf(os.Stderr, "[DTLS] Server initialized\n")
	// Store context for later use (diagnostics)
	s.context = context
	return nil
}

// handleShutdown handles the shutdown request
func (s *Server) handleShutdown(context *glsp.Context) error {
	fmt.Fprintf(os.Stderr, "[DTLS] Server shutting down\n")
	return nil
}

// handleSetTrace handles the setTrace notification
func (s *Server) handleSetTrace(context *glsp.Context, params *protocol.SetTraceParams) error {
	fmt.Fprintf(os.Stderr, "[DTLS] Trace level set to: %s\n", params.Value)
	return nil
}

// handleDidOpen handles the textDocument/didOpen notification
func (s *Server) handleDidOpen(context *glsp.Context, params *protocol.DidOpenTextDocumentParams) error {
	err := s.DidOpen(params.TextDocument.URI, params.TextDocument.LanguageID, int(params.TextDocument.Version), params.TextDocument.Text)
	if err != nil {
		return err
	}

	// Publish diagnostics for the opened document
	if s.context != nil {
		s.PublishDiagnostics(s.context, params.TextDocument.URI)
	}

	return nil
}

// DidOpen opens a document (exposed for testing)
func (s *Server) DidOpen(uri, languageID string, version int, content string) error {
	fmt.Fprintf(os.Stderr, "[DTLS] Document opened: %s (language: %s, version: %d)\n", uri, languageID, version)
	return s.documents.DidOpen(uri, languageID, version, content)
}

// handleDidChange handles the textDocument/didChange notification
func (s *Server) handleDidChange(context *glsp.Context, params *protocol.DidChangeTextDocumentParams) error {
	uri := params.TextDocument.URI
	version := int(params.TextDocument.Version)

	fmt.Fprintf(os.Stderr, "[DTLS] Document changed: %s (version: %d, changes: %d)\n", uri, version, len(params.ContentChanges))

	// Convert any[] to proper type
	changes := make([]protocol.TextDocumentContentChangeEvent, len(params.ContentChanges))
	for i, change := range params.ContentChanges {
		if changeEvent, ok := change.(protocol.TextDocumentContentChangeEvent); ok {
			changes[i] = changeEvent
		}
	}

	err := s.documents.DidChange(uri, version, changes)
	if err != nil {
		return err
	}

	// Publish diagnostics after document change
	if s.context != nil {
		s.PublishDiagnostics(s.context, uri)
	}

	return nil
}

// handleDidClose handles the textDocument/didClose notification
func (s *Server) handleDidClose(context *glsp.Context, params *protocol.DidCloseTextDocumentParams) error {
	uri := params.TextDocument.URI

	fmt.Fprintf(os.Stderr, "[DTLS] Document closed: %s\n", uri)

	return s.documents.DidClose(uri)
}

func boolPtr(b bool) *bool {
	return &b
}

func strPtr(s string) *string {
	return &s
}
