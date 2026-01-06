package color

import (
	"fmt"
	"math"
	"strings"

	"bennypowers.dev/dtls/internal/parser/common"
)

// ToCSS converts a ColorValue to a CSS-compatible string
func ToCSS(colorValue common.ColorValue) string {
	// For draft schema string colors, return as-is
	if stringColor, ok := colorValue.(*common.StringColorValue); ok {
		return stringColor.Value
	}

	// For 2025.10 structured colors
	if objColor, ok := colorValue.(*common.ObjectColorValue); ok {
		return objectColorToCSS(objColor)
	}

	// Fallback
	return ""
}

// objectColorToCSS converts a 2025.10 structured color to CSS
func objectColorToCSS(c *common.ObjectColorValue) string {
	colorSpace := strings.ToLower(c.ColorSpace)

	// If hex field is provided, use it
	if c.Hex != nil && *c.Hex != "" {
		return *c.Hex
	}

	// Get alpha value (default to 1.0 if not specified)
	alpha := 1.0
	if c.Alpha != nil {
		alpha = *c.Alpha
	}

	// Handle different color spaces
	switch colorSpace {
	case "srgb":
		return srgbToCSS(c.Components, alpha)

	case "hsl":
		return hslToCSS(c.Components, alpha)

	case "hwb":
		return hwbToCSS(c.Components, alpha)

	case "oklch":
		return oklchToCSS(c.Components, alpha)

	case "oklab":
		return oklabToCSS(c.Components, alpha)

	case "lch":
		return lchToCSS(c.Components, alpha)

	case "lab":
		return labToCSS(c.Components, alpha)

	case "display-p3", "a98-rgb", "prophoto-rgb", "rec2020", "xyz-d65", "xyz-d50", "srgb-linear":
		// Use CSS color() function for these spaces
		return colorFunctionToCSS(colorSpace, c.Components, alpha)

	default:
		// Unknown color space, use CSS color() function as fallback
		// This allows the browser to handle unknown color spaces
		return colorFunctionToCSS(colorSpace, c.Components, alpha)
	}
}

// srgbToCSS converts sRGB components to CSS
func srgbToCSS(components []interface{}, alpha float64) string {
	if len(components) < 3 {
		return ""
	}

	r := math.Max(0, math.Min(1, componentToFloat(components[0])))
	g := math.Max(0, math.Min(1, componentToFloat(components[1])))
	b := math.Max(0, math.Min(1, componentToFloat(components[2])))

	// Convert 0-1 range to 0-255
	rInt := int(math.Round(r * 255))
	gInt := int(math.Round(g * 255))
	bInt := int(math.Round(b * 255))

	// If alpha is 1.0, use hex or rgb
	if alpha >= 0.999 {
		return fmt.Sprintf("#%02x%02x%02x", rInt, gInt, bInt)
	}

	// Use rgba for transparency
	return fmt.Sprintf("rgba(%d, %d, %d, %.2f)", rInt, gInt, bInt, alpha)
}

// hslToCSS converts HSL components to CSS
func hslToCSS(components []interface{}, alpha float64) string {
	if len(components) < 3 {
		return ""
	}

	h := componentToFloat(components[0])
	s := componentToFloat(components[1])
	l := componentToFloat(components[2])

	if alpha >= 0.999 {
		return fmt.Sprintf("hsl(%.1f, %.1f%%, %.1f%%)", h, s, l)
	}

	return fmt.Sprintf("hsla(%.1f, %.1f%%, %.1f%%, %.2f)", h, s, l, alpha)
}

// hwbToCSS converts HWB components to CSS
func hwbToCSS(components []interface{}, alpha float64) string {
	if len(components) < 3 {
		return ""
	}

	h := componentToFloat(components[0])
	w := componentToFloat(components[1])
	b := componentToFloat(components[2])

	if alpha >= 0.999 {
		return fmt.Sprintf("hwb(%.1f %.1f%% %.1f%%)", h, w, b)
	}

	return fmt.Sprintf("hwb(%.1f %.1f%% %.1f%% / %.2f)", h, w, b, alpha)
}

// oklchToCSS converts OKLCH components to CSS
func oklchToCSS(components []interface{}, alpha float64) string {
	if len(components) < 3 {
		return ""
	}

	l := componentToFloat(components[0])
	c := componentToFloat(components[1])
	h := componentToFloat(components[2])

	if alpha >= 0.999 {
		return fmt.Sprintf("oklch(%.2f %.2f %.1f)", l, c, h)
	}

	return fmt.Sprintf("oklch(%.2f %.2f %.1f / %.2f)", l, c, h, alpha)
}

// oklabToCSS converts OKLAB components to CSS
func oklabToCSS(components []interface{}, alpha float64) string {
	if len(components) < 3 {
		return ""
	}

	l := componentToFloat(components[0])
	a := componentToFloat(components[1])
	b := componentToFloat(components[2])

	if alpha >= 0.999 {
		return fmt.Sprintf("oklab(%.2f %.2f %.2f)", l, a, b)
	}

	return fmt.Sprintf("oklab(%.2f %.2f %.2f / %.2f)", l, a, b, alpha)
}

// lchToCSS converts LCH components to CSS
func lchToCSS(components []interface{}, alpha float64) string {
	if len(components) < 3 {
		return ""
	}

	l := componentToFloat(components[0])
	c := componentToFloat(components[1])
	h := componentToFloat(components[2])

	if alpha >= 0.999 {
		return fmt.Sprintf("lch(%.1f %.1f %.1f)", l, c, h)
	}

	return fmt.Sprintf("lch(%.1f %.1f %.1f / %.2f)", l, c, h, alpha)
}

// labToCSS converts LAB components to CSS
func labToCSS(components []interface{}, alpha float64) string {
	if len(components) < 3 {
		return ""
	}

	l := componentToFloat(components[0])
	a := componentToFloat(components[1])
	b := componentToFloat(components[2])

	if alpha >= 0.999 {
		return fmt.Sprintf("lab(%.1f %.1f %.1f)", l, a, b)
	}

	return fmt.Sprintf("lab(%.1f %.1f %.1f / %.2f)", l, a, b, alpha)
}

// colorFunctionToCSS converts color space to CSS color() function
func colorFunctionToCSS(colorSpace string, components []interface{}, alpha float64) string {
	if len(components) < 3 {
		return ""
	}

	// CSS color space identifiers match input names
	c0 := componentToString(components[0])
	c1 := componentToString(components[1])
	c2 := componentToString(components[2])

	if alpha >= 0.999 {
		return fmt.Sprintf("color(%s %s %s %s)", colorSpace, c0, c1, c2)
	}

	return fmt.Sprintf("color(%s %s %s %s / %.2f)", colorSpace, c0, c1, c2, alpha)
}

// componentToFloat converts a component value to float64
// Handles both numeric values and "none" keyword
func componentToFloat(component interface{}) float64 {
	switch v := component.(type) {
	case float64:
		return v
	case int:
		return float64(v)
	case string:
		// "none" keyword treated as 0
		if v == "none" {
			return 0.0
		}
	}
	return 0.0
}

// componentToString converts a component to string for CSS color() function
func componentToString(component interface{}) string {
	switch v := component.(type) {
	case float64:
		return fmt.Sprintf("%.4f", v)
	case int:
		return fmt.Sprintf("%d", v)
	case string:
		return v // "none" keyword
	}
	return "0"
}

// ToHex converts a 2025.10 ObjectColorValue to hex format
// Only works for sRGB colors
func ToHex(c *common.ObjectColorValue) (string, error) {
	if c == nil {
		return "", fmt.Errorf("color value is nil")
	}

	colorSpace := strings.ToLower(c.ColorSpace)

	// Only convert sRGB to hex
	if colorSpace != "srgb" {
		return "", fmt.Errorf("can only convert sRGB colors to hex, got %s", colorSpace)
	}

	if len(c.Components) < 3 {
		return "", fmt.Errorf("invalid number of components: %d", len(c.Components))
	}

	r := math.Max(0, math.Min(1, componentToFloat(c.Components[0])))
	g := math.Max(0, math.Min(1, componentToFloat(c.Components[1])))
	b := math.Max(0, math.Min(1, componentToFloat(c.Components[2])))

	// Convert 0-1 range to 0-255
	rInt := int(math.Round(r * 255))
	gInt := int(math.Round(g * 255))
	bInt := int(math.Round(b * 255))

	return fmt.Sprintf("#%02x%02x%02x", rInt, gInt, bInt), nil
}
