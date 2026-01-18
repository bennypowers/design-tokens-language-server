package completion

import (
	"bennypowers.dev/dtls/internal/log"
	"bytes"
	"fmt"
	"strings"
	"text/template"

	"bennypowers.dev/dtls/internal/position"
	"bennypowers.dev/dtls/internal/tokens"
	"bennypowers.dev/dtls/lsp/types"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// Template for token documentation
// Note: {{.CSSVariableName}} calls the Token.CSSVariableName() method (not a field)
var tokenDocTemplate = template.Must(template.New("tokenDoc").Parse(`# {{.CSSVariableName}}
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

// renderTokenDoc renders the documentation markdown for a token
func renderTokenDoc(token *tokens.Token) (string, error) {
	var buf bytes.Buffer
	if err := tokenDocTemplate.Execute(&buf, token); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// handleCompletion handles the textDocument/completion request

// Completion handles the textDocument/completion request
func Completion(req *types.RequestContext, params *protocol.CompletionParams) (any, error) {
	uri := params.TextDocument.URI
	pos := params.Position

	log.Info("Completion requested: %s at line %d, char %d", uri, pos.Line, pos.Character)

	// Get document
	doc := req.Server.Document(uri)
	if doc == nil {
		return nil, nil
	}

	// Only process CSS files
	if doc.LanguageID() != "css" {
		return nil, nil
	}

	// Get the word at the cursor position
	word := getWordAtPosition(doc.Content(), pos)
	if word == "" {
		return nil, nil
	}

	log.Info("Completion word: '%s'", word)

	// Check if we're in a valid completion context (inside a block or property value)
	if !isInCompletionContext(doc.Content(), pos) {
		return nil, nil
	}

	// Filter tokens by the current word
	var items []protocol.CompletionItem
	normalizedWord := normalizeTokenName(word)

	for _, token := range req.Server.TokenManager().GetAll() {
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
				Data: map[string]any{
					"tokenName": cssVar,
				},
			}

			items = append(items, item)
		}
	}

	log.Info("Returning %d completion items", len(items))

	return &protocol.CompletionList{
		IsIncomplete: false,
		Items:        items,
	}, nil
}

// handleCompletionResolve handles the completionItem/resolve request

// CompletionResolve resolves a completion item with additional details
func CompletionResolve(req *types.RequestContext, item *protocol.CompletionItem) (*protocol.CompletionItem, error) {
	// Get token name from data
	var tokenName string
	if item.Data != nil {
		if data, ok := item.Data.(map[string]any); ok {
			if name, ok := data["tokenName"].(string); ok {
				tokenName = name
			}
		}
	}

	if tokenName == "" {
		tokenName = item.Label
	}

	// Look up the token
	token := req.Server.Token(tokenName)
	if token == nil {
		return item, nil
	}

	// Render documentation using template
	documentation, err := renderTokenDoc(token)
	if err != nil {
		log.Info("Failed to render token documentation: %v", err)
		return item, nil
	}

	// Add documentation
	item.Documentation = protocol.MarkupContent{
		Kind:  protocol.MarkupKindMarkdown,
		Value: documentation,
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

// isInCompletionContext checks if the position is in a valid completion req.GLSP.
// Completions are valid inside CSS blocks (between { and }) where var() calls can be used.
// This implementation counts braces up to the cursor position to determine if we're inside a block.
func isInCompletionContext(content string, pos protocol.Position) bool {
	lines := strings.Split(content, "\n")
	if int(pos.Line) >= len(lines) {
		return false
	}

	// Get all content up to and including the cursor position
	var textUpToCursor strings.Builder
	for i := 0; i <= int(pos.Line); i++ {
		if i < int(pos.Line) {
			textUpToCursor.WriteString(lines[i])
			textUpToCursor.WriteString("\n")
		} else {
			// For the cursor line, only include text up to the cursor position
			line := lines[i]
			// Convert UTF-16 character offset to byte offset
			utf16Col := int(pos.Character)
			byteOffset := position.UTF16ToByteOffset(line, utf16Col)
			if byteOffset > len(line) {
				byteOffset = len(line)
			}
			textUpToCursor.WriteString(line[:byteOffset])
		}
	}

	// Count opening and closing braces
	// If we have more opening braces than closing braces, we're inside a block
	openBraces := 0
	closeBraces := 0
	text := textUpToCursor.String()

	// Simple character-by-character scan
	// Note: This doesn't handle strings or comments, but it's good enough
	// for most cases. A more sophisticated implementation would skip
	// content inside strings and comments.
	for _, ch := range text {
		switch ch {
		case '{':
			openBraces++
		case '}':
			closeBraces++
		}
	}

	// We're inside a block if we have unclosed braces
	return openBraces > closeBraces
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
