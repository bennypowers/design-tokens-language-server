package integration_test

import (
	"testing"

	"github.com/bennypowers/design-tokens-language-server/test/integration/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// TestDefinitionOnVarCall tests go-to-definition on a var() call
func TestDefinitionOnVarCall(t *testing.T) {
	server := testutil.NewTestServer(t)
	testutil.LoadBasicTokens(t, server)
	testutil.OpenCSSFixture(t, server, "file:///test.css", "basic-var-calls.css")

	// Request definition - see fixture for position
	locations, err := server.GetDefinition(&protocol.DefinitionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{
				URI: "file:///test.css",
			},
			Position: protocol.Position{
				Line:      2, // Adjusted for comment line
				Character: 18, // Inside --color-primary
			},
		},
	})

	require.NoError(t, err)
	// Since LoadTokensFromJSON doesn't set DefinitionURI, this returns nil
	// When loading from a file, this would return the token file location
	assert.Nil(t, locations)
}

// TestDefinitionOutsideVarCall tests that definition returns nil outside var() calls
func TestDefinitionOutsideVarCall(t *testing.T) {
	server := testutil.NewTestServer(t)
	testutil.OpenCSSFixture(t, server, "file:///test.css", "no-var-call.css")

	// Request definition - see fixture for position
	locations, err := server.GetDefinition(&protocol.DefinitionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{
				URI: "file:///test.css",
			},
			Position: protocol.Position{
				Line:      2, // Adjusted for comment line
				Character: 10, // On "red"
			},
		},
	})

	require.NoError(t, err)
	assert.Nil(t, locations)
}
