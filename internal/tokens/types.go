package tokens

// Token represents a design token following the DTCG specification
// See: https://design-tokens.github.io/community-group/format/
type Token struct {
	// Name is the token's identifier (e.g., "color.primary")
	Name string `json:"name"`

	// $value is the resolved value of the token
	Value string `json:"$value"`

	// $type specifies the type of token (color, dimension, etc.)
	Type string `json:"$type,omitempty"`

	// $description is optional documentation for the token
	Description string `json:"$description,omitempty"`

	// $extensions allows for custom metadata
	Extensions map[string]interface{} `json:"$extensions,omitempty"`

	// Deprecated indicates if this token should no longer be used
	Deprecated bool `json:"deprecated,omitempty"`

	// DeprecationMessage provides context for deprecated tokens
	DeprecationMessage string `json:"deprecationMessage,omitempty"`

	// FilePath is the file this token was loaded from
	FilePath string `json:"-"`

	// Prefix is the CSS variable prefix for this token
	Prefix string `json:"-"`
}

// CSSVariableName returns the CSS custom property name for this token
// e.g., "--color-primary" or "--my-prefix-color-primary"
func (t *Token) CSSVariableName() string {
	if t.Prefix != "" {
		return "--" + t.Prefix + "-" + t.Name
	}
	return "--" + t.Name
}

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
