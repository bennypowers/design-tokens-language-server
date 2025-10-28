package workspace

import (
	"testing"

	"github.com/tliron/glsp"
)

func TestLogError_NilContext(t *testing.T) {
	// Should not panic with nil context
	LogError(nil, "test error: %s", "message")
	// If we get here, it didn't panic - success!
}

func TestLogWarning_NilContext(t *testing.T) {
	// Should not panic with nil context
	LogWarning(nil, "test warning: %s", "message")
}

func TestShowMessage_NilContext(t *testing.T) {
	// Should not panic with nil context
	ShowMessage(nil, 1, "test message")
}

func TestLogError_WithContext(t *testing.T) {
	// We can't easily test with a real context without a full LSP server
	// but we can verify it doesn't panic
	// In production, this would send messages via the LSP connection
	var ctx *glsp.Context // nil, but typed
	LogError(ctx, "test error: %s", "message")
}
