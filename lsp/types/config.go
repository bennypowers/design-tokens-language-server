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

// ServerConfig represents the server configuration (user-provided settings)
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

	// NetworkFallback enables CDN fallback for npm: specifiers
	// when local node_modules resolution fails.
	NetworkFallback bool `json:"networkFallback"`

	// NetworkTimeout is the max time in seconds for CDN requests.
	// Non-positive values (<= 0) use the default (30s). Has no effect if NetworkFallback is false.
	NetworkTimeout int `json:"networkTimeout,omitempty"`

	// CDN selects the CDN provider for network fallback of package specifiers.
	// Valid values: "unpkg", "esm.sh", "esm.run", "jspm", "jsdelivr".
	// Defaults to "unpkg" if empty. Has no effect if NetworkFallback is false.
	CDN string `json:"cdn,omitempty"`
}

// ServerState represents a snapshot of runtime state (NOT configuration)
// This is returned by GetState() for thread-safe access to runtime state.
// For configuration, use GetConfig() separately.
type ServerState struct {
	// RootPath is the workspace root path (file system)
	RootPath string
}

// DefaultConfig returns the default server configuration
func DefaultConfig() ServerConfig {
	return ServerConfig{
		TokensFiles: nil, // No default tokens - must be explicitly configured
		Prefix:      "",
		GroupMarkers: []string{
			"_",
			"@",
			"DEFAULT",
		},
		NetworkFallback: false,
		NetworkTimeout:  0,
		CDN:             "",
	}
}
