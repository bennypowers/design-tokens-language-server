package css

import (
	"fmt"
	"regexp"
	"strings"

	"bennypowers.dev/dtls/internal/collections"
	"bennypowers.dev/dtls/internal/tokens"
)

// Package-level Sets for static CSS type and value lookups
var (
	// safeCSSTypes are token types that can use raw values without quoting
	safeCSSTypes = collections.NewSet(
		"color", "dimension", "number", "duration", "cubicbezier",
	)

	// fontWeightKeywords are valid CSS font-weight keyword values
	fontWeightKeywords = collections.NewSet(
		"normal", "bold", "bolder", "lighter",
		"inherit", "initial", "unset",
	)

	// genericFontFamilies are CSS generic font family names that don't need quotes
	genericFontFamilies = collections.NewSet(
		"serif", "sans-serif", "monospace",
		"cursive", "fantasy", "system-ui",
	)

	// cssNamedColors are all valid CSS named color keywords
	cssNamedColors = collections.NewSet(
		"transparent", "black", "white", "red", "green",
		"blue", "yellow", "cyan", "magenta", "gray",
		"grey", "maroon", "purple", "fuchsia", "lime",
		"olive", "navy", "teal", "aqua", "orange",
		"aliceblue", "antiquewhite", "aquamarine", "azure",
		"beige", "bisque", "blanchedalmond", "blueviolet",
		"brown", "burlywood", "cadetblue", "chartreuse",
		"chocolate", "coral", "cornflowerblue", "cornsilk",
		"crimson", "darkblue", "darkcyan", "darkgoldenrod",
		"darkgray", "darkgrey", "darkgreen", "darkkhaki",
		"darkmagenta", "darkolivegreen", "darkorange", "darkorchid",
		"darkred", "darksalmon", "darkseagreen", "darkslateblue",
		"darkslategray", "darkslategrey", "darkturquoise", "darkviolet",
		"deeppink", "deepskyblue", "dimgray", "dimgrey",
		"dodgerblue", "firebrick", "floralwhite", "forestgreen",
		"gainsboro", "ghostwhite", "gold", "goldenrod",
		"greenyellow", "honeydew", "hotpink", "indianred",
		"indigo", "ivory", "khaki", "lavender",
		"lavenderblush", "lawngreen", "lemonchiffon", "lightblue",
		"lightcoral", "lightcyan", "lightgoldenrodyellow", "lightgray",
		"lightgrey", "lightgreen", "lightpink", "lightsalmon",
		"lightseagreen", "lightskyblue", "lightslategray", "lightslategrey",
		"lightsteelblue", "lightyellow", "limegreen", "linen",
		"mediumaquamarine", "mediumblue", "mediumorchid", "mediumpurple",
		"mediumseagreen", "mediumslateblue", "mediumspringgreen", "mediumturquoise",
		"mediumvioletred", "midnightblue", "mintcream", "mistyrose",
		"moccasin", "navajowhite", "oldlace", "olivedrab",
		"orangered", "orchid", "palegoldenrod", "palegreen",
		"paleturquoise", "palevioletred", "papayawhip", "peachpuff",
		"peru", "pink", "plum", "powderblue",
		"rosybrown", "royalblue", "saddlebrown", "salmon",
		"sandybrown", "seagreen", "seashell", "sienna",
		"silver", "skyblue", "slateblue", "slategray",
		"slategrey", "snow", "springgreen", "steelblue",
		"tan", "thistle", "tomato", "turquoise",
		"violet", "wheat", "whitesmoke", "yellowgreen",
	)

	// Regex patterns for CSS value validation
	cssNumberPattern         = regexp.MustCompile(`^-?\d+(\.\d+)?$`)
	cssDimensionPattern      = regexp.MustCompile(`^-?\d+(\.\d+)?(px|rem|em|%|vh|vw|pt|cm|mm|in|pc|ex|ch|vmin|vmax)$`)
	cssIdentifierPattern     = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9-]*$`)
	fontWeightNumericPattern = regexp.MustCompile(`^\d+$`)
)

// formatFontWeightForCSS formats a font-weight value for CSS.
// Returns the formatted value or an error if the value is invalid.
func formatFontWeightForCSS(value string) (string, error) {
	// Check if it's a valid keyword
	if fontWeightKeywords.Has(strings.ToLower(value)) {
		return value, nil
	}

	// Check if it's a numeric value
	if !fontWeightNumericPattern.MatchString(value) {
		return "", fmt.Errorf("invalid font-weight value %q (must be keyword or number)", value)
	}

	// Parse and validate range (1-1000)
	var numValue int
	n, err := fmt.Sscanf(value, "%d", &numValue)
	if err != nil || n != 1 {
		return "", fmt.Errorf("failed to parse font-weight value %q: %w", value, err)
	}

	if numValue < 1 || numValue > 1000 {
		return "", fmt.Errorf("font-weight value %q out of range (must be 1-1000)", value)
	}

	return value, nil
}

// formatUntypedTokenForCSS handles untyped tokens by inspecting the value format.
// Returns the formatted value or an error if the value format is unsupported.
func formatUntypedTokenForCSS(value string) (string, error) {
	// Named colors
	if isNamedColor(value) {
		return value, nil
	}

	// Hex colors
	if strings.HasPrefix(value, "#") {
		return value, nil
	}

	// CSS functions (rgb, rgba, hsl, hsla, var, calc, etc.)
	if strings.Contains(value, "(") && strings.Contains(value, ")") {
		return value, nil
	}

	// CSS dimensions (with units)
	if cssDimensionPattern.MatchString(value) {
		return value, nil
	}

	// Pure numbers
	if cssNumberPattern.MatchString(value) {
		return value, nil
	}

	// If it contains spaces or special characters, it might need quoting
	// but we can't be sure, so skip it for safety
	if strings.ContainsAny(value, " \t\n\"'()[]{}") {
		return "", fmt.Errorf("value %q contains special characters and cannot be safely formatted", value)
	}

	// CSS identifiers (like keyword values)
	if cssIdentifierPattern.MatchString(value) {
		return value, nil
	}

	// Unknown format
	return "", fmt.Errorf("value %q has unknown format", value)
}

// FormatTokenValueForCSS formats a token value for safe insertion into CSS.
// Returns the formatted value and an error if formatting/validation fails.
// Returns ("", error) for unsupported token types, invalid values or parse failures (caller should warn).
// Returns (value, nil) for successfully formatted values.
func FormatTokenValueForCSS(token *tokens.Token) (string, error) {
	value := token.Value
	tokenType := strings.ToLower(token.Type)

	// Dispatch to type-specific handlers
	switch tokenType {
	case "fontweight":
		return formatFontWeightForCSS(value)
	case "fontfamily":
		return FormatFontFamilyValue(value)
	case "":
		// No type specified, inspect the value to determine if it's safe
		return formatUntypedTokenForCSS(value)
	}

	// For known safe types, return the value as-is
	if safeCSSTypes.Has(tokenType) {
		return value, nil
	}

	// Composite types (border, shadow, etc.) should not be used as CSS fallback values
	return "", fmt.Errorf("token type %q cannot be used as CSS fallback value", tokenType)
}

// FormatFontFamilyValue formats a font family value for CSS.
// Returns the formatted value or an error if formatting fails.
func FormatFontFamilyValue(value string) (string, error) {
	value = strings.TrimSpace(value)

	// If it's already quoted, use as-is (assume it's properly formatted)
	if (strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"")) ||
		(strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'")) {
		return value, nil
	}

	// Generic font families don't need quotes
	if genericFontFamilies.Has(strings.ToLower(value)) {
		return value, nil
	}

	// If it contains a comma, it's likely a font-family list (e.g., "Arial, sans-serif")
	// These are typically already properly formatted, so return as-is
	if strings.Contains(value, ",") {
		// Check if it looks like a valid font-family list with quoted names
		// If it contains quotes, assume it's already properly formatted
		if strings.Contains(value, "\"") || strings.Contains(value, "'") {
			return value, nil
		}
		// Otherwise, it's a simple comma-separated list, also safe to use
		return value, nil
	}

	// If it contains spaces or special characters (but no comma), it needs quoting
	needsQuoting := strings.ContainsAny(value, " \t\n\"'")

	if needsQuoting {
		// Use %q to automatically quote and escape the string
		return fmt.Sprintf("%q", value), nil
	}

	// Single-word font names without special chars don't need quotes
	return value, nil
}

// isNamedColor checks if a value is a named CSS color
func isNamedColor(value string) bool {
	return cssNamedColors.Has(strings.ToLower(value))
}

// IsCSSValueSemanticallyEquivalent checks if two CSS values are semantically equivalent.
// It normalizes values by removing all whitespace and converting to lowercase before comparison.
//
// This is useful for comparing CSS values that may have different formatting but represent
// the same value, such as:
//   - "rgb(255, 0, 0)" vs "rgb(255,0,0)"
//   - "#FF0000" vs "#ff0000"
//   - "1.5rem" vs "1.5REM"
//
// Returns true if the values are equivalent after normalization.
func IsCSSValueSemanticallyEquivalent(a, b string) bool {
	// Normalize: remove all whitespace and convert to lowercase
	normalize := func(s string) string {
		s = strings.ReplaceAll(s, " ", "")
		s = strings.ReplaceAll(s, "\t", "")
		s = strings.ReplaceAll(s, "\n", "")
		s = strings.ToLower(s)
		return s
	}

	return normalize(a) == normalize(b)
}
