package lifecycle

import (
	"testing"

	"bennypowers.dev/dtls/lsp/testutil"
	"bennypowers.dev/dtls/lsp/types"
	"github.com/stretchr/testify/assert"
	"github.com/tliron/glsp"
)

func TestShutdown(t *testing.T) {
	t.Run("completes successfully", func(t *testing.T) {
		ctx := testutil.NewMockServerContext()
		glspCtx := &glsp.Context{}
		req := types.NewRequestContext(ctx, glspCtx)

		err := Shutdown(req)
		assert.NoError(t, err)
	})

	t.Run("cleans up CSS parser pool", func(t *testing.T) {
		ctx := testutil.NewMockServerContext()
		glspCtx := &glsp.Context{}
		req := types.NewRequestContext(ctx, glspCtx)

		// Call shutdown
		err := Shutdown(req)
		assert.NoError(t, err)

		// CSS parser pool should be closed (tested in main server tests)
	})

	t.Run("can be called multiple times safely", func(t *testing.T) {
		ctx := testutil.NewMockServerContext()
		glspCtx := &glsp.Context{}
		req := types.NewRequestContext(ctx, glspCtx)

		// Call shutdown multiple times
		err1 := Shutdown(req)
		assert.NoError(t, err1)

		err2 := Shutdown(req)
		assert.NoError(t, err2)

		// Should not panic or error
	})
}
