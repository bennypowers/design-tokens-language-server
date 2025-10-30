package lsp

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
)

// shouldSkipDirectory checks if a directory should be skipped during file discovery.
// Returns true for hidden directories (starting with .) and common build/dependency directories.
func shouldSkipDirectory(info os.FileInfo) bool {
	if !info.IsDir() {
		return false
	}

	// Skip hidden directories
	if strings.HasPrefix(info.Name(), ".") {
		return true
	}

	// Skip common build/dependency directories
	skipDirs := []string{"node_modules", "dist", "build"}
	return slices.Contains(skipDirs, info.Name())
}

// matchesAnyPattern checks if a file path matches any of the given glob patterns.
// Returns true if the file matches at least one pattern.
func matchesAnyPattern(relPath string, patterns []string) bool {
	for _, pattern := range patterns {
		matched, err := matchGlobPattern(pattern, relPath)
		if err == nil && matched {
			return true
		}
	}
	return false
}

// collectTokenFiles creates a filepath.Walk callback that collects matching token files.
// Skips directories and hidden files, and matches files against the provided patterns.
func collectTokenFiles(rootDir string, patterns []string, tokenFiles *[]string) filepath.WalkFunc {
	return func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors, continue walking
		}

		// Skip directories that should be excluded
		if shouldSkipDirectory(info) {
			return filepath.SkipDir
		}

		// Skip if it's a directory (but not excluded)
		if info.IsDir() {
			return nil
		}

		// Check if file matches any pattern
		relPath, err := filepath.Rel(rootDir, path)
		if err != nil {
			return nil
		}

		if matchesAnyPattern(relPath, patterns) {
			*tokenFiles = append(*tokenFiles, path)
		}

		return nil
	}
}

// loadDiscoveredFiles loads all discovered token files with the given options.
// Collects and returns any errors that occur during loading.
func (s *Server) loadDiscoveredFiles(tokenFiles []string, opts *TokenFileOptions) error {
	var errs []error
	for _, filePath := range tokenFiles {
		if err := s.LoadTokenFileWithOptions(filePath, opts); err != nil {
			errs = append(errs, fmt.Errorf("failed to load %s: %w", filePath, err))
			continue
		}
		fmt.Fprintf(os.Stderr, "[DTLS] Loaded: %s\n", filePath)
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

// TokenFileConfig represents configuration for loading token files
type TokenFileConfig struct {
	// Patterns to search for token files (glob patterns)
	// Default: ["**/tokens.json", "**/*.tokens.json"]
	Patterns []string

	// CSS variable prefix
	Prefix string

	// GroupMarkers indicate terminal paths that are also groups
	GroupMarkers []string

	// Root directory to search from
	RootDir string
}

// LoadTokenFiles discovers and loads all token files in the workspace
func (s *Server) LoadTokenFiles(config TokenFileConfig) error {
	if config.RootDir == "" {
		return fmt.Errorf("root directory is required")
	}

	// Default patterns if none specified
	if len(config.Patterns) == 0 {
		config.Patterns = []string{
			"**/tokens.json",
			"**/*.tokens.json",
			"**/design-tokens.json",
		}
	}

	fmt.Fprintf(os.Stderr, "[DTLS] Loading token files from: %s\n", config.RootDir)
	fmt.Fprintf(os.Stderr, "[DTLS] Patterns: %v\n", config.Patterns)

	// Collect matching token files
	var tokenFiles []string
	err := filepath.Walk(config.RootDir, collectTokenFiles(config.RootDir, config.Patterns, &tokenFiles))
	if err != nil {
		return fmt.Errorf("failed to walk directory: %w", err)
	}

	fmt.Fprintf(os.Stderr, "[DTLS] Found %d token files\n", len(tokenFiles))

	// Load discovered files
	opts := &TokenFileOptions{
		Prefix:       config.Prefix,
		GroupMarkers: config.GroupMarkers,
	}
	if err := s.loadDiscoveredFiles(tokenFiles, opts); err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "[DTLS] Total tokens loaded: %d\n", s.tokens.Count())
	return nil
}

// matchGlobPattern matches a glob pattern against a path using doublestar
// Supports full glob syntax including ** for recursive directory matching
func matchGlobPattern(pattern, path string) (bool, error) {
	// Normalize path separators to forward slashes for consistent glob matching
	// doublestar.Match expects forward slashes, but Windows paths use backslashes
	normalizedPath := filepath.ToSlash(path)
	return doublestar.Match(pattern, normalizedPath)
}

// ReloadTokens clears and reloads all token files
func (s *Server) ReloadTokens(config TokenFileConfig) error {
	fmt.Fprintf(os.Stderr, "[DTLS] Reloading all tokens\n")

	// Clear existing tokens
	s.tokens.Clear()

	// Reload from files
	return s.LoadTokenFiles(config)
}
