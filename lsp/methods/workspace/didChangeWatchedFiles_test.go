package workspace

import (
	"testing"

	"github.com/bennypowers/design-tokens-language-server/internal/documents"
	"github.com/bennypowers/design-tokens-language-server/internal/tokens"
	"github.com/bennypowers/design-tokens-language-server/lsp/types"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// mockServerContext implements types.ServerContext for testing
type mockServerContext struct {
	docs        *documents.Manager
	tokens      *tokens.Manager
	rootPath    string
	config      types.ServerConfig
	loadedFiles map[string]string
}

func (m *mockServerContext) Document(uri string) *documents.Document {
	return m.docs.Get(uri)
}

func (m *mockServerContext) DocumentManager() *documents.Manager {
	return m.docs
}

func (m *mockServerContext) AllDocuments() []*documents.Document {
	return m.docs.GetAll()
}

func (m *mockServerContext) Token(name string) *tokens.Token {
	return m.tokens.Get(name)
}

func (m *mockServerContext) TokenManager() *tokens.Manager {
	return m.tokens
}

func (m *mockServerContext) TokenCount() int {
	return m.tokens.Count()
}

func (m *mockServerContext) RootURI() string {
	return "file://" + m.rootPath
}

func (m *mockServerContext) RootPath() string {
	return m.rootPath
}

func (m *mockServerContext) SetRootURI(uri string) {}

func (m *mockServerContext) SetRootPath(path string) {
	m.rootPath = path
}

func (m *mockServerContext) GetConfig() types.ServerConfig {
	return m.config
}

func (m *mockServerContext) SetConfig(config types.ServerConfig) {
	m.config = config
}

func (m *mockServerContext) IsTokenFile(path string) bool {
	// Simple implementation for testing
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

func (m *mockServerContext) LoadTokensFromConfig() error {
	return nil
}

func (m *mockServerContext) RegisterFileWatchers(ctx *glsp.Context) error {
	return nil
}

func (m *mockServerContext) GLSPContext() *glsp.Context {
	return nil
}

func (m *mockServerContext) SetGLSPContext(ctx *glsp.Context) {}

func (m *mockServerContext) PublishDiagnostics(context *glsp.Context, uri string) error {
	return nil
}

func newMockServerContext() *mockServerContext {
	return &mockServerContext{
		docs:        documents.NewManager(),
		tokens:      tokens.NewManager(),
		config:      types.DefaultConfig(),
		loadedFiles: make(map[string]string),
	}
}

func TestHandleDidChangeWatchedFiles(t *testing.T) {
	ctx := newMockServerContext()
	ctx.rootPath = "/workspace"
	ctx.config.TokensFiles = []any{"tokens.json"}

	// Create a change event
	params := &protocol.DidChangeWatchedFilesParams{
		Changes: []protocol.FileEvent{
			{
				URI:  "file:///workspace/tokens.json",
				Type: protocol.FileChangeTypeChanged,
			},
		},
	}

	// Handle the change
	err := DidChangeWatchedFiles(ctx, nil, params)
	if err != nil {
		t.Errorf("DidChangeWatchedFiles failed: %v", err)
	}
}

func TestHandleDidChangeWatchedFiles_MultipleChanges(t *testing.T) {
	ctx := newMockServerContext()
	ctx.rootPath = "/workspace"
	ctx.config.TokensFiles = []any{"tokens.json", "design-tokens.json"}

	params := &protocol.DidChangeWatchedFilesParams{
		Changes: []protocol.FileEvent{
			{
				URI:  "file:///workspace/tokens.json",
				Type: protocol.FileChangeTypeChanged,
			},
			{
				URI:  "file:///workspace/design-tokens.json",
				Type: protocol.FileChangeTypeChanged,
			},
			{
				URI:  "file:///workspace/package.json", // Not a token file
				Type: protocol.FileChangeTypeChanged,
			},
		},
	}

	err := DidChangeWatchedFiles(ctx, nil, params)
	if err != nil {
		t.Errorf("DidChangeWatchedFiles failed: %v", err)
	}
}

func TestHandleDidChangeWatchedFiles_DeletedFile(t *testing.T) {
	ctx := newMockServerContext()
	ctx.rootPath = "/workspace"
	ctx.config.TokensFiles = []any{"tokens.json"}

	params := &protocol.DidChangeWatchedFilesParams{
		Changes: []protocol.FileEvent{
			{
				URI:  "file:///workspace/tokens.json",
				Type: protocol.FileChangeTypeDeleted,
			},
		},
	}

	err := DidChangeWatchedFiles(ctx, nil, params)
	if err != nil {
		t.Errorf("DidChangeWatchedFiles failed: %v", err)
	}
}

func TestHandleDidChangeWatchedFiles_NonTokenFile(t *testing.T) {
	ctx := newMockServerContext()
	ctx.rootPath = "/workspace"
	ctx.config.TokensFiles = []any{"tokens.json"}

	params := &protocol.DidChangeWatchedFilesParams{
		Changes: []protocol.FileEvent{
			{
				URI:  "file:///workspace/package.json", // Not a token file
				Type: protocol.FileChangeTypeChanged,
			},
		},
	}

	// Should not trigger a reload since it's not a token file
	err := DidChangeWatchedFiles(ctx, nil, params)
	if err != nil {
		t.Errorf("DidChangeWatchedFiles failed: %v", err)
	}
}
