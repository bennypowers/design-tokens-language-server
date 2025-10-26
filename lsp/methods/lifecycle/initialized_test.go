package lifecycle

import (
	"errors"
	"testing"

	"bennypowers.dev/dtls/lsp/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func TestInitialized(t *testing.T) {
	t.Run("stores GLSP context", func(t *testing.T) {
		ctx := testutil.NewMockServerContext()
		glspCtx := &glsp.Context{}

		params := &protocol.InitializedParams{}

		err := Initialized(ctx, glspCtx, params)
		assert.NoError(t, err)

		// Verify context was stored
		assert.Equal(t, glspCtx, ctx.GLSPContext())
	})

	t.Run("calls LoadTokensFromConfig", func(t *testing.T) {
		ctx := testutil.NewMockServerContext()
		glspCtx := &glsp.Context{}
		params := &protocol.InitializedParams{}

		err := Initialized(ctx, glspCtx, params)
		assert.NoError(t, err)
		assert.True(t, ctx.LoadTokensCalled, "LoadTokensFromConfig should be called")
	})

	t.Run("calls RegisterFileWatchers", func(t *testing.T) {
		ctx := testutil.NewMockServerContext()
		glspCtx := &glsp.Context{}
		params := &protocol.InitializedParams{}

		err := Initialized(ctx, glspCtx, params)
		assert.NoError(t, err)
		assert.True(t, ctx.RegisterWatchersCalled, "RegisterFileWatchers should be called")
	})

	t.Run("continues on LoadTokensFromConfig error", func(t *testing.T) {
		ctx := testutil.NewMockServerContext()
		ctx.LoadTokensFunc = func() error {
			return errors.New("load error")
		}

		glspCtx := &glsp.Context{}
		params := &protocol.InitializedParams{}

		// Should not fail, just log warning
		err := Initialized(ctx, glspCtx, params)
		assert.NoError(t, err)
		assert.True(t, ctx.LoadTokensCalled)
	})

	t.Run("continues on RegisterFileWatchers error", func(t *testing.T) {
		ctx := testutil.NewMockServerContext()
		ctx.RegisterWatchersFunc = func(*glsp.Context) error {
			return errors.New("watcher error")
		}

		glspCtx := &glsp.Context{}
		params := &protocol.InitializedParams{}

		// Should not fail, just log warning
		err := Initialized(ctx, glspCtx, params)
		assert.NoError(t, err)
		assert.True(t, ctx.RegisterWatchersCalled)
	})
}
