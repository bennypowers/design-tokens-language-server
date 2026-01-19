// Package schema provides design token schema version types.
// This package re-exports types from asimonim for compatibility.
package schema

import (
	asimonimSchema "bennypowers.dev/asimonim/schema"
)

// SchemaVersion is a type alias for asimonim's Version type.
// This maintains backward compatibility with existing dtls code.
type SchemaVersion = asimonimSchema.Version

// Re-export constants from asimonim schema
const (
	Unknown  = asimonimSchema.Unknown
	Draft    = asimonimSchema.Draft
	V2025_10 = asimonimSchema.V2025_10
)

// Re-export functions from asimonim schema
var (
	FromURL    = asimonimSchema.FromURL
	FromString = asimonimSchema.FromString
)
