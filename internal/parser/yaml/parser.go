package yaml

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"bennypowers.dev/dtls/internal/tokens"
	"gopkg.in/yaml.v3"
)

// Parser handles parsing DTCG-compliant YAML token files
type Parser struct{}

// NewParser creates a new YAML token parser
func NewParser() *Parser {
	return &Parser{}
}

// Parse parses YAML token data and returns a list of tokens
func (p *Parser) Parse(data []byte, prefix string) ([]*tokens.Token, error) {
	return p.ParseWithGroupMarkers(data, prefix, nil)
}

// ParseWithGroupMarkers parses YAML token data with support for group markers
// Group markers are token names that can be both a token (with $value) and a group (with children)
func (p *Parser) ParseWithGroupMarkers(data []byte, prefix string, groupMarkers []string) ([]*tokens.Token, error) {
	// Parse YAML
	var rawData map[string]interface{}
	if err := yaml.Unmarshal(data, &rawData); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	// Extract tokens
	result := []*tokens.Token{}
	p.extractTokensWithGroupMarkers(rawData, "", prefix, groupMarkers, &result)

	return result, nil
}

// extractTokens recursively extracts tokens from the YAML structure
func (p *Parser) extractTokens(data map[string]interface{}, path, prefix string, result *[]*tokens.Token) {
	p.extractTokensWithPath(data, []string{}, path, prefix, result)
}

// extractTokensWithGroupMarkers recursively extracts tokens with group marker support
func (p *Parser) extractTokensWithGroupMarkers(data map[string]interface{}, path, prefix string, groupMarkers []string, result *[]*tokens.Token) {
	p.extractTokensWithPathAndGroupMarkers(data, []string{}, path, prefix, groupMarkers, result)
}

// extractTokensWithPath recursively extracts tokens tracking the JSON path
func (p *Parser) extractTokensWithPath(data map[string]interface{}, jsonPath []string, path, prefix string, result *[]*tokens.Token) {
	// Sort keys to ensure deterministic order
	keys := make([]string, 0, len(data))
	for key := range data {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		value := data[key]
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

// extractTokensWithPathAndGroupMarkers recursively extracts tokens with group marker support
func (p *Parser) extractTokensWithPathAndGroupMarkers(data map[string]interface{}, jsonPath []string, path, prefix string, groupMarkers []string, result *[]*tokens.Token) {
	// Sort keys to ensure deterministic order
	keys := make([]string, 0, len(data))
	for key := range data {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		value := data[key]
		valueMap, isMap := value.(map[string]interface{})
		if !isMap {
			continue
		}

		currentPath := append(jsonPath, key)
		newPath := path
		if path == "" {
			newPath = key
		} else {
			newPath = path + "-" + key
		}

		// Check if this is a token (has $value)
		dollarValue, hasValue := valueMap["$value"]

		// Check if this key is in groupMarkers
		isGroupMarker := false
		for _, marker := range groupMarkers {
			if key == marker {
				isGroupMarker = true
				break
			}
		}

		// If has $value, extract the token
		if hasValue {
			token := p.createToken(key, path, dollarValue, valueMap, prefix, currentPath)
			*result = append(*result, token)
		}

		// Check if we should recurse into children
		shouldRecurse := false
		if !hasValue {
			// No $value means it's definitely a group
			shouldRecurse = true
		} else if isGroupMarker {
			// Has $value but is a group marker - recurse into children too
			shouldRecurse = true
		}

		// Recurse into children if needed
		if shouldRecurse {
			// Filter out DTCG metadata keys (starting with $) when recursing
			childData := make(map[string]interface{})
			for k, v := range valueMap {
				if !strings.HasPrefix(k, "$") {
					childData[k] = v
				}
			}

			if len(childData) > 0 {
				p.extractTokensWithPathAndGroupMarkers(childData, currentPath, newPath, prefix, groupMarkers, result)
			}
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

// ParseFile parses a YAML file and returns tokens
func (p *Parser) ParseFile(filename string, prefix string) ([]*tokens.Token, error) {
	return p.ParseFileWithGroupMarkers(filename, prefix, nil)
}

// ParseFileWithGroupMarkers parses a YAML file with group marker support
func (p *Parser) ParseFileWithGroupMarkers(filename string, prefix string, groupMarkers []string) ([]*tokens.Token, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", filename, err)
	}

	tokens, err := p.ParseWithGroupMarkers(data, prefix, groupMarkers)
	if err != nil {
		return nil, fmt.Errorf("failed to parse file %s: %w", filename, err)
	}

	return tokens, nil
}
