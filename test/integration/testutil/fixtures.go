package testutil

import (
	"os"
	"path/filepath"
	"testing"

	"bennypowers.dev/dtls/lsp"
	"bennypowers.dev/dtls/lsp/methods/textDocument"
	"bennypowers.dev/dtls/lsp/types"
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
	data, err := os.ReadFile(path) //nolint:gosec // G304: Test fixture path - test code only
	require.NoError(t, err, "Failed to load token fixture: %s", name)
	return data
}

// LoadCSSFixture loads a CSS fixture file and returns the content
func LoadCSSFixture(t *testing.T, name string) string {
	t.Helper()
	path := filepath.Join(FixtureRoot(), "css", name)
	data, err := os.ReadFile(path) //nolint:gosec // G304: Test fixture path - test code only
	require.NoError(t, err, "Failed to load CSS fixture: %s", name)
	return string(data)
}

// LoadGoldenFile loads a golden file for comparison
func LoadGoldenFile(t *testing.T, name string) string {
	t.Helper()
	path := filepath.Join(FixtureRoot(), "golden", name)
	data, err := os.ReadFile(path) //nolint:gosec // G304: Test fixture path - test code only
	require.NoError(t, err, "Failed to load golden file: %s", name)
	return string(data)
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

// languageIDFromURI determines the language ID based on the URI extension
func languageIDFromURI(uri string) string {
	if len(uri) > 5 && uri[len(uri)-5:] == ".yaml" {
		return "yaml"
	} else if len(uri) > 4 && uri[len(uri)-4:] == ".yml" {
		return "yaml"
	}
	return "json"
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
	req := types.NewRequestContext(server, nil)
	err := textDocument.DidOpen(req, params)
	require.NoError(t, err, "Failed to open CSS fixture: %s", fixtureName)
}

// OpenTokenFixture opens a token fixture file as a document in the server
// This is useful for testing references/definition from token files
func OpenTokenFixture(t *testing.T, server *lsp.Server, uri, fixtureName string) {
	t.Helper()
	content := LoadTokenFixture(t, fixtureName)

	params := &protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{
			URI:        uri,
			LanguageID: languageIDFromURI(uri),
			Version:    1,
			Text:       string(content),
		},
	}
	req := types.NewRequestContext(server, nil)
	err := textDocument.DidOpen(req, params)
	require.NoError(t, err, "Failed to open token fixture: %s", fixtureName)
}

// LoadNonTokenFixture loads a non-token file fixture and returns the bytes
func LoadNonTokenFixture(t *testing.T, name string) []byte {
	t.Helper()
	path := filepath.Join(FixtureRoot(), "non-token-files", name)
	data, err := os.ReadFile(path) //nolint:gosec // G304: Test fixture path - test code only
	require.NoError(t, err, "Failed to load non-token fixture: %s", name)
	return data
}

// OpenNonTokenFixture opens a non-token JSON/YAML fixture file as a document in the server
// This is for testing that LSP features are NOT provided for non-token files
func OpenNonTokenFixture(t *testing.T, server *lsp.Server, uri, fixtureName string) {
	t.Helper()
	content := LoadNonTokenFixture(t, fixtureName)

	params := &protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{
			URI:        uri,
			LanguageID: languageIDFromURI(uri),
			Version:    1,
			Text:       string(content),
		},
	}
	req := types.NewRequestContext(server, nil)
	err := textDocument.DidOpen(req, params)
	require.NoError(t, err, "Failed to open non-token fixture: %s", fixtureName)
}
