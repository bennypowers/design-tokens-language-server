package definition

import (
	"fmt"
	"os"

	"github.com/bennypowers/design-tokens-language-server/internal/parser/css"
	"github.com/bennypowers/design-tokens-language-server/lsp/types"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// handleDefinition handles the textDocument/definition request

// Definition returns the definition location for a token
func Definition(ctx types.ServerContext, context *glsp.Context, params *protocol.DefinitionParams) (any, error) {
	uri := params.TextDocument.URI
	position := params.Position

	fmt.Fprintf(os.Stderr, "[DTLS] Definition requested: %s at line %d, char %d\n", uri, position.Line, position.Character)

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
		return nil, nil
	}

	// Find var() call at the cursor position
	for _, varCall := range result.VarCalls {
		if isPositionInVarCall(position, varCall) {
			// Look up the token
			token := ctx.Token(varCall.TokenName)
			if token == nil {
				// Unknown token
				return nil, nil
			}

			// Return the definition location in the token file
			if token.DefinitionURI != "" && len(token.Path) > 0 {
				// Calculate the range in the token file
				// For now, we return line 0 since we don't have precise line tracking
				// In a full implementation, we'd parse the token file to find the exact line
				location := protocol.Location{
					URI: token.DefinitionURI,
					Range: protocol.Range{
						Start: protocol.Position{Line: 0, Character: 0},
						End:   protocol.Position{Line: 0, Character: 0},
					},
				}

				fmt.Fprintf(os.Stderr, "[DTLS] Found definition for %s in %s\n", varCall.TokenName, token.DefinitionURI)
				return []protocol.Location{location}, nil
			}

			return nil, nil
		}
	}

	return nil, nil
}

// isPositionInVarCall checks if a position is within a var() call
func isPositionInVarCall(pos protocol.Position, varCall *css.VarCall) bool {
	// Check if position is within the var call range
	if pos.Line < varCall.Range.Start.Line || pos.Line > varCall.Range.End.Line {
		return false
	}

	if pos.Line == varCall.Range.Start.Line && pos.Character < varCall.Range.Start.Character {
		return false
	}

	if pos.Line == varCall.Range.End.Line && pos.Character >= varCall.Range.End.Character {
		return false
	}

	return true
}
