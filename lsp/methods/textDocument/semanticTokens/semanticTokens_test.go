package semantictokens_test

import (
	"testing"

	"bennypowers.dev/dtls/internal/documents"
	"bennypowers.dev/dtls/internal/tokens"
	"bennypowers.dev/dtls/lsp/methods/textDocument"
	semantictokens "bennypowers.dev/dtls/lsp/methods/textDocument/semanticTokens"
	"bennypowers.dev/dtls/lsp/testutil"
	"bennypowers.dev/dtls/lsp/types"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

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

func TestSemanticTokensFullDelta_ReturnsEmptyDeltaWhenUnchanged(t *testing.T) {
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

	// Create and register document
	content := `{"secondary": {"$value": "{color.brand.primary}"}}`
	uri := "file:///test.json"
	_ = s.DocumentManager().DidOpen(uri, "json", 1, content)

	// First: get full tokens
	req := types.NewRequestContext(s, nil)
	fullResult, err := semantictokens.SemanticTokensFull(req, &protocol.SemanticTokensParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: uri},
	})
	if err != nil {
		t.Fatalf("SemanticTokensFull failed: %v", err)
	}
	if fullResult.ResultID == nil || *fullResult.ResultID == "" {
		t.Fatal("Expected ResultID in full response")
	}

	// Second: request delta with same result ID (no changes)
	deltaResult, err := semantictokens.SemanticTokensFullDelta(req, &semantictokens.SemanticTokensDeltaParams{
		TextDocument:     protocol.TextDocumentIdentifier{URI: uri},
		PreviousResultID: *fullResult.ResultID,
	})
	if err != nil {
		t.Fatalf("SemanticTokensFullDelta failed: %v", err)
	}

	// Should return delta with no edits
	delta, ok := deltaResult.(*protocol.SemanticTokensDelta)
	if !ok {
		t.Fatalf("Expected SemanticTokensDelta, got %T", deltaResult)
	}
	if len(delta.Edits) != 0 {
		t.Errorf("Expected no edits, got %d", len(delta.Edits))
	}
}

func TestSemanticTokensFullDelta_ReturnsFullWhenPreviousResultIDNotFound(t *testing.T) {
	s := testutil.NewMockServerContext()

	// Add a test token that matches the reference
	token := &tokens.Token{
		Name:  "color-brand-primary",
		Value: "#FF6B35",
		Type:  "color",
	}
	if err := s.TokenManager().Add(token); err != nil {
		t.Fatalf("Failed to add token: %v", err)
	}

	// Create and register document with DTCG schema and valid reference
	content := `{
		"$schema": "https://json.schemastore.org/design-tokens.json",
		"secondary": {"$value": "{color.brand.primary}", "$type": "color"}
	}`
	uri := "file:///test.json"
	_ = s.DocumentManager().DidOpen(uri, "json", 1, content)

	// Request delta with non-existent result ID
	req := types.NewRequestContext(s, nil)
	result, err := semantictokens.SemanticTokensFullDelta(req, &semantictokens.SemanticTokensDeltaParams{
		TextDocument:     protocol.TextDocumentIdentifier{URI: uri},
		PreviousResultID: "non-existent-result-id",
	})
	if err != nil {
		t.Fatalf("SemanticTokensFullDelta failed: %v", err)
	}

	// Should return full tokens
	fullResponse, ok := result.(*protocol.SemanticTokens)
	if !ok {
		t.Fatalf("Expected SemanticTokens (full response), got %T", result)
	}
	if fullResponse.ResultID == nil || *fullResponse.ResultID == "" {
		t.Error("Expected ResultID in full response")
	}
	// Note: Data may still be empty if the token reference doesn't resolve (schema mismatch, etc.)
	// The important check is that we got a full response, not a delta
}

func TestSemanticTokensFullDelta_DocumentNotFound(t *testing.T) {
	s := testutil.NewMockServerContext()

	req := types.NewRequestContext(s, nil)
	_, err := semantictokens.SemanticTokensFullDelta(req, &semantictokens.SemanticTokensDeltaParams{
		TextDocument:     protocol.TextDocumentIdentifier{URI: "file:///non-existent.json"},
		PreviousResultID: "some-id",
	})

	if err == nil {
		t.Error("Expected error for non-existent document")
	}
}

func TestSemanticTokensFullDelta_NonTokenFile(t *testing.T) {
	s := testutil.NewMockServerContext()
	// Configure to reject this file
	s.ShouldProcessAsTokenFileFunc = func(uri string) bool { return false }

	// Create and register a non-token file
	uri := "file:///test.css"
	_ = s.DocumentManager().DidOpen(uri, "css", 1, ".foo { color: red; }")

	req := types.NewRequestContext(s, nil)
	result, err := semantictokens.SemanticTokensFullDelta(req, &semantictokens.SemanticTokensDeltaParams{
		TextDocument:     protocol.TextDocumentIdentifier{URI: uri},
		PreviousResultID: "some-id",
	})

	// Should return nil for non-token files
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if result != nil {
		t.Errorf("Expected nil result for non-token file, got %v", result)
	}
}

func TestSemanticTokensFull_ReturnsResultID(t *testing.T) {
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

	// Create and register document
	content := `{"secondary": {"$value": "{color.brand.primary}"}}`
	uri := "file:///test.json"
	_ = s.DocumentManager().DidOpen(uri, "json", 1, content)

	req := types.NewRequestContext(s, nil)
	result, err := semantictokens.SemanticTokensFull(req, &protocol.SemanticTokensParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: uri},
	})

	if err != nil {
		t.Fatalf("SemanticTokensFull failed: %v", err)
	}
	if result == nil {
		t.Fatal("Expected non-nil result")
	}
	if result.ResultID == nil {
		t.Error("Expected ResultID to be set")
	}
	if *result.ResultID == "" {
		t.Error("Expected ResultID to be non-empty")
	}
}
