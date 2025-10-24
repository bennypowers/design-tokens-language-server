package lsp

import (
	"fmt"
	"os"

	"github.com/bennypowers/design-tokens-language-server/internal/parser/json"
	"github.com/bennypowers/design-tokens-language-server/internal/parser/yaml"
	"github.com/bennypowers/design-tokens-language-server/internal/tokens"
)

// LoadTokenFile loads a token file (JSON or YAML) and adds tokens to the manager
func (s *Server) LoadTokenFile(filepath, prefix string) error {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	var parsedTokens []*tokens.Token

	// Determine parser based on file extension
	if len(filepath) > 5 && filepath[len(filepath)-5:] == ".json" {
		parser := json.NewParser()
		parsedTokens, err = parser.Parse(data, prefix)
	} else if len(filepath) > 5 && (filepath[len(filepath)-5:] == ".yaml" || filepath[len(filepath)-4:] == ".yml") {
		parser := yaml.NewParser()
		parsedTokens, err = parser.Parse(data, prefix)
	} else {
		return fmt.Errorf("unsupported file type: %s", filepath)
	}

	if err != nil {
		return fmt.Errorf("failed to parse token file: %w", err)
	}

	// Convert filepath to URI using the helper from token_loader
	fileURI := pathToURI(filepath)

	// Add all tokens to the manager
	for _, token := range parsedTokens {
		token.FilePath = filepath
		token.DefinitionURI = fileURI
		if err := s.tokens.Add(token); err != nil {
			fmt.Fprintf(os.Stderr, "[DTLS] Warning: failed to add token %s: %v\n", token.Name, err)
		}
	}

	fmt.Fprintf(os.Stderr, "[DTLS] Loaded %d tokens from %s\n", len(parsedTokens), filepath)
	return nil
}

// LoadTokensFromJSON loads tokens from JSON data (for testing)
func (s *Server) LoadTokensFromJSON(data []byte, prefix string) error {
	parser := json.NewParser()
	parsedTokens, err := parser.Parse(data, prefix)
	if err != nil {
		return err
	}

	for _, token := range parsedTokens {
		s.tokens.Add(token)
	}

	return nil
}
