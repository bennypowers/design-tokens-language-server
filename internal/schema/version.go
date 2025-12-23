package schema

import "fmt"

// SchemaVersion represents a design tokens schema version
type SchemaVersion int

const (
	// Unknown represents an undetected or unrecognized schema version
	Unknown SchemaVersion = iota

	// Draft represents the editor's draft schema
	Draft

	// V2025_10 represents the stable 2025.10 schema
	V2025_10
)

// String returns the string representation of the schema version
func (v SchemaVersion) String() string {
	switch v {
	case Draft:
		return "draft"
	case V2025_10:
		return "v2025_10"
	default:
		return "unknown"
	}
}

// URL returns the JSON Schema URL for this version
func (v SchemaVersion) URL() string {
	switch v {
	case Draft:
		return "https://www.designtokens.org/schemas/draft.json"
	case V2025_10:
		return "https://www.designtokens.org/schemas/2025.10.json"
	default:
		return ""
	}
}

// FromURL returns the schema version from a JSON Schema URL
func FromURL(url string) (SchemaVersion, error) {
	switch url {
	case "https://www.designtokens.org/schemas/draft.json":
		return Draft, nil
	case "https://www.designtokens.org/schemas/2025.10.json":
		return V2025_10, nil
	default:
		return Unknown, fmt.Errorf("unrecognized schema URL: %s", url)
	}
}

// FromString returns the schema version from a string representation
func FromString(s string) (SchemaVersion, error) {
	switch s {
	case "draft":
		return Draft, nil
	case "v2025_10":
		return V2025_10, nil
	default:
		return Unknown, fmt.Errorf("unrecognized schema version string: %s", s)
	}
}
