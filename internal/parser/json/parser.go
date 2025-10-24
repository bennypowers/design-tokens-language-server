package json

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/bennypowers/design-tokens-language-server/internal/tokens"
	"github.com/tidwall/jsonc"
)

// Parser handles parsing DTCG-compliant JSON token files
type Parser struct{}

// NewParser creates a new JSON token parser
func NewParser() *Parser {
	return &Parser{}
}

// Parse parses JSON token data and returns a list of tokens
// Supports JSONC (JSON with comments)
func (p *Parser) Parse(data []byte, prefix string) ([]*tokens.Token, error) {
	// Remove comments using jsonc
	cleanJSON := jsonc.ToJSON(data)

	// Parse JSON
	var rawData map[string]interface{}
	if err := json.Unmarshal(cleanJSON, &rawData); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	// Extract tokens
	result := []*tokens.Token{}
	p.extractTokens(rawData, "", prefix, &result)

	return result, nil
}

// extractTokens recursively extracts tokens from the JSON structure
func (p *Parser) extractTokens(data map[string]interface{}, path, prefix string, result *[]*tokens.Token) {
	p.extractTokensWithPath(data, []string{}, path, prefix, result)
}

// extractTokensWithPath recursively extracts tokens tracking the JSON path
func (p *Parser) extractTokensWithPath(data map[string]interface{}, jsonPath []string, path, prefix string, result *[]*tokens.Token) {
	for key, value := range data {
		valueMap, isMap := value.(map[string]interface{})
		if !isMap {
			continue
		}

		currentPath := append(jsonPath, key)

		// Check if this is a token (has $value)
		if dollarValue, hasValue := valueMap["$value"]; hasValue {
			token := p.createToken(key, path, dollarValue, valueMap, prefix, currentPath)
			*result = append(*result, token)
		} else {
			// This is a group, recurse into it
			newPath := path
			if path == "" {
				newPath = key
			} else {
				newPath = path + "-" + key
			}
			p.extractTokensWithPath(valueMap, currentPath, newPath, prefix, result)
		}
	}
}

// createToken creates a Token from the parsed data
func (p *Parser) createToken(key, path string, value interface{}, data map[string]interface{}, prefix string, jsonPath []string) *tokens.Token {
	// Build token name from path
	name := path
	if name == "" {
		name = key
	} else {
		name = path + "-" + key
	}

	// Build reference format (e.g., "{color.primary}")
	reference := "{" + strings.Join(jsonPath, ".") + "}"

	token := &tokens.Token{
		Name:      name,
		Value:     fmt.Sprintf("%v", value),
		Prefix:    prefix,
		Path:      jsonPath,
		Reference: reference,
	}

	// Extract $type
	if tokenType, ok := data["$type"].(string); ok {
		token.Type = tokenType
	}

	// Extract $description
	if desc, ok := data["$description"].(string); ok {
		token.Description = desc
	}

	// Extract $deprecated flag (can be bool or string with message)
	if deprecated, ok := data["$deprecated"].(bool); ok {
		token.Deprecated = deprecated
	} else if depMsg, ok := data["$deprecated"].(string); ok {
		token.Deprecated = true
		token.DeprecationMessage = depMsg
	}

	// Extract $extensions
	if extensions, ok := data["$extensions"].(map[string]interface{}); ok {
		token.Extensions = extensions
	}

	return token
}

// ParseFile parses a JSON file and returns tokens
func (p *Parser) ParseFile(filename string, prefix string) ([]*tokens.Token, error) {
	// This will be implemented when we add file I/O
	// For now, keeping it simple for testing
	return nil, fmt.Errorf("ParseFile not yet implemented")
}

// NormalizeTokenName converts a token path to a CSS variable name format
// e.g., "color.brand.primary" -> "color-brand-primary"
func NormalizeTokenName(name string) string {
	// Replace dots with hyphens
	name = strings.ReplaceAll(name, ".", "-")
	// Convert to lowercase
	name = strings.ToLower(name)
	return name
}
