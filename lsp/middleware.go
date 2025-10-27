package lsp

import (
	"fmt"
	"os"
	"runtime/debug"

	"bennypowers.dev/dtls/lsp/methods/workspace"
	"bennypowers.dev/dtls/lsp/types"
	"github.com/tliron/glsp"
)

// method wraps an LSP handler that returns (result, error) with middleware
// Returns the underlying function type so it's compatible with protocol.Handler field types
func method[P, R any](
	s types.ServerContext,
	methodName string,
	handler func(types.ServerContext, *glsp.Context, P) (R, error),
) func(*glsp.Context, P) (R, error) {
	return func(ctx *glsp.Context, params P) (result R, err error) {
		// Panic recovery - prevents LSP server crashes
		defer func() {
			if r := recover(); r != nil {
				stackTrace := string(debug.Stack())
				fmt.Fprintf(os.Stderr, "[LSP] PANIC in %s: %v\nStack trace:\n%s",
					methodName, r, stackTrace)
				// Log panic to LSP client
				workspace.LogError(ctx, "Internal error in %s: %v", methodName, r)
				err = fmt.Errorf("internal error in %s", methodName)
				var zero R
				result = zero
			}
		}()

		// Request logging
		fmt.Fprintf(os.Stderr, "[LSP] %s started\n", methodName)

		// Execute handler
		result, err = handler(s, ctx, params)

		// Error context wrapping
		if err != nil {
			fmt.Fprintf(os.Stderr, "[LSP] %s error: %v\n", methodName, err)
			// Log error to LSP client via window/logMessage
			workspace.LogError(ctx, "%s: %v", methodName, err)
			return result, fmt.Errorf("%s: %w", methodName, err)
		}

		// Success logging
		fmt.Fprintf(os.Stderr, "[LSP] %s completed successfully\n", methodName)
		return result, nil
	}
}

// notify wraps an LSP notification handler that returns only error
func notify[P any](
	s types.ServerContext,
	methodName string,
	handler func(types.ServerContext, *glsp.Context, P) error,
) func(*glsp.Context, P) error {
	return func(ctx *glsp.Context, params P) (err error) {
		defer func() {
			if r := recover(); r != nil {
				stackTrace := string(debug.Stack())
				fmt.Fprintf(os.Stderr, "[LSP] PANIC in %s: %v\nStack trace:\n%s",
					methodName, r, stackTrace)
				// Log panic to LSP client
				workspace.LogError(ctx, "Internal error in %s: %v", methodName, r)
				err = fmt.Errorf("internal error in %s", methodName)
			}
		}()

		fmt.Fprintf(os.Stderr, "[LSP] %s started\n", methodName)
		err = handler(s, ctx, params)

		if err != nil {
			fmt.Fprintf(os.Stderr, "[LSP] %s error: %v\n", methodName, err)
			// Log error to LSP client via window/logMessage
			workspace.LogError(ctx, "%s: %v", methodName, err)
			return fmt.Errorf("%s: %w", methodName, err)
		}

		fmt.Fprintf(os.Stderr, "[LSP] %s completed successfully\n", methodName)
		return nil
	}
}

// noParam wraps an LSP handler that takes no params (like Shutdown)
func noParam(
	s types.ServerContext,
	methodName string,
	handler func(types.ServerContext, *glsp.Context) error,
) func(*glsp.Context) error {
	return func(ctx *glsp.Context) (err error) {
		defer func() {
			if r := recover(); r != nil {
				stackTrace := string(debug.Stack())
				fmt.Fprintf(os.Stderr, "[LSP] PANIC in %s: %v\nStack trace:\n%s",
					methodName, r, stackTrace)
				// Log panic to LSP client
				workspace.LogError(ctx, "Internal error in %s: %v", methodName, r)
				err = fmt.Errorf("internal error in %s", methodName)
			}
		}()

		fmt.Fprintf(os.Stderr, "[LSP] %s started\n", methodName)
		err = handler(s, ctx)

		if err != nil {
			fmt.Fprintf(os.Stderr, "[LSP] %s error: %v\n", methodName, err)
			// Log error to LSP client via window/logMessage
			workspace.LogError(ctx, "%s: %v", methodName, err)
			return fmt.Errorf("%s: %w", methodName, err)
		}

		fmt.Fprintf(os.Stderr, "[LSP] %s completed successfully\n", methodName)
		return nil
	}
}
