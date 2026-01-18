package testutil

import (
	"path/filepath"

	"bennypowers.dev/dtls/internal/documents"
	"bennypowers.dev/dtls/internal/tokens"
	semantictokens "bennypowers.dev/dtls/lsp/methods/textDocument/semanticTokens"
	"bennypowers.dev/dtls/lsp/types"
	"github.com/tliron/glsp"
)

// MockServerContext implements types.ServerContext for testing.
// It provides a minimal implementation with configurable behavior via callback functions.
type MockServerContext struct {
	docs        *documents.Manager
	tokens      *tokens.Manager
	rootURI     string
	rootPath    string
	config                     types.ServerConfig
	loadedFiles                map[string]string
	glspContext                *glsp.Context
	clientDiagnosticCapability *bool
	usePullDiagnostics         bool
	semanticTokenCache         *semantictokens.TokenCache

	// Optional callbacks for custom behavior in tests
	LoadTokensFunc                func() error
	RegisterWatchersFunc          func(*glsp.Context) error
	IsTokenFileFunc               func(string) bool
	ShouldProcessAsTokenFileFunc  func(string) bool
	PublishDiagnosticsFunc        func(*glsp.Context, string) error

	// Tracking flags for tests that need to verify methods were called
	LoadTokensCalled       bool
	RegisterWatchersCalled bool
}

// NewMockServerContext creates a new mock server context with default behavior
func NewMockServerContext() *MockServerContext {
	return &MockServerContext{
		docs:               documents.NewManager(),
		tokens:             tokens.NewManager(),
		config:             types.DefaultConfig(),
		loadedFiles:        make(map[string]string),
		rootURI:            "",
		rootPath:           "",
		semanticTokenCache: semantictokens.NewTokenCache(),
	}
}

// Document returns the document with the given URI
func (m *MockServerContext) Document(uri string) *documents.Document {
	return m.docs.Get(uri)
}

// DocumentManager returns the document manager
func (m *MockServerContext) DocumentManager() *documents.Manager {
	return m.docs
}

// AllDocuments returns all tracked documents
func (m *MockServerContext) AllDocuments() []*documents.Document {
	return m.docs.GetAll()
}

// Token returns the token with the given name
func (m *MockServerContext) Token(name string) *tokens.Token {
	return m.tokens.Get(name)
}

// TokenManager returns the token manager
func (m *MockServerContext) TokenManager() *tokens.Manager {
	return m.tokens
}

// TokenCount returns the number of tokens
func (m *MockServerContext) TokenCount() int {
	return m.tokens.Count()
}

// RootURI returns the workspace root URI
func (m *MockServerContext) RootURI() string {
	return m.rootURI
}

// RootPath returns the workspace root path
func (m *MockServerContext) RootPath() string {
	return m.rootPath
}

// SetRootURI sets the workspace root URI
func (m *MockServerContext) SetRootURI(uri string) {
	m.rootURI = uri
}

// SetRootPath sets the workspace root path
func (m *MockServerContext) SetRootPath(path string) {
	m.rootPath = path
}

// GetConfig returns the server configuration
func (m *MockServerContext) GetConfig() types.ServerConfig {
	return m.config
}

// SetConfig sets the server configuration
func (m *MockServerContext) SetConfig(config types.ServerConfig) {
	m.config = config
}

// IsTokenFile checks if a file path is a token file
func (m *MockServerContext) IsTokenFile(path string) bool {
	if m.IsTokenFileFunc != nil {
		return m.IsTokenFileFunc(path)
	}

	// Default implementation: check loadedFiles and config
	// Normalize path to match production code behavior
	cleanPath := filepath.Clean(path)
	if _, exists := m.loadedFiles[cleanPath]; exists {
		return true
	}

	for _, item := range m.config.TokensFiles {
		// Handle string entries
		if str, ok := item.(string); ok {
			if str == path {
				return true
			}
		}
		// Handle object-style entries like {"path": "..."}
		if obj, ok := item.(map[string]any); ok {
			if pathVal, exists := obj["path"]; exists {
				if pathStr, ok := pathVal.(string); ok && pathStr == path {
					return true
				}
			}
		}
	}

	return false
}

// ShouldProcessAsTokenFile checks if a document should receive token file features
func (m *MockServerContext) ShouldProcessAsTokenFile(uri string) bool {
	if m.ShouldProcessAsTokenFileFunc != nil {
		return m.ShouldProcessAsTokenFileFunc(uri)
	}

	// Default implementation: always return true for mock (tests expect features to work)
	return true
}

// LoadTokensFromConfig loads tokens from configuration
func (m *MockServerContext) LoadTokensFromConfig() error {
	m.LoadTokensCalled = true
	if m.LoadTokensFunc != nil {
		return m.LoadTokensFunc()
	}
	return nil
}

// LoadPackageJsonConfig loads configuration from package.json (mock stub)
func (m *MockServerContext) LoadPackageJsonConfig() error {
	// Mock implementation - does nothing by default
	return nil
}

// RegisterFileWatchers registers file watchers with the client
func (m *MockServerContext) RegisterFileWatchers(ctx *glsp.Context) error {
	m.RegisterWatchersCalled = true
	if m.RegisterWatchersFunc != nil {
		return m.RegisterWatchersFunc(ctx)
	}
	return nil
}

// LoadTokensFromDocumentContent loads tokens from document content (no-op for mock)
func (m *MockServerContext) LoadTokensFromDocumentContent(uri, languageID, content string) error {
	return nil
}

// RemoveLoadedFile removes a file from the loaded files tracking map
func (m *MockServerContext) RemoveLoadedFile(path string) {
	// Normalize path to match production code behavior
	cleanPath := filepath.Clean(path)
	delete(m.loadedFiles, cleanPath)
}

// GLSPContext returns the GLSP context
func (m *MockServerContext) GLSPContext() *glsp.Context {
	return m.glspContext
}

// SetGLSPContext sets the GLSP context
func (m *MockServerContext) SetGLSPContext(ctx *glsp.Context) {
	m.glspContext = ctx
}

// ClientDiagnosticCapability returns the detected client diagnostic capability
func (m *MockServerContext) ClientDiagnosticCapability() *bool {
	return m.clientDiagnosticCapability
}

// SetClientDiagnosticCapability sets the client's diagnostic capability
func (m *MockServerContext) SetClientDiagnosticCapability(hasCapability bool) {
	m.clientDiagnosticCapability = &hasCapability
}

// PublishDiagnostics publishes diagnostics for a document
func (m *MockServerContext) PublishDiagnostics(context *glsp.Context, uri string) error {
	if m.PublishDiagnosticsFunc != nil {
		return m.PublishDiagnosticsFunc(context, uri)
	}
	return nil
}

// UsePullDiagnostics returns whether to use pull diagnostics (LSP 3.17)
func (m *MockServerContext) UsePullDiagnostics() bool {
	return m.usePullDiagnostics
}

// SetUsePullDiagnostics sets whether to use pull diagnostics
func (m *MockServerContext) SetUsePullDiagnostics(use bool) {
	m.usePullDiagnostics = use
}

// SemanticTokenCache returns the semantic tokens cache for delta support
func (m *MockServerContext) SemanticTokenCache() types.SemanticTokenCacher {
	return m.semanticTokenCache
}

// AddDocument adds a document to the manager
func (m *MockServerContext) AddDocument(doc *documents.Document) {
	_ = m.docs.DidOpen(doc.URI(), doc.LanguageID(), doc.Version(), doc.Content())
}

// AddToken adds a token to the manager
func (m *MockServerContext) AddToken(token *tokens.Token) {
	m.tokens.Add(token)
}

// NewMockServer is an alias for NewMockServerContext for convenience
func NewMockServer() *MockServerContext {
	return NewMockServerContext()
}
