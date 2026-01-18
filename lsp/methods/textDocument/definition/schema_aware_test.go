package definition_test

import (
	"testing"

	"bennypowers.dev/dtls/internal/documents"
	"bennypowers.dev/dtls/internal/tokens"
	"bennypowers.dev/dtls/lsp/methods/textDocument/definition"
	"bennypowers.dev/dtls/lsp/testutil"
	"bennypowers.dev/dtls/lsp/types"
	protocol "github.com/tliron/glsp/protocol_3_16"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefinition_Draft_CurlyBraceReference(t *testing.T) {
	// Test go-to-definition for curly brace references in draft schema
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
	mockServer.AddDocument(doc)

	// Add the token with definition location
	mockServer.TokenManager().Add(&tokens.Token{
		Name:          "color-primary",
		Value:         "#FF0000",
		DefinitionURI: "file:///test.json",
		Line:          3,
		Character:     4,
		Path:          []string{"color", "primary"},
	})

	req := &types.RequestContext{
		Server: mockServer,
	}

	params := &protocol.DefinitionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{
				URI: "file:///test.json",
			},
			// Position in the middle of "{color.primary}" on line 9
			Position: protocol.Position{Line: 9, Character: 20},
		},
	}

	result, err := definition.Definition(req, params)
	require.NoError(t, err)

	// Should return location pointing to the definition of color.primary (line 3)
	locations, ok := result.([]protocol.Location)
	require.True(t, ok, "Result should be []protocol.Location")
	assert.NotEmpty(t, locations, "Should find definition location")

	if len(locations) > 0 {
		assert.Equal(t, "file:///test.json", string(locations[0].URI))
		assert.Equal(t, uint32(3), locations[0].Range.Start.Line, "Should point to line where 'primary' is defined")
	}
}

func TestDefinition_2025_JSONPointerReference(t *testing.T) {
	// Test go-to-definition for JSON Pointer references in 2025.10 schema
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
	mockServer.AddDocument(doc)

	// Add the token with definition location
	mockServer.TokenManager().Add(&tokens.Token{
		Name:          "color-primary",
		Value:         "srgb color",
		DefinitionURI: "file:///test.json",
		Line:          3,
		Character:     4,
		Path:          []string{"color", "primary"},
	})

	req := &types.RequestContext{
		Server: mockServer,
	}

	params := &protocol.DefinitionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{
				URI: "file:///test.json",
			},
			// Position in the JSON Pointer path on line 12
			Position: protocol.Position{Line: 12, Character: 20},
		},
	}

	result, err := definition.Definition(req, params)
	require.NoError(t, err)

	// Should return location pointing to the definition of color/primary (line 3)
	locations, ok := result.([]protocol.Location)
	require.True(t, ok, "Result should be []protocol.Location")
	assert.NotEmpty(t, locations, "Should find definition location for JSON Pointer")

	if len(locations) > 0 {
		assert.Equal(t, "file:///test.json", string(locations[0].URI))
		assert.Equal(t, uint32(3), locations[0].Range.Start.Line, "Should point to line where 'primary' is defined")
	}
}

func TestDefinition_NoReferenceCursor(t *testing.T) {
	// Test that definition returns nil when cursor is not on a reference
	content := `{
  "$schema": "https://www.designtokens.org/schemas/2025.10.json",
  "color": {
    "primary": {
      "$type": "color",
      "$value": {
        "colorSpace": "srgb",
        "components": [1.0, 0, 0]
      }
    }
  }
}`

	mockServer := testutil.NewMockServer()
	doc := documents.NewDocument("file:///test.json", "json", 1, content)
	mockServer.AddDocument(doc)

	req := &types.RequestContext{
		Server: mockServer,
	}

	params := &protocol.DefinitionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{
				URI: "file:///test.json",
			},
			// Position on "$type" keyword (not a reference)
			Position: protocol.Position{Line: 4, Character: 10},
		},
	}

	result, err := definition.Definition(req, params)
	require.NoError(t, err)

	// Should return nil when not on a reference
	assert.Nil(t, result)
}
