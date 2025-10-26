package lifecycle

import (
	"testing"

	"github.com/bennypowers/design-tokens-language-server/internal/documents"
	"github.com/bennypowers/design-tokens-language-server/internal/tokens"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// mockServerContext implements types.ServerContext for testing
type mockServerContext struct {
	rootURI  string
	rootPath string
	docs     *documents.Manager
	tokens   *tokens.Manager
	context  *glsp.Context
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
	return m.rootURI
}

func (m *mockServerContext) RootPath() string {
	return m.rootPath
}

func (m *mockServerContext) SetRootURI(uri string) {
	m.rootURI = uri
}

func (m *mockServerContext) SetRootPath(path string) {
	m.rootPath = path
}

func (m *mockServerContext) LoadTokensFromConfig() error {
	return nil
}

func (m *mockServerContext) RegisterFileWatchers(ctx *glsp.Context) error {
	return nil
}

func (m *mockServerContext) GLSPContext() *glsp.Context {
	return m.context
}

func (m *mockServerContext) SetGLSPContext(ctx *glsp.Context) {
	m.context = ctx
}

func (m *mockServerContext) PublishDiagnostics(context *glsp.Context, uri string) error {
	return nil
}

func newMockServerContext() *mockServerContext {
	return &mockServerContext{
		docs:   documents.NewManager(),
		tokens: tokens.NewManager(),
	}
}

func TestInitialize(t *testing.T) {
	t.Run("sets root URI from params.RootURI", func(t *testing.T) {
		ctx := newMockServerContext()
		glspCtx := &glsp.Context{}
		rootURI := "file:///workspace"

		params := &protocol.InitializeParams{
			RootURI: &rootURI,
		}

		result, err := Initialize(ctx, glspCtx, params)
		require.NoError(t, err)
		require.NotNil(t, result)

		// Verify root was set
		assert.Equal(t, "file:///workspace", ctx.RootURI())
		assert.Equal(t, "/workspace", ctx.RootPath())
	})

	t.Run("sets root path from params.RootPath", func(t *testing.T) {
		ctx := newMockServerContext()
		glspCtx := &glsp.Context{}
		rootPath := "/workspace"

		params := &protocol.InitializeParams{
			RootPath: &rootPath,
		}

		result, err := Initialize(ctx, glspCtx, params)
		require.NoError(t, err)
		require.NotNil(t, result)

		// Verify root was set
		assert.Equal(t, "/workspace", ctx.RootPath())
		assert.Equal(t, "file:///workspace", ctx.RootURI())
	})

	t.Run("returns server capabilities", func(t *testing.T) {
		ctx := newMockServerContext()
		glspCtx := &glsp.Context{}

		params := &protocol.InitializeParams{}

		result, err := Initialize(ctx, glspCtx, params)
		require.NoError(t, err)
		require.NotNil(t, result)

		// Result should have Capabilities and ServerInfo fields
		initResult := result.(struct {
			Capabilities interface{}                           `json:"capabilities"`
			ServerInfo   *protocol.InitializeResultServerInfo `json:"serverInfo,omitempty"`
		})

		assert.NotNil(t, initResult.Capabilities)
		assert.NotNil(t, initResult.ServerInfo)
		assert.Equal(t, "design-tokens-language-server", initResult.ServerInfo.Name)
	})

	t.Run("capabilities include all LSP features", func(t *testing.T) {
		ctx := newMockServerContext()
		glspCtx := &glsp.Context{}

		params := &protocol.InitializeParams{}

		result, err := Initialize(ctx, glspCtx, params)
		require.NoError(t, err)

		initResult := result.(struct {
			Capabilities interface{}                           `json:"capabilities"`
			ServerInfo   *protocol.InitializeResultServerInfo `json:"serverInfo,omitempty"`
		})

		caps, ok := initResult.Capabilities.(map[string]interface{})
		require.True(t, ok, "Capabilities should be a map")

		// Verify all expected capabilities are present
		assert.Contains(t, caps, "textDocumentSync")
		assert.Contains(t, caps, "hoverProvider")
		assert.Contains(t, caps, "completionProvider")
		assert.Contains(t, caps, "definitionProvider")
		assert.Contains(t, caps, "referencesProvider")
		assert.Contains(t, caps, "codeActionProvider")
		assert.Contains(t, caps, "colorProvider")
		assert.Contains(t, caps, "semanticTokensProvider")
		assert.Contains(t, caps, "diagnosticProvider")

		// Verify resolve providers are enabled
		completionProvider, ok := caps["completionProvider"].(protocol.CompletionOptions)
		assert.True(t, ok)
		assert.NotNil(t, completionProvider.ResolveProvider)
		assert.True(t, *completionProvider.ResolveProvider)

		codeActionProvider, ok := caps["codeActionProvider"].(protocol.CodeActionOptions)
		assert.True(t, ok)
		assert.NotNil(t, codeActionProvider.ResolveProvider)
		assert.True(t, *codeActionProvider.ResolveProvider)
	})

	t.Run("handles client info", func(t *testing.T) {
		ctx := newMockServerContext()
		glspCtx := &glsp.Context{}

		clientVersion := "1.85.0"
		params := &protocol.InitializeParams{
			ClientInfo: &struct {
				Name    string  `json:"name"`
				Version *string `json:"version,omitempty"`
			}{
				Name:    "vscode",
				Version: &clientVersion,
			},
		}

		result, err := Initialize(ctx, glspCtx, params)
		require.NoError(t, err)
		require.NotNil(t, result)
	})

	t.Run("handles nil params gracefully", func(t *testing.T) {
		ctx := newMockServerContext()
		glspCtx := &glsp.Context{}

		params := &protocol.InitializeParams{}

		result, err := Initialize(ctx, glspCtx, params)
		require.NoError(t, err)
		require.NotNil(t, result)

		// Should still return valid capabilities
		assert.Empty(t, ctx.RootURI())
		assert.Empty(t, ctx.RootPath())
	})
}

func TestPathConversion(t *testing.T) {
	t.Run("uriToPath strips file:// prefix", func(t *testing.T) {
		tests := []struct {
			name string
			uri  string
			want string
		}{
			{
				name: "simple path",
				uri:  "file:///workspace",
				want: "/workspace",
			},
			{
				name: "nested path",
				uri:  "file:///home/user/project",
				want: "/home/user/project",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got := uriToPath(tt.uri)
				assert.Equal(t, tt.want, got)
			})
		}
	})

	t.Run("pathToURI adds file:// prefix", func(t *testing.T) {
		tests := []struct {
			name string
			path string
			want string
		}{
			{
				name: "simple path",
				path: "/workspace",
				want: "file:///workspace",
			},
			{
				name: "nested path",
				path: "/home/user/project",
				want: "file:///home/user/project",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got := pathToURI(tt.path)
				assert.Equal(t, tt.want, got)
			})
		}
	})

	t.Run("round trip conversion", func(t *testing.T) {
		paths := []string{
			"/workspace",
			"/home/user/project",
		}

		for _, path := range paths {
			uri := pathToURI(path)
			got := uriToPath(uri)
			assert.Equal(t, path, got, "round trip should preserve path")
		}
	})
}
