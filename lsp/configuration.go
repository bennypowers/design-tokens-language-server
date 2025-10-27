package lsp

import (
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"

	"bennypowers.dev/dtls/lsp/types"
)

// GetConfig returns the current server configuration
func (s *Server) GetConfig() types.ServerConfig {
	s.configMu.RLock()
	defer s.configMu.RUnlock()
	return s.config
}

// SetConfig updates the server configuration
func (s *Server) SetConfig(config types.ServerConfig) {
	s.configMu.Lock()
	defer s.configMu.Unlock()
	s.config = config
}

// loadTokensFromConfig loads tokens based on current configuration
func (s *Server) LoadTokensFromConfig() error {
	// If tokensFiles is specified, load those files
	if len(s.config.TokensFiles) > 0 {
		// Clear existing tokens before loading configured files
		s.tokens.Clear()
		s.autoDiscoveryMode = false
		return s.loadExplicitTokenFiles()
	}

	// If we're in auto-discovery mode, always re-discover to pick up new files
	if s.autoDiscoveryMode && s.rootPath != "" {
		// Clear existing tokens before auto-discover
		s.tokens.Clear()
		return s.loadTokenFilesAutoDiscover()
	}

	// If we have previously loaded files (from tests or programmatic loading),
	// reload them
	s.loadedFilesMu.RLock()
	hasLoadedFiles := len(s.loadedFiles) > 0
	s.loadedFilesMu.RUnlock()
	if hasLoadedFiles {
		return s.reloadPreviouslyLoadedFiles()
	}

	// Otherwise, auto-discover token files
	if s.rootPath != "" {
		// Clear existing tokens before auto-discover
		s.tokens.Clear()
		s.autoDiscoveryMode = true
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

		// Load the file with per-file options
		opts := &TokenFileOptions{
			Prefix:       prefix,
			GroupMarkers: groupMarkers,
		}
		if err := s.LoadTokenFileWithOptions(path, opts); err != nil {
			errs = append(errs, fmt.Errorf("failed to load %s: %w", path, err))
			continue
		}

		if len(groupMarkers) > 0 {
			fmt.Fprintf(os.Stderr, "[DTLS] Loaded %s (prefix: %s, groupMarkers: %v)\n", path, prefix, groupMarkers)
		} else {
			fmt.Fprintf(os.Stderr, "[DTLS] Loaded %s (prefix: %s)\n", path, prefix)
		}
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
		Prefix:       s.config.Prefix,
		GroupMarkers: s.config.GroupMarkers,
	}

	return s.LoadTokenFiles(tokenConfig)
}

// reloadPreviouslyLoadedFiles reloads all files that were previously loaded
// This is used for programmatic loading (e.g., tests using LoadTokenFile)
// For auto-discovery mode, LoadTokensFromConfig handles re-discovery directly
func (s *Server) reloadPreviouslyLoadedFiles() error {
	// Clear existing tokens
	s.tokens.Clear()

	// Copy loadedFiles to avoid holding the lock during file I/O
	s.loadedFilesMu.RLock()
	filesToReload := make(map[string]*TokenFileOptions, len(s.loadedFiles))
	maps.Copy(filesToReload, s.loadedFiles)
	s.loadedFilesMu.RUnlock()

	// Reload each previously loaded file with its original options (prefix, groupMarkers)
	var errs []error
	for path, opts := range filesToReload {
		if err := s.loadTokenFileInternal(path, opts); err != nil {
			errs = append(errs, fmt.Errorf("failed to reload %s: %w", path, err))
			continue
		}
		if len(opts.GroupMarkers) > 0 {
			fmt.Fprintf(os.Stderr, "[DTLS] Reloaded %s (prefix: %s, groupMarkers: %v)\n", path, opts.Prefix, opts.GroupMarkers)
		} else {
			fmt.Fprintf(os.Stderr, "[DTLS] Reloaded %s (prefix: %s)\n", path, opts.Prefix)
		}
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}
