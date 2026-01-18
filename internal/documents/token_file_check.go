package documents

import (
	"strings"

	"bennypowers.dev/dtls/internal/parser/common"
)

// designTokensSchemaPrefix is the required prefix for Design Tokens schema URLs
const designTokensSchemaPrefix = "https://www.designtokens.org/schemas/"

// IsDesignTokensSchema checks if the content contains a $schema field pointing
// to a Design Tokens schema URL (https://www.designtokens.org/schemas/**/*.json).
// This function only matches top-level $schema declarations.
func IsDesignTokensSchema(content string) bool {
	matches := common.SchemaFieldRegexp.FindStringSubmatch(content)
	if len(matches) < 2 {
		return false
	}

	schemaURL := matches[1]

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
