package documentcolor

import (
	"bennypowers.dev/dtls/internal/log"
	"fmt"
	"strings"

	"bennypowers.dev/dtls/internal/parser/css"
	"bennypowers.dev/dtls/lsp/types"
	"github.com/mazznoer/csscolorparser"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// DocumentColor handles the textDocument/documentColor request
func DocumentColor(req *types.RequestContext, params *protocol.DocumentColorParams) ([]protocol.ColorInformation, error) {
	uri := params.TextDocument.URI

	log.Info("DocumentColor requested: %s", uri)

	// Get document
	doc := req.Server.Document(uri)
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
		return nil, fmt.Errorf("failed to parse CSS: %w", err)
	}

	var colors []protocol.ColorInformation
	var parseErrors []error

	// Find all var() calls that reference color tokens
	for _, varCall := range result.VarCalls {
		// Look up the token
		token := req.Server.Token(varCall.TokenName)
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
			log.Info("Failed to parse color %s: %v", token.Value, err)
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
		token := req.Server.Token(variable.Name)
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
			log.Info("Failed to parse color %s: %v", token.Value, err)
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

	log.Info("Found %d colors", len(colors))

	// Add parse errors as warnings
	// Don't fail the operation - we can still return partial results
	// Middleware will log these warnings after successful completion
	for _, err := range parseErrors {
		req.AddWarning(err)
	}

	return colors, nil
}

// ColorPresentation handles the textDocument/colorPresentation request
// Returns token names that have the same color value as the requested color
func ColorPresentation(req *types.RequestContext, params *protocol.ColorPresentationParams) ([]protocol.ColorPresentation, error) {
	uri := params.TextDocument.URI
	color := params.Color

	log.Info("ColorPresentation requested: %s", uri)

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
	for _, token := range req.Server.TokenManager().GetAll() {
		// Only process color tokens
		if token.Type != "color" {
			continue
		}

		// Parse the token's color value
		tokenColor, err := parseColor(token.Value)
		if err != nil {
			log.Info("Failed to parse color token %s (value: %s): %v", token.Name, token.Value, err)
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

	log.Info("Found %d matching color tokens", len(presentations))

	// Add parse errors as warnings
	// Don't fail the operation - we can still return partial results
	// Middleware will log these warnings after successful completion
	for _, err := range parseErrors {
		req.AddWarning(err)
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
