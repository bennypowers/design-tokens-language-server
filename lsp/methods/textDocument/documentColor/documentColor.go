package documentcolor

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"bennypowers.dev/dtls/internal/parser/css"
	"bennypowers.dev/dtls/lsp/types"
	"github.com/mazznoer/csscolorparser"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

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
	var parseErrors []error

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
			parseErrors = append(parseErrors, fmt.Errorf("failed to parse color token %s (value: %s): %w", varCall.TokenName, token.Value, err))
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
			fmt.Fprintf(os.Stderr, "[DTLS] Failed to parse color %s: %v\n", token.Value, err)
			parseErrors = append(parseErrors, fmt.Errorf("failed to parse color token %s (value: %s): %w", variable.Name, token.Value, err))
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

	// If there were parse errors, return them as a single aggregated error
	// The middleware will log this error to the LSP client via window/logMessage
	if len(parseErrors) > 0 {
		return colors, errors.Join(parseErrors...)
	}

	return colors, nil
}

// ColorPresentation handles the textDocument/colorPresentation request
// Returns token names that have the same color value as the requested color
func ColorPresentation(ctx types.ServerContext, context *glsp.Context, params *protocol.ColorPresentationParams) ([]protocol.ColorPresentation, error) {
	uri := params.TextDocument.URI
	color := params.Color

	fmt.Fprintf(os.Stderr, "[DTLS] ColorPresentation requested: %s\n", uri)

	// Convert protocol.Color to csscolorparser.Color for comparison
	requestedColor := csscolorparser.Color{
		R: float64(color.Red),
		G: float64(color.Green),
		B: float64(color.Blue),
		A: float64(color.Alpha),
	}
	requestedHex := requestedColor.HexString() // Includes alpha if < 1.0

	var presentations []protocol.ColorPresentation
	var parseErrors []error

	// Find all tokens with matching color values
	for _, token := range ctx.TokenManager().GetAll() {
		// Only process color tokens
		if token.Type != "color" {
			continue
		}

		// Parse the token's color value
		tokenColor, err := parseColor(token.Value)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[DTLS] Failed to parse color token %s (value: %s): %v\n", token.Name, token.Value, err)
			parseErrors = append(parseErrors, fmt.Errorf("failed to parse color token %s (value: %s): %w", token.Name, token.Value, err))
			continue
		}

		// Convert to csscolorparser.Color for comparison
		c := csscolorparser.Color{
			R: float64(tokenColor.Red),
			G: float64(tokenColor.Green),
			B: float64(tokenColor.Blue),
			A: float64(tokenColor.Alpha),
		}

		// Compare hex strings (includes alpha channel)
		if c.HexString() == requestedHex {
			presentations = append(presentations, protocol.ColorPresentation{
				Label: token.Name,
			})
		}
	}

	fmt.Fprintf(os.Stderr, "[DTLS] Found %d matching color tokens\n", len(presentations))

	// If there were parse errors, return them as a single aggregated error
	// The middleware will log this error to the LSP client via window/logMessage
	if len(parseErrors) > 0 {
		return presentations, errors.Join(parseErrors...)
	}

	return presentations, nil
}

// parseColor parses a color string (hex, rgb, rgba, hsl, hsla, etc.) and returns a protocol.Color
func parseColor(value string) (*protocol.Color, error) {
	value = strings.TrimSpace(value)

	// Use csscolorparser for all color formats (hex, rgb, rgba, hsl, hsla, named colors, etc.)
	// This is a battle-tested library that handles all CSS color formats correctly
	parsed, err := csscolorparser.Parse(value)
	if err != nil {
		return nil, fmt.Errorf("unsupported color format: %s", value)
	}

	// Convert csscolorparser.Color to protocol.Color
	// csscolorparser.Color has R, G, B, A fields as float64 values (0-1)
	return &protocol.Color{
		Red:   protocol.Decimal(parsed.R),
		Green: protocol.Decimal(parsed.G),
		Blue:  protocol.Decimal(parsed.B),
		Alpha: protocol.Decimal(parsed.A),
	}, nil
}
