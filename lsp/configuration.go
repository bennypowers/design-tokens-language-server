package lsp

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/bennypowers/design-tokens-language-server/lsp/types"
)

// GetConfig returns the current server configuration
func (s *Server) GetConfig() types.ServerConfig {
	if s.config.TokensFiles == nil {
		return types.DefaultConfig()
	}
	return s.config
}

// SetConfig updates the server configuration
func (s *Server) SetConfig(config types.ServerConfig) {
	s.config = config
}

// loadTokensFromConfig loads tokens based on current configuration
func (s *Server) LoadTokensFromConfig() error {
	// Clear existing tokens
	s.tokens.Clear()

	// If tokensFiles is specified, load those files
	if len(s.config.TokensFiles) > 0 {
		return s.loadExplicitTokenFiles()
	}

	// Otherwise, auto-discover token files
	if s.rootPath != "" {
		return s.loadTokenFilesAutoDiscover()
	}

	return nil
}

// loadExplicitTokenFiles loads tokens from explicitly configured files
func (s *Server) loadExplicitTokenFiles() error {
	var errs []error

	for _, item := range s.config.TokensFiles {
		var path, prefix string
		var groupMarkers []string

		// Parse the item - can be string or object
		switch v := item.(type) {
		case string:
			path = v
			prefix = s.config.Prefix
			groupMarkers = s.config.GroupMarkers
		case map[string]any:
			// Convert to TokenFileSpec
			pathVal, ok := v["path"]
			if !ok {
				continue
			}
			path, _ = pathVal.(string)
			if prefixVal, ok := v["prefix"]; ok {
				prefix, _ = prefixVal.(string)
			} else {
				prefix = s.config.Prefix
			}
			if gmVal, ok := v["groupMarkers"]; ok {
				if gmSlice, ok := gmVal.([]any); ok {
					for _, gm := range gmSlice {
						if gmStr, ok := gm.(string); ok {
							groupMarkers = append(groupMarkers, gmStr)
						}
					}
				}
			}
			if len(groupMarkers) == 0 {
				groupMarkers = s.config.GroupMarkers
			}
		default:
			continue
		}

		// Resolve path relative to workspace
		if s.rootPath != "" && !filepath.IsAbs(path) {
			path = filepath.Join(s.rootPath, path)
		}

		// TODO: Handle npm: protocol

		// Load the file
		if err := s.LoadTokenFile(path, prefix); err != nil {
			errs = append(errs, fmt.Errorf("failed to load %s: %w", path, err))
			continue
		}

		fmt.Fprintf(os.Stderr, "[DTLS] Loaded %s (prefix: %s)\n", path, prefix)
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

// loadTokenFilesAutoDiscover auto-discovers and loads token files
func (s *Server) loadTokenFilesAutoDiscover() error {
	tokenConfig := TokenFileConfig{
		RootDir: s.rootPath,
		Patterns: []string{
			"**/tokens.json",
			"**/*.tokens.json",
			"**/design-tokens.json",
		},
		Prefix: s.config.Prefix,
	}

	return s.LoadTokenFiles(tokenConfig)
}
