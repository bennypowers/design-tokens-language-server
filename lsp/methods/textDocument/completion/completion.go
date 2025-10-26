package completion

import (
	"fmt"
	"os"
	"strings"

	"bennypowers.dev/dtls/internal/parser/css"
	"bennypowers.dev/dtls/internal/position"
	"bennypowers.dev/dtls/lsp/types"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// handleCompletion handles the textDocument/completion request

// Completion handles the textDocument/completion request
func Completion(ctx types.ServerContext, context *glsp.Context, params *protocol.CompletionParams) (any, error) {
	uri := params.TextDocument.URI
	position := params.Position

	fmt.Fprintf(os.Stderr, "[DTLS] Completion requested: %s at line %d, char %d\n", uri, position.Line, position.Character)

	// Get document
	doc := ctx.Document(uri)
	if doc == nil {
		return nil, nil
	}

	// Only process CSS files
	if doc.LanguageID() != "css" {
		return nil, nil
	}

	// Parse CSS to find context
	parser := css.AcquireParser()
	defer css.ReleaseParser(parser)
	result, err := parser.Parse(doc.Content())
	if err != nil {
		return nil, nil
	}

	// Get the word at the cursor position
	word := getWordAtPosition(doc.Content(), position)
	if word == "" {
		return nil, nil
	}

	fmt.Fprintf(os.Stderr, "[DTLS] Completion word: '%s'\n", word)

	// Check if we're in a valid completion context (inside a block or property value)
	if !isInCompletionContext(result, position) {
		return nil, nil
	}

	// Filter tokens by the current word
	var items []protocol.CompletionItem
	normalizedWord := normalizeTokenName(word)

	for _, token := range ctx.TokenManager().GetAll() {
		cssVar := token.CSSVariableName()
		normalizedLabel := normalizeTokenName(cssVar)

		// Check if the token matches the current word
		if strings.HasPrefix(normalizedLabel, normalizedWord) {
			insertTextFormat := protocol.InsertTextFormatSnippet
			kind := protocol.CompletionItemKindVariable
			insertText := fmt.Sprintf("var(%s${1:, %s})$0", cssVar, token.Value)
			item := protocol.CompletionItem{
				Label:            cssVar,
				Kind:             &kind,
				InsertTextFormat: &insertTextFormat,
				InsertText:       &insertText,
				Data: map[string]interface{}{
					"tokenName": cssVar,
				},
			}

			items = append(items, item)
		}
	}

	fmt.Fprintf(os.Stderr, "[DTLS] Returning %d completion items\n", len(items))

	return &protocol.CompletionList{
		IsIncomplete: false,
		Items:        items,
	}, nil
}

// handleCompletionResolve handles the completionItem/resolve request

// CompletionResolve resolves a completion item with additional details
func CompletionResolve(ctx types.ServerContext, context *glsp.Context, item *protocol.CompletionItem) (*protocol.CompletionItem, error) {
	// Get token name from data
	var tokenName string
	if item.Data != nil {
		if data, ok := item.Data.(map[string]interface{}); ok {
			if name, ok := data["tokenName"].(string); ok {
				tokenName = name
			}
		}
	}

	if tokenName == "" {
		tokenName = item.Label
	}

	// Look up the token
	token := ctx.Token(tokenName)
	if token == nil {
		return item, nil
	}

	// Build documentation
	var doc strings.Builder
	doc.WriteString(fmt.Sprintf("# %s\n\n", token.CSSVariableName()))

	if token.Description != "" {
		doc.WriteString(fmt.Sprintf("%s\n\n", token.Description))
	}

	doc.WriteString(fmt.Sprintf("**Value**: `%s`\n", token.Value))

	if token.Type != "" {
		doc.WriteString(fmt.Sprintf("**Type**: `%s`\n", token.Type))
	}

	if token.Deprecated {
		doc.WriteString("\n⚠️ **DEPRECATED**")
		if token.DeprecationMessage != "" {
			doc.WriteString(fmt.Sprintf(": %s", token.DeprecationMessage))
		}
		doc.WriteString("\n")
	}

	if token.FilePath != "" {
		doc.WriteString(fmt.Sprintf("\n*Defined in: %s*\n", token.FilePath))
	}

	// Add documentation
	item.Documentation = protocol.MarkupContent{
		Kind:  protocol.MarkupKindMarkdown,
		Value: doc.String(),
	}

	// Add detail (value preview)
	detail := fmt.Sprintf(": %s", token.Value)
	item.Detail = &detail

	return item, nil
}

// getWordAtPosition extracts the word at the given position.
// LSP positions use UTF-16 code units, so this function converts them to byte offsets.
func getWordAtPosition(content string, pos protocol.Position) string {
	lines := strings.Split(content, "\n")
	if int(pos.Line) >= len(lines) {
		return ""
	}

	line := lines[pos.Line]

	// Convert UTF-16 column to byte offset
	utf16Col := int(pos.Character)
	byteOffset := position.UTF16ToByteOffset(line, utf16Col)

	// Bounds check
	if byteOffset > len(line) {
		return ""
	}

	// Find the start of the word (going backwards from cursor)
	start := byteOffset
	for start > 0 && isWordChar(line[start-1]) {
		start--
	}

	// Find the end of the word (going forwards from cursor)
	end := byteOffset
	for end < len(line) && isWordChar(line[end]) {
		end++
	}

	return line[start:end]
}

// isWordChar checks if a character is part of a CSS identifier
func isWordChar(c byte) bool {
	return (c >= 'a' && c <= 'z') ||
		(c >= 'A' && c <= 'Z') ||
		(c >= '0' && c <= '9') ||
		c == '-' || c == '_'
}

// isInCompletionContext checks if the position is in a valid completion context
func isInCompletionContext(result *css.ParseResult, pos protocol.Position) bool {
	// For now, we'll accept completions anywhere in CSS
	// In the future, we can be more specific about only completing inside blocks
	return true
}

// normalizeTokenName normalizes a token name for comparison
func normalizeTokenName(name string) string {
	// Remove leading dashes and convert to lowercase
	name = strings.TrimPrefix(name, "--")
	name = strings.ToLower(name)
	// Remove all hyphens for fuzzy matching
	name = strings.ReplaceAll(name, "-", "")
	return name
}
