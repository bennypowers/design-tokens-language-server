package lsp

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

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
	documents         *documents.Manager
	tokens            *tokens.Manager
	glspServer        *server.Server
	context           *glsp.Context
	rootURI           string                // Workspace root URI
	rootPath          string                // Workspace root path (file system)
	config            types.ServerConfig    // Server configuration
	loadedFiles       map[string]string     // Track loaded files: filepath -> prefix
	autoDiscoveryMode bool                  // True if using auto-discovery instead of explicit files
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

// IsTokenFile checks if a file path is one of our token files
func (s *Server) IsTokenFile(path string) bool {
	// Check if it's a JSON or YAML file
	ext := filepath.Ext(path)
	if ext != ".json" && ext != ".yaml" && ext != ".yml" {
		return false
	}

	// Check if it's in our loaded files map (for programmatically loaded tokens)
	if _, exists := s.loadedFiles[path]; exists {
		return true
	}

	// Check if it matches any of our configured token files
	for _, item := range s.config.TokensFiles {
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
		if s.rootPath != "" && !filepath.IsAbs(tokenPath) {
			tokenPath = filepath.Join(s.rootPath, tokenPath)
		}

		// Normalize both paths before comparison to handle redundant separators
		// and relative components (e.g., /foo//bar vs /foo/bar, /foo/./bar vs /foo/bar)
		cleanPath := filepath.Clean(path)
		cleanTokenPath := filepath.Clean(tokenPath)

		// Check if the paths match
		if cleanPath == cleanTokenPath {
			return true
		}
	}

	// If we're in auto-discover mode, check common patterns
	if len(s.config.TokensFiles) == 0 {
		filename := filepath.Base(path)
		if filename == "tokens.json" ||
			strings.HasSuffix(filename, ".tokens.json") ||
			filename == "design-tokens.json" ||
			filename == "tokens.yaml" ||
			strings.HasSuffix(filename, ".tokens.yaml") ||
			filename == "design-tokens.yaml" ||
			filename == "tokens.yml" ||
			strings.HasSuffix(filename, ".tokens.yml") ||
			filename == "design-tokens.yml" {
			return true
		}
	}

	return false
}

// RemoveLoadedFile removes a file from the loaded files tracking map
// This should be called when a token file is deleted to prevent stale entries
func (s *Server) RemoveLoadedFile(path string) {
	delete(s.loadedFiles, path)
}

// RegisterFileWatchers registers file watchers with the client
func (s *Server) RegisterFileWatchers(context *glsp.Context) error {
	// Guard against nil context (can happen in tests without real LSP connection)
	if context == nil {
		fmt.Fprintf(os.Stderr, "[DTLS] Skipping file watcher registration (no client context)\n")
		return nil
	}

	// Build list of watchers based on configuration
	watchers := []protocol.FileSystemWatcher{}

	if len(s.config.TokensFiles) > 0 {
		// Watch explicitly configured files
		for _, item := range s.config.TokensFiles {
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
			} else if s.rootPath != "" {
				// Relative path: join with root and convert to forward slashes
				absPath := filepath.Join(s.rootPath, tokenPath)
				pattern = filepath.ToSlash(filepath.Clean(absPath))
			} else {
				// No root path: keep relative, convert to forward slashes
				pattern = filepath.ToSlash(tokenPath)
			}

			watchers = append(watchers, protocol.FileSystemWatcher{
				GlobPattern: pattern,
			})
		}
	} else if s.rootPath != "" {
		// Auto-discover mode: watch common patterns
		// Convert root path to forward-slash separated filesystem path
		rootPattern := filepath.ToSlash(filepath.Clean(s.rootPath))
		watchers = append(watchers,
			protocol.FileSystemWatcher{
				GlobPattern: rootPattern + "/**/tokens.json",
			},
			protocol.FileSystemWatcher{
				GlobPattern: rootPattern + "/**/*.tokens.json",
			},
			protocol.FileSystemWatcher{
				GlobPattern: rootPattern + "/**/design-tokens.json",
			},
			protocol.FileSystemWatcher{
				GlobPattern: rootPattern + "/**/tokens.yaml",
			},
			protocol.FileSystemWatcher{
				GlobPattern: rootPattern + "/**/*.tokens.yaml",
			},
			protocol.FileSystemWatcher{
				GlobPattern: rootPattern + "/**/design-tokens.yaml",
			},
			protocol.FileSystemWatcher{
				GlobPattern: rootPattern + "/**/tokens.yml",
			},
			protocol.FileSystemWatcher{
				GlobPattern: rootPattern + "/**/*.tokens.yml",
			},
			protocol.FileSystemWatcher{
				GlobPattern: rootPattern + "/**/design-tokens.yml",
			},
		)
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
	go func() {
		var result interface{}
		context.Call("client/registerCapability", params, &result)
		fmt.Fprintf(os.Stderr, "[DTLS] File watcher registration completed\n")
	}()

	fmt.Fprintf(os.Stderr, "[DTLS] Sent file watcher registration request (%d watchers)\n", len(watchers))
	return nil
}
