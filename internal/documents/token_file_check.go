package documents

import (
	"strings"

	"gopkg.in/yaml.v3"
)

// designTokensSchemaPrefix is the required prefix for Design Tokens schema URLs
const designTokensSchemaPrefix = "https://www.designtokens.org/schemas/"

// IsDesignTokensSchema checks if the content contains a top-level $schema field
// pointing to a Design Tokens schema URL (https://www.designtokens.org/schemas/**/*.json).
// It parses the content as YAML (which also handles JSON) to ensure only the
// document root's $schema is considered, not nested $schema fields.
func IsDesignTokensSchema(content string) bool {
	// Parse content as YAML (works for both YAML and JSON)
	var doc map[string]any
	if err := yaml.Unmarshal([]byte(content), &doc); err != nil {
		return false
	}

	// Check for top-level $schema key
	schemaValue, exists := doc["$schema"]
	if !exists {
		return false
	}

	// Ensure $schema value is a string
	schemaURL, ok := schemaValue.(string)
	if !ok {
		return false
	}

	// Check if schema URL starts with the Design Tokens schema prefix
	if !strings.HasPrefix(schemaURL, designTokensSchemaPrefix) {
		return false
	}

	// Check if schema URL ends with .json
	if !strings.HasSuffix(schemaURL, ".json") {
		return false
	}

	return true
}
