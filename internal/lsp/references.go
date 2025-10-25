package lsp

import (
	"fmt"
	"os"
	"strings"

	"github.com/bennypowers/design-tokens-language-server/internal/parser/css"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// handleReferences handles the textDocument/references request
func (s *Server) handleReferences(context *glsp.Context, params *protocol.ReferenceParams) ([]protocol.Location, error) {
	return s.GetReferences(params)
}

// GetReferences returns all references to a token
func (s *Server) GetReferences(params *protocol.ReferenceParams) ([]protocol.Location, error) {
	uri := params.TextDocument.URI
	position := params.Position

	fmt.Fprintf(os.Stderr, "[DTLS] References requested: %s at line %d, char %d\n", uri, position.Line, position.Character)

	// Get document
	doc := s.documents.Get(uri)
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
		if s.isPositionInVarCall(position, varCall) {
			targetTokenName = varCall.TokenName
			break
		}
	}

	if targetTokenName == "" {
		return nil, nil
	}

	// Look up the token
	token := s.tokens.Get(targetTokenName)
	if token == nil {
		return nil, nil
	}

	fmt.Fprintf(os.Stderr, "[DTLS] Finding references for %s\n", targetTokenName)

	// Find all references across all CSS documents
	var locations []protocol.Location

	for _, document := range s.documents.GetAll() {
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

// getRangesForSubstring finds all ranges where a substring appears in the document
func (s *Server) getRangesForSubstring(content, substring string) []protocol.Range {
	var ranges []protocol.Range
	lines := strings.Split(content, "\n")

	for lineNum, line := range lines {
		startIdx := 0
		for {
			idx := strings.Index(line[startIdx:], substring)
			if idx == -1 {
				break
			}

			actualIdx := startIdx + idx
			ranges = append(ranges, protocol.Range{
				Start: protocol.Position{
					Line:      uint32(lineNum),
					Character: uint32(actualIdx),
				},
				End: protocol.Position{
					Line:      uint32(lineNum),
					Character: uint32(actualIdx + len(substring)),
				},
			})

			startIdx = actualIdx + 1
		}
	}

	return ranges
}
