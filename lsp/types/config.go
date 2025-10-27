package types

// TokenFileSpec represents a token file specification
type TokenFileSpec struct {
	// Path to the token file (required)
	// Can be absolute, relative, or npm: protocol
	Path string `json:"path"`

	// Prefix for CSS variables from this file (optional)
	Prefix string `json:"prefix,omitempty"`

	// GroupMarkers are token names that can also be groups (optional)
	GroupMarkers []string `json:"groupMarkers,omitempty"`
}

// ServerConfig represents the server configuration
type ServerConfig struct {
	// TokensFiles specifies token files to load
	// Can be:
	//  - strings (paths): ["./tokens.json", "npm:@foo/tokens"]
	//  - objects: [{"path": "./tokens.json", "prefix": "ds"}]
	// If empty, falls back to searching for common patterns
	TokensFiles []any `json:"tokensFiles"`

	// Prefix is the global CSS variable prefix (can be overridden per-file)
	// Example: "ds" will generate "--ds-color-primary"
	Prefix string `json:"prefix"`

	// GroupMarkers are token names which will be treated as group names as well
	// Default: ["_", "@", "DEFAULT"]
	GroupMarkers []string `json:"groupMarkers"`
}

// DefaultConfig returns the default server configuration
func DefaultConfig() ServerConfig {
	return ServerConfig{
		TokensFiles: []any{}, // Empty = auto-discover
		Prefix:      "",
		GroupMarkers: []string{
			"_",
			"@",
			"DEFAULT",
		},
	}
}

// AutoDiscoverPatterns are the glob patterns used to auto-discover token files
// when TokensFiles is not explicitly configured.
// These patterns match common token file naming conventions:
//   - tokens.{ext}           (e.g., tokens.json, tokens.yaml)
//   - *.tokens.{ext}         (e.g., colors.tokens.json, spacing.tokens.yaml)
//   - design-tokens.{ext}    (e.g., design-tokens.json, design-tokens.yaml)
//
// Supported extensions: json, yaml, yml
var AutoDiscoverPatterns = []string{
	"**/tokens.json",
	"**/*.tokens.json",
	"**/design-tokens.json",
	"**/tokens.yaml",
	"**/*.tokens.yaml",
	"**/design-tokens.yaml",
	"**/tokens.yml",
	"**/*.tokens.yml",
	"**/design-tokens.yml",
}
