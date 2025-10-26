package lifecycle

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// mockServerContextWithCallbacks extends mockServerContext with callbacks for testing
type mockServerContextWithCallbacks struct {
	*mockServerContext
	loadTokensFunc        func() error
	registerWatchersFunc  func(*glsp.Context) error
	loadTokensCalled      bool
	registerWatchersCalled bool
}

func (m *mockServerContextWithCallbacks) LoadTokensFromConfig() error {
	m.loadTokensCalled = true
	if m.loadTokensFunc != nil {
		return m.loadTokensFunc()
	}
	return nil
}

func (m *mockServerContextWithCallbacks) RegisterFileWatchers(ctx *glsp.Context) error {
	m.registerWatchersCalled = true
	if m.registerWatchersFunc != nil {
		return m.registerWatchersFunc(ctx)
	}
	return nil
}

func newMockWithCallbacks() *mockServerContextWithCallbacks {
	return &mockServerContextWithCallbacks{
		mockServerContext: newMockServerContext(),
	}
}

func TestInitialized(t *testing.T) {
	t.Run("stores GLSP context", func(t *testing.T) {
		ctx := newMockServerContext()
		glspCtx := &glsp.Context{}

		params := &protocol.InitializedParams{}

		err := Initialized(ctx, glspCtx, params)
		assert.NoError(t, err)

		// Verify context was stored
		assert.Equal(t, glspCtx, ctx.GLSPContext())
	})

	t.Run("calls LoadTokensFromConfig", func(t *testing.T) {
		ctx := newMockWithCallbacks()
		glspCtx := &glsp.Context{}
		params := &protocol.InitializedParams{}

		err := Initialized(ctx, glspCtx, params)
		assert.NoError(t, err)
		assert.True(t, ctx.loadTokensCalled, "LoadTokensFromConfig should be called")
	})

	t.Run("calls RegisterFileWatchers", func(t *testing.T) {
		ctx := newMockWithCallbacks()
		glspCtx := &glsp.Context{}
		params := &protocol.InitializedParams{}

		err := Initialized(ctx, glspCtx, params)
		assert.NoError(t, err)
		assert.True(t, ctx.registerWatchersCalled, "RegisterFileWatchers should be called")
	})

	t.Run("continues on LoadTokensFromConfig error", func(t *testing.T) {
		ctx := newMockWithCallbacks()
		ctx.loadTokensFunc = func() error {
			return errors.New("load error")
		}

		glspCtx := &glsp.Context{}
		params := &protocol.InitializedParams{}

		// Should not fail, just log warning
		err := Initialized(ctx, glspCtx, params)
		assert.NoError(t, err)
		assert.True(t, ctx.loadTokensCalled)
	})

	t.Run("continues on RegisterFileWatchers error", func(t *testing.T) {
		ctx := newMockWithCallbacks()
		ctx.registerWatchersFunc = func(*glsp.Context) error {
			return errors.New("watcher error")
		}

		glspCtx := &glsp.Context{}
		params := &protocol.InitializedParams{}

		// Should not fail, just log warning
		err := Initialized(ctx, glspCtx, params)
		assert.NoError(t, err)
		assert.True(t, ctx.registerWatchersCalled)
	})
}
