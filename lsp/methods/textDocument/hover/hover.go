package hover

import (
	"bennypowers.dev/dtls/internal/log"
	"bytes"
	"fmt"
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

// Plaintext template for token hover content
var tokenHoverPlaintextTemplate = template.Must(template.New("tokenHoverPlaintext").Parse(`{{.CSSVariableName}}
{{if .Description}}
{{.Description}}
{{end}}
Value: {{.Value}}
{{if .Type}}Type: {{.Type}}
{{end}}{{if .Deprecated}}
DEPRECATED{{if .DeprecationMessage}}: {{.DeprecationMessage}}{{end}}
{{end}}{{if .FilePath}}
Defined in: {{.FilePath}}
{{end}}`))

// Plaintext template for unknown token message
var unknownTokenPlaintextTemplate = template.Must(template.New("unknownTokenPlaintext").Parse(`Unknown token: {{.}}

This token is not defined in any loaded token files.`))

// renderTokenHover renders the hover content for a token in the specified format
func renderTokenHover(token *tokens.Token, format protocol.MarkupKind) (string, error) {
	var buf bytes.Buffer
	var tmpl *template.Template
	if format == protocol.MarkupKindPlainText {
		tmpl = tokenHoverPlaintextTemplate
	} else {
		tmpl = tokenHoverTemplate
	}
	if err := tmpl.Execute(&buf, token); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// renderUnknownToken renders the hover content for an unknown token in the specified format
func renderUnknownToken(tokenName string, format protocol.MarkupKind) (string, error) {
	var buf bytes.Buffer
	var tmpl *template.Template
	if format == protocol.MarkupKindPlainText {
		tmpl = unknownTokenPlaintextTemplate
	} else {
		tmpl = unknownTokenTemplate
	}
	if err := tmpl.Execute(&buf, tokenName); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// findInnermostVarCall finds the innermost (smallest) var() call containing the cursor position.
// Returns nil if no var() call contains the position.
func findInnermostVarCall(position protocol.Position, varCalls []*css.VarCall) *css.VarCall {
	var bestVarCall *css.VarCall
	var smallestRangeSize = -1

	for _, varCall := range varCalls {
		if isPositionInRange(position, varCall.Range) {
			rangeSize := calculateRangeSize(varCall.Range)
			if smallestRangeSize == -1 || rangeSize < smallestRangeSize {
				smallestRangeSize = rangeSize
				bestVarCall = varCall
			}
		}
	}

	return bestVarCall
}

// findInnermostVariable finds the innermost (smallest) variable declaration containing the cursor position.
// Returns nil if no variable declaration contains the position.
func findInnermostVariable(position protocol.Position, variables []*css.Variable) *css.Variable {
	var bestVariable *css.Variable
	var smallestRangeSize = -1

	for _, variable := range variables {
		if isPositionInRange(position, variable.Range) {
			rangeSize := calculateRangeSize(variable.Range)
			if smallestRangeSize == -1 || rangeSize < smallestRangeSize {
				smallestRangeSize = rangeSize
				bestVariable = variable
			}
		}
	}

	return bestVariable
}

// createHoverResponse creates a protocol.Hover response with content in the specified format.
// This is a common helper to avoid duplication across different hover scenarios.
func createHoverResponse(content string, cssRange css.Range, format protocol.MarkupKind) *protocol.Hover {
	return &protocol.Hover{
		Contents: protocol.MarkupContent{
			Kind:  format,
			Value: content,
		},
		Range: &protocol.Range{
			Start: protocol.Position{
				Line:      cssRange.Start.Line,
				Character: cssRange.Start.Character,
			},
			End: protocol.Position{
				Line:      cssRange.End.Line,
				Character: cssRange.End.Character,
			},
		},
	}
}

// processVarCallHover processes hover for a var() call, looking up the token and rendering content.
// Returns hover response or error. Shows "unknown token" message if token is not found.
func processVarCallHover(req *types.RequestContext, varCall *css.VarCall) (*protocol.Hover, error) {
	format := req.Server.PreferredHoverFormat()
	token := req.Server.Token(varCall.TokenName)

	if token == nil {
		// Token not found - render unknown token message
		content, err := renderUnknownToken(varCall.TokenName, format)
		if err != nil {
			return nil, fmt.Errorf("failed to render unknown token message: %w", err)
		}
		return createHoverResponse(content, varCall.Range, format), nil
	}

	// Render token hover content
	content, err := renderTokenHover(token, format)
	if err != nil {
		return nil, fmt.Errorf("failed to render token hover: %w", err)
	}

	return createHoverResponse(content, varCall.Range, format), nil
}

// processVariableHover processes hover for a variable declaration, looking up the token and rendering content.
// Returns nil if the token is not found (local CSS variables without token definitions).
func processVariableHover(req *types.RequestContext, variable *css.Variable) (*protocol.Hover, error) {
	format := req.Server.PreferredHoverFormat()
	token := req.Server.Token(variable.Name)
	if token == nil {
		return nil, nil
	}

	// Render token hover content
	content, err := renderTokenHover(token, format)
	if err != nil {
		return nil, fmt.Errorf("failed to render token hover for declaration: %w", err)
	}

	return createHoverResponse(content, variable.Range, format), nil
}

// Hover handles the textDocument/hover request
func Hover(req *types.RequestContext, params *protocol.HoverParams) (*protocol.Hover, error) {
	uri := params.TextDocument.URI
	position := params.Position

	log.Info("Hover requested: %s at line %d, char %d", uri, position.Line, position.Character)

	// Get document
	doc := req.Server.Document(uri)
	if doc == nil {
		return nil, nil
	}

	// Only process CSS files
	if doc.LanguageID() != "css" {
		return nil, nil
	}

	// Parse CSS to find var() calls and variable declarations
	parser := css.AcquireParser()
	defer css.ReleaseParser(parser)
	result, err := parser.Parse(doc.Content())
	if err != nil {
		return nil, fmt.Errorf("failed to parse CSS: %w", err)
	}

	// Check for var() calls first (priority for nested cases)
	if varCall := findInnermostVarCall(position, result.VarCalls); varCall != nil {
		return processVarCallHover(req, varCall)
	}

	// Check for variable declarations
	if variable := findInnermostVariable(position, result.Variables); variable != nil {
		return processVariableHover(req, variable)
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
