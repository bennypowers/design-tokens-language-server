package semantictokens_test

import (
	"os"
	"path/filepath"
	"testing"

	"bennypowers.dev/dtls/internal/documents"
	"bennypowers.dev/dtls/internal/tokens"
	"bennypowers.dev/dtls/lsp/methods/textDocument"
	semantictokens "bennypowers.dev/dtls/lsp/methods/textDocument/semanticTokens"
	"bennypowers.dev/dtls/lsp/testutil"
	"bennypowers.dev/dtls/lsp/types"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// loadFixture loads a test fixture file from test/fixtures/
func loadFixture(t *testing.T, path string) string {
	t.Helper()
	// Get the project root (go up from lsp/methods/textDocument/semanticTokens/)
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	projectRoot := filepath.Join(wd, "..", "..", "..", "..")
	fixturePath := filepath.Join(projectRoot, "test", "fixtures", path)
	content, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("Failed to load fixture %s: %v", path, err)
	}
	return string(content)
}

func TestGetSemanticTokensForDocument(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		langID   string
		tokens   map[string]*tokens.Token
		expected []semantictokens.SemanticTokenIntermediate
	}{
		{
			name:   "JSON with single reference",
			langID: "json",
			content: `{
  "secondary": {
    "$value": "{color.brand.primary}",
    "$type": "color"
  }
}`,
			tokens: map[string]*tokens.Token{
				"color-brand-primary": {
					Name:  "color-brand-primary",
					Value: "#FF6B35",
					Type:  "color",
				},
			},
			expected: []semantictokens.SemanticTokenIntermediate{
				{Line: 2, StartChar: 16, Length: 5, TokenType: 0, TokenModifiers: 0}, // "color"
				{Line: 2, StartChar: 22, Length: 5, TokenType: 1, TokenModifiers: 0}, // "brand"
				{Line: 2, StartChar: 28, Length: 7, TokenType: 1, TokenModifiers: 0}, // "primary"
			},
		},
		{
			name:   "JSON with multiple references",
			langID: "json",
			content: `{
  "secondary": {
    "$value": "{color.brand.primary}",
    "$type": "color"
  },
  "background": {
    "$value": "{color.ui.background}",
    "$type": "color"
  }
}`,
			tokens: map[string]*tokens.Token{
				"color-brand-primary": {
					Name:  "color-brand-primary",
					Value: "#FF6B35",
					Type:  "color",
				},
				"color-ui-background": {
					Name:  "color-ui-background",
					Value: "#F7F7F7",
					Type:  "color",
				},
			},
			expected: []semantictokens.SemanticTokenIntermediate{
				{Line: 2, StartChar: 16, Length: 5, TokenType: 0, TokenModifiers: 0},  // "color"
				{Line: 2, StartChar: 22, Length: 5, TokenType: 1, TokenModifiers: 0},  // "brand"
				{Line: 2, StartChar: 28, Length: 7, TokenType: 1, TokenModifiers: 0},  // "primary"
				{Line: 6, StartChar: 16, Length: 5, TokenType: 0, TokenModifiers: 0},  // "color"
				{Line: 6, StartChar: 22, Length: 2, TokenType: 1, TokenModifiers: 0},  // "ui"
				{Line: 6, StartChar: 25, Length: 10, TokenType: 1, TokenModifiers: 0}, // "background"
			},
		},
		{
			name:   "JSON with non-existent reference - should be skipped",
			langID: "json",
			content: `{
  "secondary": {
    "$value": "{color.nonexistent}",
    "$type": "color"
  }
}`,
			tokens:   map[string]*tokens.Token{},
			expected: []semantictokens.SemanticTokenIntermediate{},
		},
		{
			name:   "JSON without references",
			langID: "json",
			content: `{
  "primary": {
    "$value": "#FF6B35",
    "$type": "color"
  }
}`,
			tokens:   map[string]*tokens.Token{},
			expected: []semantictokens.SemanticTokenIntermediate{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a server with token manager
			s := testutil.NewMockServerContext()

			// Add test tokens
			for _, token := range tt.tokens {
				if err := s.TokenManager().Add(token); err != nil {
					t.Fatalf("Failed to add token: %v", err)
				}
			}

			// Create a document
			doc := documents.NewDocument("file:///test.json", tt.langID, 1, tt.content)

			// Get semantic tokens
			result := semantictokens.GetSemanticTokensForDocument(s, doc)

			// Check result count
			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d tokens, got %d", len(tt.expected), len(result))
				t.Logf("Result: %+v", result)
				return
			}

			// Check each token
			for i, expected := range tt.expected {
				if i >= len(result) {
					break
				}
				actual := result[i]
				if actual.Line != expected.Line {
					t.Errorf("Token %d: Expected line %d, got %d", i, expected.Line, actual.Line)
				}
				if actual.StartChar != expected.StartChar {
					t.Errorf("Token %d: Expected startChar %d, got %d", i, expected.StartChar, actual.StartChar)
				}
				if actual.Length != expected.Length {
					t.Errorf("Token %d: Expected length %d, got %d", i, expected.Length, actual.Length)
				}
				if actual.TokenType != expected.TokenType {
					t.Errorf("Token %d: Expected tokenType %d, got %d", i, expected.TokenType, actual.TokenType)
				}
				if actual.TokenModifiers != expected.TokenModifiers {
					t.Errorf("Token %d: Expected tokenModifiers %d, got %d", i, expected.TokenModifiers, actual.TokenModifiers)
				}
			}
		})
	}
}

func TestSemanticTokensDeltaEncoding(t *testing.T) {
	// Test that the delta encoding is correct
	s := testutil.NewMockServerContext()

	// Add a test token
	token := &tokens.Token{
		Name:  "color-brand-primary",
		Value: "#FF6B35",
		Type:  "color",
	}
	if err := s.TokenManager().Add(token); err != nil {
		t.Fatalf("Failed to add token: %v", err)
	}

	// Create document with references on different lines
	content := `{
  "line1": "{color.brand.primary}",
  "line2": "{color.brand.primary}"
}`
	doc := documents.NewDocument("file:///test.json", "json", 1, content)

	// Register the document
	req := types.NewRequestContext(s, nil)
	_ = textDocument.DidOpen(req, &protocol.DidOpenTextDocumentParams{TextDocument: protocol.TextDocumentItem{URI: doc.URI(), LanguageID: doc.LanguageID(), Version: protocol.Integer(doc.Version()), Text: doc.Content()}})

	// Get intermediate tokens first
	intermediateTokens := semantictokens.GetSemanticTokensForDocument(s, doc)

	// Expected intermediate tokens
	// Line 1: "color" at char 13, "brand" at char 19, "primary" at char 25
	// Line 2: "color" at char 13, "brand" at char 19, "primary" at char 25
	if len(intermediateTokens) != 6 {
		t.Fatalf("Expected 6 intermediate tokens, got %d", len(intermediateTokens))
	}

	// Now test the full handler with delta encoding
	params := &protocol.SemanticTokensParams{
		TextDocument: protocol.TextDocumentIdentifier{
			URI: doc.URI(),
		},
	}
	req = types.NewRequestContext(s, nil)
	result, err := semantictokens.SemanticTokensFull(req, params)

	if err != nil {
		t.Fatalf("handleSemanticTokensFull failed: %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	// Delta encoding should produce 5 values per token (deltaLine, deltaStart, length, tokenType, tokenModifiers)
	expectedDataLength := 6 * 5
	if len(result.Data) != expectedDataLength {
		t.Errorf("Expected %d data values, got %d", expectedDataLength, len(result.Data))
	}

	// First token (line 1, "color"): deltaLine=1, deltaStart=13
	if result.Data[0] != 1 {
		t.Errorf("First token deltaLine: expected 1, got %d", result.Data[0])
	}
	if result.Data[1] != 13 {
		t.Errorf("First token deltaStart: expected 13, got %d", result.Data[1])
	}

	// Second token (line 1, "brand"): deltaLine=0, deltaStart=6 (19-13)
	if result.Data[5] != 0 {
		t.Errorf("Second token deltaLine: expected 0, got %d", result.Data[5])
	}
	if result.Data[6] != 6 {
		t.Errorf("Second token deltaStart: expected 6, got %d", result.Data[6])
	}
}

func TestSemanticTokensRange(t *testing.T) {
	s := testutil.NewMockServerContext()

	// Add test tokens
	token1 := &tokens.Token{
		Name:  "color-brand-primary",
		Value: "#FF6B35",
		Type:  "color",
	}
	token2 := &tokens.Token{
		Name:  "color-ui-background",
		Value: "#F7F7F7",
		Type:  "color",
	}
	_ = s.TokenManager().Add(token1)
	_ = s.TokenManager().Add(token2)

	// Load fixture document
	content := loadFixture(t, "semantic-tokens/range-test.json")
	doc := documents.NewDocument("file:///test.json", "json", 1, content)
	req := types.NewRequestContext(s, nil)
	_ = textDocument.DidOpen(req, &protocol.DidOpenTextDocumentParams{TextDocument: protocol.TextDocumentItem{URI: doc.URI(), LanguageID: doc.LanguageID(), Version: protocol.Integer(doc.Version()), Text: doc.Content()}})

	// Request semantic tokens for range (lines 1-2 only)
	params := &protocol.SemanticTokensRangeParams{
		TextDocument: protocol.TextDocumentIdentifier{
			URI: doc.URI(),
		},
		Range: protocol.Range{
			Start: protocol.Position{Line: 1, Character: 0},
			End:   protocol.Position{Line: 2, Character: 100},
		},
	}

	req = types.NewRequestContext(s, nil)
	result, err := semantictokens.SemanticTokensRange(req, params)
	if err != nil {
		t.Fatalf("handleSemanticTokensRange failed: %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	// Should only include tokens from lines 1-2 (6 tokens total: 3 per reference)
	// Line 1: color, brand, primary
	// Line 2: color, ui, background
	expectedDataLength := 6 * 5 // 6 tokens × 5 values each
	if len(result.Data) != expectedDataLength {
		t.Errorf("Expected %d data values for range, got %d", expectedDataLength, len(result.Data))
	}
}

