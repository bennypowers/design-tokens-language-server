package lsp

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"bennypowers.dev/dtls/lsp/types"
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

// discoverTokenFiles discovers token files using auto-discovery patterns
// Returns a map of file paths to prefixes (empty string prefix for auto-discovered files)
func (s *Server) discoverTokenFiles() (map[string]string, error) {
	if s.rootPath == "" {
		return nil, nil
	}

	tokenConfig := TokenFileConfig{
		RootDir: s.rootPath,
		Patterns: []string{
			"**/tokens.json",
			"**/*.tokens.json",
			"**/design-tokens.json",
		},
		Prefix: s.config.Prefix,
	}

	discovered := make(map[string]string)

	// Walk the directory tree to find matching files
	err := filepath.Walk(tokenConfig.RootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors, continue walking
		}

		// Skip directories and hidden files/directories
		if info.IsDir() {
			if strings.HasPrefix(info.Name(), ".") {
				return filepath.SkipDir
			}
			// Skip node_modules and other common directories
			if info.Name() == "node_modules" || info.Name() == "dist" || info.Name() == "build" {
				return filepath.SkipDir
			}
			return nil
		}

		// Check if file matches any pattern
		relPath, err := filepath.Rel(tokenConfig.RootDir, path)
		if err != nil {
			return nil
		}

		for _, pattern := range tokenConfig.Patterns {
			matched, err := matchGlobPattern(pattern, relPath)
			if err == nil && matched {
				discovered[path] = tokenConfig.Prefix
				break
			}
		}

		return nil
	})

	return discovered, err
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
	for path, opts := range s.loadedFiles {
		filesToReload[path] = opts
	}
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
