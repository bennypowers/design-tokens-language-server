package hover

import (
	"bytes"
	"fmt"
	"os"
	"text/template"

	"bennypowers.dev/dtls/internal/parser/css"
	"bennypowers.dev/dtls/internal/tokens"
	"bennypowers.dev/dtls/lsp/types"
	"github.com/tliron/glsp"
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
func Hover(ctx types.ServerContext, context *glsp.Context, params *protocol.HoverParams) (*protocol.Hover, error) {
	uri := params.TextDocument.URI
	position := params.Position

	fmt.Fprintf(os.Stderr, "[DTLS] Hover requested: %s at line %d, char %d\n", uri, position.Line, position.Character)

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
		return nil, fmt.Errorf("failed to parse CSS: %w", err)
	}

	// Find var() call at the cursor position
	for _, varCall := range result.VarCalls {
		if isPositionInRange(position, varCall.Range) {
			// Look up the token
			token := ctx.Token(varCall.TokenName)
			if token == nil {
				// Token not found - render unknown token message
				content, err := renderUnknownToken(varCall.TokenName)
				if err != nil {
					return nil, fmt.Errorf("failed to render unknown token message: %w", err)
				}
				return &protocol.Hover{
					Contents: protocol.MarkupContent{
						Kind:  protocol.MarkupKindMarkdown,
						Value: content,
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
						Line:      varCall.Range.Start.Line,
						Character: varCall.Range.Start.Character,
					},
					End: protocol.Position{
						Line:      varCall.Range.End.Line,
						Character: varCall.Range.End.Character,
					},
				},
			}, nil
		}
	}

	// Also check variable declarations
	for _, variable := range result.Variables {
		if isPositionInRange(position, variable.Range) {
			// Look up the token by the variable name
			token := ctx.Token(variable.Name)
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
						Line:      variable.Range.Start.Line,
						Character: variable.Range.Start.Character,
					},
					End: protocol.Position{
						Line:      variable.Range.End.Line,
						Character: variable.Range.End.Character,
					},
				},
			}, nil
		}
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
