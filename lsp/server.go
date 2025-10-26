package lsp

import (
	"fmt"
	"os"

	"github.com/bennypowers/design-tokens-language-server/internal/documents"
	"github.com/bennypowers/design-tokens-language-server/internal/parser/css"
	"github.com/bennypowers/design-tokens-language-server/internal/tokens"
	"github.com/bennypowers/design-tokens-language-server/lsp/methods/lifecycle"
	"github.com/bennypowers/design-tokens-language-server/lsp/methods/textDocument"
	codeaction "github.com/bennypowers/design-tokens-language-server/lsp/methods/textDocument/codeAction"
	"github.com/bennypowers/design-tokens-language-server/lsp/methods/textDocument/completion"
	"github.com/bennypowers/design-tokens-language-server/lsp/methods/textDocument/definition"
	"github.com/bennypowers/design-tokens-language-server/lsp/methods/textDocument/diagnostic"
	documentcolor "github.com/bennypowers/design-tokens-language-server/lsp/methods/textDocument/documentColor"
	"github.com/bennypowers/design-tokens-language-server/lsp/methods/textDocument/hover"
	"github.com/bennypowers/design-tokens-language-server/lsp/methods/textDocument/references"
	semantictokens "github.com/bennypowers/design-tokens-language-server/lsp/methods/textDocument/semanticTokens"
	"github.com/bennypowers/design-tokens-language-server/lsp/methods/workspace"
	"github.com/bennypowers/design-tokens-language-server/lsp/types"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
	"github.com/tliron/glsp/server"
)

// Verify that Server implements ServerContext interface
var _ types.ServerContext = (*Server)(nil)

// Server represents the Design Tokens Language Server
type Server struct {
	documents   *documents.Manager
	tokens      *tokens.Manager
	glspServer  *server.Server
	context     *glsp.Context
	rootURI     string                // Workspace root URI
	rootPath    string                // Workspace root path (file system)
	config      types.ServerConfig    // Server configuration
	loadedFiles map[string]string     // Track loaded files: filepath -> prefix
}

// NewServer creates a new Design Tokens LSP server
func NewServer() (*Server, error) {
	s := &Server{
		documents:   documents.NewManager(),
		tokens:      tokens.NewManager(),
		config:      types.DefaultConfig(),
		loadedFiles: make(map[string]string),
	}

	// Create the GLSP server with our handlers wrapped with middleware
	protocolHandler := protocol.Handler{
		Initialize:                      method(s, "initialize", lifecycle.Initialize),
		Initialized:                     notify(s, "initialized", lifecycle.Initialized),
		Shutdown:                        noParam(s, "shutdown", lifecycle.Shutdown),
		SetTrace:                        notify(s, "$/setTrace", lifecycle.SetTrace),
		WorkspaceDidChangeConfiguration: notify(s, "workspace/didChangeConfiguration", workspace.DidChangeConfiguration),
		WorkspaceDidChangeWatchedFiles:  notify(s, "workspace/didChangeWatchedFiles", workspace.DidChangeWatchedFiles),
		TextDocumentDidOpen:             notify(s, "textDocument/didOpen", textDocument.DidOpen),
		TextDocumentDidChange:           notify(s, "textDocument/didChange", textDocument.DidChange),
		TextDocumentDidClose:            notify(s, "textDocument/didClose", textDocument.DidClose),
		TextDocumentHover:               method(s, "textDocument/hover", hover.Hover),
		TextDocumentCompletion:          method(s, "textDocument/completion", completion.Completion),
		CompletionItemResolve:           method(s, "completionItem/resolve", completion.CompletionResolve),
		TextDocumentDefinition:          method(s, "textDocument/definition", definition.Definition),
		TextDocumentReferences:          method(s, "textDocument/references", references.References),
		TextDocumentColor:               method(s, "textDocument/documentColor", documentcolor.DocumentColor),
		TextDocumentColorPresentation:   method(s, "textDocument/colorPresentation", documentcolor.ColorPresentation),
		TextDocumentCodeAction:          method(s, "textDocument/codeAction", codeaction.CodeAction),
		CodeActionResolve:               method(s, "codeAction/resolve", codeaction.CodeActionResolve),
		TextDocumentSemanticTokensFull:  method(s, "textDocument/semanticTokens/full", semantictokens.SemanticTokensFull),
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

// Close releases server resources including the CSS parser pool.
// It is safe to call Close multiple times.
// This method should be called when the server is no longer needed,
// typically in test cleanup via defer server.Close().
func (s *Server) Close() error {
	// Clean up the CSS parser pool
	css.ClosePool()
	return nil
}

// ServerContext interface implementation

// Document returns the document with the given URI
func (s *Server) Document(uri string) *documents.Document {
	return s.documents.Get(uri)
}

// DocumentManager returns the document manager
func (s *Server) DocumentManager() *documents.Manager {
	return s.documents
}

// AllDocuments returns all tracked documents
func (s *Server) AllDocuments() []*documents.Document {
	return s.documents.GetAll()
}

// Token returns the token with the given name
func (s *Server) Token(name string) *tokens.Token {
	return s.tokens.Get(name)
}

// TokenManager returns the token manager
func (s *Server) TokenManager() *tokens.Manager {
	return s.tokens
}

// TokenCount returns the number of tokens
func (s *Server) TokenCount() int {
	return s.tokens.Count()
}

// RootURI returns the workspace root URI
func (s *Server) RootURI() string {
	return s.rootURI
}

// RootPath returns the workspace root path
func (s *Server) RootPath() string {
	return s.rootPath
}

// SetRootURI sets the workspace root URI
func (s *Server) SetRootURI(uri string) {
	s.rootURI = uri
}

// SetRootPath sets the workspace root path
func (s *Server) SetRootPath(path string) {
	s.rootPath = path
}

// GLSPContext returns the GLSP context
func (s *Server) GLSPContext() *glsp.Context {
	return s.context
}

// SetGLSPContext sets the GLSP context
func (s *Server) SetGLSPContext(ctx *glsp.Context) {
	s.context = ctx
}

// PublishDiagnostics publishes diagnostics for a document
func (s *Server) PublishDiagnostics(context *glsp.Context, uri string) error {
	fmt.Fprintf(os.Stderr, "[DTLS] Publishing diagnostics for: %s\n", uri)

	diagnostics, err := diagnostic.GetDiagnostics(s, uri)
	if err != nil {
		return err
	}

	// Publish diagnostics to the client
	context.Notify(protocol.ServerTextDocumentPublishDiagnostics, protocol.PublishDiagnosticsParams{
		URI:         uri,
		Diagnostics: diagnostics,
	})

	return nil
}

// GetDiagnostics returns diagnostics for a document (for testing)
func (s *Server) GetDiagnostics(uri string) ([]protocol.Diagnostic, error) {
	return diagnostic.GetDiagnostics(s, uri)
}
