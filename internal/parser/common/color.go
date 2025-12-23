package common

import (
	"fmt"

	"bennypowers.dev/dtls/internal/schema"
)

// ColorValue represents a color token value in any schema format
type ColorValue interface {
	ToCSS() string
	SchemaVersion() schema.SchemaVersion
	IsValid() bool
}

// StringColorValue represents a draft schema color value (string format)
type StringColorValue struct {
	Value  string
	Schema schema.SchemaVersion
}

func (s *StringColorValue) ToCSS() string {
	return s.Value
}

func (s *StringColorValue) SchemaVersion() schema.SchemaVersion {
	return s.Schema
}

func (s *StringColorValue) IsValid() bool {
	return s.Value != ""
}

// ObjectColorValue represents a 2025.10 schema color value (structured format)
type ObjectColorValue struct {
	ColorSpace string
	Components []interface{} // Can be float64 or "none" keyword
	Alpha      *float64
	Hex        *string
	Schema     schema.SchemaVersion
}

func (o *ObjectColorValue) ToCSS() string {
	// If hex field is provided, use it
	if o.Hex != nil && *o.Hex != "" {
		return *o.Hex
	}

	// Otherwise, convert to CSS color() function
	// For simplicity, we'll generate basic CSS color() syntax
	// Full implementation would handle all 14 color spaces
	alpha := 1.0
	if o.Alpha != nil {
		alpha = *o.Alpha
	}

	// Convert components to string
	var compStr string
	for i, comp := range o.Components {
		if i > 0 {
			compStr += " "
		}
		switch v := comp.(type) {
		case float64:
			compStr += fmt.Sprintf("%.4g", v)
		case string:
			compStr += v // "none" keyword
		default:
			compStr += fmt.Sprintf("%v", v)
		}
	}

	// Generate CSS color() function
	return fmt.Sprintf("color(%s %s / %.4g)", o.ColorSpace, compStr, alpha)
}

func (o *ObjectColorValue) SchemaVersion() schema.SchemaVersion {
	return o.Schema
}

func (o *ObjectColorValue) IsValid() bool {
	return o.ColorSpace != "" && len(o.Components) > 0
}

// ParseColorValue parses a color value according to the schema version
func ParseColorValue(value interface{}, version schema.SchemaVersion) (ColorValue, error) {
	switch version {
	case schema.Draft:
		// Draft expects string values
		str, ok := value.(string)
		if !ok {
			return nil, schema.NewInvalidColorFormatError("", "", "draft", "structured object with colorSpace", "string value")
		}
		return &StringColorValue{
			Value:  str,
			Schema: schema.Draft,
		}, nil

	case schema.V2025_10:
		// 2025.10 expects structured objects
		obj, ok := value.(map[string]interface{})
		if !ok {
			return nil, schema.NewInvalidColorFormatError("", "", "v2025_10", "string value", "structured object with colorSpace")
		}

		// Extract colorSpace
		colorSpace, ok := obj["colorSpace"].(string)
		if !ok {
			return nil, fmt.Errorf("missing or invalid colorSpace field in color object")
		}

		// Extract components
		componentsRaw, ok := obj["components"]
		if !ok {
			return nil, fmt.Errorf("missing components field in color object")
		}

		components, ok := componentsRaw.([]interface{})
		if !ok {
			return nil, fmt.Errorf("components must be an array")
		}

		// Extract optional alpha
		var alpha *float64
		if alphaRaw, exists := obj["alpha"]; exists {
			if alphaVal, ok := alphaRaw.(float64); ok {
				alpha = &alphaVal
			}
		}

		// Extract optional hex
		var hex *string
		if hexRaw, exists := obj["hex"]; exists {
			if hexVal, ok := hexRaw.(string); ok {
				hex = &hexVal
			}
		}

		return &ObjectColorValue{
			ColorSpace: colorSpace,
			Components: components,
			Alpha:      alpha,
			Hex:        hex,
			Schema:     schema.V2025_10,
		}, nil

	default:
		return nil, fmt.Errorf("unknown schema version: %v", version)
	}
}
