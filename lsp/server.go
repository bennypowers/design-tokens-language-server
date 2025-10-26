package lsp

import (
	"fmt"
	"os"

	"github.com/bennypowers/design-tokens-language-server/internal/documents"
	"github.com/bennypowers/design-tokens-language-server/internal/parser/css"
	"github.com/bennypowers/design-tokens-language-server/internal/tokens"
	"github.com/bennypowers/design-tokens-language-server/lsp/methods/lifecycle"
	"github.com/bennypowers/design-tokens-language-server/lsp/methods/textDocument"
	"github.com/bennypowers/design-tokens-language-server/lsp/methods/textDocument/codeAction"
	"github.com/bennypowers/design-tokens-language-server/lsp/methods/textDocument/completion"
	"github.com/bennypowers/design-tokens-language-server/lsp/methods/textDocument/definition"
	"github.com/bennypowers/design-tokens-language-server/lsp/methods/textDocument/diagnostic"
	documentcolor "github.com/bennypowers/design-tokens-language-server/lsp/methods/textDocument/documentColor"
	"github.com/bennypowers/design-tokens-language-server/lsp/methods/textDocument/hover"
	"github.com/bennypowers/design-tokens-language-server/lsp/methods/textDocument/references"
	semantictokens "github.com/bennypowers/design-tokens-language-server/lsp/methods/textDocument/semanticTokens"
	"github.com/bennypowers/design-tokens-language-server/lsp/types"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
	"github.com/tliron/glsp/server"
)

// Verify that Server implements ServerContext interface
var _ types.ServerContext = (*Server)(nil)

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

	// Create the GLSP server with our handlers wrapped with middleware
	protocolHandler := protocol.Handler{
		Initialize:                      method(s, "initialize", lifecycle.Initialize),
		Initialized:                     notify(s, "initialized", lifecycle.Initialized),
		Shutdown:                        noParam(s, "shutdown", lifecycle.Shutdown),
		SetTrace:                        notify(s, "$/setTrace", lifecycle.SetTrace),
		WorkspaceDidChangeConfiguration: notify(s, "workspace/didChangeConfiguration", DidChangeConfiguration),
		WorkspaceDidChangeWatchedFiles:  notify(s, "workspace/didChangeWatchedFiles", DidChangeWatchedFiles),
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

// TokenCount returns the number of loaded tokens (exposed for testing)
func (s *Server) TokenCount() int {
	return s.tokens.Count()
}

// DidOpen opens a document (exposed for testing)
func (s *Server) DidOpen(uri, languageID string, version int, content string) error {
	return s.documents.DidOpen(uri, languageID, version, content)
}

// Hover provides hover information (exposed for testing)
func (s *Server) Hover(params *protocol.HoverParams) (*protocol.Hover, error) {
	return hover.Hover(s, nil, params)
}

// GetDefinition provides definition information (exposed for testing)
func (s *Server) GetDefinition(params *protocol.DefinitionParams) ([]protocol.Location, error) {
	result, err := definition.Definition(s, nil, params)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, nil
	}
	if locations, ok := result.([]protocol.Location); ok {
		return locations, nil
	}
	return nil, nil
}

// GetReferences provides reference information (exposed for testing)
func (s *Server) GetReferences(params *protocol.ReferenceParams) ([]protocol.Location, error) {
	return references.References(s, nil, params)
}

// GetCompletions provides completion information (exposed for testing)
func (s *Server) GetCompletions(params *protocol.CompletionParams) (*protocol.CompletionList, error) {
	result, err := completion.Completion(s, nil, params)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, nil
	}
	if list, ok := result.(*protocol.CompletionList); ok {
		return list, nil
	}
	return nil, nil
}

// ResolveCompletion provides completion resolve information (exposed for testing)
func (s *Server) ResolveCompletion(item *protocol.CompletionItem) (*protocol.CompletionItem, error) {
	return completion.CompletionResolve(s, nil, item)
}

// DocumentColor provides color information (exposed for testing)
func (s *Server) DocumentColor(params *protocol.DocumentColorParams) ([]protocol.ColorInformation, error) {
	return documentcolor.DocumentColor(s, nil, params)
}

// ColorPresentation provides color presentations (exposed for testing)
func (s *Server) ColorPresentation(params *protocol.ColorPresentationParams) ([]protocol.ColorPresentation, error) {
	return documentcolor.ColorPresentation(s, nil, params)
}

// CodeAction provides code actions (exposed for testing)
func (s *Server) CodeAction(params *protocol.CodeActionParams) ([]protocol.CodeAction, error) {
	result, err := codeaction.CodeAction(s, nil, params)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, nil
	}
	// CodeAction returns any, could be []protocol.CodeAction
	if actions, ok := result.([]protocol.CodeAction); ok {
		return actions, nil
	}
	return nil, nil
}

// CodeActionResolve resolves a code action (exposed for testing)
func (s *Server) CodeActionResolve(action *protocol.CodeAction) (*protocol.CodeAction, error) {
	return codeaction.CodeActionResolve(s, nil, action)
}

// Initialize handles the initialize request (exposed for testing)
func (s *Server) Initialize(context *glsp.Context, params *protocol.InitializeParams) (interface{}, error) {
	return lifecycle.Initialize(s, context, params)
}

// Shutdown handles the shutdown request (exposed for testing)
func (s *Server) Shutdown(context *glsp.Context) error {
	return lifecycle.Shutdown(s, context)
}

// SetTrace handles the setTrace notification (exposed for testing)
func (s *Server) SetTrace(context *glsp.Context, params *protocol.SetTraceParams) error {
	return lifecycle.SetTrace(s, context, params)
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

// LoadTokensFromConfig loads tokens based on current configuration
func (s *Server) LoadTokensFromConfig() error {
	return s.loadTokensFromConfig()
}

// RegisterFileWatchers registers file watchers for token files
func (s *Server) RegisterFileWatchers(ctx *glsp.Context) error {
	return s.registerFileWatchers(ctx)
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

// Workspace method wrappers that adapt Server methods to ServerContext interface

func DidChangeConfiguration(ctx types.ServerContext, context *glsp.Context, params *protocol.DidChangeConfigurationParams) error {
	return ctx.(*Server).handleDidChangeConfiguration(context, params)
}

func DidChangeWatchedFiles(ctx types.ServerContext, context *glsp.Context, params *protocol.DidChangeWatchedFilesParams) error {
	return ctx.(*Server).handleDidChangeWatchedFiles(context, params)
}

// Completion is defined in completion.go

// CompletionResolve is defined in completion.go

// Definition is defined in definition.go

// References is defined in references.go

// DocumentColor is defined in color.go

// ColorPresentation is defined in color.go

// CodeAction is defined in code_actions.go

// CodeActionResolve is defined in code_actions.go

// SemanticTokensFull is defined in semantic_tokens.go
