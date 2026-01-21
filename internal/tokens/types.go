// Package tokens provides design token types and management.
// The Token type is re-exported from asimonim for compatibility.
package tokens

import (
	asimonimToken "bennypowers.dev/asimonim/token"
)

// Token is a type alias for asimonim's Token type.
// This eliminates conversion overhead and maintains backward compatibility.
type Token = asimonimToken.Token

// TokenGroup represents a group of tokens (can be nested)
type TokenGroup struct {
	Name        string                 `json:"-"`
	Description string                 `json:"$description,omitempty"`
	Type        string                 `json:"$type,omitempty"`
	Tokens      map[string]*Token      `json:"-"`
	Groups      map[string]*TokenGroup `json:"-"`
}

// TokenFile represents a design token file configuration
type TokenFile struct {
	// Path to the token file
	Path string

	// Prefix for CSS variables from this file
	Prefix string

	// GroupMarkers indicate terminal paths that are also groups
	GroupMarkers []string
}

// RawTokenData represents the raw JSON/YAML structure of a token file
type RawTokenData map[string]interface{}

// TokenReference represents a reference to another token
// e.g., "{color.primary}" or "$color.primary"
type TokenReference struct {
	// Raw is the original reference string
	Raw string

	// TokenName is the resolved token name being referenced
	TokenName string

	// Valid indicates if this reference could be resolved
	Valid bool

	// ResolvedValue is the value of the referenced token (if Valid)
	ResolvedValue string
}
