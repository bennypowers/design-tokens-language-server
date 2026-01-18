package definition

import (
	"bennypowers.dev/dtls/internal/log"
	"fmt"

	"bennypowers.dev/dtls/internal/parser/css"
	"bennypowers.dev/dtls/lsp/types"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// handleDefinition handles the textDocument/definition request

// Definition returns the definition location for a token
func Definition(req *types.RequestContext, params *protocol.DefinitionParams) (any, error) {
	uri := params.TextDocument.URI
	position := params.Position

	log.Info("Definition requested: %s at line %d, char %d", uri, position.Line, position.Character)

	// Get document
	doc := req.Server.Document(uri)
	if doc == nil {
		return nil, nil
	}

	// Handle token files (JSON/YAML)
	if doc.LanguageID() == "json" || doc.LanguageID() == "yaml" {
		if !req.Server.ShouldProcessAsTokenFile(uri) {
			return nil, nil
		}
		return DefinitionForTokenFile(req, doc, position)
	}

	// Only process CSS files beyond this point
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
		if isPositionInVarCall(position, varCall) {
			// Look up the token
			token := req.Server.Token(varCall.TokenName)
			if token == nil {
				// Unknown token
				return nil, nil
			}

			// Return the definition location in the token file
			if token.DefinitionURI != "" && len(token.Path) > 0 {
				location := protocol.Location{
					URI: token.DefinitionURI,
					Range: protocol.Range{
						Start: protocol.Position{Line: token.Line, Character: token.Character},
						End:   protocol.Position{Line: token.Line, Character: token.Character},
					},
				}

				log.Info("Found definition for %s in %s at line %d, char %d",
					varCall.TokenName, token.DefinitionURI, token.Line, token.Character)
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
