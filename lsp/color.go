package lsp

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/bennypowers/design-tokens-language-server/internal/parser/css"
	"github.com/bennypowers/design-tokens-language-server/lsp/types"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// handleDocumentColor handles the textDocument/documentColor request
func (s *Server) handleDocumentColor(context *glsp.Context, params *protocol.DocumentColorParams) ([]protocol.ColorInformation, error) {
	return DocumentColor(s, context, params)
}

// DocumentColor handles the textDocument/documentColor request
func DocumentColor(ctx types.ServerContext, context *glsp.Context, params *protocol.DocumentColorParams) ([]protocol.ColorInformation, error) {
	uri := params.TextDocument.URI

	fmt.Fprintf(os.Stderr, "[DTLS] DocumentColor requested: %s\n", uri)

	// Get document
	doc := ctx.Document(uri)
	if doc == nil {
		return nil, nil
	}

	// Only process CSS files
	if doc.LanguageID() != "css" {
		return nil, nil
	}

	// Parse CSS to find var() calls
	parser := css.AcquireParser()
	defer css.ReleaseParser(parser)
	result, err := parser.Parse(doc.Content())
	if err != nil {
		fmt.Fprintf(os.Stderr, "[DTLS] Failed to parse CSS: %v\n", err)
		return nil, nil
	}

	var colors []protocol.ColorInformation

	// Find all var() calls that reference color tokens
	for _, varCall := range result.VarCalls {
		// Look up the token
		token := ctx.Token(varCall.TokenName)
		if token == nil {
			continue
		}

		// Only process color tokens
		if token.Type != "color" {
			continue
		}

		// Parse the color value
		color, err := parseColor(token.Value)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[DTLS] Failed to parse color %s: %v\n", token.Value, err)
			continue
		}

		colors = append(colors, protocol.ColorInformation{
			Range: protocol.Range{
				Start: protocol.Position{
					Line:      varCall.Range.Start.Line,
					Character: varCall.Range.Start.Character,
				},
				End: protocol.Position{
					Line:      varCall.Range.End.Line,
					Character: varCall.Range.End.Character,
				},
			},
			Color: *color,
		})
	}

	// Also check variable declarations
	for _, variable := range result.Variables {
		// Look up the token
		token := ctx.Token(variable.Name)
		if token == nil {
			continue
		}

		// Only process color tokens
		if token.Type != "color" {
			continue
		}

		// Parse the color value
		color, err := parseColor(token.Value)
		if err != nil {
			continue
		}

		colors = append(colors, protocol.ColorInformation{
			Range: protocol.Range{
				Start: protocol.Position{
					Line:      variable.Range.Start.Line,
					Character: variable.Range.Start.Character,
				},
				End: protocol.Position{
					Line:      variable.Range.End.Line,
					Character: variable.Range.End.Character,
				},
			},
			Color: *color,
		})
	}

	fmt.Fprintf(os.Stderr, "[DTLS] Found %d colors\n", len(colors))

	return colors, nil
}

// handleColorPresentation handles the textDocument/colorPresentation request
func (s *Server) handleColorPresentation(context *glsp.Context, params *protocol.ColorPresentationParams) ([]protocol.ColorPresentation, error) {
	return ColorPresentation(s, context, params)
}

// ColorPresentation handles the textDocument/colorPresentation request
func ColorPresentation(ctx types.ServerContext, context *glsp.Context, params *protocol.ColorPresentationParams) ([]protocol.ColorPresentation, error) {
	uri := params.TextDocument.URI
	color := params.Color

	fmt.Fprintf(os.Stderr, "[DTLS] ColorPresentation requested: %s\n", uri)

	// Convert the color to different formats
	presentations := []protocol.ColorPresentation{
		{
			Label: formatColorHex(color),
		},
		{
			Label: formatColorRGB(color),
		},
		{
			Label: formatColorRGBA(color),
		},
		{
			Label: formatColorHSL(color),
		},
	}

	return presentations, nil
}

// parseColor parses a color string (hex, rgb, etc.) and returns a protocol.Color
func parseColor(value string) (*protocol.Color, error) {
	value = strings.TrimSpace(value)

	// Parse hex colors (#RGB, #RRGGBB, #RRGGBBAA)
	if strings.HasPrefix(value, "#") {
		hex := strings.TrimPrefix(value, "#")

		// Expand 3-digit hex to 6-digit
		if len(hex) == 3 {
			hex = string([]byte{hex[0], hex[0], hex[1], hex[1], hex[2], hex[2]})
		}

		// Parse 6 or 8 digit hex
		if len(hex) == 6 || len(hex) == 8 {
			r, err := strconv.ParseUint(hex[0:2], 16, 8)
			if err != nil {
				return nil, err
			}
			g, err := strconv.ParseUint(hex[2:4], 16, 8)
			if err != nil {
				return nil, err
			}
			b, err := strconv.ParseUint(hex[4:6], 16, 8)
			if err != nil {
				return nil, err
			}

			alpha := 1.0
			if len(hex) == 8 {
				a, err := strconv.ParseUint(hex[6:8], 16, 8)
				if err != nil {
					return nil, err
				}
				alpha = float64(a) / 255.0
			}

			return &protocol.Color{
				Red:   protocol.Decimal(float64(r) / 255.0),
				Green: protocol.Decimal(float64(g) / 255.0),
				Blue:  protocol.Decimal(float64(b) / 255.0),
				Alpha: protocol.Decimal(alpha),
			}, nil
		}
	}

	// TODO: Parse rgb(), rgba(), hsl(), hsla() formats

	return nil, fmt.Errorf("unsupported color format: %s", value)
}

// formatColorHex formats a color as hex
func formatColorHex(color protocol.Color) string {
	r := uint8(color.Red * 255)
	g := uint8(color.Green * 255)
	b := uint8(color.Blue * 255)

	if color.Alpha < 1.0 {
		a := uint8(color.Alpha * 255)
		return fmt.Sprintf("#%02x%02x%02x%02x", r, g, b, a)
	}

	return fmt.Sprintf("#%02x%02x%02x", r, g, b)
}

// formatColorRGB formats a color as rgb()
func formatColorRGB(color protocol.Color) string {
	r := uint8(color.Red * 255)
	g := uint8(color.Green * 255)
	b := uint8(color.Blue * 255)
	return fmt.Sprintf("rgb(%d, %d, %d)", r, g, b)
}

// formatColorRGBA formats a color as rgba()
func formatColorRGBA(color protocol.Color) string {
	r := uint8(color.Red * 255)
	g := uint8(color.Green * 255)
	b := uint8(color.Blue * 255)
	return fmt.Sprintf("rgba(%d, %d, %d, %.2f)", r, g, b, color.Alpha)
}

// formatColorHSL formats a color as hsl()
func formatColorHSL(color protocol.Color) string {
	h, s, l := rgbToHSL(float64(color.Red), float64(color.Green), float64(color.Blue))
	return fmt.Sprintf("hsl(%.0f, %.0f%%, %.0f%%)", h, s*100, l*100)
}

// rgbToHSL converts RGB to HSL
func rgbToHSL(r, g, b float64) (h, s, l float64) {
	max := r
	if g > max {
		max = g
	}
	if b > max {
		max = b
	}

	min := r
	if g < min {
		min = g
	}
	if b < min {
		min = b
	}

	l = (max + min) / 2

	if max == min {
		h = 0
		s = 0
	} else {
		d := max - min
		if l > 0.5 {
			s = d / (2 - max - min)
		} else {
			s = d / (max + min)
		}

		switch max {
		case r:
			h = (g - b) / d
			if g < b {
				h += 6
			}
		case g:
			h = (b-r)/d + 2
		case b:
			h = (r-g)/d + 4
		}

		h *= 60
	}

	return h, s, l
}
