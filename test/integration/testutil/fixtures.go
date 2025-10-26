package testutil

import (
	"os"
	"path/filepath"
	"testing"

	"bennypowers.dev/dtls/lsp"
	"bennypowers.dev/dtls/lsp/methods/textDocument"
	"github.com/stretchr/testify/require"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// FixtureRoot returns the path to the test fixtures directory
func FixtureRoot() string {
	return filepath.Join("..", "fixtures")
}

// LoadTokenFixture loads a token fixture file and returns the bytes
func LoadTokenFixture(t *testing.T, name string) []byte {
	t.Helper()
	path := filepath.Join(FixtureRoot(), "tokens", name)
	data, err := os.ReadFile(path)
	require.NoError(t, err, "Failed to load token fixture: %s", name)
	return data
}

// LoadCSSFixture loads a CSS fixture file and returns the content
func LoadCSSFixture(t *testing.T, name string) string {
	t.Helper()
	path := filepath.Join(FixtureRoot(), "css", name)
	data, err := os.ReadFile(path)
	require.NoError(t, err, "Failed to load CSS fixture: %s", name)
	return string(data)
}

// LoadGoldenFile loads a golden file for comparison
func LoadGoldenFile(t *testing.T, name string) string {
	t.Helper()
	path := filepath.Join(FixtureRoot(), "golden", name)
	data, err := os.ReadFile(path)
	require.NoError(t, err, "Failed to load golden file: %s", name)
	return string(data)
}

// UpdateGoldenFile updates a golden file with new content
// Set UPDATE_GOLDEN=1 environment variable to update golden files
func UpdateGoldenFile(t *testing.T, name, content string) {
	t.Helper()
	if os.Getenv("UPDATE_GOLDEN") != "1" {
		return
	}
	path := filepath.Join(FixtureRoot(), "golden", name)
	err := os.WriteFile(path, []byte(content), 0644)
	require.NoError(t, err, "Failed to update golden file: %s", name)
}

// NewTestServer creates a new LSP server for testing
func NewTestServer(t *testing.T) *lsp.Server {
	t.Helper()
	server, err := lsp.NewServer()
	require.NoError(t, err, "Failed to create test server")
	return server
}

// LoadBasicTokens loads the basic-colors.json fixture into the server
func LoadBasicTokens(t *testing.T, server *lsp.Server) {
	t.Helper()
	tokens := LoadTokenFixture(t, "basic-colors.json")
	err := server.LoadTokensFromJSON(tokens, "")
	require.NoError(t, err, "Failed to load basic tokens")
}

// LoadTokensWithPrefix loads the with-prefix.json fixture into the server
func LoadTokensWithPrefix(t *testing.T, server *lsp.Server, prefix string) {
	t.Helper()
	tokens := LoadTokenFixture(t, "with-prefix.json")
	err := server.LoadTokensFromJSON(tokens, prefix)
	require.NoError(t, err, "Failed to load tokens with prefix")
}

// OpenCSSFixture opens a CSS fixture file in the server
func OpenCSSFixture(t *testing.T, server *lsp.Server, uri, fixtureName string) {
	t.Helper()
	content := LoadCSSFixture(t, fixtureName)
	params := &protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{
			URI:        uri,
			LanguageID: "css",
			Version:    1,
			Text:       content,
		},
	}
	err := textDocument.DidOpen(server, nil, params)
	require.NoError(t, err, "Failed to open CSS fixture: %s", fixtureName)
}
