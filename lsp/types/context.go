package types

import (
	"github.com/bennypowers/design-tokens-language-server/internal/documents"
	"github.com/bennypowers/design-tokens-language-server/internal/tokens"
	"github.com/tliron/glsp"
)

// ServerContext provides all dependencies needed for LSP handlers.
// This unified context eliminates the need for handler-specific interfaces
// and enables dependency injection for testing.
type ServerContext interface {
	// Document operations
	Document(uri string) *documents.Document
	DocumentManager() *documents.Manager
	AllDocuments() []*documents.Document

	// Token operations
	Token(name string) *tokens.Token
	TokenManager() *tokens.Manager
	TokenCount() int

	// Workspace operations
	RootURI() string
	RootPath() string
	SetRootURI(uri string)
	SetRootPath(path string)

	// Configuration
	GetConfig() ServerConfig
	SetConfig(config ServerConfig)
	IsTokenFile(path string) bool

	// Workspace initialization (called by Initialize handler)
	LoadTokensFromConfig() error
	RegisterFileWatchers(ctx *glsp.Context) error

	// LSP context (for publishing diagnostics, etc.)
	GLSPContext() *glsp.Context
	SetGLSPContext(ctx *glsp.Context)

	// Diagnostics publishing
	PublishDiagnostics(context *glsp.Context, uri string) error
}

// No need for ServerConfig interface - handlers can access fields directly
