package lsp

import (
	"fmt"
	"os"
	"strings"

	"github.com/bennypowers/design-tokens-language-server/internal/parser/css"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// handleHover handles the textDocument/hover request
func (s *Server) handleHover(context *glsp.Context, params *protocol.HoverParams) (*protocol.Hover, error) {
	uri := params.TextDocument.URI
	position := params.Position

	fmt.Fprintf(os.Stderr, "[DTLS] Hover requested: %s at line %d, char %d\n", uri, position.Line, position.Character)

	// Get document
	doc := s.documents.Get(uri)
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

	// Find var() call at the cursor position
	for _, varCall := range result.VarCalls {
		if s.isPositionInRange(position, varCall.Range) {
			// Look up the token
			token := s.tokens.Get(varCall.TokenName)
			if token == nil {
				// Token not found
				return &protocol.Hover{
					Contents: protocol.MarkupContent{
						Kind:  protocol.MarkupKindMarkdown,
						Value: fmt.Sprintf("❌ **Unknown token**: `%s`\n\nThis token is not defined in any loaded token files.", varCall.TokenName),
					},
				}, nil
			}

			// Build hover content
			var content strings.Builder
			content.WriteString(fmt.Sprintf("# %s\n\n", token.CSSVariableName()))

			if token.Description != "" {
				content.WriteString(fmt.Sprintf("%s\n\n", token.Description))
			}

			content.WriteString(fmt.Sprintf("**Value**: `%s`\n", token.Value))

			if token.Type != "" {
				content.WriteString(fmt.Sprintf("**Type**: `%s`\n", token.Type))
			}

			if token.Deprecated {
				content.WriteString("\n⚠️ **DEPRECATED**")
				if token.DeprecationMessage != "" {
					content.WriteString(fmt.Sprintf(": %s", token.DeprecationMessage))
				}
				content.WriteString("\n")
			}

			if token.FilePath != "" {
				content.WriteString(fmt.Sprintf("\n*Defined in: %s*\n", token.FilePath))
			}

			return &protocol.Hover{
				Contents: protocol.MarkupContent{
					Kind:  protocol.MarkupKindMarkdown,
					Value: content.String(),
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
		if s.isPositionInRange(position, variable.Range) {
			// Look up the token by the variable name
			token := s.tokens.Get(variable.Name)
			if token == nil {
				return nil, nil
			}

			// Build hover content for declaration
			var content strings.Builder
			content.WriteString(fmt.Sprintf("# %s\n\n", token.CSSVariableName()))

			if token.Description != "" {
				content.WriteString(fmt.Sprintf("%s\n\n", token.Description))
			}

			content.WriteString(fmt.Sprintf("**Value**: `%s`\n", token.Value))

			if token.Type != "" {
				content.WriteString(fmt.Sprintf("**Type**: `%s`\n", token.Type))
			}

			return &protocol.Hover{
				Contents: protocol.MarkupContent{
					Kind:  protocol.MarkupKindMarkdown,
					Value: content.String(),
				},
			}, nil
		}
	}

	return nil, nil
}

// isPositionInRange checks if a position is within a range
func (s *Server) isPositionInRange(pos protocol.Position, r css.Range) bool {
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
