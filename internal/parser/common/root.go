package common

import (
	"bennypowers.dev/dtls/internal/schema"
)

// IsRootToken checks if a token name represents a root token for the given schema
func IsRootToken(name string, version schema.SchemaVersion, groupMarkers []string) bool {
	switch version {
	case schema.V2025_10:
		// In 2025.10, only "$root" is the reserved root token name
		return name == "$root"

	case schema.Draft:
		// In draft, use configured groupMarkers
		for _, marker := range groupMarkers {
			if name == marker {
				return true
			}
		}
		return false

	default:
		return false
	}
}

// GenerateRootTokenPath generates the token path for a root token.
// Both $root and groupMarkers should produce the same path.
//
// Example: ["color", "primary"] + "$root" -> ["color", "primary"]
// Example: ["color", "primary"] + "_" -> ["color", "primary"]
//
// Parameters rootTokenName and version are reserved for potential future
// schema-specific behavior. Currently, all root tokens inherit the group path
// regardless of schema version to ensure CSS variable consistency:
// color.primary.$root -> --color-primary
// color.primary._ -> --color-primary
func GenerateRootTokenPath(groupPath []string, rootTokenName string, version schema.SchemaVersion) []string {
	// Root token inherits the group path (doesn't add itself to path)
	return groupPath
}
