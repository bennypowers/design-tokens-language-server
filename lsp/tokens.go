package lsp

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	asimonimParser "bennypowers.dev/asimonim/parser"
	"bennypowers.dev/asimonim/schema"
	asimonimToken "bennypowers.dev/asimonim/token"
	"bennypowers.dev/asimonim/validator"
	"bennypowers.dev/dtls/internal/log"
	"bennypowers.dev/dtls/internal/uriutil"
)

// detectSchemaVersion returns the schema version from the first token that has one set.
// Falls back to schema.Draft if no token has a schema version.
func detectSchemaVersion(tokens []*asimonimToken.Token) schema.Version {
	for _, t := range tokens {
		if t.SchemaVersion != schema.Unknown {
			return t.SchemaVersion
		}
	}
	return schema.Draft
}

// logValidationErrors logs schema validation errors as warnings.
func logValidationErrors(validationErrors []validator.ValidationError) {
	for _, ve := range validationErrors {
		log.Warn("Schema validation: %s", ve.Error())
	}
}

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
	data, err := os.ReadFile(filepath.Clean(filePath))
	if err != nil {
		return fmt.Errorf("failed to read file %s: %w", filePath, err)
	}

	// Convert filepath to URI
	fileURI := uriutil.PathToURI(filePath)

	_, err = s.parseAndAddTokens(data, filePath, fileURI, opts)
	return err
}

// parseAndAddTokens parses token data, validates it, and adds the tokens to the manager.
// filePath and fileURI are set on each token for definition tracking.
// Returns the number of successfully added tokens.
func (s *Server) parseAndAddTokens(data []byte, filePath, fileURI string, opts *TokenFileOptions) (int, error) {
	if opts == nil {
		opts = &TokenFileOptions{}
	}

	// Parse tokens using asimonim (handles both JSON and YAML)
	parser := asimonimParser.NewJSONParser()
	parsedTokens, err := parser.Parse(data, asimonimParser.Options{
		Prefix:       opts.Prefix,
		GroupMarkers: opts.GroupMarkers,
	})
	if err != nil {
		return 0, err
	}

	// Validate schema consistency
	version := detectSchemaVersion(parsedTokens)
	if filePath != "" {
		if validationErrors := validator.ValidateConsistencyWithPath(data, version, filePath); len(validationErrors) > 0 {
			logValidationErrors(validationErrors)
		}
	} else {
		if validationErrors := validator.ValidateConsistency(data, version); len(validationErrors) > 0 {
			logValidationErrors(validationErrors)
		}
	}

	// Add all tokens to the manager
	var errs []error
	successCount := 0
	source := filePath
	if source == "" {
		source = fileURI
	}
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
				successCount, len(parsedTokens), source, len(errs))
		} else {
			log.Info("Failed to load any tokens from %s", source)
		}
		return successCount, fmt.Errorf("failed to add %d/%d tokens: %w", len(errs), len(parsedTokens), errors.Join(errs...))
	}

	// Log groupMarkers if provided (for future use when parsers support them)
	if len(opts.GroupMarkers) > 0 {
		log.Info("Loaded %d tokens from %s (prefix: %s, groupMarkers: %v)",
			successCount, source, opts.Prefix, opts.GroupMarkers)
	} else {
		log.Info("Loaded %d tokens from %s", successCount, source)
	}
	return successCount, nil
}

// LoadTokensFromJSON loads tokens from JSON data (for testing)
// errors from this function should be presented to the user via window/logMessage
// further up the call stack
func (s *Server) LoadTokensFromJSON(data []byte, prefix string) error {
	_, err := s.parseAndAddTokens(data, "", "", &TokenFileOptions{Prefix: prefix})
	if err != nil {
		return err
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

	// Convert URI to file path for FilePath field
	filePath := uriutil.URIToPath(uri)

	successCount, err := s.parseAndAddTokens([]byte(content), filePath, uri, &TokenFileOptions{})
	if successCount > 0 {
		// Resolve all aliases after loading tokens
		s.ResolveAllTokens()
	}

	return err
}
