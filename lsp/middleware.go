package lsp

import (
	"bennypowers.dev/dtls/internal/log"
	"fmt"
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
	handler func(*types.RequestContext, P) (R, error),
) func(*glsp.Context, P) (R, error) {
	return func(glspCtx *glsp.Context, params P) (result R, err error) {
		// Panic recovery - prevents LSP server crashes
		defer func() {
			if r := recover(); r != nil {
				stackTrace := string(debug.Stack())
				log.Error("PANIC in %s: %v\nStack trace:\n%s",
					methodName, r, stackTrace)
				// Log panic to LSP client
				workspace.LogError(glspCtx, "Internal error in %s: %v", methodName, r)
				err = fmt.Errorf("internal error in %s", methodName)
				var zero R
				result = zero
			}
		}()

		// Request logging
		log.Debug("%s started", methodName)

		// Create request context
		req := types.NewRequestContext(s, glspCtx)

		// Execute handler with request context
		result, err = handler(req, params)

		// Log warnings if operation succeeded
		if err == nil && req.HasWarnings() {
			for _, w := range req.Warnings() {
				workspace.LogWarning(glspCtx, "%s warning: %v", methodName, w)
			}
		}

		// Error context wrapping
		if err != nil {
			log.Error("%s error: %v", methodName, err)
			// Log error to LSP client via window/logMessage
			workspace.LogError(glspCtx, "%s: %v", methodName, err)
			return result, fmt.Errorf("%s: %w", methodName, err)
		}

		// Success logging
		log.Debug("%s completed successfully", methodName)
		return result, nil
	}
}

// notify wraps an LSP notification handler that returns only error
func notify[P any](
	s types.ServerContext,
	methodName string,
	handler func(*types.RequestContext, P) error,
) func(*glsp.Context, P) error {
	return func(glspCtx *glsp.Context, params P) (err error) {
		defer func() {
			if r := recover(); r != nil {
				stackTrace := string(debug.Stack())
				log.Error("PANIC in %s: %v\nStack trace:\n%s",
					methodName, r, stackTrace)
				// Log panic to LSP client
				workspace.LogError(glspCtx, "Internal error in %s: %v", methodName, r)
				err = fmt.Errorf("internal error in %s", methodName)
			}
		}()

		log.Debug("%s started", methodName)

		// Create request context
		req := types.NewRequestContext(s, glspCtx)

		// Execute handler
		err = handler(req, params)

		// Log warnings if operation succeeded
		if err == nil && req.HasWarnings() {
			for _, w := range req.Warnings() {
				workspace.LogWarning(glspCtx, "%s warning: %v", methodName, w)
			}
		}

		if err != nil {
			log.Error("%s error: %v", methodName, err)
			// Log error to LSP client via window/logMessage
			workspace.LogError(glspCtx, "%s: %v", methodName, err)
			return fmt.Errorf("%s: %w", methodName, err)
		}

		log.Debug("%s completed successfully", methodName)
		return nil
	}
}

// noParam wraps an LSP handler that takes no params (like Shutdown)
func noParam(
	s types.ServerContext,
	methodName string,
	handler func(*types.RequestContext) error,
) func(*glsp.Context) error {
	return func(glspCtx *glsp.Context) (err error) {
		defer func() {
			if r := recover(); r != nil {
				stackTrace := string(debug.Stack())
				log.Error("PANIC in %s: %v\nStack trace:\n%s",
					methodName, r, stackTrace)
				// Log panic to LSP client
				workspace.LogError(glspCtx, "Internal error in %s: %v", methodName, r)
				err = fmt.Errorf("internal error in %s", methodName)
			}
		}()

		log.Debug("%s started", methodName)

		// Create request context
		req := types.NewRequestContext(s, glspCtx)

		// Execute handler
		err = handler(req)

		// Log warnings if operation succeeded
		if err == nil && req.HasWarnings() {
			for _, w := range req.Warnings() {
				workspace.LogWarning(glspCtx, "%s warning: %v", methodName, w)
			}
		}

		if err != nil {
			log.Error("%s error: %v", methodName, err)
			// Log error to LSP client via window/logMessage
			workspace.LogError(glspCtx, "%s: %v", methodName, err)
			return fmt.Errorf("%s: %w", methodName, err)
		}

		log.Debug("%s completed successfully", methodName)
		return nil
	}
}
