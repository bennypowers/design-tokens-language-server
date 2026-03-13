package lsp

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"bennypowers.dev/asimonim/config"
	"bennypowers.dev/asimonim/load"
	"bennypowers.dev/asimonim/specifier"
	"bennypowers.dev/dtls/internal/log"
)

// isResolverDocument checks if JSON data represents a resolver document
// by looking for the "resolutionOrder" field at the root.
func isResolverDocument(data []byte) bool {
	var doc map[string]json.RawMessage
	if err := json.Unmarshal(data, &doc); err != nil {
		return false
	}
	_, hasResolutionOrder := doc["resolutionOrder"]
	return hasResolutionOrder
}

// loadResolverDocument reads a resolver document and loads its source token files.
func (s *Server) loadResolverDocument(resolverPath string, opts *TokenFileOptions) error {
	data, err := os.ReadFile(filepath.Clean(resolverPath))
	if err != nil {
		return fmt.Errorf("failed to read resolver document %s: %w", resolverPath, err)
	}

	resolverDir := filepath.Dir(resolverPath)
	sourcePaths, err := config.ExtractSourcePaths(data, resolverDir)
	if err != nil {
		return fmt.Errorf("failed to extract sources from resolver %s: %w", resolverPath, err)
	}

	log.Info("Resolver %s has %d source files", resolverPath, len(sourcePaths))

	cfg := s.GetConfig()
	state := s.GetState()

	// Create fetcher once if network fallback is enabled
	var fetcher load.Fetcher
	if cfg.NetworkFallback {
		fetcher = load.NewHTTPFetcher(load.DefaultMaxSize)
	}

	var errs []error
	for _, srcPath := range sourcePaths {
		normalizedPath, err := normalizePath(srcPath, state.RootPath)
		if err != nil {
			// Try CDN fallback for package specifiers
			if fetcher != nil && specifier.IsPackageSpecifier(srcPath) {
				count, cdnErr := s.loadFromCDN(fetcher, srcPath, opts, cfg)
				if cdnErr != nil && count == 0 {
					errs = append(errs, fmt.Errorf("failed to resolve resolver source %s: %w (CDN fallback also failed: %v)", srcPath, err, cdnErr))
				} else if cdnErr != nil {
					log.Warn("CDN fallback for resolver source %s loaded %d tokens but had errors: %v", srcPath, count, cdnErr)
				}
				continue
			}
			errs = append(errs, fmt.Errorf("failed to resolve resolver source %s: %w", srcPath, err))
			continue
		}

		if err := s.loadTokenFileAndLog(normalizedPath, opts); err != nil {
			errs = append(errs, fmt.Errorf("failed to load resolver source %s: %w", normalizedPath, err))
		}
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}
