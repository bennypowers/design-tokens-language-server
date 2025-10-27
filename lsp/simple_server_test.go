package lsp

import (
	"testing"

	"bennypowers.dev/dtls/internal/documents"
	"bennypowers.dev/dtls/internal/tokens"
	"bennypowers.dev/dtls/lsp/types"
	"github.com/stretchr/testify/assert"
)

// Simple tests for server methods that don't require complex setup

func TestServer_AllDocuments(t *testing.T) {
	server := &Server{
		documents:   documents.NewManager(),
		tokens:      tokens.NewManager(),
		loadedFiles: make(map[string]*TokenFileOptions),
	}

	// Add some documents
	_ = server.documents.DidOpen("file:///test1.css", "css", 1, ".button { }")
	_ = server.documents.DidOpen("file:///test2.css", "css", 1, ".link { }")

	all := server.AllDocuments()
	assert.Len(t, all, 2)
}

func TestServer_TokenManager(t *testing.T) {
	server := &Server{
		documents:   documents.NewManager(),
		tokens:      tokens.NewManager(),
		loadedFiles: make(map[string]*TokenFileOptions),
	}

	tm := server.TokenManager()
	assert.NotNil(t, tm)
}

func TestServer_TokenCount(t *testing.T) {
	server := &Server{
		documents:   documents.NewManager(),
		tokens:      tokens.NewManager(),
		loadedFiles: make(map[string]*TokenFileOptions),
	}

	// Add a token
	_ = server.tokens.Add(&tokens.Token{
		Name:  "color.primary",
		Value: "#ff0000",
		Type:  "color",
	})

	count := server.TokenCount()
	assert.Equal(t, 1, count)
}

func TestServer_GetSetConfig(t *testing.T) {
	server := &Server{
		documents:   documents.NewManager(),
		tokens:      tokens.NewManager(),
		loadedFiles: make(map[string]*TokenFileOptions),
	}

	// Set a config
	newConfig := types.ServerConfig{
		Prefix:      "--test",
		TokensFiles: []any{"tokens.json"},
	}
	server.SetConfig(newConfig)

	// Get the config back
	config := server.GetConfig()
	assert.Equal(t, "--test", config.Prefix)
}

func TestServer_RootPaths(t *testing.T) {
	server := &Server{
		documents:   documents.NewManager(),
		tokens:      tokens.NewManager(),
		loadedFiles: make(map[string]*TokenFileOptions),
	}

	server.SetRootURI("file:///workspace")
	assert.Equal(t, "file:///workspace", server.RootURI())

	server.SetRootPath("/workspace")
	assert.Equal(t, "/workspace", server.RootPath())
}
