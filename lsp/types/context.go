package types

import (
	"bennypowers.dev/dtls/internal/documents"
	"bennypowers.dev/dtls/internal/tokens"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
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
	LoadPackageJsonConfig() error
	IsTokenFile(path string) bool

	// Token file detection
	// ShouldProcessAsTokenFile checks if a document should receive token file features.
	// Returns true if the file is configured as a token file OR has a valid Design Tokens $schema.
	ShouldProcessAsTokenFile(uri string) bool

	// Workspace initialization (called by Initialize handler)
	LoadTokensFromConfig() error
	RegisterFileWatchers(ctx *glsp.Context) error

	// Load tokens from an open document (for files with Design Tokens schema)
	LoadTokensFromDocumentContent(uri, languageID, content string) error

	// File tracking (for managing loaded token files)
	RemoveLoadedFile(path string)

	// LSP context (for publishing diagnostics, etc.)
	GLSPContext() *glsp.Context
	SetGLSPContext(ctx *glsp.Context)

	// Client capability detection (for LSP 3.17 features)
	ClientDiagnosticCapability() *bool
	SetClientDiagnosticCapability(hasCapability bool)

	// Full client capabilities (stored during initialize)
	ClientCapabilities() *protocol.ClientCapabilities
	SetClientCapabilities(caps protocol.ClientCapabilities)

	// Capability helpers derived from ClientCapabilities
	SupportsSnippets() bool
	PreferredHoverFormat() protocol.MarkupKind
	SupportsDefinitionLinks() bool
	SupportsDiagnosticRelatedInfo() bool

	// Diagnostics mode (pull vs push)
	UsePullDiagnostics() bool
	SetUsePullDiagnostics(use bool)

	// Diagnostics publishing
	PublishDiagnostics(context *glsp.Context, uri string) error

	// Semantic tokens delta support
	SemanticTokenCache() SemanticTokenCacher
}

// SemanticTokenCacheEntry holds cached semantic tokens for a document
type SemanticTokenCacheEntry struct {
	ResultID string
	Data     []uint32
	Version  int
}

// SemanticTokenCacher is the interface for semantic token cache operations
type SemanticTokenCacher interface {
	Store(uri string, data []uint32, version int) string
	Get(resultID string) *SemanticTokenCacheEntry
	GetForURI(resultID, uri string) *SemanticTokenCacheEntry
	GetByURI(uri string) *SemanticTokenCacheEntry
	Invalidate(uri string)
}

// No need for ServerConfig interface - handlers can access fields directly
