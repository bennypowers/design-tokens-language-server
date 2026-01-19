// Package asimonim provides an adapter layer to use the asimonim parser library
// within dtls. Since dtls.Token is now a type alias for asimonim.Token,
// this adapter simply wraps the parser with no conversion overhead.
package asimonim

import (
	asimonimParser "bennypowers.dev/asimonim/parser"
	"bennypowers.dev/dtls/internal/schema"
	"bennypowers.dev/dtls/internal/tokens"
)

// ParseWithSchemaVersion parses JSON token data using the asimonim parser.
// Since tokens.Token is now a type alias for asimonim/token.Token,
// no conversion is needed - tokens are returned directly.
func ParseWithSchemaVersion(data []byte, prefix string, version schema.SchemaVersion, groupMarkers []string) ([]*tokens.Token, error) {
	return ParseWithOptions(data, prefix, version, groupMarkers, false)
}

// ParseWithOptions parses JSON token data with additional options.
// skipSort disables alphabetical sorting for better performance.
func ParseWithOptions(data []byte, prefix string, version schema.SchemaVersion, groupMarkers []string, skipSort bool) ([]*tokens.Token, error) {
	parser := asimonimParser.NewJSONParser()

	opts := asimonimParser.Options{
		Prefix:        prefix,
		SchemaVersion: version, // schema.SchemaVersion is now an alias for asimonim schema.Version
		GroupMarkers:  groupMarkers,
		SkipSort:      skipSort,
	}

	// tokens.Token is now a type alias for asimonim/token.Token
	// so we can return directly without conversion
	return parser.Parse(data, opts)
}
