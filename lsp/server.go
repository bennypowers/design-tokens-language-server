package lsp

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"bennypowers.dev/dtls/internal/documents"
	"bennypowers.dev/dtls/internal/parser/css"
	"bennypowers.dev/dtls/internal/tokens"
	"bennypowers.dev/dtls/lsp/methods/lifecycle"
	"bennypowers.dev/dtls/lsp/methods/textDocument"
	codeaction "bennypowers.dev/dtls/lsp/methods/textDocument/codeAction"
	"bennypowers.dev/dtls/lsp/methods/textDocument/completion"
	"bennypowers.dev/dtls/lsp/methods/textDocument/definition"
	"bennypowers.dev/dtls/lsp/methods/textDocument/diagnostic"
	documentcolor "bennypowers.dev/dtls/lsp/methods/textDocument/documentColor"
	"bennypowers.dev/dtls/lsp/methods/textDocument/hover"
	"bennypowers.dev/dtls/lsp/methods/textDocument/references"
	semantictokens "bennypowers.dev/dtls/lsp/methods/textDocument/semanticTokens"
	"bennypowers.dev/dtls/lsp/methods/workspace"
	"bennypowers.dev/dtls/lsp/types"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
	"github.com/tliron/glsp/server"
)

// Verify that Server implements ServerContext interface
var _ types.ServerContext = (*Server)(nil)

// Server represents the Design Tokens Language Server
type Server struct {
	documents          *documents.Manager
	tokens             *tokens.Manager
	glspServer         *server.Server
	context            *glsp.Context
	rootURI            string                       // Workspace root URI
	rootPath           string                       // Workspace root path (file system)
	config             types.ServerConfig           // Server configuration
	configMu           sync.RWMutex                 // Protects config, context, and usePullDiagnostics from concurrent access
	loadedFiles        map[string]*TokenFileOptions // Track loaded files: filepath -> options (prefix, groupMarkers)
	loadedFilesMu      sync.RWMutex                 // Protects loadedFiles from concurrent access
	usePullDiagnostics bool                         // Whether to use pull diagnostics (LSP 3.17) vs push (LSP 3.0)
}

// NewServer creates a new Design Tokens LSP server
func NewServer() (*Server, error) {
	s := &Server{
		documents:   documents.NewManager(),
		tokens:      tokens.NewManager(),
		config:      types.DefaultConfig(),
		loadedFiles: make(map[string]*TokenFileOptions),
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
		Handler: &protocolHandler,
		server:  s,
	}

	// Create GLSP server with debug enabled for stdio
	s.glspServer = server.NewServer(customHandler, "design-tokens-language-server", true)

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
	s.configMu.RLock()
	defer s.configMu.RUnlock()
	return s.rootURI
}

// RootPath returns the workspace root path
func (s *Server) RootPath() string {
	s.configMu.RLock()
	defer s.configMu.RUnlock()
	return s.rootPath
}

// SetRootURI sets the workspace root URI
func (s *Server) SetRootURI(uri string) {
	s.configMu.Lock()
	defer s.configMu.Unlock()
	s.rootURI = uri
}

// SetRootPath sets the workspace root path
func (s *Server) SetRootPath(path string) {
	s.configMu.Lock()
	defer s.configMu.Unlock()
	s.rootPath = path
}

// GLSPContext returns the GLSP context.
// Access is protected by configMu to prevent concurrent races.
func (s *Server) GLSPContext() *glsp.Context {
	s.configMu.RLock()
	defer s.configMu.RUnlock()
	return s.context
}

// SetGLSPContext sets the GLSP context.
// Access is protected by configMu to prevent concurrent races.
func (s *Server) SetGLSPContext(ctx *glsp.Context) {
	s.configMu.Lock()
	defer s.configMu.Unlock()
	s.context = ctx
}

// UsePullDiagnostics returns whether the client supports pull diagnostics (LSP 3.17)
// If true, the server should NOT send push diagnostics (textDocument/publishDiagnostics)
// and instead wait for the client to request diagnostics via textDocument/diagnostic
func (s *Server) UsePullDiagnostics() bool {
	s.configMu.RLock()
	defer s.configMu.RUnlock()
	return s.usePullDiagnostics
}

// SetUsePullDiagnostics sets whether to use pull diagnostics based on client capabilities
func (s *Server) SetUsePullDiagnostics(use bool) {
	s.configMu.Lock()
	defer s.configMu.Unlock()
	s.usePullDiagnostics = use
}

// PublishDiagnostics publishes diagnostics for a document
func (s *Server) PublishDiagnostics(context *glsp.Context, uri string) error {
	fmt.Fprintf(os.Stderr, "[DTLS] Publishing diagnostics for: %s\n", uri)

	// Select a working context: use passed-in context if non-nil, otherwise fall back to server's context
	workingContext := context
	if workingContext == nil {
		workingContext = s.context
	}

	// If we still don't have a context, fail fast
	if workingContext == nil {
		return fmt.Errorf("cannot publish diagnostics: no client context available")
	}

	diagnostics, err := diagnostic.GetDiagnostics(s, uri)
	if err != nil {
		return err
	}

	// Publish diagnostics to the client using the selected context
	workingContext.Notify(protocol.ServerTextDocumentPublishDiagnostics, protocol.PublishDiagnosticsParams{
		URI:         uri,
		Diagnostics: diagnostics,
	})

	return nil
}

// IsTokenFile checks if a file path is one of our token files
func (s *Server) IsTokenFile(path string) bool {
	// Check if it's a JSON or YAML file
	ext := filepath.Ext(path)
	if ext != ".json" && ext != ".yaml" && ext != ".yml" {
		return false
	}

	// Normalize input path early for consistent comparisons
	cleanPath := filepath.Clean(path)

	// Check if it's in our loaded files map (for programmatically loaded tokens)
	s.loadedFilesMu.RLock()
	_, exists := s.loadedFiles[cleanPath]
	s.loadedFilesMu.RUnlock()
	if exists {
		return true
	}

	// Get config and state snapshots for thread-safe access
	cfg := s.GetConfig()
	state := s.GetState()

	// Check if it matches any of our configured token files
	for _, item := range cfg.TokensFiles {
		var tokenPath string
		switch v := item.(type) {
		case string:
			tokenPath = v
		case map[string]any:
			if pathVal, ok := v["path"]; ok {
				tokenPath, _ = pathVal.(string)
			}
		}

		if tokenPath == "" {
			continue
		}

		// Resolve relative paths
		if state.RootPath != "" && !filepath.IsAbs(tokenPath) {
			tokenPath = filepath.Join(state.RootPath, tokenPath)
		}

		// Normalize token path before comparison
		cleanTokenPath := filepath.Clean(tokenPath)

		// Check if the paths match
		if cleanPath == cleanTokenPath {
			return true
		}
	}

	// Not in loadedFiles: this is not a tracked token file
	return false
}

// RemoveLoadedFile removes a file from the loaded files tracking map
// This should be called when a token file is deleted to prevent stale entries
func (s *Server) RemoveLoadedFile(path string) {
	// Normalize path to match keys used during insertion
	cleanPath := filepath.Clean(path)

	s.loadedFilesMu.Lock()
	delete(s.loadedFiles, cleanPath)
	s.loadedFilesMu.Unlock()
}

// RegisterFileWatchers registers file watchers with the client
func (s *Server) RegisterFileWatchers(context *glsp.Context) error {
	// Guard against nil or empty context (can happen in tests without real LSP connection)
	// An empty context (created with &glsp.Context{}) won't have Call initialized
	if context == nil || context.Call == nil {
		fmt.Fprintf(os.Stderr, "[DTLS] Skipping file watcher registration (no client context)\n")
		return nil
	}

	// Get config and state snapshots for thread-safe access
	cfg := s.GetConfig()
	state := s.GetState()

	// Build file watchers for configured token files
	watchers := []protocol.FileSystemWatcher{}

	if len(cfg.TokensFiles) > 0 {
		for _, item := range cfg.TokensFiles {
			var tokenPath string
			switch v := item.(type) {
			case string:
				tokenPath = v
			case map[string]any:
				if pathVal, ok := v["path"]; ok {
					tokenPath, _ = pathVal.(string)
				}
			}

			if tokenPath == "" {
				continue
			}

			// Convert to filesystem path pattern (forward-slash separated)
			// Glob patterns use filesystem paths, not URIs
			var pattern string
			if filepath.IsAbs(tokenPath) {
				// Absolute path: convert to forward slashes
				pattern = filepath.ToSlash(filepath.Clean(tokenPath))
			} else if state.RootPath != "" {
				// Relative path: join with root and convert to forward slashes
				absPath := filepath.Join(state.RootPath, tokenPath)
				pattern = filepath.ToSlash(filepath.Clean(absPath))
			} else {
				// No root path: keep relative, convert to forward slashes
				pattern = filepath.ToSlash(tokenPath)
			}

			watchers = append(watchers, protocol.FileSystemWatcher{
				GlobPattern: pattern,
			})
		}
	}

	if len(watchers) == 0 {
		fmt.Fprintf(os.Stderr, "[DTLS] No file watchers to register\n")
		return nil
	}

	// Register the watchers with the client
	params := protocol.RegistrationParams{
		Registrations: []protocol.Registration{
			{
				ID:     "design-tokens-file-watcher",
				Method: "workspace/didChangeWatchedFiles",
				RegisterOptions: protocol.DidChangeWatchedFilesRegistrationOptions{
					Watchers: watchers,
				},
			},
		},
	}

	// Send registration request to client
	// Note: client/registerCapability is a request (not notification) per LSP spec.
	// We use context.Call instead of context.Notify to properly send a request.
	//
	// IMPORTANT: We must call this in a goroutine to avoid blocking the main message
	// handler loop. If we call context.Call synchronously, the server cannot read the
	// client's response because the message handler is blocked waiting for it (deadlock).
	//
	// Error handling note: glsp.Context.Call doesn't return errors - the underlying
	// jsonrpc2.Conn.Call errors are caught and logged by the glsp wrapper
	// (see github.com/tliron/glsp@v0.2.2/server/handle.go:24-28).
	// If the client rejects the registration, the error response will be logged
	// to stderr by the glsp library. Since client capability registration failures
	// are not fatal (the client continues working, just without file watching),
	// this fire-and-forget approach with logging is acceptable.
	go func(ctx *glsp.Context) {
		var result any
		ctx.Call("client/registerCapability", params, &result)
		fmt.Fprintf(os.Stderr, "[DTLS] File watcher registration completed\n")
	}(context)

	fmt.Fprintf(os.Stderr, "[DTLS] Sent file watcher registration request (%d watchers)\n", len(watchers))
	return nil
}
