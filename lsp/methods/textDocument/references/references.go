package references

import (
	"fmt"
	"os"
	"strings"

	"bennypowers.dev/dtls/lsp/types"
	"github.com/tidwall/jsonc"
	protocol "github.com/tliron/glsp/protocol_3_16"
	"gopkg.in/yaml.v3"
)

// References returns all references to a token
// For CSS files: returns nil (let css-ls handle it)
// For JSON/YAML files: finds all references to the token at cursor
func References(req *types.RequestContext, params *protocol.ReferenceParams) ([]protocol.Location, error) {
	uri := params.TextDocument.URI
	position := params.Position

	fmt.Fprintf(os.Stderr, "[DTLS] References requested: %s at line %d, char %d\n", uri, position.Line, position.Character)

	// Get document
	doc := req.Server.Document(uri)
	if doc == nil {
		return nil, nil
	}

	// Return nil for CSS files (let css-ls handle it)
	if doc.LanguageID() == "css" {
		return nil, nil
	}

	// For JSON/YAML files, find the token at cursor position
	tokenName := findTokenAtPosition(doc.Content(), position, doc.LanguageID())
	fmt.Fprintf(os.Stderr, "[DTLS] Token name at cursor: '%s'\n", tokenName)
	if tokenName == "" {
		return nil, nil
	}

	// Look up the token
	token := req.Server.Token(tokenName)
	fmt.Fprintf(os.Stderr, "[DTLS] Token lookup result: %v\n", token != nil)
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

	for _, document := range req.Server.AllDocuments() {
		docContent := document.Content()
		docURI := document.URI()

		if document.LanguageID() == "css" {
			// In CSS files, search for var(--token-name, or var(--token-name)
			// Find all occurrences of the CSS variable name
			ranges := findSubstringRanges(docContent, cssVarName)
			for _, r := range ranges {
				// Check character after the match - should be , or )
				// This excludes things like --token-color-red: and --token-color-reddish)
				endPos := r.End
				if endPos.Character < uint32(len(getLine(docContent, int(endPos.Line)))) {
					charAfter := getCharAt(docContent, endPos)
					if charAfter == ',' || charAfter == ')' {
						loc := protocol.Location{
							URI:   docURI,
							Range: r,
						}
						key := fmt.Sprintf("%s:%d:%d", docURI, r.Start.Line, r.Start.Character)
						locationMap[key] = loc
					}
				}
			}
		} else {
			// In JSON/YAML files, search for token references like {color.primary}
			if tokenReference != "" {
				ranges := findSubstringRanges(docContent, tokenReference)
				for _, r := range ranges {
					loc := protocol.Location{
						URI:   docURI,
						Range: r,
					}
					key := fmt.Sprintf("%s:%d:%d", docURI, r.Start.Line, r.Start.Character)
					locationMap[key] = loc
				}
			}
		}
	}

	// Convert map to slice
	for _, loc := range locationMap {
		locations = append(locations, loc)
	}

	// Include declaration if requested
	if params.Context.IncludeDeclaration && token.DefinitionURI != "" {
		// Find the range for this token in its definition file
		defDoc := req.Server.Document(token.DefinitionURI)
		if defDoc != nil {
			// Find the token definition in the document
			// For now, use a simple approach: find the token path
			defRange := findTokenDefinitionRange(defDoc.Content(), token.Path, defDoc.LanguageID())
			location := protocol.Location{
				URI:   token.DefinitionURI,
				Range: defRange,
			}
			locations = append(locations, location)
		}
	}

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
func findSubstringRanges(content string, substring string) []protocol.Range {
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
			offset = actualIdx + len(substring)
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
				return protocol.Range{
					Start: protocol.Position{
						Line:      uint32(lineNum),
						Character: uint32(idx),
					},
					End: protocol.Position{
						Line:      uint32(lineNum),
						Character: uint32(idx + len(lastKey)),
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
