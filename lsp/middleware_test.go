package lsp

import (
	"bytes"
	"errors"
	"testing"

	"bennypowers.dev/dtls/internal/documents"
	"bennypowers.dev/dtls/internal/log"
	"bennypowers.dev/dtls/internal/tokens"
	"bennypowers.dev/dtls/lsp/types"
	"github.com/stretchr/testify/assert"
	"github.com/tliron/glsp"
)

// Verify compile-time interface satisfaction
var _ = (*glsp.Context)(nil)

// mockServerContext implements types.ServerContext for testing
type mockServerContext struct{}

func (m *mockServerContext) Document(uri string) *documents.Document      { return nil }
func (m *mockServerContext) DocumentManager() *documents.Manager          { return nil }
func (m *mockServerContext) AllDocuments() []*documents.Document          { return nil }
func (m *mockServerContext) Token(name string) *tokens.Token              { return nil }
func (m *mockServerContext) TokenManager() *tokens.Manager                { return nil }
func (m *mockServerContext) TokenCount() int                              { return 0 }
func (m *mockServerContext) RootURI() string                              { return "" }
func (m *mockServerContext) RootPath() string                             { return "" }
func (m *mockServerContext) SetRootURI(uri string)                        {}
func (m *mockServerContext) SetRootPath(path string)                      {}
func (m *mockServerContext) GetConfig() types.ServerConfig                { return types.ServerConfig{} }
func (m *mockServerContext) SetConfig(config types.ServerConfig)          {}
func (m *mockServerContext) LoadPackageJsonConfig() error                 { return nil }
func (m *mockServerContext) IsTokenFile(path string) bool                 { return false }
func (m *mockServerContext) LoadTokensFromConfig() error                  { return nil }
func (m *mockServerContext) RegisterFileWatchers(ctx *glsp.Context) error { return nil }
func (m *mockServerContext) RemoveLoadedFile(path string)                 {}
func (m *mockServerContext) GLSPContext() *glsp.Context                   { return nil }
func (m *mockServerContext) SetGLSPContext(ctx *glsp.Context)             {}
func (m *mockServerContext) ClientDiagnosticCapability() *bool            { return nil }
func (m *mockServerContext) SetClientDiagnosticCapability(hasCapability bool) {}
func (m *mockServerContext) PublishDiagnostics(context *glsp.Context, uri string) error {
	return nil
}
func (m *mockServerContext) UsePullDiagnostics() bool         { return false }
func (m *mockServerContext) SetUsePullDiagnostics(use bool)   {}
func (m *mockServerContext) AddWarning(err error)             {}
func (m *mockServerContext) TakeWarnings() []error            { return nil }
func (m *mockServerContext) ShouldProcessAsTokenFile(uri string) bool { return true }

func TestMethod_PanicRecovery(t *testing.T) {
	// Capture log output
	var logBuf bytes.Buffer
	log.SetOutput(&logBuf)
	defer log.SetOutput(nil)

	// Create a handler that panics
	panicHandler := func(req *types.RequestContext, params string) (string, error) {
		panic("test panic")
	}

	// Wrap with middleware
	server := &mockServerContext{}
	wrapped := method(server, "testMethod", panicHandler)

	// Use nil context to avoid LogError trying to Notify (which panics with nil Notify)
	// The panic recovery will still work, it just won't notify the client
	result, err := wrapped(nil, "test params")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "internal error")
	assert.Contains(t, err.Error(), "testMethod")
	assert.Empty(t, result)
	assert.Contains(t, logBuf.String(), "PANIC")
}

func TestMethod_ErrorWrapping(t *testing.T) {
	// Capture log output
	var logBuf bytes.Buffer
	log.SetOutput(&logBuf)
	defer log.SetOutput(nil)

	// Create a handler that returns an error
	errHandler := func(req *types.RequestContext, params string) (string, error) {
		return "", errors.New("handler error")
	}

	server := &mockServerContext{}
	wrapped := method(server, "testMethod", errHandler)

	// Use nil context to avoid LogError trying to Notify
	result, err := wrapped(nil, "params")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "testMethod")
	assert.Contains(t, err.Error(), "handler error")
	assert.Empty(t, result)
	assert.Contains(t, logBuf.String(), "error")
}

func TestMethod_SuccessLogging(t *testing.T) {
	// Capture log output and enable debug level
	var logBuf bytes.Buffer
	log.SetOutput(&logBuf)
	log.SetLevel(log.LevelDebug)
	defer func() {
		log.SetOutput(nil)
		log.SetLevel(log.LevelInfo)
	}()

	// Create a successful handler
	successHandler := func(req *types.RequestContext, params string) (string, error) {
		return "success result", nil
	}

	server := &mockServerContext{}
	wrapped := method(server, "testMethod", successHandler)

	// Use nil context for testing - no client notification needed
	result, err := wrapped(nil, "params")

	assert.NoError(t, err)
	assert.Equal(t, "success result", result)
	assert.Contains(t, logBuf.String(), "started")
	assert.Contains(t, logBuf.String(), "completed")
}

func TestNotify_PanicRecovery(t *testing.T) {
	var logBuf bytes.Buffer
	log.SetOutput(&logBuf)
	defer log.SetOutput(nil)

	panicHandler := func(req *types.RequestContext, params int) error {
		panic("notify panic")
	}

	server := &mockServerContext{}
	wrapped := notify(server, "testNotify", panicHandler)

	// Use nil context to avoid LogError trying to Notify
	err := wrapped(nil, 42)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "internal error")
	assert.Contains(t, logBuf.String(), "PANIC")
}

func TestNoParam_PanicRecovery(t *testing.T) {
	var logBuf bytes.Buffer
	log.SetOutput(&logBuf)
	defer log.SetOutput(nil)

	panicHandler := func(req *types.RequestContext) error {
		panic("noParam panic")
	}

	server := &mockServerContext{}
	wrapped := noParam(server, "shutdown", panicHandler)

	// Use nil context to avoid LogError trying to Notify
	err := wrapped(nil)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "internal error")
	assert.Contains(t, logBuf.String(), "PANIC")
}

func TestNoParam_Success(t *testing.T) {
	var logBuf bytes.Buffer
	log.SetOutput(&logBuf)
	log.SetLevel(log.LevelDebug)
	defer func() {
		log.SetOutput(nil)
		log.SetLevel(log.LevelInfo)
	}()

	successHandler := func(req *types.RequestContext) error {
		return nil
	}

	server := &mockServerContext{}
	wrapped := noParam(server, "shutdown", successHandler)

	// Use nil context for testing
	err := wrapped(nil)

	assert.NoError(t, err)
	assert.Contains(t, logBuf.String(), "completed")
}
