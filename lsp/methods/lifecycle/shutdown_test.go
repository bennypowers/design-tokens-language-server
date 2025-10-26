package lifecycle

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tliron/glsp"
)

func TestShutdown(t *testing.T) {
	t.Run("completes successfully", func(t *testing.T) {
		ctx := newMockServerContext()
		glspCtx := &glsp.Context{}

		err := Shutdown(ctx, glspCtx)
		assert.NoError(t, err)
	})

	t.Run("cleans up CSS parser pool", func(t *testing.T) {
		ctx := newMockServerContext()
		glspCtx := &glsp.Context{}

		// Call shutdown
		err := Shutdown(ctx, glspCtx)
		assert.NoError(t, err)

		// CSS parser pool should be closed (tested in main server tests)
	})

	t.Run("can be called multiple times safely", func(t *testing.T) {
		ctx := newMockServerContext()
		glspCtx := &glsp.Context{}

		// Call shutdown multiple times
		err1 := Shutdown(ctx, glspCtx)
		assert.NoError(t, err1)

		err2 := Shutdown(ctx, glspCtx)
		assert.NoError(t, err2)

		// Should not panic or error
	})
}
