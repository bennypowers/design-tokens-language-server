package lifecycle

import (
	"testing"

	"bennypowers.dev/dtls/lsp/testutil"
	"bennypowers.dev/dtls/lsp/types"
	"github.com/stretchr/testify/assert"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func TestSetTrace(t *testing.T) {
	t.Run("handles off trace level", func(t *testing.T) {
		ctx := testutil.NewMockServerContext()
		glspCtx := &glsp.Context{}
		req := types.NewRequestContext(ctx, glspCtx)

		params := &protocol.SetTraceParams{
			Value: "off",
		}

		err := SetTrace(req, params)
		assert.NoError(t, err)
	})

	t.Run("handles messages trace level", func(t *testing.T) {
		ctx := testutil.NewMockServerContext()
		glspCtx := &glsp.Context{}
		req := types.NewRequestContext(ctx, glspCtx)

		params := &protocol.SetTraceParams{
			Value: "messages",
		}

		err := SetTrace(req, params)
		assert.NoError(t, err)
	})

	t.Run("handles verbose trace level", func(t *testing.T) {
		ctx := testutil.NewMockServerContext()
		glspCtx := &glsp.Context{}
		req := types.NewRequestContext(ctx, glspCtx)

		params := &protocol.SetTraceParams{
			Value: "verbose",
		}

		err := SetTrace(req, params)
		assert.NoError(t, err)
	})

	t.Run("handles invalid trace level gracefully", func(t *testing.T) {
		ctx := testutil.NewMockServerContext()
		glspCtx := &glsp.Context{}
		req := types.NewRequestContext(ctx, glspCtx)

		params := &protocol.SetTraceParams{
			Value: "invalid",
		}

		// Should not error, just log
		err := SetTrace(req, params)
		assert.NoError(t, err)
	})
}
