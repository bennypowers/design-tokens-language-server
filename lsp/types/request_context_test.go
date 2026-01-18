package types

import (
	"errors"
	"testing"

	"bennypowers.dev/dtls/internal/documents"
	"bennypowers.dev/dtls/internal/tokens"
	"github.com/stretchr/testify/assert"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func TestRequestContext_AddWarning(t *testing.T) {
	mockServer := NewMockServerContextForTest()
	glspCtx := &glsp.Context{Method: "test"}
	req := NewRequestContext(mockServer, glspCtx)

	// Should start with no warnings
	assert.False(t, req.HasWarnings())
	assert.Nil(t, req.Warnings())

	// Add warnings
	err1 := errors.New("warning 1")
	err2 := errors.New("warning 2")
	req.AddWarning(err1)
	req.AddWarning(err2)

	// Should have warnings
	assert.True(t, req.HasWarnings())
	warnings := req.Warnings()
	assert.Len(t, warnings, 2)
	assert.Equal(t, err1, warnings[0])
	assert.Equal(t, err2, warnings[1])
}

func TestRequestContext_AddWarning_Nil(t *testing.T) {
	req := NewRequestContext(nil, nil)

	// Adding nil should be ignored
	req.AddWarning(nil)

	assert.False(t, req.HasWarnings())
}

func TestRequestContext_ContextAccess(t *testing.T) {
	mockServer := NewMockServerContextForTest()
	glspCtx := &glsp.Context{Method: "testMethod"}
	req := NewRequestContext(mockServer, glspCtx)

	// Should be able to access both contexts
	assert.Equal(t, mockServer, req.Server)
	assert.Equal(t, glspCtx, req.GLSP)
	assert.Equal(t, "testMethod", req.GLSP.Method)
}

// Helper to create mock for these tests
func NewMockServerContextForTest() *mockServerContextMinimal {
	return &mockServerContextMinimal{}
}

// mockServerContextMinimal is a minimal mock just for request context tests.
// It provides stub implementations for all ServerContext methods.
type mockServerContextMinimal struct {
	// cache holds a stable semantic token cache instance, lazily initialized on first access.
	// This ensures consistent behavior across multiple SemanticTokenCache() calls.
	cache SemanticTokenCacher
}

func (m *mockServerContextMinimal) Document(uri string) *documents.Document      { return nil }
func (m *mockServerContextMinimal) DocumentManager() *documents.Manager          { return nil }
func (m *mockServerContextMinimal) AllDocuments() []*documents.Document          { return nil }
func (m *mockServerContextMinimal) Token(name string) *tokens.Token              { return nil }
func (m *mockServerContextMinimal) TokenManager() *tokens.Manager                { return nil }
func (m *mockServerContextMinimal) TokenCount() int                              { return 0 }
func (m *mockServerContextMinimal) RootURI() string                              { return "" }
func (m *mockServerContextMinimal) RootPath() string                             { return "" }
func (m *mockServerContextMinimal) SetRootURI(uri string)                        {}
func (m *mockServerContextMinimal) SetRootPath(path string)                      {}
func (m *mockServerContextMinimal) GetConfig() ServerConfig                      { return ServerConfig{} }
func (m *mockServerContextMinimal) SetConfig(config ServerConfig)                {}
func (m *mockServerContextMinimal) LoadPackageJsonConfig() error                 { return nil }
func (m *mockServerContextMinimal) IsTokenFile(path string) bool                 { return false }
func (m *mockServerContextMinimal) LoadTokensFromConfig() error                  { return nil }
func (m *mockServerContextMinimal) RegisterFileWatchers(ctx *glsp.Context) error { return nil }
func (m *mockServerContextMinimal) RemoveLoadedFile(path string)                 {}
func (m *mockServerContextMinimal) GLSPContext() *glsp.Context                   { return nil }
func (m *mockServerContextMinimal) SetGLSPContext(ctx *glsp.Context)             {}
func (m *mockServerContextMinimal) ClientDiagnosticCapability() *bool            { return nil }
func (m *mockServerContextMinimal) SetClientDiagnosticCapability(hasCapability bool) {}
func (m *mockServerContextMinimal) ClientCapabilities() *protocol.ClientCapabilities { return nil }
func (m *mockServerContextMinimal) SetClientCapabilities(caps protocol.ClientCapabilities) {}
func (m *mockServerContextMinimal) SupportsSnippets() bool { return false }
func (m *mockServerContextMinimal) PreferredHoverFormat() protocol.MarkupKind { return protocol.MarkupKindMarkdown }
func (m *mockServerContextMinimal) SupportsDefinitionLinks() bool { return false }
func (m *mockServerContextMinimal) SupportsDiagnosticRelatedInfo() bool { return false }
func (m *mockServerContextMinimal) PublishDiagnostics(context *glsp.Context, uri string) error {
	return nil
}
func (m *mockServerContextMinimal) UsePullDiagnostics() bool         { return false }
func (m *mockServerContextMinimal) SetUsePullDiagnostics(use bool)   {}
func (m *mockServerContextMinimal) AddWarning(err error)             {}
func (m *mockServerContextMinimal) TakeWarnings() []error            { return nil }
func (m *mockServerContextMinimal) ShouldProcessAsTokenFile(uri string) bool { return true }
func (m *mockServerContextMinimal) LoadTokensFromDocumentContent(uri, languageID, content string) error {
	return nil
}
func (m *mockServerContextMinimal) SemanticTokenCache() SemanticTokenCacher {
	if m.cache == nil {
		m.cache = &mockSemanticTokenCache{}
	}
	return m.cache
}

// mockSemanticTokenCache is a minimal mock for SemanticTokenCacher
type mockSemanticTokenCache struct{}

func (m *mockSemanticTokenCache) Store(uri string, data []uint32, version int) string  { return "" }
func (m *mockSemanticTokenCache) Get(resultID string) *SemanticTokenCacheEntry         { return nil }
func (m *mockSemanticTokenCache) GetForURI(resultID, uri string) *SemanticTokenCacheEntry { return nil }
func (m *mockSemanticTokenCache) GetByURI(uri string) *SemanticTokenCacheEntry         { return nil }
func (m *mockSemanticTokenCache) Invalidate(uri string)                                {}
