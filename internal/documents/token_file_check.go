package documents

import (
	"strings"

	"gopkg.in/yaml.v3"
)

// dtcgSchemaURLPrefix is the required prefix for Design Tokens schema URLs
const dtcgSchemaURLPrefix = "https://www.designtokens.org/schemas/"

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
	if !strings.HasPrefix(schemaURL, dtcgSchemaURLPrefix) {
		return false
	}

	// Check if schema URL ends with .json
	if !strings.HasSuffix(schemaURL, ".json") {
		return false
	}

	return true
}

// LooksLikeDTCGContent checks if the content looks like a DTCG token file
// by detecting characteristic patterns like $value, $type fields.
// This is a heuristic detection used when $schema is not present.
func LooksLikeDTCGContent(content string) bool {
	// Parse content as YAML (works for both YAML and JSON)
	var doc map[string]any
	if err := yaml.Unmarshal([]byte(content), &doc); err != nil {
		return false
	}

	// Check if it has DTCG patterns
	return hasDTCGPatterns(doc, 0)
}

// hasDTCGPatterns recursively checks for DTCG token patterns
// maxDepth prevents excessive recursion on large files
func hasDTCGPatterns(data map[string]any, depth int) bool {
	// Limit recursion depth to avoid performance issues on large files
	if depth > 10 {
		return false
	}

	// Check for $value field (primary DTCG indicator)
	if _, hasValue := data["$value"]; hasValue {
		return true
	}

	// Check for $type field at current level
	if _, hasType := data["$type"]; hasType {
		return true
	}

	// Recurse into nested objects
	for key, value := range data {
		// Skip $-prefixed keys at root level as they're metadata
		if depth == 0 && strings.HasPrefix(key, "$") {
			continue
		}

		if nested, ok := value.(map[string]any); ok {
			if hasDTCGPatterns(nested, depth+1) {
				return true
			}
		}
	}

	return false
}
