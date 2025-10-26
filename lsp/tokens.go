package lsp

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bennypowers/design-tokens-language-server/internal/parser/json"
	"github.com/bennypowers/design-tokens-language-server/internal/parser/yaml"
	"github.com/bennypowers/design-tokens-language-server/internal/tokens"
)

// LoadTokenFile loads a token file (JSON or YAML) and adds tokens to the manager
func (s *Server) LoadTokenFile(filepath, prefix string) error {
	err := s.loadTokenFileInternal(filepath, prefix)
	if err != nil {
		return err
	}

	// Track this file for reload on change (only on successful load)
	s.loadedFiles[filepath] = prefix

	return nil
}

// loadTokenFileInternal loads a token file without tracking it
func (s *Server) loadTokenFileInternal(filePath, prefix string) error {
	var parsedTokens []*tokens.Token
	var err error

	// Determine parser based on file extension
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".json":
		parser := json.NewParser()
		parsedTokens, err = parser.ParseFile(filePath, prefix)
	case ".yaml", ".yml":
		parser := yaml.NewParser()
		parsedTokens, err = parser.ParseFile(filePath, prefix)
	default:
		return fmt.Errorf("unsupported file type %s: %s", ext, filePath)
	}

	if err != nil {
		return err // Error already wrapped by parser
	}

	// Convert filepath to URI using the helper from token_loader
	fileURI := pathToURI(filePath)

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

	fmt.Fprintf(os.Stderr, "[DTLS] Loaded %d tokens from %s\n", successCount, filePath)
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
