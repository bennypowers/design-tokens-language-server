package schema

import "gopkg.in/yaml.v3"

// SchemaHandler defines version-specific token processing operations
// Implementations handle parsing, validation, and formatting for specific schema versions
type SchemaHandler interface {
	// Version returns the schema version this handler supports
	Version() SchemaVersion

	// ValidateTokenNode validates a token node structure for this schema
	// Returns error if the token structure is invalid for this schema
	ValidateTokenNode(node *yaml.Node) error

	// FormatColorForCSS converts a color value to CSS format for this schema
	// Returns the CSS representation (hex, rgb(), color(), etc.)
	FormatColorForCSS(colorValue interface{}) string

	// SupportsFeature checks if a feature is supported in this schema
	// Features: "json-pointer", "extends", "root", "resolution-order"
	SupportsFeature(feature string) bool
}
