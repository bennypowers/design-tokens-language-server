package lsp

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// HandleDidChangeWatchedFiles is exported for testing
func (s *Server) HandleDidChangeWatchedFiles(params *protocol.DidChangeWatchedFilesParams) error {
	return s.handleDidChangeWatchedFiles(nil, params)
}

// handleDidChangeWatchedFiles handles the workspace/didChangeWatchedFiles notification
func (s *Server) handleDidChangeWatchedFiles(context *glsp.Context, params *protocol.DidChangeWatchedFilesParams) error {
	fmt.Fprintf(os.Stderr, "[DTLS] Watched files changed: %d files\n", len(params.Changes))

	// Track if we need to reload tokens
	needsReload := false

	for _, change := range params.Changes {
		uri := change.URI
		path := uriToPath(uri)
		fmt.Fprintf(os.Stderr, "[DTLS] File change: %s (type: %d)\n", path, change.Type)

		// Check if this is a token file we're watching
		if s.isTokenFile(path) {
			needsReload = true

			// If the file was deleted, we might want to handle it differently
			if change.Type == protocol.FileChangeTypeDeleted {
				fmt.Fprintf(os.Stderr, "[DTLS] Token file deleted: %s\n", path)
			}
		}
	}

	// Reload all token files if any token file changed
	if needsReload {
		fmt.Fprintf(os.Stderr, "[DTLS] Reloading token files due to changes\n")
		if err := s.reloadTokenFiles(); err != nil {
			fmt.Fprintf(os.Stderr, "[DTLS] Warning: failed to reload tokens: %v\n", err)
		}

		// Republish diagnostics for all open documents
		if s.context != nil {
			for _, doc := range s.documents.GetAll() {
				if err := s.PublishDiagnostics(s.context, doc.URI()); err != nil {
					fmt.Fprintf(os.Stderr, "[DTLS] Warning: failed to publish diagnostics for %s: %v\n", doc.URI(), err)
				}
			}
		}
	}

	return nil
}

// reloadTokenFiles reloads all tracked token files
func (s *Server) reloadTokenFiles() error {
	// Clear existing tokens
	s.tokens.Clear()

	// Reload all tracked files
	var errs []error
	for filepath, prefix := range s.loadedFiles {
		if err := s.loadTokenFileInternal(filepath, prefix); err != nil {
			errs = append(errs, fmt.Errorf("failed to reload %s: %w", filepath, err))
		}
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

// isTokenFile checks if a file path is one of our token files
func (s *Server) isTokenFile(path string) bool {
	// Check if it's a JSON or YAML file
	ext := filepath.Ext(path)
	if ext != ".json" && ext != ".yaml" && ext != ".yml" {
		return false
	}

	// Check if it's in our loaded files map (for programmatically loaded tokens)
	if _, exists := s.loadedFiles[path]; exists {
		return true
	}

	// Check if it matches any of our configured token files
	for _, item := range s.config.TokensFiles {
		var tokenPath string
		switch v := item.(type) {
		case string:
			tokenPath = v
		case map[string]any:
			if pathVal, ok := v["path"]; ok {
				tokenPath, _ = pathVal.(string)
			}
		}

		if tokenPath == "" {
			continue
		}

		// Resolve relative paths
		if s.rootPath != "" && !filepath.IsAbs(tokenPath) {
			tokenPath = filepath.Join(s.rootPath, tokenPath)
		}

		// Check if the paths match
		if path == tokenPath {
			return true
		}
	}

	// If we're in auto-discover mode, check common patterns
	if len(s.config.TokensFiles) == 0 {
		filename := filepath.Base(path)
		if filename == "tokens.json" ||
			strings.HasSuffix(filename, ".tokens.json") ||
			filename == "design-tokens.json" ||
			filename == "tokens.yaml" ||
			strings.HasSuffix(filename, ".tokens.yaml") ||
			filename == "design-tokens.yaml" {
			return true
		}
	}

	return false
}

// registerFileWatchers registers file watchers with the client
func (s *Server) registerFileWatchers(context *glsp.Context) error {
	// Build list of watchers based on configuration
	watchers := []protocol.FileSystemWatcher{}

	if len(s.config.TokensFiles) > 0 {
		// Watch explicitly configured files
		for _, item := range s.config.TokensFiles {
			var tokenPath string
			switch v := item.(type) {
			case string:
				tokenPath = v
			case map[string]any:
				if pathVal, ok := v["path"]; ok {
					tokenPath, _ = pathVal.(string)
				}
			}

			if tokenPath == "" {
				continue
			}

			// Convert to filesystem path pattern (forward-slash separated)
			// Glob patterns use filesystem paths, not URIs
			var pattern string
			if filepath.IsAbs(tokenPath) {
				// Absolute path: convert to forward slashes
				pattern = filepath.ToSlash(filepath.Clean(tokenPath))
			} else if s.rootPath != "" {
				// Relative path: join with root and convert to forward slashes
				absPath := filepath.Join(s.rootPath, tokenPath)
				pattern = filepath.ToSlash(filepath.Clean(absPath))
			} else {
				// No root path: keep relative, convert to forward slashes
				pattern = filepath.ToSlash(tokenPath)
			}

			watchers = append(watchers, protocol.FileSystemWatcher{
				GlobPattern: pattern,
			})
		}
	} else if s.rootPath != "" {
		// Auto-discover mode: watch common patterns
		// Convert root path to forward-slash separated filesystem path
		rootPattern := filepath.ToSlash(filepath.Clean(s.rootPath))
		watchers = append(watchers,
			protocol.FileSystemWatcher{
				GlobPattern: rootPattern + "/**/tokens.json",
			},
			protocol.FileSystemWatcher{
				GlobPattern: rootPattern + "/**/*.tokens.json",
			},
			protocol.FileSystemWatcher{
				GlobPattern: rootPattern + "/**/design-tokens.json",
			},
			protocol.FileSystemWatcher{
				GlobPattern: rootPattern + "/**/tokens.yaml",
			},
			protocol.FileSystemWatcher{
				GlobPattern: rootPattern + "/**/*.tokens.yaml",
			},
			protocol.FileSystemWatcher{
				GlobPattern: rootPattern + "/**/design-tokens.yaml",
			},
		)
	}

	if len(watchers) == 0 {
		fmt.Fprintf(os.Stderr, "[DTLS] No file watchers to register\n")
		return nil
	}

	// Register the watchers with the client
	params := protocol.RegistrationParams{
		Registrations: []protocol.Registration{
			{
				ID:     "design-tokens-file-watcher",
				Method: "workspace/didChangeWatchedFiles",
				RegisterOptions: protocol.DidChangeWatchedFilesRegistrationOptions{
					Watchers: watchers,
				},
			},
		},
	}

	// Send registration request to client
	// Note: client/registerCapability is a request (not notification) per LSP spec.
	// We use context.Call instead of context.Notify to properly send a request.
	//
	// Error handling note: glsp.Context.Call doesn't return errors - the underlying
	// jsonrpc2.Conn.Call errors are caught and logged by the glsp wrapper
	// (see github.com/tliron/glsp@v0.2.2/server/handle.go:24-28).
	// If the client rejects the registration, the error response will be logged
	// to stderr by the glsp library. Since client capability registration failures
	// are not fatal (the client continues working, just without file watching),
	// this fire-and-forget approach with logging is acceptable.
	var result interface{}
	context.Call("client/registerCapability", params, &result)

	fmt.Fprintf(os.Stderr, "[DTLS] Sent file watcher registration request (%d watchers)\n", len(watchers))
	return nil
}
