package lsp

import (
	"fmt"
	"os"

	"github.com/bennypowers/design-tokens-language-server/internal/documents"
	"github.com/bennypowers/design-tokens-language-server/internal/parser/css"
	"github.com/bennypowers/design-tokens-language-server/internal/tokens"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
	"github.com/tliron/glsp/server"
)

// Server represents the Design Tokens Language Server
type Server struct {
	documents    *documents.Manager
	tokens       *tokens.Manager
	glspServer   *server.Server
	context      *glsp.Context
	rootURI      string                 // Workspace root URI
	rootPath     string                 // Workspace root path (file system)
	config       ServerConfig           // Server configuration
	loadedFiles  map[string]string      // Track loaded files: filepath -> prefix
}

// NewServer creates a new Design Tokens LSP server
func NewServer() (*Server, error) {
	s := &Server{
		documents:   documents.NewManager(),
		tokens:      tokens.NewManager(),
		config:      DefaultConfig(),
		loadedFiles: make(map[string]string),
	}

	// Create the GLSP server with our handlers
	protocolHandler := protocol.Handler{
		Initialize:                      s.handleInitialize,
		Initialized:                     s.handleInitialized,
		Shutdown:                        s.handleShutdown,
		SetTrace:                        s.handleSetTrace,
		WorkspaceDidChangeConfiguration: s.handleDidChangeConfiguration,
		WorkspaceDidChangeWatchedFiles:  s.handleDidChangeWatchedFiles,
		TextDocumentDidOpen:             s.handleDidOpen,
		TextDocumentDidChange:           s.handleDidChange,
		TextDocumentDidClose:            s.handleDidClose,
		TextDocumentHover:               s.handleHover,
		TextDocumentCompletion:          s.handleCompletion,
		CompletionItemResolve:           s.handleCompletionResolve,
		TextDocumentDefinition:          s.handleDefinition,
		TextDocumentReferences:          s.handleReferences,
		TextDocumentColor:               s.handleDocumentColor,
		TextDocumentColorPresentation:   s.handleColorPresentation,
		TextDocumentCodeAction:          s.handleCodeAction,
		CodeActionResolve:               s.handleCodeActionResolve,
		TextDocumentSemanticTokensFull:  s.handleSemanticTokensFull,
	}

	// WORKAROUND: Wrap with custom handler to support LSP 3.17 features
	// The CustomHandler intercepts LSP 3.17 methods (like textDocument/diagnostic)
	// before they reach protocol.Handler, which only knows about LSP 3.16 methods.
	// When glsp is updated to LSP 3.17, we can remove CustomHandler and use protocol_3_17.Handler directly.
	customHandler := &CustomHandler{
		Handler: protocolHandler,
		server:  s,
	}

	// Create GLSP server with debug enabled for stdio
	s.glspServer = server.NewServer(customHandler, "design-tokens-lsp", true)

	return s, nil
}

// RunStdio starts the LSP server using stdio transport
func (s *Server) RunStdio() error {
	return s.glspServer.RunStdio()
}

// TokenCount returns the number of loaded tokens (for testing)
func (s *Server) TokenCount() int {
	return s.tokens.Count()
}

// Initialize handles the initialize request (exposed for testing)
func (s *Server) Initialize(context *glsp.Context, params *protocol.InitializeParams) (interface{}, error) {
	return s.handleInitialize(context, params)
}

// Initialized handles the initialized notification (exposed for testing)
func (s *Server) Initialized(context *glsp.Context, params *protocol.InitializedParams) error {
	return s.handleInitialized(context, params)
}

// Shutdown handles the shutdown request (exposed for testing)
func (s *Server) Shutdown(context *glsp.Context) error {
	return s.handleShutdown(context)
}

// SetTrace handles the setTrace notification (exposed for testing)
func (s *Server) SetTrace(context *glsp.Context, params *protocol.SetTraceParams) error {
	return s.handleSetTrace(context, params)
}

// handleInitialize handles the initialize request
func (s *Server) handleInitialize(context *glsp.Context, params *protocol.InitializeParams) (interface{}, error) {
	clientName := "unknown"
	if params.ClientInfo != nil {
		clientName = params.ClientInfo.Name
	}

	fmt.Fprintf(os.Stderr, "[DTLS] Initializing for client: %s\n", clientName)

	// Store the workspace root
	if params.RootURI != nil {
		s.rootURI = *params.RootURI
		// Convert URI to file path
		s.rootPath = uriToPath(*params.RootURI)
		fmt.Fprintf(os.Stderr, "[DTLS] Workspace root: %s\n", s.rootPath)
	} else if params.RootPath != nil {
		s.rootPath = *params.RootPath
		s.rootURI = pathToURI(s.rootPath)
		fmt.Fprintf(os.Stderr, "[DTLS] Workspace root (from rootPath): %s\n", s.rootPath)
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
		"semanticTokensProvider": protocol.SemanticTokensOptions{
			Legend: protocol.SemanticTokensLegend{
				TokenTypes:     []string{"class", "property"}, // Match TypeScript: class for first part, property for rest
				TokenModifiers: []string{},
			},
			Full: boolPtr(true),
		},
		// LSP 3.17: Pull diagnostics support
		"diagnosticProvider": DiagnosticOptions{
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

// handleInitialized handles the initialized notification
func (s *Server) handleInitialized(context *glsp.Context, params *protocol.InitializedParams) error {
	fmt.Fprintf(os.Stderr, "[DTLS] Server initialized\n")
	// Store context for later use (diagnostics)
	s.context = context

	// Load token files from workspace using configuration
	if err := s.loadTokensFromConfig(); err != nil {
		fmt.Fprintf(os.Stderr, "[DTLS] Warning: failed to load token files: %v\n", err)
		// Don't fail initialization, just log the error
	}

	// Register file watchers for token files
	if err := s.registerFileWatchers(context); err != nil {
		fmt.Fprintf(os.Stderr, "[DTLS] Warning: failed to register file watchers: %v\n", err)
		// Don't fail initialization, just log the error
	}

	return nil
}

// Close releases server resources including the CSS parser pool.
// It is safe to call Close multiple times.
// This method should be called when the server is no longer needed,
// typically in test cleanup via defer server.Close().
func (s *Server) Close() error {
	// Clean up the CSS parser pool
	css.ClosePool()
	return nil
}

// handleShutdown handles the shutdown request
func (s *Server) handleShutdown(context *glsp.Context) error {
	fmt.Fprintf(os.Stderr, "[DTLS] Server shutting down\n")

	// Delegate to Close() for resource cleanup
	return s.Close()
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

	// Convert any[] to proper type, filtering out invalid entries
	changes := make([]protocol.TextDocumentContentChangeEvent, 0, len(params.ContentChanges))
	for _, change := range params.ContentChanges {
		if changeEvent, ok := change.(protocol.TextDocumentContentChangeEvent); ok {
			changes = append(changes, changeEvent)
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
