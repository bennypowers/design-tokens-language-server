package workspace

import (
	"fmt"
	"os"

	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// LogError logs an error message to stderr and optionally to the LSP client
func LogError(context *glsp.Context, format string, args ...any) {
	message := fmt.Sprintf(format, args...)
	// Always log to stderr for debugging
	fmt.Fprintf(os.Stderr, "[DTLS ERROR] %s\n", message)

	// Optionally notify client if context available
	if context != nil {
		go func() {
			context.Notify(protocol.ServerWindowLogMessage, &protocol.LogMessageParams{
				Type:    protocol.MessageTypeError,
				Message: message,
			})
		}()
	}
}

// LogWarning logs a warning message to stderr and optionally to the LSP client
func LogWarning(context *glsp.Context, format string, args ...any) {
	message := fmt.Sprintf(format, args...)
	// Always log to stderr for debugging
	fmt.Fprintf(os.Stderr, "[DTLS WARNING] %s\n", message)

	// Optionally notify client if context available
	if context != nil {
		go func() {
			context.Notify(protocol.ServerWindowLogMessage, &protocol.LogMessageParams{
				Type:    protocol.MessageTypeWarning,
				Message: message,
			})
		}()
	}
}

// ShowMessage sends a message to be displayed to the user
func ShowMessage(context *glsp.Context, messageType protocol.MessageType, message string) {
	if context != nil {
		go func() {
			context.Notify(protocol.ServerWindowShowMessage, &protocol.ShowMessageParams{
				Type:    messageType,
				Message: message,
			})
		}()
	}
}
