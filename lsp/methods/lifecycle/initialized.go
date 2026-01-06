package lifecycle

import (
	"bennypowers.dev/dtls/internal/log"

	"bennypowers.dev/dtls/lsp/types"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// Initialized handles the LSP initialized notification
func Initialized(req *types.RequestContext, params *protocol.InitializedParams) error {
	log.Info("Server initialized")

	// Store context for later use (diagnostics)
	req.Server.SetGLSPContext(req.GLSP)

	// Read configuration from package.json if it exists
	// This provides the "zero-config" experience for projects with package.json config
	if err := req.Server.LoadPackageJsonConfig(); err != nil {
		log.Info("Warning: failed to load package.json config: %v", err)
		// Don't fail initialization, just log the error
	}

	// Load token files from workspace using configuration
	if err := req.Server.LoadTokensFromConfig(); err != nil {
		log.Info("Warning: failed to load token files: %v", err)
		// Don't fail initialization, just log the error
	}

	// Register file watchers for token files
	if err := req.Server.RegisterFileWatchers(req.GLSP); err != nil {
		log.Info("Warning: failed to register file watchers: %v", err)
		// Don't fail initialization, just log the error
	}

	return nil
}
