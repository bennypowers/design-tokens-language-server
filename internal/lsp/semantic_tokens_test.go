package lsp

import (
	"testing"

	"github.com/bennypowers/design-tokens-language-server/internal/documents"
	"github.com/bennypowers/design-tokens-language-server/internal/tokens"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func TestGetSemanticTokensForDocument(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		langID   string
		tokens   map[string]*tokens.Token
		expected []SemanticTokenIntermediate
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
				"color.brand.primary": {
					Name:  "color.brand.primary",
					Value: "#FF6B35",
					Type:  "color",
				},
			},
			expected: []SemanticTokenIntermediate{
				{Line: 2, StartChar: 16, Length: 5, TokenType: 0, TokenModifiers: 0},  // "color"
				{Line: 2, StartChar: 22, Length: 5, TokenType: 1, TokenModifiers: 0},  // "brand"
				{Line: 2, StartChar: 28, Length: 7, TokenType: 1, TokenModifiers: 0},  // "primary"
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
				"color.brand.primary": {
					Name:  "color.brand.primary",
					Value: "#FF6B35",
					Type:  "color",
				},
				"color.ui.background": {
					Name:  "color.ui.background",
					Value: "#F7F7F7",
					Type:  "color",
				},
			},
			expected: []SemanticTokenIntermediate{
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
			expected: []SemanticTokenIntermediate{},
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
			expected: []SemanticTokenIntermediate{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a server with token manager
			s, err := NewServer()
			if err != nil {
				t.Fatalf("Failed to create server: %v", err)
			}

			// Add test tokens
			for _, token := range tt.tokens {
				if err := s.tokens.Add(token); err != nil {
					t.Fatalf("Failed to add token: %v", err)
				}
			}

			// Create a document
			doc := documents.NewDocument("file:///test.json", tt.langID, 1, tt.content)

			// Get semantic tokens
			result := s.getSemanticTokensForDocument(doc)

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
	s, err := NewServer()
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// Add a test token
	token := &tokens.Token{
		Name:  "color.brand.primary",
		Value: "#FF6B35",
		Type:  "color",
	}
	if err := s.tokens.Add(token); err != nil {
		t.Fatalf("Failed to add token: %v", err)
	}

	// Create document with references on different lines
	content := `{
  "line1": "{color.brand.primary}",
  "line2": "{color.brand.primary}"
}`
	doc := documents.NewDocument("file:///test.json", "json", 1, content)

	// Register the document
	s.documents.DidOpen(doc.URI(), doc.LanguageID(), doc.Version(), doc.Content())

	// Get intermediate tokens first
	intermediateTokens := s.getSemanticTokensForDocument(doc)

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
	result, err := s.handleSemanticTokensFull(nil, params)

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
