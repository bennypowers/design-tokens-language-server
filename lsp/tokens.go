package lsp

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	asimonimParser "bennypowers.dev/asimonim/parser"
	"bennypowers.dev/asimonim/schema"
	"bennypowers.dev/asimonim/validator"
	"bennypowers.dev/dtls/internal/log"
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
func (s *Server) LoadTokenFile(filePath, prefix string) error {
	return s.LoadTokenFileWithOptions(filePath, &TokenFileOptions{
		Prefix:       prefix,
		GroupMarkers: nil, // Use global defaults
	})
}

// LoadTokenFileWithOptions loads a token file with per-file configuration options
func (s *Server) LoadTokenFileWithOptions(filePath string, opts *TokenFileOptions) error {
	if opts == nil {
		opts = &TokenFileOptions{}
	}

	err := s.loadTokenFileInternal(filePath, opts)
	if err != nil {
		return err
	}

	// Track this file for reload on change (only on successful load)
	// Normalize the path to match IsTokenFile's lookup behavior
	cleanPath := filepath.Clean(filePath)
	s.loadedFilesMu.Lock()
	s.loadedFiles[cleanPath] = opts
	s.loadedFilesMu.Unlock()

	return nil
}

// loadTokenFileInternal loads a token file without tracking it
func (s *Server) loadTokenFileInternal(filePath string, opts *TokenFileOptions) error {
	if opts == nil {
		opts = &TokenFileOptions{}
	}

	// Validate file extension
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".json", ".yaml", ".yml":
		// Supported
	default:
		return fmt.Errorf("unsupported file type %s: %s", ext, filePath)
	}

	// Read file content
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file %s: %w", filePath, err)
	}

	// Parse tokens using asimonim (handles both JSON and YAML)
	parser := asimonimParser.NewJSONParser()
	parsedTokens, err := parser.Parse(data, asimonimParser.Options{
		Prefix:       opts.Prefix,
		GroupMarkers: opts.GroupMarkers,
	})
	if err != nil {
		return err
	}

	// Determine schema version from parsed tokens for validation
	version := schema.Draft
	for _, t := range parsedTokens {
		if t.SchemaVersion != schema.Unknown {
			version = t.SchemaVersion
			break
		}
	}

	// Validate schema consistency
	if validationErrors := validator.ValidateConsistencyWithPath(data, version, filePath); len(validationErrors) > 0 {
		for _, ve := range validationErrors {
			log.Warn("Schema validation: %s", ve.Error())
		}
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
			log.Info("Loaded %d/%d tokens from %s (%d failed)",
				successCount, len(parsedTokens), filePath, len(errs))
		} else {
			log.Info("Failed to load any tokens from %s", filePath)
		}
		return fmt.Errorf("failed to add %d/%d tokens: %w", len(errs), len(parsedTokens), errors.Join(errs...))
	}

	// Log groupMarkers if provided (for future use when parsers support them)
	if len(opts.GroupMarkers) > 0 {
		log.Info("Loaded %d tokens from %s (prefix: %s, groupMarkers: %v)",
			successCount, filePath, opts.Prefix, opts.GroupMarkers)
	} else {
		log.Info("Loaded %d tokens from %s", successCount, filePath)
	}
	return nil
}

// LoadTokensFromJSON loads tokens from JSON data (for testing)
// errors from this function should be presented to the user via window/logMessage
// further up the call stack
func (s *Server) LoadTokensFromJSON(data []byte, prefix string) error {
	parser := asimonimParser.NewJSONParser()
	parsedTokens, err := parser.Parse(data, asimonimParser.Options{
		Prefix: prefix,
	})
	if err != nil {
		return err
	}

	// Determine schema version from parsed tokens for validation
	version := schema.Draft
	for _, t := range parsedTokens {
		if t.SchemaVersion != schema.Unknown {
			version = t.SchemaVersion
			break
		}
	}

	// Validate schema consistency
	if validationErrors := validator.ValidateConsistency(data, version); len(validationErrors) > 0 {
		for _, ve := range validationErrors {
			log.Warn("Schema validation: %s", ve.Error())
		}
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

	// Resolve all aliases after loading tokens
	s.ResolveAllTokens()

	return nil
}

// LoadTokensFromDocumentContent loads tokens from a document's content into the token manager.
// This is used when opening a file with Design Tokens schema that isn't configured in tokensFiles.
// The uri is used to set the DefinitionURI for go-to-definition support.
func (s *Server) LoadTokensFromDocumentContent(uri, languageID, content string) error {
	// Only parse JSON and YAML files
	switch languageID {
	case "json", "yaml":
		// Supported
	default:
		// Not a supported token file format
		return nil
	}

	// Parse tokens using asimonim (handles both JSON and YAML)
	parser := asimonimParser.NewJSONParser()
	contentBytes := []byte(content)
	parsedTokens, err := parser.Parse(contentBytes, asimonimParser.Options{})
	if err != nil {
		return fmt.Errorf("failed to parse tokens from document: %w", err)
	}

	// Convert URI to file path for FilePath field
	filePath := uriutil.URIToPath(uri)

	// Determine schema version from parsed tokens for validation
	version := schema.Draft
	for _, t := range parsedTokens {
		if t.SchemaVersion != schema.Unknown {
			version = t.SchemaVersion
			break
		}
	}

	// Validate schema consistency
	if validationErrors := validator.ValidateConsistencyWithPath(contentBytes, version, filePath); len(validationErrors) > 0 {
		for _, ve := range validationErrors {
			log.Warn("Schema validation: %s", ve.Error())
		}
	}

	var errs []error
	successCount := 0
	for _, token := range parsedTokens {
		token.FilePath = filePath
		token.DefinitionURI = uri
		if err := s.tokens.Add(token); err != nil {
			errs = append(errs, fmt.Errorf("failed to add token %s: %w", token.Name, err))
		} else {
			successCount++
		}
	}

	if successCount > 0 {
		log.Info("Auto-loaded %d tokens from %s", successCount, uri)
		// Resolve all aliases after loading tokens
		s.ResolveAllTokens()
	}

	if len(errs) > 0 {
		return fmt.Errorf("failed to add %d/%d tokens: %w", len(errs), len(parsedTokens), errors.Join(errs...))
	}
	return nil
}
