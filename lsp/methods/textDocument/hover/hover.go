package hover

import (
	"bytes"
	"fmt"
	"os"
	"text/template"

	"bennypowers.dev/dtls/internal/parser/css"
	"bennypowers.dev/dtls/internal/tokens"
	"bennypowers.dev/dtls/lsp/types"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// Template for token hover content
// Note: {{.CSSVariableName}} calls the Token.CSSVariableName() method (not a field)
var tokenHoverTemplate = template.Must(template.New("tokenHover").Parse(`# {{.CSSVariableName}}
{{if .Description}}
{{.Description}}
{{end}}
**Value**: ` + "`{{.Value}}`" + `
{{if .Type}}**Type**: ` + "`{{.Type}}`" + `
{{end}}{{if .Deprecated}}
⚠️ **DEPRECATED**{{if .DeprecationMessage}}: {{.DeprecationMessage}}{{end}}
{{end}}{{if .FilePath}}
*Defined in: {{.FilePath}}*
{{end}}`))

// Template for unknown token message
var unknownTokenTemplate = template.Must(template.New("unknownToken").Parse(`❌ **Unknown token**: ` + "`{{.}}`" + `

This token is not defined in any loaded token files.`))

// renderTokenHover renders the hover markdown for a token
func renderTokenHover(token *tokens.Token) (string, error) {
	var buf bytes.Buffer
	if err := tokenHoverTemplate.Execute(&buf, token); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// renderUnknownToken renders the hover markdown for an unknown token
func renderUnknownToken(tokenName string) (string, error) {
	var buf bytes.Buffer
	if err := unknownTokenTemplate.Execute(&buf, tokenName); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// Hover handles the textDocument/hover request
func Hover(req *types.RequestContext, params *protocol.HoverParams) (*protocol.Hover, error) {
	uri := params.TextDocument.URI
	position := params.Position

	fmt.Fprintf(os.Stderr, "[DTLS] Hover requested: %s at line %d, char %d\n", uri, position.Line, position.Character)

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

	// Find the innermost var() call at the cursor position
	// When var() calls are nested (e.g., var(--outer, var(--inner, fallback))),
	// we want to find the smallest (innermost) range that contains the cursor
	var bestVarCall *css.VarCall
	var smallestRangeSize = -1

	for _, varCall := range result.VarCalls {
		if isPositionInRange(position, varCall.Range) {
			rangeSize := calculateRangeSize(varCall.Range)
			if smallestRangeSize == -1 || rangeSize < smallestRangeSize {
				smallestRangeSize = rangeSize
				bestVarCall = varCall
			}
		}
	}

	// Process the innermost var() call if found
	if bestVarCall != nil {
		// Look up the token
		token := req.Server.Token(bestVarCall.TokenName)
		if token == nil {
			// Token not found - render unknown token message
			content, err := renderUnknownToken(bestVarCall.TokenName)
			if err != nil {
				return nil, fmt.Errorf("failed to render unknown token message: %w", err)
			}
			return &protocol.Hover{
				Contents: protocol.MarkupContent{
					Kind:  protocol.MarkupKindMarkdown,
					Value: content,
				},
				Range: &protocol.Range{
					Start: protocol.Position{
						Line:      bestVarCall.Range.Start.Line,
						Character: bestVarCall.Range.Start.Character,
					},
					End: protocol.Position{
						Line:      bestVarCall.Range.End.Line,
						Character: bestVarCall.Range.End.Character,
					},
				},
			}, nil
		}

		// Render token hover content using template
		content, err := renderTokenHover(token)
		if err != nil {
			return nil, fmt.Errorf("failed to render token hover: %w", err)
		}

		return &protocol.Hover{
			Contents: protocol.MarkupContent{
				Kind:  protocol.MarkupKindMarkdown,
				Value: content,
			},
			Range: &protocol.Range{
				Start: protocol.Position{
					Line:      bestVarCall.Range.Start.Line,
					Character: bestVarCall.Range.Start.Character,
				},
				End: protocol.Position{
					Line:      bestVarCall.Range.End.Line,
					Character: bestVarCall.Range.End.Character,
				},
			},
		}, nil
	}

	// Also check variable declarations (using same innermost-match logic)
	var bestVariable *css.Variable
	smallestRangeSize = -1

	for _, variable := range result.Variables {
		if isPositionInRange(position, variable.Range) {
			rangeSize := calculateRangeSize(variable.Range)
			if smallestRangeSize == -1 || rangeSize < smallestRangeSize {
				smallestRangeSize = rangeSize
				bestVariable = variable
			}
		}
	}

	// Process the innermost variable declaration if found
	if bestVariable != nil {
		// Look up the token by the variable name
		token := req.Server.Token(bestVariable.Name)
		if token == nil {
			return nil, nil
		}

		// Render token hover content using template
		content, err := renderTokenHover(token)
		if err != nil {
			return nil, fmt.Errorf("failed to render token hover for declaration: %w", err)
		}

		return &protocol.Hover{
			Contents: protocol.MarkupContent{
				Kind:  protocol.MarkupKindMarkdown,
				Value: content,
			},
			Range: &protocol.Range{
				Start: protocol.Position{
					Line:      bestVariable.Range.Start.Line,
					Character: bestVariable.Range.Start.Character,
				},
				End: protocol.Position{
					Line:      bestVariable.Range.End.Line,
					Character: bestVariable.Range.End.Character,
				},
			},
		}, nil
	}

	return nil, nil
}

// isPositionInRange checks if a position is within a range
func isPositionInRange(pos protocol.Position, r css.Range) bool {
	// Convert to comparable format
	posLine := pos.Line
	posChar := pos.Character

	// Check if position is within range
	if posLine < r.Start.Line || posLine > r.End.Line {
		return false
	}

	if posLine == r.Start.Line && posChar < r.Start.Character {
		return false
	}

	if posLine == r.End.Line && posChar >= r.End.Character {
		return false
	}

	return true
}

// calculateRangeSize computes a metric for range size to find the smallest (innermost) range
// For nested ranges, we want to select the innermost one containing the cursor position
func calculateRangeSize(r css.Range) int {
	// Calculate total character span across all lines
	// Simple metric: (line difference * 10000) + character difference
	// This ensures multi-line ranges are always larger than single-line ranges
	lineDiff := r.End.Line - r.Start.Line
	charDiff := r.End.Character - r.Start.Character

	if lineDiff == 0 {
		// Single line: just use character difference
		return int(charDiff)
	}

	// Multi-line: weight lines heavily to ensure they're always larger
	return int(lineDiff)*10000 + int(charDiff)
}
