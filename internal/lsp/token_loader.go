package lsp

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// TokenFileConfig represents configuration for loading token files
type TokenFileConfig struct {
	// Patterns to search for token files (glob patterns)
	// Default: ["**/tokens.json", "**/*.tokens.json"]
	Patterns []string

	// CSS variable prefix
	Prefix string

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

	var tokenFiles []string

	// Walk the directory tree
	err := filepath.Walk(config.RootDir, func(path string, info os.FileInfo, err error) error {
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
		relPath, err := filepath.Rel(config.RootDir, path)
		if err != nil {
			return nil
		}

		for _, pattern := range config.Patterns {
			matched, err := matchGlobPattern(pattern, relPath)
			if err == nil && matched {
				tokenFiles = append(tokenFiles, path)
				break
			}
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to walk directory: %w", err)
	}

	fmt.Fprintf(os.Stderr, "[DTLS] Found %d token files\n", len(tokenFiles))

	// Load each token file
	var errs []error
	for _, filePath := range tokenFiles {
		if err := s.LoadTokenFile(filePath, config.Prefix); err != nil {
			errs = append(errs, fmt.Errorf("failed to load %s: %w", filePath, err))
			continue
		}
		fmt.Fprintf(os.Stderr, "[DTLS] Loaded: %s\n", filePath)
	}

	fmt.Fprintf(os.Stderr, "[DTLS] Total tokens loaded: %d\n", s.tokens.Count())

	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

// matchGlobPattern matches a glob pattern against a path
// Simplified glob matching: ** matches any path, * matches any file
func matchGlobPattern(pattern, path string) (bool, error) {
	// Convert pattern to filepath pattern
	// Handle ** for directory recursion
	if strings.Contains(pattern, "**") {
		// **/*.json should match any path ending in .json
		suffix := strings.TrimPrefix(pattern, "**")
		suffix = strings.TrimPrefix(suffix, "/")

		if strings.HasSuffix(path, suffix) ||
		   strings.Contains(path, suffix) {
			// Do a proper filepath match for the suffix part
			matched, err := filepath.Match(suffix, filepath.Base(path))
			if err != nil {
				return false, err
			}
			if matched {
				return true, nil
			}

			// Also try matching the full remaining path
			return strings.HasSuffix(path, suffix), nil
		}
		return false, nil
	}

	// Regular glob pattern
	return filepath.Match(pattern, path)
}

// pathToURI converts a file path to a file:// URI
func pathToURI(path string) string {
	// Convert to absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		absPath = path
	}

	// Convert to forward slashes
	absPath = filepath.ToSlash(absPath)

	// Add file:// prefix
	if !strings.HasPrefix(absPath, "/") {
		absPath = "/" + absPath
	}

	return "file://" + absPath
}

// uriToPath converts a file:// URI to a file path
func uriToPath(uri string) string {
	// Remove file:// prefix
	path := strings.TrimPrefix(uri, "file://")

	// Convert forward slashes to OS-specific separators
	path = filepath.FromSlash(path)

	// On Windows, remove leading slash from /C:/path
	if len(path) > 2 && path[0] == filepath.Separator && path[2] == ':' {
		path = path[1:]
	}

	return path
}

// ReloadTokens clears and reloads all token files
func (s *Server) ReloadTokens(config TokenFileConfig) error {
	fmt.Fprintf(os.Stderr, "[DTLS] Reloading all tokens\n")

	// Clear existing tokens
	s.tokens.Clear()

	// Reload from files
	return s.LoadTokenFiles(config)
}
