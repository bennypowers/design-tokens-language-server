package references

import (
	"fmt"
	"math"
	"os"
	"strings"

	"bennypowers.dev/dtls/internal/documents"
	"bennypowers.dev/dtls/internal/tokens"
	"bennypowers.dev/dtls/lsp/types"
	"github.com/tidwall/jsonc"
	protocol "github.com/tliron/glsp/protocol_3_16"
	"gopkg.in/yaml.v3"
)

// validateTokenContext validates the basic request context and extracts the token at cursor.
// Returns the token and tokenName, or (nil, "") if validation fails.
func validateTokenContext(req *types.RequestContext, uri protocol.DocumentUri, position protocol.Position) (*tokens.Token, string) {
	// Get document
	doc := req.Server.Document(uri)
	if doc == nil {
		return nil, ""
	}

	// Return nil for CSS files (let css-ls handle it)
	if doc.LanguageID() == "css" {
		return nil, ""
	}

	// For JSON/YAML files, find the token at cursor position
	tokenName := findTokenAtPosition(doc.Content(), position, doc.LanguageID())
	fmt.Fprintf(os.Stderr, "[DTLS] Token name at cursor: '%s'\n", tokenName)
	if tokenName == "" {
		return nil, ""
	}

	// Look up the token
	token := req.Server.Token(tokenName)
	fmt.Fprintf(os.Stderr, "[DTLS] Token lookup result: %v\n", token != nil)
	if token == nil {
		return nil, ""
	}

	return token, tokenName
}

// isValidCSSReference checks if a CSS variable reference is valid by verifying
// the character after the match is either ',' or ')'.
// This excludes things like --token-color-red: and --token-color-reddish)
func isValidCSSReference(content string, endPos protocol.Position) bool {
	line := getLine(content, int(endPos.Line))
	lineLen := len(line)

	// Check for overflow - if line length exceeds uint32, position is considered out of bounds
	if lineLen > math.MaxUint32 {
		return false
	}

	if endPos.Character < uint32(lineLen) {
		charAfter := getCharAt(content, endPos)
		return charAfter == ',' || charAfter == ')'
	}
	return false
}

// findCSSReferences finds all CSS var() references to a token across documents.
// Adds valid references to the locationMap for deduplication.
func findCSSReferences(docs []*documents.Document, cssVarName string, locationMap map[string]protocol.Location) {
	for _, document := range docs {
		if document.LanguageID() != "css" {
			continue
		}

		docContent := document.Content()
		docURI := document.URI()
		ranges := findSubstringRanges(docContent, cssVarName)

		for _, r := range ranges {
			if isValidCSSReference(docContent, r.End) {
				loc := protocol.Location{URI: docURI, Range: r}
				key := fmt.Sprintf("%s:%d:%d", docURI, r.Start.Line, r.Start.Character)
				locationMap[key] = loc
			}
		}
	}
}

// findJSONReferences finds all JSON/YAML token references across documents.
// Adds references to the locationMap for deduplication.
func findJSONReferences(docs []*documents.Document, tokenReference string, locationMap map[string]protocol.Location) {
	if tokenReference == "" {
		return
	}

	for _, document := range docs {
		if document.LanguageID() == "css" {
			continue
		}

		docContent := document.Content()
		docURI := document.URI()
		ranges := findSubstringRanges(docContent, tokenReference)

		for _, r := range ranges {
			loc := protocol.Location{URI: docURI, Range: r}
			key := fmt.Sprintf("%s:%d:%d", docURI, r.Start.Line, r.Start.Character)
			locationMap[key] = loc
		}
	}
}

// addDeclarationIfRequested adds the token declaration location if requested.
// Modifies the locations slice in place.
func addDeclarationIfRequested(req *types.RequestContext, params *protocol.ReferenceParams, token *tokens.Token, locations *[]protocol.Location) {
	if !params.Context.IncludeDeclaration || token.DefinitionURI == "" {
		return
	}

	defDoc := req.Server.Document(token.DefinitionURI)
	if defDoc == nil {
		return
	}

	defRange := findTokenDefinitionRange(defDoc.Content(), token.Path, defDoc.LanguageID())
	location := protocol.Location{
		URI:   token.DefinitionURI,
		Range: defRange,
	}
	*locations = append(*locations, location)
}

// References returns all references to a token
// For CSS files: returns nil (let css-ls handle it)
// For JSON/YAML files: finds all references to the token at cursor
func References(req *types.RequestContext, params *protocol.ReferenceParams) ([]protocol.Location, error) {
	uri := params.TextDocument.URI
	position := params.Position

	fmt.Fprintf(os.Stderr, "[DTLS] References requested: %s at line %d, char %d\n", uri, position.Line, position.Character)

	// Validate context and get token
	token, tokenName := validateTokenContext(req, uri, position)
	if token == nil {
		return nil, nil
	}

	fmt.Fprintf(os.Stderr, "[DTLS] Finding references for %s (CSS name: %s, reference: %s)\n",
		tokenName, token.CSSVariableName(), token.Reference)

	// Find all references across all documents
	var locations []protocol.Location
	cssVarName := token.CSSVariableName()
	tokenReference := token.Reference

	// Deduplicate locations using a map (JSON.stringify equivalent)
	locationMap := make(map[string]protocol.Location)

	// Find CSS var() references
	findCSSReferences(req.Server.AllDocuments(), cssVarName, locationMap)

	// Find JSON/YAML token references
	findJSONReferences(req.Server.AllDocuments(), tokenReference, locationMap)

	// Convert map to slice
	for _, loc := range locationMap {
		locations = append(locations, loc)
	}

	// Include declaration if requested
	addDeclarationIfRequested(req, params, token, &locations)

	fmt.Fprintf(os.Stderr, "[DTLS] Found %d references\n", len(locations))
	return locations, nil
}

// findTokenAtPosition finds the token name at the given position in a JSON/YAML file
// Uses yaml.v3 AST for robust parsing (YAML is a superset of JSON)
func findTokenAtPosition(content string, pos protocol.Position, languageID string) string {
	var data []byte

	// For JSON/JSONC, strip comments first (preserves line numbers)
	if languageID == "json" || languageID == "jsonc" {
		data = jsonc.ToJSON([]byte(content))
	} else {
		data = []byte(content)
	}

	// Parse with yaml.v3 (works for both JSON and YAML)
	var root yaml.Node
	if err := yaml.Unmarshal(data, &root); err != nil {
		return ""
	}

	if len(root.Content) == 0 {
		return ""
	}

	// yaml.v3 uses 1-based line/column, LSP uses 0-based
	yamlLine := int(pos.Line) + 1
	yamlCol := int(pos.Character) + 1

	// Find the path at the cursor position
	path := findPathAtPosition(root.Content[0], yamlLine, yamlCol, nil)
	if len(path) > 0 {
		return strings.Join(path, "-")
	}

	return ""
}

// findPathAtPosition recursively finds the token path at a given position in the AST
func findPathAtPosition(node *yaml.Node, targetLine, targetCol int, currentPath []string) []string {
	if node.Kind == yaml.MappingNode {
		// Mapping nodes contain key-value pairs in Content
		// Content[0] = key1, Content[1] = value1, Content[2] = key2, Content[3] = value2, ...
		for i := 0; i < len(node.Content); i += 2 {
			if i >= len(node.Content)-1 {
				break
			}

			keyNode := node.Content[i]
			valueNode := node.Content[i+1]

			// Skip $-prefixed keys (DTCG metadata)
			if strings.HasPrefix(keyNode.Value, "$") {
				continue
			}

			// Check if cursor is on this key
			// keyNode.Column points to the start of the key (including quote for JSON)
			keyStartCol := keyNode.Column
			keyEndCol := keyStartCol + len(keyNode.Value)

			if keyNode.Line == targetLine && targetCol >= keyStartCol && targetCol < keyEndCol {
				// Cursor is on this key - return the path including this key
				newPath := append(append([]string{}, currentPath...), keyNode.Value)
				return newPath
			}

			// Recurse into the value to check nested structures
			newPath := append(append([]string{}, currentPath...), keyNode.Value)
			if result := findPathAtPosition(valueNode, targetLine, targetCol, newPath); result != nil {
				return result
			}
		}
	}

	return nil
}

// findSubstringRanges finds all occurrences of a substring in content
func findSubstringRanges(content, substring string) []protocol.Range {
	var ranges []protocol.Range
	lines := strings.Split(content, "\n")

	for lineNum, line := range lines {
		offset := 0
		for {
			idx := strings.Index(line[offset:], substring)
			if idx == -1 {
				break
			}
			actualIdx := offset + idx
			endIdx := actualIdx + len(substring)

			// Check for overflow - skip positions that exceed uint32 limits
			if lineNum > math.MaxUint32 || actualIdx > math.MaxUint32 || endIdx > math.MaxUint32 {
				fmt.Fprintf(os.Stderr, "[DTLS] Warning: Skipping reference at line %d, char %d (exceeds uint32 limit)\n", lineNum, actualIdx)
				offset = endIdx
				continue
			}

			// Convert to uint32 after validation (gosec doesn't recognize validation above)
			lineU32 := uint32(lineNum)           //nolint:gosec // G115: validated above
			actualIdxU32 := uint32(actualIdx)    //nolint:gosec // G115: validated above
			endIdxU32 := uint32(endIdx)          //nolint:gosec // G115: validated above

			ranges = append(ranges, protocol.Range{
				Start: protocol.Position{
					Line:      lineU32,
					Character: actualIdxU32,
				},
				End: protocol.Position{
					Line:      lineU32,
					Character: endIdxU32,
				},
			})
			offset = endIdx
		}
	}

	return ranges
}

// findTokenDefinitionRange finds the range of a token definition in a JSON/YAML file
func findTokenDefinitionRange(content string, path []string, languageID string) protocol.Range {
	// Simple approach: find the last key in the path
	if len(path) == 0 {
		return protocol.Range{
			Start: protocol.Position{Line: 0, Character: 0},
			End:   protocol.Position{Line: 0, Character: 0},
		}
	}

	lastKey := path[len(path)-1]
	lines := strings.Split(content, "\n")

	for lineNum, line := range lines {
		if strings.Contains(line, lastKey) {
			// Find the position of the key
			idx := strings.Index(line, lastKey)
			if idx != -1 {
				endIdx := idx + len(lastKey)

				// Check for overflow - skip if exceeds uint32 limits
				if lineNum > math.MaxUint32 || idx > math.MaxUint32 || endIdx > math.MaxUint32 {
					fmt.Fprintf(os.Stderr, "[DTLS] Warning: Token definition position exceeds uint32 limit at line %d\n", lineNum)
					continue
				}

				// Convert to uint32 after validation (gosec doesn't recognize validation above)
				lineU32 := uint32(lineNum)    //nolint:gosec // G115: validated above
				idxU32 := uint32(idx)          //nolint:gosec // G115: validated above
				endIdxU32 := uint32(endIdx)    //nolint:gosec // G115: validated above

				return protocol.Range{
					Start: protocol.Position{
						Line:      lineU32,
						Character: idxU32,
					},
					End: protocol.Position{
						Line:      lineU32,
						Character: endIdxU32,
					},
				}
			}
		}
	}

	return protocol.Range{
		Start: protocol.Position{Line: 0, Character: 0},
		End:   protocol.Position{Line: 0, Character: 0},
	}
}

// getLine returns the line at the given index
func getLine(content string, lineNum int) string {
	lines := strings.Split(content, "\n")
	if lineNum >= 0 && lineNum < len(lines) {
		return lines[lineNum]
	}
	return ""
}

// getCharAt returns the character at the given position
func getCharAt(content string, pos protocol.Position) rune {
	lines := strings.Split(content, "\n")
	if int(pos.Line) < len(lines) {
		line := lines[pos.Line]
		if int(pos.Character) < len(line) {
			return rune(line[pos.Character])
		}
	}
	return 0
}
