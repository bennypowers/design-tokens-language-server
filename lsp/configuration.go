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

// LoadPackageJsonConfig reads and merges configuration from package.json
// Client-sent configuration takes precedence over package.json
func (s *Server) LoadPackageJsonConfig() error {
	state := s.GetState()
	if state.RootPath == "" {
		return nil // No workspace, nothing to load
	}

	pkgConfig, err := ReadPackageJsonConfig(state.RootPath)
	if err != nil {
		return err
	}

	if pkgConfig == nil {
		return nil // No package.json config, not an error
	}

	// Merge with existing config (client config takes precedence)
	s.configMu.Lock()
	defer s.configMu.Unlock()

	// Only set fields if not already configured by client
	// For fields with defaults, we check if they're still at default values
	defaults := types.DefaultConfig()

	if s.config.Prefix == "" && pkgConfig.Prefix != "" {
		s.config.Prefix = pkgConfig.Prefix
		fmt.Fprintf(os.Stderr, "[DTLS] Loaded prefix from package.json: %s\n", pkgConfig.Prefix)
	}

	// Allow package.json to override if groupMarkers are still at defaults
	if isGroupMarkersDefault(s.config.GroupMarkers, defaults.GroupMarkers) && len(pkgConfig.GroupMarkers) > 0 {
		s.config.GroupMarkers = pkgConfig.GroupMarkers
		fmt.Fprintf(os.Stderr, "[DTLS] Loaded groupMarkers from package.json: %v\n", pkgConfig.GroupMarkers)
	}

	if len(s.config.TokensFiles) == 0 && len(pkgConfig.TokensFiles) > 0 {
		s.config.TokensFiles = pkgConfig.TokensFiles
		fmt.Fprintf(os.Stderr, "[DTLS] Loaded tokensFiles from package.json: %v\n", pkgConfig.TokensFiles)
	}

	return nil
}

// isGroupMarkersDefault checks if group markers are equal to the default values
func isGroupMarkersDefault(current, defaults []string) bool {
	if len(current) != len(defaults) {
		return false
	}
	for i := range current {
		if current[i] != defaults[i] {
			return false
		}
	}
	return true
}

// GetState returns a snapshot of runtime state (NOT configuration)
// For configuration, use GetConfig() separately.
// This separation allows clear distinction between user configuration and runtime state.
func (s *Server) GetState() types.ServerState {
	s.configMu.RLock()
	defer s.configMu.RUnlock()
	return types.ServerState{
		RootPath: s.rootPath,
	}
}

// SetConfig updates the server configuration
func (s *Server) SetConfig(config types.ServerConfig) {
	s.configMu.Lock()
	defer s.configMu.Unlock()
	s.config = config
}

// loadTokensFromConfig loads tokens based on current configuration
// Matches TypeScript behavior: explicit configuration only, no auto-discovery
func (s *Server) LoadTokensFromConfig() error {
	cfg := s.GetConfig()

	// If tokensFiles is explicitly provided and non-empty, load those files
	if len(cfg.TokensFiles) > 0 {
		// Clear existing tokens before loading configured files
		s.tokens.Clear()
		return s.loadExplicitTokenFiles()
	}

	// If tokensFiles is empty or nil, check if we have programmatically loaded files to reload
	// (This supports test scenarios where files are loaded via LoadTokenFile)
	s.loadedFilesMu.RLock()
	hasLoadedFiles := len(s.loadedFiles) > 0
	s.loadedFilesMu.RUnlock()

	if hasLoadedFiles {
		return s.reloadPreviouslyLoadedFiles()
	}

	// No configuration and no previously loaded files: do nothing
	// Users must explicitly configure tokensFiles
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

// reloadPreviouslyLoadedFiles reloads all files that were previously loaded
// This is used for programmatic loading (e.g., tests using LoadTokenFile)
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
