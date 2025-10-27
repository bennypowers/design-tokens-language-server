package completion

import (
	"testing"

	"bennypowers.dev/dtls/internal/tokens"
	"bennypowers.dev/dtls/lsp/testutil"
	"bennypowers.dev/dtls/lsp/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)


func TestCompletion_CSSVariableCompletion(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}
		req := types.NewRequestContext(ctx, glspCtx)

	// Add some tokens
	ctx.TokenManager().Add(&tokens.Token{
		Name:        "color.primary",
		Value:       "#ff0000",
		Type:        "color",
		Description: "Primary brand color",
	})
	ctx.TokenManager().Add(&tokens.Token{
		Name:  "color.secondary",
		Value: "#00ff00",
		Type:  "color",
	})
	ctx.TokenManager().Add(&tokens.Token{
		Name:  "spacing.small",
		Value: "8px",
		Type:  "dimension",
	})

	uri := "file:///test.css"
	cssContent := `.button { color: --col }`
	ctx.DocumentManager().DidOpen(uri, "css", 1, cssContent)

	result, err := Completion(req, &protocol.CompletionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     protocol.Position{Line: 0, Character: 20}, // Inside "--col"
		},
	})

	require.NoError(t, err)
	require.NotNil(t, result)

	completionList, ok := result.(*protocol.CompletionList)
	require.True(t, ok)

	// Should return color tokens (filtered by "col" prefix)
	assert.GreaterOrEqual(t, len(completionList.Items), 2)

	// Check that items have correct structure
	for _, item := range completionList.Items {
		assert.NotNil(t, item.Kind)
		assert.Equal(t, protocol.CompletionItemKindVariable, *item.Kind)
		assert.NotNil(t, item.InsertTextFormat)
		assert.Equal(t, protocol.InsertTextFormatSnippet, *item.InsertTextFormat)
		assert.NotNil(t, item.InsertText)
		assert.Contains(t, *item.InsertText, "var(")
	}
}

func TestCompletion_AllTokens(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}
		req := types.NewRequestContext(ctx, glspCtx)

	ctx.TokenManager().Add(&tokens.Token{
		Name:  "color.primary",
		Value: "#ff0000",
	})
	ctx.TokenManager().Add(&tokens.Token{
		Name:  "spacing.small",
		Value: "8px",
	})

	uri := "file:///test.css"
	cssContent := `.button { color: -- }`
	ctx.DocumentManager().DidOpen(uri, "css", 1, cssContent)

	result, err := Completion(req, &protocol.CompletionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     protocol.Position{Line: 0, Character: 18}, // Inside "--"
		},
	})

	require.NoError(t, err)
	require.NotNil(t, result)

	completionList, ok := result.(*protocol.CompletionList)
	require.True(t, ok)

	// Should return all tokens when prefix is just "--"
	assert.Equal(t, 2, len(completionList.Items))
}

func TestCompletion_NonCSSDocument(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}
		req := types.NewRequestContext(ctx, glspCtx)

	uri := "file:///test.json"
	jsonContent := `{"color": {"$value": "#ff0000"}}`
	ctx.DocumentManager().DidOpen(uri, "json", 1, jsonContent)

	result, err := Completion(req, &protocol.CompletionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     protocol.Position{Line: 0, Character: 10},
		},
	})

	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestCompletion_DocumentNotFound(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}
		req := types.NewRequestContext(ctx, glspCtx)

	result, err := Completion(req, &protocol.CompletionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: "file:///nonexistent.css"},
			Position:     protocol.Position{Line: 0, Character: 10},
		},
	})

	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestCompletion_NoWordAtPosition(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}
		req := types.NewRequestContext(ctx, glspCtx)

	ctx.TokenManager().Add(&tokens.Token{
		Name:  "color.primary",
		Value: "#ff0000",
	})

	uri := "file:///test.css"
	cssContent := `.button {  }`
	ctx.DocumentManager().DidOpen(uri, "css", 1, cssContent)

	result, err := Completion(req, &protocol.CompletionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     protocol.Position{Line: 0, Character: 10},
		},
	})

	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestCompletionResolve_AddsDocumentation(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}
		req := types.NewRequestContext(ctx, glspCtx)

	ctx.TokenManager().Add(&tokens.Token{
		Name:        "color.primary",
		Value:       "#ff0000",
		Type:        "color",
		Description: "Primary brand color",
		FilePath:    "tokens.json",
	})

	item := &protocol.CompletionItem{
		Label: "--color-primary",
		Data: map[string]interface{}{
			"tokenName": "--color-primary",
		},
	}

	resolved, err := CompletionResolve(req, item)

	require.NoError(t, err)
	require.NotNil(t, resolved)

	// Check documentation was added
	doc, ok := resolved.Documentation.(protocol.MarkupContent)
	require.True(t, ok)
	assert.Equal(t, protocol.MarkupKindMarkdown, doc.Kind)
	assert.Contains(t, doc.Value, "--color-primary")
	assert.Contains(t, doc.Value, "#ff0000")
	assert.Contains(t, doc.Value, "color")
	assert.Contains(t, doc.Value, "Primary brand color")
	assert.Contains(t, doc.Value, "tokens.json")

	// Check detail was added
	assert.NotNil(t, resolved.Detail)
	assert.Contains(t, *resolved.Detail, "#ff0000")
}

func TestCompletionResolve_DeprecatedToken(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}
		req := types.NewRequestContext(ctx, glspCtx)

	ctx.TokenManager().Add(&tokens.Token{
		Name:               "color.old-primary",
		Value:              "#cc0000",
		Type:               "color",
		Deprecated:         true,
		DeprecationMessage: "Use color.primary instead",
	})

	item := &protocol.CompletionItem{
		Label: "--color-old-primary",
		Data: map[string]interface{}{
			"tokenName": "--color-old-primary",
		},
	}

	resolved, err := CompletionResolve(req, item)

	require.NoError(t, err)
	require.NotNil(t, resolved)

	doc, ok := resolved.Documentation.(protocol.MarkupContent)
	require.True(t, ok)
	assert.Contains(t, doc.Value, "DEPRECATED")
	assert.Contains(t, doc.Value, "Use color.primary instead")
}

func TestCompletionResolve_UnknownToken(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}
		req := types.NewRequestContext(ctx, glspCtx)

	item := &protocol.CompletionItem{
		Label: "--unknown-token",
		Data: map[string]interface{}{
			"tokenName": "--unknown-token",
		},
	}

	resolved, err := CompletionResolve(req, item)

	require.NoError(t, err)
	assert.Equal(t, item, resolved) // Should return item unchanged
}

func TestCompletionResolve_NoData(t *testing.T) {
	ctx := testutil.NewMockServerContext()
	glspCtx := &glsp.Context{}
		req := types.NewRequestContext(ctx, glspCtx)

	ctx.TokenManager().Add(&tokens.Token{
		Name:  "color.primary",
		Value: "#ff0000",
	})

	item := &protocol.CompletionItem{
		Label: "--color-primary",
		// No Data field - should fall back to Label
	}

	resolved, err := CompletionResolve(req, item)

	require.NoError(t, err)
	require.NotNil(t, resolved)

	// Should still resolve using Label
	doc, ok := resolved.Documentation.(protocol.MarkupContent)
	require.True(t, ok)
	assert.Contains(t, doc.Value, "--color-primary")
}

// TestGetWordAtPosition tests the getWordAtPosition helper function
func TestGetWordAtPosition(t *testing.T) {
	tests := []struct{
		name     string
		content  string
		position protocol.Position
		expected string
	}{
		{
			name:     "word at start of line",
			content:  "color-primary: #ff0000;",
			position: protocol.Position{Line: 0, Character: 5},
			expected: "color-primary",
		},
		{
			name:     "word in middle of line",
			content:  "  var(--color-primary, #ff0000);",
			position: protocol.Position{Line: 0, Character: 12},
			expected: "--color-primary",
		},
		{
			name:     "cursor at end of word",
			content:  "spacing-large",
			position: protocol.Position{Line: 0, Character: 13},
			expected: "spacing-large",
		},
		{
			name:     "cursor on whitespace before word",
			content:  "  color-primary",
			position: protocol.Position{Line: 0, Character: 1},
			expected: "", // cursor is on space, not touching the word
		},
		{
			name:     "cursor at start of word",
			content:  "color-primary",
			position: protocol.Position{Line: 0, Character: 0},
			expected: "color-primary",
		},
		{
			name:     "empty line",
			content:  "",
			position: protocol.Position{Line: 0, Character: 0},
			expected: "",
		},
		{
			name:     "position out of bounds",
			content:  "color",
			position: protocol.Position{Line: 5, Character: 0},
			expected: "",
		},
		{
			name:     "word with underscores",
			content:  "color_primary_500",
			position: protocol.Position{Line: 0, Character: 8},
			expected: "color_primary_500",
		},
		{
			name:     "word with numbers",
			content:  "spacing-16px",
			position: protocol.Position{Line: 0, Character: 10},
			expected: "spacing-16px",
		},
		{
			name:     "multiline content",
			content:  "line1\ncolor-primary\nline3",
			position: protocol.Position{Line: 1, Character: 5},
			expected: "color-primary",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getWordAtPosition(tt.content, tt.position)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestIsWordChar tests the isWordChar helper function
func TestIsWordChar(t *testing.T) {
	tests := []struct {
		name     string
		char     byte
		expected bool
	}{
		{"lowercase letter", 'a', true},
		{"uppercase letter", 'Z', true},
		{"digit", '5', true},
		{"hyphen", '-', true},
		{"underscore", '_', true},
		{"space", ' ', false},
		{"dot", '.', false},
		{"colon", ':', false},
		{"semicolon", ';', false},
		{"comma", ',', false},
		{"paren", '(', false},
		{"bracket", '{', false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isWordChar(tt.char)
			assert.Equal(t, tt.expected, result, "character: %c (%d)", tt.char, tt.char)
		})
	}
}

// TestIsInCompletionContext tests the isInCompletionContext helper function
func TestIsInCompletionContext(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		position protocol.Position
		expected bool
	}{
		{
			name: "inside CSS block",
			content: `.button {
  color: red;
}`,
			position: protocol.Position{Line: 1, Character: 5},
			expected: true,
		},
		{
			name: "outside CSS block - before opening brace",
			content: `.button {
  color: red;
}`,
			position: protocol.Position{Line: 0, Character: 5},
			expected: false,
		},
		{
			name: "outside CSS block - after closing brace",
			content: `.button {
  color: red;
}`,
			position: protocol.Position{Line: 2, Character: 2},
			expected: false,
		},
		{
			name: "nested blocks - inside inner block",
			content: `.outer {
  .inner {
    color: red;
  }
}`,
			position: protocol.Position{Line: 2, Character: 10},
			expected: true,
		},
		{
			name: "at opening brace",
			content: `.button {
  color: red;
}`,
			position: protocol.Position{Line: 0, Character: 8},
			expected: false, // At the brace itself, not inside yet
		},
		{
			name: "after opening brace",
			content: `.button {
  color: red;
}`,
			position: protocol.Position{Line: 0, Character: 9},
			expected: true, // Now inside the block
		},
		{
			name: "empty file",
			content:  "",
			position: protocol.Position{Line: 0, Character: 0},
			expected: false,
		},
		{
			name:     "single line with block",
			content:  `.button { color: red; }`,
			position: protocol.Position{Line: 0, Character: 15},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isInCompletionContext(tt.content, tt.position)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestNormalizeTokenName tests the normalizeTokenName helper function
func TestNormalizeTokenName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "CSS variable with dashes",
			input:    "--color-primary",
			expected: "colorprimary",
		},
		{
			name:     "token name without prefix",
			input:    "color-primary",
			expected: "colorprimary",
		},
		{
			name:     "uppercase token name",
			input:    "COLOR-PRIMARY",
			expected: "colorprimary",
		},
		{
			name:     "mixed case with dashes",
			input:    "--Color-Primary-500",
			expected: "colorprimary500",
		},
		{
			name:     "token with multiple hyphens",
			input:    "--spacing-large-xl",
			expected: "spacinglargexl",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "just dashes",
			input:    "--",
			expected: "",
		},
		{
			name:     "single word",
			input:    "primary",
			expected: "primary",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeTokenName(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
