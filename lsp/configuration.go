package lsp

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"time"

	"bennypowers.dev/asimonim/load"
	"bennypowers.dev/asimonim/resolver"
	"bennypowers.dev/asimonim/schema"
	"bennypowers.dev/asimonim/specifier"
	"bennypowers.dev/dtls/internal/log"
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
		log.Info("Loaded prefix from package.json: %s\n", pkgConfig.Prefix)
	}

	// Allow package.json to override if groupMarkers are still at defaults
	if isGroupMarkersDefault(s.config.GroupMarkers, defaults.GroupMarkers) && len(pkgConfig.GroupMarkers) > 0 {
		s.config.GroupMarkers = pkgConfig.GroupMarkers
		log.Info("Loaded groupMarkers from package.json: %v\n", pkgConfig.GroupMarkers)
	}

	if len(s.config.TokensFiles) == 0 && len(pkgConfig.TokensFiles) > 0 {
		s.config.TokensFiles = pkgConfig.TokensFiles
		log.Info("Loaded %d tokensFiles from config", len(pkgConfig.TokensFiles))
	}

	if !s.config.NetworkFallback && pkgConfig.NetworkFallback {
		s.config.NetworkFallback = true
		log.Info("Loaded networkFallback from package.json: %v", pkgConfig.NetworkFallback)
	}

	if s.config.NetworkTimeout == 0 && pkgConfig.NetworkTimeout != 0 {
		s.config.NetworkTimeout = pkgConfig.NetworkTimeout
		log.Info("Loaded networkTimeout from package.json: %d", pkgConfig.NetworkTimeout)
	}

	if s.config.CDN == "" && pkgConfig.CDN != "" {
		s.config.CDN = pkgConfig.CDN
		log.Info("Loaded cdn from package.json: %s", pkgConfig.CDN)
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
		log.Info("Loading %d token files from config", len(cfg.TokensFiles))
		// Clear existing tokens before loading configured files
		s.tokens.Clear()
		err := s.loadExplicitTokenFiles()
		if err != nil {
			return err
		}
		// Resolve all aliases after loading all tokens
		s.ResolveAllTokens()
		log.Info("Loaded %d tokens total", s.tokens.Count())
		return nil
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

// ResolveAllTokens resolves all alias references in the loaded tokens.
// This should be called after all token files are loaded.
func (s *Server) ResolveAllTokens() {
	tokens := s.tokens.GetAll()
	if len(tokens) == 0 {
		return
	}

	// Determine the schema version to use for resolution
	// Use the first token's schema version as a heuristic
	// (in practice, all tokens in a file should have the same version)
	version := schema.Draft
	for _, t := range tokens {
		if t.SchemaVersion != schema.Unknown {
			version = t.SchemaVersion
			break
		}
	}

	if err := resolver.ResolveAliases(tokens, version); err != nil {
		log.Warn("Failed to resolve token aliases: %v", err)
	}
}

// validateTokenFilePath validates that a token file path is not empty.
// Returns an error if the path is empty, nil otherwise.
func validateTokenFilePath(path, context string) error {
	if path == "" {
		return fmt.Errorf("%s must not be empty", context)
	}
	return nil
}

// parseGroupMarkersFromItem extracts groupMarkers from a map[string]any item.
// Handles both []string and []any types, falling back to defaults if not present or empty.
func parseGroupMarkersFromItem(itemMap map[string]any, defaultGroupMarkers []string) []string {
	gmVal, ok := itemMap["groupMarkers"]
	if !ok {
		return defaultGroupMarkers
	}

	var groupMarkers []string
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

	if len(groupMarkers) == 0 {
		return defaultGroupMarkers
	}
	return groupMarkers
}

// parseTokenFileItem parses a token file item (string or map[string]any) into path and options.
// Returns path, prefix, groupMarkers, and error.
func parseTokenFileItem(item any, defaultPrefix string, defaultGroupMarkers []string) (path, prefix string, groupMarkers []string, err error) {
	switch v := item.(type) {
	case string:
		if err := validateTokenFilePath(v, "token file path"); err != nil {
			return "", "", nil, err
		}
		return v, defaultPrefix, defaultGroupMarkers, nil

	case map[string]any:
		// Extract path
		pathVal, ok := v["path"]
		if !ok {
			return "", "", nil, fmt.Errorf("token file entry missing required 'path' field: %v", v)
		}
		path, _ = pathVal.(string)
		if err := validateTokenFilePath(path, "token file entry 'path'"); err != nil {
			return "", "", nil, fmt.Errorf("%w: %v", err, v)
		}

		// Extract prefix (optional)
		if prefixVal, ok := v["prefix"]; ok {
			prefix, _ = prefixVal.(string)
		} else {
			prefix = defaultPrefix
		}

		// Extract groupMarkers (optional)
		groupMarkers = parseGroupMarkersFromItem(v, defaultGroupMarkers)

		return path, prefix, groupMarkers, nil

	default:
		// Silently skip unsupported types (matches current behavior)
		return "", "", nil, nil
	}
}

// loadTokenFileAndLog loads a token file with options.
// Returns an error if loading fails. Success is logged by parseAndAddTokens.
func (s *Server) loadTokenFileAndLog(path string, opts *TokenFileOptions) error {
	return s.LoadTokenFileWithOptions(path, opts)
}

// networkTimeout returns the configured timeout duration for CDN requests,
// falling back to load.DefaultTimeout if not configured.
func networkTimeout(cfg types.ServerConfig) time.Duration {
	if cfg.NetworkTimeout > 0 {
		return time.Duration(cfg.NetworkTimeout) * time.Second
	}
	return load.DefaultTimeout
}

// loadExplicitTokenFiles loads tokens from explicitly configured files
func (s *Server) loadExplicitTokenFiles() error {
	// Snapshot config and state separately for semantic clarity
	cfg := s.GetConfig()
	state := s.GetState()

	// Create fetcher once if network fallback is enabled
	var fetcher load.Fetcher
	if cfg.NetworkFallback {
		fetcher = load.NewHTTPFetcher(load.DefaultMaxSize)
	}

	var errs []error

	for _, item := range cfg.TokensFiles {
		// Parse the item - can be string or object
		path, prefix, groupMarkers, err := parseTokenFileItem(item, cfg.Prefix, cfg.GroupMarkers)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		if path == "" {
			// Skip items that returned empty (default case in parseTokenFileItem)
			continue
		}

		opts := &TokenFileOptions{
			Prefix:       prefix,
			GroupMarkers: groupMarkers,
		}

		// Normalize path (handles relative, ~/, npm:, and absolute paths)
		normalizedPath, err := normalizePath(path, state.RootPath)
		if err != nil {
			// Try CDN fallback for package specifiers when local resolution fails
			if fetcher != nil && specifier.IsPackageSpecifier(path) {
				count, cdnErr := s.loadFromCDN(fetcher, path, opts, cfg)
				if cdnErr != nil && count == 0 {
					errs = append(errs, fmt.Errorf("failed to resolve path %s: %w (CDN fallback also failed: %v)", path, err, cdnErr))
				} else if cdnErr != nil {
					log.Warn("CDN fallback for %s loaded %d tokens but had errors: %v", path, count, cdnErr)
				}
				continue
			}
			errs = append(errs, fmt.Errorf("failed to resolve path %s: %w", path, err))
			continue
		}

		// Load the file with per-file options and log results
		if err := s.loadTokenFileAndLog(normalizedPath, opts); err != nil {
			errs = append(errs, fmt.Errorf("failed to load %s: %w", normalizedPath, err))
			continue
		}
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

// loadFromCDN fetches token data from a CDN for a package specifier and adds the tokens.
// Returns the number of tokens successfully added and any error.
func (s *Server) loadFromCDN(fetcher load.Fetcher, specPath string, opts *TokenFileOptions, cfg types.ServerConfig) (int, error) {
	cdnURL, ok := specifier.CDNURL(specPath, specifier.CDN(cfg.CDN))
	if !ok {
		return 0, fmt.Errorf("cannot determine CDN URL for %s", specPath)
	}

	ctx, cancel := context.WithTimeout(context.Background(), networkTimeout(cfg))
	defer cancel()

	content, err := fetcher.Fetch(ctx, cdnURL)
	if err != nil {
		return 0, fmt.Errorf("CDN fetch failed for %s: %w", cdnURL, err)
	}

	return s.parseAndAddTokens(content, "", cdnURL, opts)
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
			log.Info("Reloaded %s (prefix: %s, groupMarkers: %v)\n", path, opts.Prefix, opts.GroupMarkers)
		} else {
			log.Info("Reloaded %s (prefix: %s)\n", path, opts.Prefix)
		}
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	// Resolve all aliases after reloading all tokens
	s.ResolveAllTokens()

	return nil
}
