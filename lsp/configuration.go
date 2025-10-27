package lsp

import (
	"errors"
	"fmt"
	"maps"
	"os"

	"bennypowers.dev/dtls/lsp/types"
)

// GetConfig returns the current server configuration (user settings only)
func (s *Server) GetConfig() types.ServerConfig {
	s.configMu.RLock()
	defer s.configMu.RUnlock()
	return s.config
}

// GetState returns a snapshot of runtime state (NOT configuration)
// For configuration, use GetConfig() separately.
// This separation allows clear distinction between user configuration and runtime state.
func (s *Server) GetState() types.ServerState {
	s.configMu.RLock()
	defer s.configMu.RUnlock()
	return types.ServerState{
		AutoDiscoveryMode: s.autoDiscoveryMode,
		RootPath:          s.rootPath,
	}
}

// SetConfig updates the server configuration
func (s *Server) SetConfig(config types.ServerConfig) {
	s.configMu.Lock()
	defer s.configMu.Unlock()
	s.config = config
}

// setAutoDiscoveryMode updates the auto-discovery mode
func (s *Server) setAutoDiscoveryMode(mode bool) {
	s.configMu.Lock()
	defer s.configMu.Unlock()
	s.autoDiscoveryMode = mode
}

// loadTokensFromConfig loads tokens based on current configuration
func (s *Server) LoadTokensFromConfig() error {
	// Snapshot config and state separately for semantic clarity
	cfg := s.GetConfig()
	state := s.GetState()

	// If tokensFiles is explicitly provided (nil vs empty are distinct):
	//  - empty slice => switch to auto-discovery or reload previously loaded files
	//  - non-empty   => load explicit files
	if cfg.TokensFiles != nil {
		// Clear existing tokens before loading configured files
		s.tokens.Clear()
		s.setAutoDiscoveryMode(false)
		if len(cfg.TokensFiles) == 0 {
			// Empty TokensFiles: try auto-discovery if we have a workspace root
			if state.RootPath != "" {
				s.setAutoDiscoveryMode(true)
				return s.loadTokenFilesAutoDiscover()
			}
			// No workspace root: check if we have programmatically loaded files to reload
			s.loadedFilesMu.RLock()
			hasLoadedFiles := len(s.loadedFiles) > 0
			s.loadedFilesMu.RUnlock()
			if hasLoadedFiles {
				return s.reloadPreviouslyLoadedFiles()
			}
			return nil
		}
		return s.loadExplicitTokenFiles()
	}

	// If we're in auto-discovery mode, always re-discover to pick up new files
	if state.AutoDiscoveryMode && state.RootPath != "" {
		// Clear existing tokens before auto-discover
		s.tokens.Clear()
		return s.loadTokenFilesAutoDiscover()
	}

	// If we have previously loaded files (from tests or programmatic loading),
	// reload them
	s.loadedFilesMu.RLock()
	hasLoadedFiles := len(s.loadedFiles) > 0
	s.loadedFilesMu.RUnlock()
	// Prefer discovery when we have a workspace root but no active discovery and no files yet.
	if state.RootPath != "" && !hasLoadedFiles {
		s.tokens.Clear()
		s.setAutoDiscoveryMode(true)
		return s.loadTokenFilesAutoDiscover()
	}
	if hasLoadedFiles {
		return s.reloadPreviouslyLoadedFiles()
	}

	return nil
}

// loadExplicitTokenFiles loads tokens from explicitly configured files
func (s *Server) loadExplicitTokenFiles() error {
	// Snapshot config and state separately for semantic clarity
	cfg := s.GetConfig()
	state := s.GetState()

	var errs []error

	for _, item := range cfg.TokensFiles {
		var path, prefix string
		var groupMarkers []string

		// Parse the item - can be string or object
		switch v := item.(type) {
		case string:
			path = v
			prefix = cfg.Prefix
			groupMarkers = cfg.GroupMarkers
		case map[string]any:
			// Convert to TokenFileSpec
			pathVal, ok := v["path"]
			if !ok {
				errs = append(errs, fmt.Errorf("token file entry missing required 'path' field: %v", v))
				continue
			}
			path, _ = pathVal.(string)
			if prefixVal, ok := v["prefix"]; ok {
				prefix, _ = prefixVal.(string)
			} else {
				prefix = cfg.Prefix
			}
			if gmVal, ok := v["groupMarkers"]; ok {
				switch gm := gmVal.(type) {
				case []string:
					groupMarkers = append(groupMarkers, gm...)
				case []any:
					for _, item := range gm {
						if gmStr, ok := item.(string); ok {
							groupMarkers = append(groupMarkers, gmStr)
						}
					}
				}
			}
			if len(groupMarkers) == 0 {
				groupMarkers = cfg.GroupMarkers
			}
		default:
			continue
		}

		// Normalize path (handles relative, ~/, npm:, and absolute paths)
		normalizedPath, err := normalizePath(path, state.RootPath)
		if err != nil {
			errs = append(errs, fmt.Errorf("failed to resolve path %s: %w", path, err))
			continue
		}
		path = normalizedPath

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
	// Snapshot config and state separately for semantic clarity
	cfg := s.GetConfig()
	state := s.GetState()

	tokenConfig := TokenFileConfig{
		RootDir:      state.RootPath,
		Patterns:     types.AutoDiscoverPatterns,
		Prefix:       cfg.Prefix,
		GroupMarkers: cfg.GroupMarkers,
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
