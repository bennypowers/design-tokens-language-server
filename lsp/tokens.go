package lsp

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"bennypowers.dev/dtls/internal/parser/json"
	"bennypowers.dev/dtls/internal/parser/yaml"
	"bennypowers.dev/dtls/internal/tokens"
	"bennypowers.dev/dtls/internal/uriutil"
)

// TokenFileOptions holds per-file configuration for token loading
type TokenFileOptions struct {
	// Prefix is the CSS variable prefix for tokens in this file
	Prefix string

	// GroupMarkers indicate terminal paths that are also groups
	// e.g., a token named "color" that is also the parent of "color.primary"
	GroupMarkers []string
}

// LoadTokenFile loads a token file (JSON or YAML) and adds tokens to the manager
// This is a convenience wrapper around LoadTokenFileWithOptions for backward compatibility
func (s *Server) LoadTokenFile(filepath, prefix string) error {
	return s.LoadTokenFileWithOptions(filepath, &TokenFileOptions{
		Prefix:       prefix,
		GroupMarkers: nil, // Use global defaults
	})
}

// LoadTokenFileWithOptions loads a token file with per-file configuration options
func (s *Server) LoadTokenFileWithOptions(filepath string, opts *TokenFileOptions) error {
	if opts == nil {
		opts = &TokenFileOptions{}
	}

	err := s.loadTokenFileInternal(filepath, opts)
	if err != nil {
		return err
	}

	// Track this file for reload on change (only on successful load)
	s.loadedFilesMu.Lock()
	s.loadedFiles[filepath] = opts
	s.loadedFilesMu.Unlock()

	return nil
}

// loadTokenFileInternal loads a token file without tracking it
func (s *Server) loadTokenFileInternal(filePath string, opts *TokenFileOptions) error {
	if opts == nil {
		opts = &TokenFileOptions{}
	}

	var parsedTokens []*tokens.Token
	var err error

	// Determine parser based on file extension
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".json":
		parser := json.NewParser()
		parsedTokens, err = parser.ParseFileWithGroupMarkers(filePath, opts.Prefix, opts.GroupMarkers)
	case ".yaml", ".yml":
		parser := yaml.NewParser()
		parsedTokens, err = parser.ParseFileWithGroupMarkers(filePath, opts.Prefix, opts.GroupMarkers)
	default:
		return fmt.Errorf("unsupported file type %s: %s", ext, filePath)
	}

	if err != nil {
		return err // Error already wrapped by parser
	}

	// Convert filepath to URI
	fileURI := uriutil.PathToURI(filePath)

	// Add all tokens to the manager
	var errs []error
	successCount := 0
	for _, token := range parsedTokens {
		token.FilePath = filePath
		token.DefinitionURI = fileURI
		if err := s.tokens.Add(token); err != nil {
			errs = append(errs, fmt.Errorf("failed to add token %s: %w", token.Name, err))
		} else {
			successCount++
		}
	}

	if len(errs) > 0 {
		// Report partial success if some tokens loaded
		if successCount > 0 {
			fmt.Fprintf(os.Stderr, "[DTLS] Loaded %d/%d tokens from %s (%d failed)\n",
				successCount, len(parsedTokens), filePath, len(errs))
		} else {
			fmt.Fprintf(os.Stderr, "[DTLS] Failed to load any tokens from %s\n", filePath)
		}
		return fmt.Errorf("failed to add %d/%d tokens: %w", len(errs), len(parsedTokens), errors.Join(errs...))
	}

	// Log groupMarkers if provided (for future use when parsers support them)
	if len(opts.GroupMarkers) > 0 {
		fmt.Fprintf(os.Stderr, "[DTLS] Loaded %d tokens from %s (prefix: %s, groupMarkers: %v)\n",
			successCount, filePath, opts.Prefix, opts.GroupMarkers)
	} else {
		fmt.Fprintf(os.Stderr, "[DTLS] Loaded %d tokens from %s\n", successCount, filePath)
	}
	return nil
}

// LoadTokensFromJSON loads tokens from JSON data (for testing)
// errors from this function should be presented to the user via window/logMessage
// further up the call stack
func (s *Server) LoadTokensFromJSON(data []byte, prefix string) error {
	parser := json.NewParser()
	parsedTokens, err := parser.Parse(data, prefix)
	if err != nil {
		return err
	}

	var errs []error
	successCount := 0
	for _, token := range parsedTokens {
		if err := s.tokens.Add(token); err != nil {
			errs = append(errs, fmt.Errorf("failed to add token %s: %w", token.Name, err))
		} else {
			successCount++
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("failed to add %d/%d tokens: %w", len(errs), len(parsedTokens), errors.Join(errs...))
	}
	return nil
}
