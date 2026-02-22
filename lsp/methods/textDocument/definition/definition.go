package definition

import (
	"bennypowers.dev/dtls/internal/log"
	"fmt"

	"bennypowers.dev/dtls/internal/parser"
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

	// Only process CSS-supported files beyond this point
	if !parser.IsCSSSupportedLanguage(doc.LanguageID()) {
		return nil, nil
	}

	// Parse CSS to find var() calls
	result, err := parser.ParseCSSFromDocument(doc.Content(), doc.LanguageID())
	if err != nil {
		return nil, fmt.Errorf("failed to parse CSS: %w", err)
	}
	if result == nil {
		return nil, nil
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
				targetRange := protocol.Range{
					Start: protocol.Position{Line: token.Line, Character: token.Character},
					End:   protocol.Position{Line: token.Line, Character: token.Character},
				}

				log.Info("Found definition for %s in %s at line %d, char %d",
					varCall.TokenName, token.DefinitionURI, token.Line, token.Character)

				// Return LocationLink when client supports it (includes origin selection range)
				if req.Server.SupportsDefinitionLinks() {
					originRange := protocol.Range{
						Start: protocol.Position{
							Line:      varCall.Range.Start.Line,
							Character: varCall.Range.Start.Character,
						},
						End: protocol.Position{
							Line:      varCall.Range.End.Line,
							Character: varCall.Range.End.Character,
						},
					}
					return []protocol.LocationLink{{
						OriginSelectionRange: &originRange,
						TargetURI:            protocol.DocumentUri(token.DefinitionURI),
						TargetRange:          targetRange,
						TargetSelectionRange: targetRange,
					}}, nil
				}

				// Return Location for legacy clients
				return []protocol.Location{{
					URI:   token.DefinitionURI,
					Range: targetRange,
				}}, nil
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
