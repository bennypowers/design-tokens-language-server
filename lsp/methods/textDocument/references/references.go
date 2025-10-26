package references

import (
	"fmt"
	"os"

	"github.com/bennypowers/design-tokens-language-server/internal/parser/css"
	"github.com/bennypowers/design-tokens-language-server/lsp/types"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// handleReferences handles the textDocument/references request

// References returns all references to a token
func References(ctx types.ServerContext, context *glsp.Context, params *protocol.ReferenceParams) ([]protocol.Location, error) {
	uri := params.TextDocument.URI
	position := params.Position

	fmt.Fprintf(os.Stderr, "[DTLS] References requested: %s at line %d, char %d\n", uri, position.Line, position.Character)

	// Get document
	doc := ctx.Document(uri)
	if doc == nil {
		return nil, nil
	}

	// Only process CSS files for now
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

	// Find which token is at the cursor
	var targetTokenName string
	for _, varCall := range result.VarCalls {
		if isPositionInVarCall(position, varCall) {
			targetTokenName = varCall.TokenName
			break
		}
	}

	if targetTokenName == "" {
		return nil, nil
	}

	// Look up the token
	token := ctx.Token(targetTokenName)
	if token == nil {
		return nil, nil
	}

	fmt.Fprintf(os.Stderr, "[DTLS] Finding references for %s\n", targetTokenName)

	// Find all references across all CSS documents
	var locations []protocol.Location

	for _, document := range ctx.AllDocuments() {
		if document.LanguageID() != "css" {
			continue
		}

		// Parse the document
		docParser := css.AcquireParser()
		docResult, err := docParser.Parse(document.Content())
		css.ReleaseParser(docParser)
		if err != nil {
			continue
		}

		// Find all var() calls to this token
		for _, varCall := range docResult.VarCalls {
			if varCall.TokenName == targetTokenName {
				location := protocol.Location{
					URI: document.URI(),
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
				}
				locations = append(locations, location)
			}
		}
	}

	// Include declaration if requested
	if params.Context.IncludeDeclaration && token.DefinitionURI != "" {
		location := protocol.Location{
			URI: token.DefinitionURI,
			Range: protocol.Range{
				Start: protocol.Position{Line: 0, Character: 0},
				End:   protocol.Position{Line: 0, Character: 0},
			},
		}
		locations = append(locations, location)
	}

	fmt.Fprintf(os.Stderr, "[DTLS] Found %d references\n", len(locations))
	return locations, nil
}
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
