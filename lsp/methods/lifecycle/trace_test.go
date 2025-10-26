package lifecycle

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func TestSetTrace(t *testing.T) {
	t.Run("handles off trace level", func(t *testing.T) {
		ctx := newMockServerContext()
		glspCtx := &glsp.Context{}

		params := &protocol.SetTraceParams{
			Value: "off",
		}

		err := SetTrace(ctx, glspCtx, params)
		assert.NoError(t, err)
	})

	t.Run("handles messages trace level", func(t *testing.T) {
		ctx := newMockServerContext()
		glspCtx := &glsp.Context{}

		params := &protocol.SetTraceParams{
			Value: "messages",
		}

		err := SetTrace(ctx, glspCtx, params)
		assert.NoError(t, err)
	})

	t.Run("handles verbose trace level", func(t *testing.T) {
		ctx := newMockServerContext()
		glspCtx := &glsp.Context{}

		params := &protocol.SetTraceParams{
			Value: "verbose",
		}

		err := SetTrace(ctx, glspCtx, params)
		assert.NoError(t, err)
	})

	t.Run("handles invalid trace level gracefully", func(t *testing.T) {
		ctx := newMockServerContext()
		glspCtx := &glsp.Context{}

		params := &protocol.SetTraceParams{
			Value: "invalid",
		}

		// Should not error, just log
		err := SetTrace(ctx, glspCtx, params)
		assert.NoError(t, err)
	})
}
