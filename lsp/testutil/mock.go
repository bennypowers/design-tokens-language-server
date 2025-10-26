package testutil

import (
	"github.com/bennypowers/design-tokens-language-server/internal/documents"
	"github.com/bennypowers/design-tokens-language-server/internal/tokens"
	"github.com/bennypowers/design-tokens-language-server/lsp/types"
	"github.com/tliron/glsp"
)

// MockServerContext implements types.ServerContext for testing.
// It provides a minimal implementation with configurable behavior via callback functions.
type MockServerContext struct {
	docs        *documents.Manager
	tokens      *tokens.Manager
	rootURI     string
	rootPath    string
	config      types.ServerConfig
	loadedFiles map[string]string
	glspContext *glsp.Context

	// Optional callbacks for custom behavior in tests
	LoadTokensFunc        func() error
	RegisterWatchersFunc  func(*glsp.Context) error
	IsTokenFileFunc       func(string) bool
	PublishDiagnosticsFunc func(*glsp.Context, string) error

	// Tracking flags for tests that need to verify methods were called
	LoadTokensCalled       bool
	RegisterWatchersCalled bool
}

// NewMockServerContext creates a new mock server context with default behavior
func NewMockServerContext() *MockServerContext {
	return &MockServerContext{
		docs:        documents.NewManager(),
		tokens:      tokens.NewManager(),
		config:      types.DefaultConfig(),
		loadedFiles: make(map[string]string),
		rootURI:     "",
		rootPath:    "",
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
	if _, exists := m.loadedFiles[path]; exists {
		return true
	}

	for _, item := range m.config.TokensFiles {
		if str, ok := item.(string); ok {
			if str == path {
				return true
			}
		}
	}

	return false
}

// LoadTokensFromConfig loads tokens from configuration
func (m *MockServerContext) LoadTokensFromConfig() error {
	m.LoadTokensCalled = true
	if m.LoadTokensFunc != nil {
		return m.LoadTokensFunc()
	}
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

// GLSPContext returns the GLSP context
func (m *MockServerContext) GLSPContext() *glsp.Context {
	return m.glspContext
}

// SetGLSPContext sets the GLSP context
func (m *MockServerContext) SetGLSPContext(ctx *glsp.Context) {
	m.glspContext = ctx
}

// PublishDiagnostics publishes diagnostics for a document
func (m *MockServerContext) PublishDiagnostics(context *glsp.Context, uri string) error {
	if m.PublishDiagnosticsFunc != nil {
		return m.PublishDiagnosticsFunc(context, uri)
	}
	return nil
}
