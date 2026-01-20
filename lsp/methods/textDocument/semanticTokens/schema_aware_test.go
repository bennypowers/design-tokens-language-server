package semantictokens_test

import (
	"testing"

	"bennypowers.dev/dtls/internal/documents"
	"bennypowers.dev/dtls/internal/tokens"
	semantictokens "bennypowers.dev/dtls/lsp/methods/textDocument/semanticTokens"
	"bennypowers.dev/dtls/lsp/testutil"
	"github.com/stretchr/testify/assert"
)

func TestSemanticTokens_Draft_CurlyBraceReferences(t *testing.T) {
	// Test that curly brace references are highlighted in draft schema
	content := `{
  "$schema": "https://www.designtokens.org/schemas/draft.json",
  "color": {
    "primary": {
      "$type": "color",
      "$value": "#FF0000"
    },
    "secondary": {
      "$type": "color",
      "$value": "{color.primary}"
    }
  }
}`

	mockServer := testutil.NewMockServer()
	doc := documents.NewDocument("file:///test.json", "json", 1, content)

	// Add the token that the reference points to
	mockServer.TokenManager().Add(&tokens.Token{
		Name:  "color-primary",
		Value: "#FF0000",
	})

	semanticTokens := semantictokens.GetSemanticTokensForDocument(mockServer, doc)

	// Should find tokens for "color.primary" reference on line 9
	assert.NotEmpty(t, semanticTokens, "Should find semantic tokens for curly brace reference")

	// Verify we have tokens for both parts: "color" and "primary"
	foundColorPart := false
	foundPrimaryPart := false
	for _, token := range semanticTokens {
		if token.Line == 9 {
			if token.TokenType == semantictokens.TokenTypeVariable {
				foundColorPart = true
			}
			if token.TokenType == semantictokens.TokenTypeProperty {
				foundPrimaryPart = true
			}
		}
	}

	assert.True(t, foundColorPart, "Should highlight 'color' part of reference")
	assert.True(t, foundPrimaryPart, "Should highlight 'primary' part of reference")
}

func TestSemanticTokens_2025_JSONPointerReferences(t *testing.T) {
	// Test that $ref JSON Pointer references are highlighted in 2025.10 schema
	content := `{
  "$schema": "https://www.designtokens.org/schemas/2025.10.json",
  "color": {
    "primary": {
      "$type": "color",
      "$value": {
        "colorSpace": "srgb",
        "components": [1.0, 0, 0]
      }
    },
    "secondary": {
      "$type": "color",
      "$ref": "#/color/primary"
    }
  }
}`

	mockServer := testutil.NewMockServer()
	doc := documents.NewDocument("file:///test.json", "json", 1, content)

	semTokens := semantictokens.GetSemanticTokensForDocument(mockServer, doc)

	// Should find tokens for "$ref" keyword and JSON Pointer path
	foundRefKeyword := false
	foundPointerPath := false

	for _, token := range semTokens {
		if token.Line == 12 {
			// $ref should be highlighted as keyword
			if token.TokenType == semantictokens.TokenTypeKeyword {
				foundRefKeyword = true
			}
			// JSON Pointer path should be highlighted
			if token.TokenType == semantictokens.TokenTypeString {
				foundPointerPath = true
			}
		}
	}

	assert.True(t, foundRefKeyword, "Should highlight '$ref' keyword")
	assert.True(t, foundPointerPath, "Should highlight JSON Pointer path")
}

func TestSemanticTokens_2025_RootKeyword(t *testing.T) {
	// Test that $root is highlighted as a special keyword in 2025.10 schema
	content := `{
  "$schema": "https://www.designtokens.org/schemas/2025.10.json",
  "spacing": {
    "$root": {
      "$type": "dimension",
      "$value": "16px"
    }
  }
}`

	mockServer := testutil.NewMockServer()
	doc := documents.NewDocument("file:///test.json", "json", 1, content)

	semTokens := semantictokens.GetSemanticTokensForDocument(mockServer, doc)

	// Should find token for "$root" keyword on line 3
	foundRootKeyword := false
	for _, token := range semTokens {
		if token.Line == 3 && token.TokenType == semantictokens.TokenTypeKeyword {
			foundRootKeyword = true
		}
	}

	assert.True(t, foundRootKeyword, "Should highlight '$root' as keyword")
}

func TestSemanticTokens_Draft_NoJSONPointerHighlighting(t *testing.T) {
	// Test that JSON Pointers are NOT highlighted in draft schema
	// (they shouldn't exist, but if they do, we ignore them)
	content := `{
  "$schema": "https://www.designtokens.org/schemas/draft.json",
  "color": {
    "primary": {
      "$type": "color",
      "$ref": "#/some/path"
    }
  }
}`

	mockServer := testutil.NewMockServer()
	doc := documents.NewDocument("file:///test.json", "json", 1, content)

	semTokens := semantictokens.GetSemanticTokensForDocument(mockServer, doc)

	// Should NOT highlight $ref in draft schema
	for _, token := range semTokens {
		if token.Line == 5 {
			assert.NotEqual(t, semantictokens.TokenTypeKeyword, token.TokenType, "Should not highlight $ref as keyword in draft schema")
		}
	}
}
