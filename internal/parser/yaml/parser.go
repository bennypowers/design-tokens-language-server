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
	// Parse YAML with yaml.v3 to get AST with position data
	var root yaml.Node
	if err := yaml.Unmarshal(data, &root); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	// Extract tokens from AST
	result := []*tokens.Token{}
	if len(root.Content) > 0 {
		p.extractTokensWithPathAndGroupMarkers(root.Content[0], []string{}, "", prefix, groupMarkers, &result)
	}

	return result, nil
}

// getNodeValue finds a child node by key name in a mapping node
func getNodeValue(node *yaml.Node, key string) *yaml.Node {
	if node.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i < len(node.Content); i += 2 {
		keyNode := node.Content[i]
		if keyNode.Value == key {
			return node.Content[i+1]
		}
	}
	return nil
}

// extractTokensWithPathAndGroupMarkers recursively extracts tokens with group marker support from AST
func (p *Parser) extractTokensWithPathAndGroupMarkers(node *yaml.Node, jsonPath []string, path, prefix string, groupMarkers []string, result *[]*tokens.Token) {
	if node.Kind != yaml.MappingNode {
		return
	}

	// Collect key-value pairs and sort by key for deterministic order
	type kvPair struct {
		keyNode   *yaml.Node
		valueNode *yaml.Node
	}
	pairs := make([]kvPair, 0)
	for i := 0; i < len(node.Content); i += 2 {
		pairs = append(pairs, kvPair{
			keyNode:   node.Content[i],
			valueNode: node.Content[i+1],
		})
	}
	sort.Slice(pairs, func(i, j int) bool {
		return pairs[i].keyNode.Value < pairs[j].keyNode.Value
	})

	for _, pair := range pairs {
		keyNode := pair.keyNode
		valueNode := pair.valueNode
		key := keyNode.Value

		// Skip non-mapping values
		if valueNode.Kind != yaml.MappingNode {
			continue
		}

		currentPath := append([]string{}, jsonPath...)
		currentPath = append(currentPath, key)
		var newPath string
		if path == "" {
			newPath = key
		} else {
			newPath = path + "-" + key
		}

		// Check if this is a token (has $value)
		dollarValueNode := getNodeValue(valueNode, "$value")
		hasValue := dollarValueNode != nil

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
			token := p.createToken(keyNode, path, valueNode, prefix, currentPath)
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
			// Create child node with filtered content (skip $ keys)
			childNode := &yaml.Node{
				Kind:    yaml.MappingNode,
				Content: make([]*yaml.Node, 0),
			}

			for i := 0; i < len(valueNode.Content); i += 2 {
				k := valueNode.Content[i].Value
				v := valueNode.Content[i+1]

				// Skip DTCG metadata keys
				if !strings.HasPrefix(k, "$") {
					childNode.Content = append(childNode.Content, valueNode.Content[i], v)
				}
			}

			if len(childNode.Content) > 0 {
				p.extractTokensWithPathAndGroupMarkers(childNode, currentPath, newPath, prefix, groupMarkers, result)
			}
		}
	}
}

// createToken creates a Token from AST nodes with accurate position data
func (p *Parser) createToken(keyNode *yaml.Node, path string, valueNode *yaml.Node, prefix string, jsonPath []string) *tokens.Token {
	key := keyNode.Value

	// Build token name from path
	name := path
	if name == "" {
		name = key
	} else {
		name = path + "-" + key
	}

	// Build reference format (e.g., "{color.primary}")
	reference := "{" + strings.Join(jsonPath, ".") + "}"

	// Extract position from AST node (yaml.v3 uses 1-based, convert to 0-based for LSP)
	line := uint32(0)
	character := uint32(0)
	if keyNode.Line > 0 {
		line = uint32(keyNode.Line - 1)
	}
	if keyNode.Column > 0 {
		character = uint32(keyNode.Column - 1)
	}

	// Extract $value from value node
	dollarValueNode := getNodeValue(valueNode, "$value")
	value := ""
	if dollarValueNode != nil {
		value = dollarValueNode.Value
	}

	token := &tokens.Token{
		Name:      name,
		Value:     value,
		Prefix:    prefix,
		Path:      jsonPath,
		Reference: reference,
		Line:      line,
		Character: character,
	}

	// Extract $type
	if typeNode := getNodeValue(valueNode, "$type"); typeNode != nil {
		token.Type = typeNode.Value
	}

	// Extract $description
	if descNode := getNodeValue(valueNode, "$description"); descNode != nil {
		token.Description = descNode.Value
	}

	// Extract $deprecated flag (can be bool or string with message)
	if deprecatedNode := getNodeValue(valueNode, "$deprecated"); deprecatedNode != nil {
		if deprecatedNode.Kind == yaml.ScalarNode {
			if deprecatedNode.Tag == "!!bool" {
				token.Deprecated = deprecatedNode.Value == "true"
			} else {
				// String deprecation message
				token.Deprecated = true
				token.DeprecationMessage = deprecatedNode.Value
			}
		}
	}

	// Extract $extensions
	if extensionsNode := getNodeValue(valueNode, "$extensions"); extensionsNode != nil {
		var extensions map[string]interface{}
		if err := extensionsNode.Decode(&extensions); err == nil {
			token.Extensions = extensions
		}
		// Note: Decode errors are silently ignored - malformed extensions shouldn't break token parsing
	}

	return token
}

// ParseFile parses a YAML file and returns tokens
func (p *Parser) ParseFile(filename, prefix string) ([]*tokens.Token, error) {
	return p.ParseFileWithGroupMarkers(filename, prefix, nil)
}

// ParseFileWithGroupMarkers parses a YAML file with group marker support
func (p *Parser) ParseFileWithGroupMarkers(filename, prefix string, groupMarkers []string) ([]*tokens.Token, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", filename, err)
	}

	parsed, err := p.ParseWithGroupMarkers(data, prefix, groupMarkers)
	if err != nil {
		return nil, fmt.Errorf("failed to parse file %s: %w", filename, err)
	}

	return parsed, nil
}
